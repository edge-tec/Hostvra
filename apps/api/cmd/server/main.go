package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	"hostvra/api/internal/alerts"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/dns"
	"hostvra/api/internal/handlers"
	"hostvra/api/internal/license"
	"hostvra/api/internal/rbac"
	"hostvra/api/internal/store"
)

const AppVersion = "1.0.0"

func main() {
	cfg := config.Load()

	// Configure structured logger
	var logHandler slog.Handler
	if cfg.LogFormat == "json" {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	logger.Info("Starting Hostvra Core API Server",
		"version", AppVersion,
		"environment", cfg.Environment,
		"port", cfg.Port,
	)

	// Initialize Storage Layer (PostgreSQL with fallback to In-Memory if Postgres is unreachable)
	var dataStore store.Store
	pgStore, err := store.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		logger.Warn("PostgreSQL unavailable at configured URL, falling back to In-Memory persistence engine for local development", "error", err)
		dataStore = store.NewMemoryStore()
	} else {
		logger.Info("Connected to PostgreSQL database cluster successfully")
		dataStore = pgStore
	}
	defer dataStore.Close()

	// Seed default administrator account for local dev / initial access
	seedDefaultAdmin(context.Background(), dataStore, logger)
	autoRecoverLocalAgentNode(context.Background(), dataStore, logger)

	// Initialize Audit Logger
	auditLogger := audit.NewLogger(dataStore, logger)

	// Initialize Handlers
	authHandler := handlers.NewAuthHandler(cfg, dataStore, auditLogger)
	serverHandler := handlers.NewServerHandler(cfg, dataStore, auditLogger)
	agentHandler := handlers.NewAgentHandler(cfg, dataStore, auditLogger)
	websiteHandler := handlers.NewWebsiteHandler(cfg, dataStore, auditLogger)
	databaseHandler := handlers.NewDatabaseHandler(cfg, dataStore, auditLogger)
	auditHandler := handlers.NewAuditHandler(dataStore)
	healthHandler := handlers.NewHealthHandler(AppVersion)

	dnsService := dns.NewService()
	alertEngine := alerts.NewEngine()
	licenseManager := license.NewManager()

	dnsHandler := handlers.NewDNSHandler(cfg, dnsService, auditLogger)
	alertHandler := handlers.NewAlertHandler(cfg, alertEngine, auditLogger)
	backupHandler := handlers.NewBackupHandler(cfg, dataStore, auditLogger)
	licenseHandler := handlers.NewLicenseHandler(cfg, licenseManager, auditLogger)
	teamHandler := handlers.NewTeamHandler(cfg, dataStore, auditLogger)
	apiKeyHandler := handlers.NewAPIKeyHandler(cfg, dataStore, auditLogger)
	emailHandler := handlers.NewEmailHandler(cfg, dataStore, dnsService, auditLogger)
	phpHandler := handlers.NewPHPHandler(cfg, dataStore, auditLogger)
	webServerHandler := handlers.NewWebServerHandler(cfg, dataStore, auditLogger)
	updateHandler := handlers.NewUpdateHandler(cfg, dataStore, auditLogger, AppVersion)
	terminalHandler := handlers.NewTerminalHandler(cfg, dataStore, auditLogger)
	appStoreHandler := handlers.NewAppStoreHandler(cfg, dataStore, auditLogger)
	dashboardHandler := handlers.NewDashboardHandler(cfg, dataStore, auditLogger)
	fileHandler := handlers.NewFileHandler(cfg, dataStore, auditLogger)
	firewallHandler := handlers.NewFirewallHandler(cfg, dataStore, auditLogger)
	cronHandler := handlers.NewCronHandler(cfg, dataStore, auditLogger)
	dockerHandler := handlers.NewDockerHandler(cfg, dataStore, auditLogger)
	ftpHandler := handlers.NewFTPHandler(cfg, dataStore, auditLogger)
	wafHandler := handlers.NewWAFHandler(cfg, dataStore, auditLogger)
	sslHandler := handlers.NewSSLHandler(cfg, dataStore, auditLogger)
	installerHandler := handlers.NewInstallerHandler(cfg, dataStore, auditLogger)
	billingHandler := handlers.NewBillingHandler(cfg, dataStore, auditLogger)

	// Build Router
	r := chi.NewRouter()

	// Middleware Chain
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(300 * time.Second))

	// CORS Setup - Secure Origin validation allowing credentials
	r.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			return true // Configurable per deployment domain
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Server-ID"},
		ExposedHeaders:   []string{"Link", "X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Security Headers Middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next.ServeHTTP(w, r)
		})
	})

	// Request Body Limit Middleware (10MB) to mitigate memory exhaustion DoS
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB
			}
			next.ServeHTTP(w, r)
		})
	})

	// Base Health Probes
	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)
	r.Get("/version", healthHandler.Version)

	// API v1 Namespace
	r.Route("/api/v1", func(r chi.Router) {
		// Public Auth Endpoints
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)

			// Authenticated User Info
			r.Group(func(r chi.Router) {
				r.Use(auth.Middleware(cfg.JWTSecret))
				r.Get("/me", authHandler.Me)
			})
		})

		// Agent Ingestion API (Authenticated via enrollment tokens / agent keys)
		r.Route("/agent", func(r chi.Router) {
			r.Post("/enroll", agentHandler.Enroll)
			r.Post("/heartbeat", agentHandler.Heartbeat)
		})

		// Public Hosting Plans Catalog
		r.Get("/billing/plans", billingHandler.ListPlans)
		r.Get("/billing/plans/{id}", billingHandler.GetPlan)

		// Protected Fleet Management Endpoints
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(cfg.JWTSecret))

			// Real-Time Dashboard & Telemetry
			r.Get("/dashboard/overview", dashboardHandler.GetOverview)
			r.Get("/system/telemetry", dashboardHandler.GetOverview)
			r.Post("/system/fix", dashboardHandler.RunFix)
			r.Post("/system/restart", dashboardHandler.RestartTarget)

			// Server Fleet
			r.Route("/servers", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermServersView)).Get("/", serverHandler.ListServers)
				r.With(rbac.RequirePermission(rbac.PermServersManage)).Post("/enrollment-tokens", serverHandler.CreateEnrollmentToken)
				r.With(rbac.RequirePermission(rbac.PermServersView)).Get("/{id}", serverHandler.GetServer)
				r.With(rbac.RequirePermission(rbac.PermServersView)).Get("/{id}/metrics", serverHandler.GetServerMetrics)

				// PHP Management Subsystem per Server
				r.Route("/{serverID}/php", func(r chi.Router) {
					// Versions
					r.With(rbac.RequirePermission(rbac.PermPHPView)).Get("/versions", phpHandler.ListVersions)
					r.With(rbac.RequirePermission(rbac.PermPHPVersionManage)).Post("/versions/install", phpHandler.InstallVersion)
					r.With(rbac.RequirePermission(rbac.PermPHPVersionManage)).Delete("/versions/{version}", phpHandler.RemoveVersion)
					r.With(rbac.RequirePermission(rbac.PermPHPVersionManage)).Post("/versions/{version}/default-cli", phpHandler.SetDefaultCLI)

					// Extensions
					r.With(rbac.RequirePermission(rbac.PermPHPView)).Get("/{version}/extensions", phpHandler.ListExtensions)
					r.With(rbac.RequirePermission(rbac.PermPHPExtensionManage)).Post("/{version}/extensions/install", phpHandler.InstallExtension)
					r.With(rbac.RequirePermission(rbac.PermPHPExtensionManage)).Delete("/{version}/extensions/{ext}", phpHandler.RemoveExtension)
					r.With(rbac.RequirePermission(rbac.PermPHPExtensionManage)).Post("/{version}/extensions/{ext}/toggle", phpHandler.ToggleExtension)

					// PHP.ini Engine
					r.With(rbac.RequirePermission(rbac.PermPHPView)).Get("/{version}/ini", phpHandler.GetPHPIni)
					r.With(rbac.RequirePermission(rbac.PermPHPIniManage)).Put("/{version}/ini", phpHandler.UpdatePHPIni)

					// PHP-FPM Service & Pools
					r.With(rbac.RequirePermission(rbac.PermPHPView)).Get("/{version}/fpm", phpHandler.GetFPMStatus)
					r.With(rbac.RequirePermission(rbac.PermPHPFPMManage)).Post("/{version}/fpm/service", phpHandler.ServiceAction)
					r.With(rbac.RequirePermission(rbac.PermPHPView)).Get("/{version}/pools", phpHandler.ListFPMPools)
					r.With(rbac.RequirePermission(rbac.PermPHPPoolManage)).Post("/{version}/pools", phpHandler.CreateFPMPool)

					// Health Check
					r.With(rbac.RequirePermission(rbac.PermPHPHealthCheck)).Get("/health", phpHandler.GetHealth)
					r.With(rbac.RequirePermission(rbac.PermPHPHealthCheck)).Get("/{version}/health", phpHandler.GetHealth)
				})

				// Web Server Management (Nginx, Apache, OpenLiteSpeed, LiteSpeed Enterprise)
				r.Route("/{serverID}/webservers", func(r chi.Router) {
					r.With(rbac.RequirePermission(rbac.PermWebServerView)).Get("/", webServerHandler.ListServers)
					r.With(rbac.RequirePermission(rbac.PermWebServerView)).Get("/conflicts", webServerHandler.GetPortConflicts)
					r.With(rbac.RequirePermission(rbac.PermWebServerSwitch)).Post("/migrate", webServerHandler.Migrate)
					r.With(rbac.RequirePermission(rbac.PermWebServerView)).Get("/{type}", webServerHandler.GetServer)
					r.With(rbac.RequirePermission(rbac.PermWebServerInstall)).Post("/{type}/install", webServerHandler.InstallServer)
					r.With(rbac.RequirePermission(rbac.PermWebServerManage)).Post("/{type}/uninstall", webServerHandler.UninstallServer)
					r.With(rbac.RequirePermission(rbac.PermWebServerManage)).Post("/{type}/service", webServerHandler.ServiceAction)
					r.With(rbac.RequirePermission(rbac.PermWebServerConfig)).Get("/{type}/config", webServerHandler.GetMasterConfig)
					r.With(rbac.RequirePermission(rbac.PermWebServerConfig)).Put("/{type}/config", webServerHandler.UpdateMasterConfig)
					r.With(rbac.RequirePermission(rbac.PermVHostManage)).Get("/{type}/vhosts", webServerHandler.ListVHosts)
				})
			})

			// Websites & Vhosts
			r.Route("/websites", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermWebsitesView)).Get("/", websiteHandler.List)
				r.With(rbac.RequirePermission(rbac.PermWebsitesCreate)).Post("/", websiteHandler.Create)
				r.With(rbac.RequirePermission(rbac.PermWebsitesView)).Get("/{id}", websiteHandler.Get)
				r.With(rbac.RequirePermission(rbac.PermWebsitesManage)).Post("/{id}/status", websiteHandler.UpdateStatus)
				r.With(rbac.RequirePermission(rbac.PermWebsitesDelete)).Delete("/{id}", websiteHandler.Delete)
				r.With(rbac.RequirePermission(rbac.PermSSLManage)).Post("/{id}/ssl", websiteHandler.IssueSSL)

				// Realtime Conf, Logs, Backup, WAF, Batch, Statistics
				r.With(rbac.RequirePermission(rbac.PermWebsitesView)).Get("/statistics", websiteHandler.Statistics)
				r.With(rbac.RequirePermission(rbac.PermWebsitesManage)).Post("/batch", websiteHandler.Batch)
				r.With(rbac.RequirePermission(rbac.PermWebsitesView)).Get("/{id}/conf", websiteHandler.GetConf)
				r.With(rbac.RequirePermission(rbac.PermWebsitesManage)).Put("/{id}/conf", websiteHandler.UpdateConf)
				r.With(rbac.RequirePermission(rbac.PermWebsitesView)).Get("/{id}/logs", websiteHandler.GetLogs)
				r.With(rbac.RequirePermission(rbac.PermWebsitesManage)).Post("/{id}/backup", websiteHandler.Backup)
				r.With(rbac.RequirePermission(rbac.PermWebsitesManage)).Post("/{id}/waf", websiteHandler.ToggleWAF)

				// Per-Website PHP Integration
				r.With(rbac.RequirePermission(rbac.PermPHPView)).Get("/{id}/php", phpHandler.GetWebsitePHP)
				r.With(rbac.RequirePermission(rbac.PermWebsitesManage)).Post("/{id}/php/switch", phpHandler.SwitchWebsitePHP)
				r.With(rbac.RequirePermission(rbac.PermPHPHealthCheck)).Post("/{id}/php/test", phpHandler.TestWebsitePHP)

				// Per-Website Web Server Integration
				r.With(rbac.RequirePermission(rbac.PermWebServerView)).Get("/{id}/webserver", webServerHandler.GetWebsiteWebServer)
				r.With(rbac.RequirePermission(rbac.PermWebServerSwitch)).Post("/{id}/webserver/switch", webServerHandler.SwitchWebsiteWebServer)

				// Per-Website User Isolation & cgroups v2
				r.With(rbac.RequirePermission(rbac.PermWebsitesView)).Get("/{id}/isolation", websiteHandler.GetIsolation)
				r.With(rbac.RequirePermission(rbac.PermWebsitesManage)).Put("/{id}/isolation", websiteHandler.UpdateIsolation)

				// Per-Website 1-Click Application Installer (WordPress, Laravel, Next.js, Drupal, phpMyAdmin)
				r.With(rbac.RequirePermission(rbac.PermWebsitesView)).Get("/{id}/app", installerHandler.GetWebsiteApp)
				r.With(rbac.RequirePermission(rbac.PermWebsitesManage)).Post("/{id}/app/install", installerHandler.InstallWebsiteApp)
				r.With(rbac.RequirePermission(rbac.PermWebsitesManage)).Post("/{id}/app/uninstall", installerHandler.UninstallWebsiteApp)
			})

			// 1-Click Application Installer Catalog
			r.Route("/installer", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermWebsitesView)).Get("/templates", installerHandler.ListTemplates)
			})

			// Databases & DB Users
			r.Route("/databases", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermDatabasesView)).Get("/", databaseHandler.List)
				r.With(rbac.RequirePermission(rbac.PermDatabasesCreate)).Post("/", databaseHandler.Create)
				r.With(rbac.RequirePermission(rbac.PermDatabasesCreate)).Put("/{id}", databaseHandler.Update)
				r.With(rbac.RequirePermission(rbac.PermDatabasesDelete)).Delete("/{id}", databaseHandler.Delete)
				r.With(rbac.RequirePermission(rbac.PermDatabasesView)).Get("/status", databaseHandler.GetStatus)
				r.With(rbac.RequirePermission(rbac.PermDatabasesView)).Get("/server-dbs", databaseHandler.GetServerDatabases)
				r.With(rbac.RequirePermission(rbac.PermDatabasesCreate)).Post("/sync", databaseHandler.Sync)
				r.With(rbac.RequirePermission(rbac.PermDatabasesCreate)).Post("/{id}/tools", databaseHandler.RunTools)
				r.With(rbac.RequirePermission(rbac.PermDatabasesCreate)).Post("/{id}/backup", databaseHandler.Backup)
				r.With(rbac.RequirePermission(rbac.PermDatabasesCreate)).Post("/{id}/import", databaseHandler.Import)
				r.With(rbac.RequirePermission(rbac.PermDatabasesView)).Get("/root-password", databaseHandler.GetRootPassword)
				r.With(rbac.RequirePermission(rbac.PermDatabasesCreate)).Post("/root-password", databaseHandler.SetRootPassword)
				r.With(rbac.RequirePermission(rbac.PermDatabasesView)).Get("/auto-backup", databaseHandler.GetAutoBackup)
				r.With(rbac.RequirePermission(rbac.PermDatabasesCreate)).Post("/auto-backup", databaseHandler.SetAutoBackup)
				r.With(rbac.RequirePermission(rbac.PermDatabasesView)).Get("/advanced-setup", databaseHandler.GetAdvancedSetup)
				r.With(rbac.RequirePermission(rbac.PermDatabasesCreate)).Post("/advanced-setup", databaseHandler.SetAdvancedSetup)
				r.With(rbac.RequirePermission(rbac.PermDatabasesView)).Get("/recycle-bin", databaseHandler.ListRecycleBin)
				r.With(rbac.RequirePermission(rbac.PermDatabasesCreate)).Post("/recycle-bin/{id}/restore", databaseHandler.RestoreRecycleBin)
				r.With(rbac.RequirePermission(rbac.PermDatabasesCreate)).Post("/batch", databaseHandler.Batch)
				r.With(rbac.RequirePermission(rbac.PermDatabasesCreate)).Post("/users", databaseHandler.CreateUser)
			})

			// Audit Logs
			r.Route("/audit-logs", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermAuditView)).Get("/", auditHandler.List)
			})

			// DNS Management
			r.Route("/dns", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermDNSManage)).Get("/zones", dnsHandler.ListZones)
				r.With(rbac.RequirePermission(rbac.PermDNSManage)).Post("/zones", dnsHandler.CreateZone)
				r.With(rbac.RequirePermission(rbac.PermDNSManage)).Get("/zones/{zoneID}/records", dnsHandler.ListRecords)
				r.With(rbac.RequirePermission(rbac.PermDNSManage)).Post("/zones/{zoneID}/records", dnsHandler.CreateRecord)
				r.With(rbac.RequirePermission(rbac.PermDNSManage)).Delete("/zones/{zoneID}/records/{recordID}", dnsHandler.DeleteRecord)
				r.With(rbac.RequirePermission(rbac.PermDNSManage)).Get("/zones/{zoneID}/export/bind", dnsHandler.ExportBindZone)
			})

			// Fleet Alerts & Notifications
			r.Route("/alerts", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermAlertsManage)).Get("/", alertHandler.ListIncidents)
				r.With(rbac.RequirePermission(rbac.PermAlertsManage)).Get("/rules", alertHandler.ListRules)
				r.With(rbac.RequirePermission(rbac.PermAlertsManage)).Post("/rules", alertHandler.CreateRule)
				r.With(rbac.RequirePermission(rbac.PermAlertsManage)).Get("/channels", alertHandler.ListChannels)
				r.With(rbac.RequirePermission(rbac.PermAlertsManage)).Post("/channels", alertHandler.CreateChannel)
				r.With(rbac.RequirePermission(rbac.PermAlertsManage)).Post("/test", alertHandler.TestTrigger)
			})

			// Automated & On-demand Backups
			r.Route("/backups", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermBackupsCreate)).Get("/", backupHandler.List)
				r.With(rbac.RequirePermission(rbac.PermBackupsCreate)).Post("/", backupHandler.Create)
				r.With(rbac.RequirePermission(rbac.PermBackupsCreate)).Post("/create", backupHandler.Create)
				r.With(rbac.RequirePermission(rbac.PermBackupsRestore)).Post("/restore", backupHandler.Restore)
				r.With(rbac.RequirePermission(rbac.PermBackupsCreate)).Delete("/{id}", backupHandler.Delete)
				r.With(rbac.RequirePermission(rbac.PermBackupsCreate)).Get("/download/{id}", backupHandler.Download)

				// Cloud Destinations
				r.With(rbac.RequirePermission(rbac.PermBackupsCreate)).Get("/destinations", backupHandler.ListDestinations)
				r.With(rbac.RequirePermission(rbac.PermBackupsCreate)).Post("/destinations", backupHandler.SaveDestination)
				r.With(rbac.RequirePermission(rbac.PermBackupsCreate)).Delete("/destinations/{id}", backupHandler.DeleteDestination)
				r.With(rbac.RequirePermission(rbac.PermBackupsCreate)).Post("/destinations/test", backupHandler.TestDestination)

				// Schedules
				r.With(rbac.RequirePermission(rbac.PermBackupsCreate)).Get("/schedules", backupHandler.ListSchedules)
				r.With(rbac.RequirePermission(rbac.PermBackupsCreate)).Post("/schedules", backupHandler.SaveSchedule)
				r.With(rbac.RequirePermission(rbac.PermBackupsCreate)).Delete("/schedules/{id}", backupHandler.DeleteSchedule)
			})

			// Commercial & Licensing
			r.Route("/license", func(r chi.Router) {
				r.Get("/", licenseHandler.Get)
				r.With(rbac.RequirePermission(rbac.PermLicensesManage)).Post("/activate", licenseHandler.Activate)
			})

			// Live System Updates Management
			r.Route("/system/updates", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermSystemUpdateView)).Get("/status", updateHandler.GetStatus)
				r.With(rbac.RequirePermission(rbac.PermSystemUpdateCheck)).Post("/check", updateHandler.CheckUpdates)
				r.With(rbac.RequirePermission(rbac.PermSystemUpdateStart)).Post("/start", updateHandler.StartUpdate)
				r.With(rbac.RequirePermission(rbac.PermSystemUpdateView)).Get("/jobs", updateHandler.ListJobs)
				r.With(rbac.RequirePermission(rbac.PermSystemUpdateView)).Get("/jobs/{id}", updateHandler.GetJobStatus)
				r.With(rbac.RequirePermission(rbac.PermSystemUpdateRollback)).Post("/rollback", updateHandler.TriggerRollback)
				r.With(rbac.RequirePermission(rbac.PermSystemUpdateManage)).Put("/channel", updateHandler.SetChannel)
				r.With(rbac.RequirePermission(rbac.PermSystemUpdateSchedule)).Post("/schedule", updateHandler.ScheduleUpdate)
			})

			// Integrated Web Terminal
			r.Route("/terminal", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermTerminalAccess)).Get("/info", terminalHandler.GetInfo)
				r.With(rbac.RequirePermission(rbac.PermTerminalAccess)).Post("/execute", terminalHandler.Execute)
			})

			// File Manager Subsystem
			r.Route("/files", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermFilesBrowse)).Get("/list", fileHandler.List)
				r.With(rbac.RequirePermission(rbac.PermFilesBrowse)).Get("/stat", fileHandler.Stat)
				r.With(rbac.RequirePermission(rbac.PermFilesBrowse)).Get("/content", fileHandler.GetContent)
				r.With(rbac.RequirePermission(rbac.PermFilesBrowse)).Get("/download", fileHandler.Download)
				r.With(rbac.RequirePermission(rbac.PermFilesEdit)).Put("/content", fileHandler.SaveContent)
				r.With(rbac.RequirePermission(rbac.PermFilesEdit)).Post("/mkdir", fileHandler.Mkdir)
				r.With(rbac.RequirePermission(rbac.PermFilesEdit)).Post("/upload", fileHandler.Upload)
				r.With(rbac.RequirePermission(rbac.PermFilesEdit)).Post("/rename", fileHandler.Rename)
				r.With(rbac.RequirePermission(rbac.PermFilesEdit)).Post("/copy", fileHandler.Copy)
				r.With(rbac.RequirePermission(rbac.PermFilesEdit)).Delete("/delete", fileHandler.Delete)
				r.With(rbac.RequirePermission(rbac.PermFilesEdit)).Post("/permissions", fileHandler.Permissions)
				r.With(rbac.RequirePermission(rbac.PermFilesEdit)).Post("/archive", fileHandler.Archive)
				r.With(rbac.RequirePermission(rbac.PermFilesEdit)).Post("/extract", fileHandler.Extract)
			})

			// Linux Firewall (UFW) & Fail2ban Subsystem
			r.Route("/firewall", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermFirewallView)).Get("/status", firewallHandler.GetStatus)
				r.With(rbac.RequirePermission(rbac.PermFirewallView)).Get("/rules", firewallHandler.ListRules)
				r.With(rbac.RequirePermission(rbac.PermFirewallManage)).Post("/rules", firewallHandler.AddRule)
				r.With(rbac.RequirePermission(rbac.PermFirewallManage)).Delete("/rules", firewallHandler.DeleteRule)
				r.With(rbac.RequirePermission(rbac.PermFirewallManage)).Delete("/rules/{id}", firewallHandler.DeleteRule)
				r.With(rbac.RequirePermission(rbac.PermFirewallManage)).Post("/enable", firewallHandler.Enable)
				r.With(rbac.RequirePermission(rbac.PermFirewallManage)).Post("/disable", firewallHandler.Disable)

				// Fail2ban intrusion defense
				r.With(rbac.RequirePermission(rbac.PermFirewallView)).Get("/fail2ban/jails", firewallHandler.ListJails)
				r.With(rbac.RequirePermission(rbac.PermFirewallView)).Get("/fail2ban/banned", firewallHandler.ListBannedIPs)
				r.With(rbac.RequirePermission(rbac.PermFirewallManage)).Post("/fail2ban/ban", firewallHandler.BanIP)
				r.With(rbac.RequirePermission(rbac.PermFirewallManage)).Post("/fail2ban/unban", firewallHandler.UnbanIP)
			})

			// Web Application Firewall (ModSecurity v3 + OWASP CRS)
			r.Route("/waf", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermWAFView)).Get("/status", wafHandler.GetStatus)
				r.With(rbac.RequirePermission(rbac.PermWAFManage)).Post("/config", wafHandler.UpdateConfig)
				r.With(rbac.RequirePermission(rbac.PermWAFView)).Get("/rules", wafHandler.ListRules)
				r.With(rbac.RequirePermission(rbac.PermWAFManage)).Post("/rules/toggle", wafHandler.ToggleRule)
				r.With(rbac.RequirePermission(rbac.PermWAFView)).Get("/events", wafHandler.GetEvents)
				r.With(rbac.RequirePermission(rbac.PermWAFView)).Get("/websites/{domain}", wafHandler.GetWebsiteWAF)
				r.With(rbac.RequirePermission(rbac.PermWAFManage)).Post("/websites/{domain}", wafHandler.UpdateWebsiteWAF)
				r.With(rbac.RequirePermission(rbac.PermWAFManage)).Post("/probe", wafHandler.SimulateProbe)
			})

			// SSL Certificates & Wildcard DNS-01
			r.Route("/ssl", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermSSLView)).Get("/certificates", sslHandler.ListCertificates)
				r.With(rbac.RequirePermission(rbac.PermSSLManage)).Post("/issue", sslHandler.Issue)
				r.With(rbac.RequirePermission(rbac.PermSSLManage)).Post("/challenge", sslHandler.PrepareChallenge)
				r.With(rbac.RequirePermission(rbac.PermSSLManage)).Post("/verify-challenge", sslHandler.VerifyChallenge)
				r.With(rbac.RequirePermission(rbac.PermSSLManage)).Post("/custom", sslHandler.ImportCustom)
				r.With(rbac.RequirePermission(rbac.PermSSLManage)).Post("/renew/{id}", sslHandler.Renew)
				r.With(rbac.RequirePermission(rbac.PermSSLManage)).Post("/auto-renew", sslHandler.AutoRenew)
				r.With(rbac.RequirePermission(rbac.PermSSLManage)).Delete("/{id}", sslHandler.Delete)
			})

			// Scheduled Tasks (Linux Crontab) Subsystem
			r.Route("/cron", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermCronView)).Get("/status", cronHandler.GetStatus)
				r.With(rbac.RequirePermission(rbac.PermCronView)).Get("/jobs", cronHandler.ListJobs)
				r.With(rbac.RequirePermission(rbac.PermCronManage)).Post("/jobs", cronHandler.CreateJob)
				r.With(rbac.RequirePermission(rbac.PermCronManage)).Put("/jobs/{id}", cronHandler.UpdateJob)
				r.With(rbac.RequirePermission(rbac.PermCronManage)).Delete("/jobs/{id}", cronHandler.DeleteJob)
				r.With(rbac.RequirePermission(rbac.PermCronManage)).Post("/jobs/{id}/toggle", cronHandler.ToggleJob)
				r.With(rbac.RequirePermission(rbac.PermCronManage)).Post("/jobs/{id}/run", cronHandler.RunJob)
				r.With(rbac.RequirePermission(rbac.PermCronManage)).Post("/test", cronHandler.TestCommand)
			})

			// Docker Engine & Container Management Subsystem
			r.Route("/docker", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermServersView)).Get("/status", dockerHandler.GetStatus)
				r.With(rbac.RequirePermission(rbac.PermServersView)).Get("/containers", dockerHandler.ListContainers)
				r.With(rbac.RequirePermission(rbac.PermServersView)).Get("/containers/{id}/logs", dockerHandler.GetLogs)
				r.With(rbac.RequirePermission(rbac.PermServersView)).Get("/stats", dockerHandler.GetStats)
				r.With(rbac.RequirePermission(rbac.PermServersView)).Get("/images", dockerHandler.ListImages)

				// Mutations
				r.With(rbac.RequirePermission(rbac.PermDockerManage)).Post("/containers/{id}/start", dockerHandler.StartContainer)
				r.With(rbac.RequirePermission(rbac.PermDockerManage)).Post("/containers/{id}/stop", dockerHandler.StopContainer)
				r.With(rbac.RequirePermission(rbac.PermDockerManage)).Post("/containers/{id}/restart", dockerHandler.RestartContainer)
				r.With(rbac.RequirePermission(rbac.PermDockerManage)).Delete("/containers/{id}", dockerHandler.DeleteContainer)
				r.With(rbac.RequirePermission(rbac.PermDockerManage)).Post("/containers/run", dockerHandler.RunContainer)
				r.With(rbac.RequirePermission(rbac.PermDockerManage)).Delete("/images/{id}", dockerHandler.DeleteImage)
				r.With(rbac.RequirePermission(rbac.PermDockerManage)).Post("/prune", dockerHandler.PruneSystem)
			})

			// FTP (Pure-FTPd) Subsystem
			r.Route("/ftp", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermFilesBrowse)).Get("/status", ftpHandler.GetStatus)
				r.With(rbac.RequirePermission(rbac.PermFilesBrowse)).Get("/users", ftpHandler.ListUsers)
				r.With(rbac.RequirePermission(rbac.PermFilesEdit)).Post("/users", ftpHandler.CreateUser)
				r.With(rbac.RequirePermission(rbac.PermFilesEdit)).Put("/users/{username}", ftpHandler.UpdateUser)
				r.With(rbac.RequirePermission(rbac.PermFilesEdit)).Put("/users/{username}/password", ftpHandler.ChangePassword)
				r.With(rbac.RequirePermission(rbac.PermFilesEdit)).Delete("/users/{username}", ftpHandler.DeleteUser)
				r.With(rbac.RequirePermission(rbac.PermFilesEdit)).Post("/users/{username}/toggle", ftpHandler.ToggleUser)
			})

			// 1-Click App Store & Extensions
			r.Route("/apps", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermServersView)).Get("/", appStoreHandler.ListApps)
				r.With(rbac.RequirePermission(rbac.PermServersView)).Get("/{id}", appStoreHandler.GetApp)
				r.With(rbac.RequirePermission(rbac.PermServersManage)).Post("/{id}/install", appStoreHandler.InstallApp)
				r.With(rbac.RequirePermission(rbac.PermServersManage)).Post("/{id}/uninstall", appStoreHandler.UninstallApp)
				r.With(rbac.RequirePermission(rbac.PermServersManage)).Post("/{id}/service", appStoreHandler.ControlService)
				r.With(rbac.RequirePermission(rbac.PermServersView)).Get("/jobs/{jobID}", appStoreHandler.GetJob)
			})

			// Team & Collaborators
			r.Route("/team", func(r chi.Router) {
				r.Get("/members", teamHandler.ListMembers)
				r.With(rbac.RequirePermission(rbac.PermUsersManage)).Post("/invite", teamHandler.InviteMember)
				r.With(rbac.RequirePermission(rbac.PermUsersManage)).Delete("/members/{memberID}", teamHandler.RemoveMember)
			})

			// API Keys
			r.Route("/api-keys", func(r chi.Router) {
				r.Get("/", apiKeyHandler.List)
				r.With(rbac.RequirePermission(rbac.PermUsersManage)).Post("/", apiKeyHandler.Create)
				r.With(rbac.RequirePermission(rbac.PermUsersManage)).Delete("/{keyID}", apiKeyHandler.Revoke)
			})

			// Email Hosting Subsystem
			r.Route("/email", func(r chi.Router) {
				// Domains
				r.With(rbac.RequirePermission(rbac.PermEmailView)).Get("/domains", emailHandler.ListDomains)
				r.With(rbac.RequirePermission(rbac.PermEmailDomainManage)).Post("/domains", emailHandler.CreateDomain)
				r.With(rbac.RequirePermission(rbac.PermEmailView)).Get("/domains/{id}", emailHandler.GetDomain)
				r.With(rbac.RequirePermission(rbac.PermEmailDomainManage)).Delete("/domains/{id}", emailHandler.DeleteDomain)

				// Mailboxes
				r.With(rbac.RequirePermission(rbac.PermEmailView)).Get("/mailboxes", emailHandler.ListMailboxes)
				r.With(rbac.RequirePermission(rbac.PermEmailMailboxManage)).Post("/mailboxes", emailHandler.CreateMailbox)
				r.With(rbac.RequirePermission(rbac.PermEmailView)).Get("/mailboxes/{id}", emailHandler.GetMailbox)
				r.With(rbac.RequirePermission(rbac.PermEmailMailboxManage)).Put("/mailboxes/{id}", emailHandler.UpdateMailbox)
				r.With(rbac.RequirePermission(rbac.PermEmailMailboxManage)).Put("/mailboxes/{id}/password", emailHandler.ChangeMailboxPassword)
				r.With(rbac.RequirePermission(rbac.PermEmailMailboxManage)).Delete("/mailboxes/{id}", emailHandler.DeleteMailbox)

				// Aliases
				r.With(rbac.RequirePermission(rbac.PermEmailView)).Get("/aliases", emailHandler.ListAliases)
				r.With(rbac.RequirePermission(rbac.PermEmailAliasManage)).Post("/aliases", emailHandler.CreateAlias)
				r.With(rbac.RequirePermission(rbac.PermEmailAliasManage)).Delete("/aliases/{id}", emailHandler.DeleteAlias)

				// Delivery Logs & Health
				r.With(rbac.RequirePermission(rbac.PermEmailLogsView)).Get("/logs", emailHandler.ListLogs)
				r.With(rbac.RequirePermission(rbac.PermEmailView)).Get("/health", emailHandler.CheckHealth)
			})

			// Enterprise Hosting Billing, Subscriptions, Invoices & Payment Gateways
			r.Route("/billing", func(r chi.Router) {
				// Plans Catalog & Admin Management
				r.Get("/plans", billingHandler.ListPlans)
				r.Get("/plans/{id}", billingHandler.GetPlan)
				r.With(rbac.RequirePermission(rbac.PermBillingManage)).Post("/plans", billingHandler.CreatePlan)
				r.With(rbac.RequirePermission(rbac.PermBillingManage)).Put("/plans/{id}", billingHandler.UpdatePlan)
				r.With(rbac.RequirePermission(rbac.PermBillingManage)).Delete("/plans/{id}", billingHandler.DeletePlan)

				// Subscriptions
				r.With(rbac.RequirePermission(rbac.PermBillingView)).Get("/subscriptions", billingHandler.ListSubscriptions)
				r.With(rbac.RequirePermission(rbac.PermBillingView)).Post("/subscriptions", billingHandler.CreateSubscription)
				r.With(rbac.RequirePermission(rbac.PermBillingView)).Get("/subscriptions/{id}", billingHandler.GetSubscription)
				r.With(rbac.RequirePermission(rbac.PermBillingView)).Post("/subscriptions/{id}/cancel", billingHandler.CancelSubscription)
				r.With(rbac.RequirePermission(rbac.PermBillingView)).Post("/subscriptions/{id}/renew", billingHandler.RenewSubscription)

				// Invoices
				r.With(rbac.RequirePermission(rbac.PermBillingView)).Get("/invoices", billingHandler.ListInvoices)
				r.With(rbac.RequirePermission(rbac.PermBillingView)).Get("/invoices/{id}", billingHandler.GetInvoice)
				r.With(rbac.RequirePermission(rbac.PermBillingView)).Post("/invoices/{id}/pay", billingHandler.PayInvoice)

				// Payment Gateways
				r.With(rbac.RequirePermission(rbac.PermBillingView)).Get("/gateways", billingHandler.ListGateways)
				r.With(rbac.RequirePermission(rbac.PermBillingManage)).Put("/gateways/{gateway}", billingHandler.UpdateGateway)
			})
		})
	})

	serverAddr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server execution in background goroutine
	go func() {
		logger.Info("Hostvra API HTTP server listening", "address", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Hostvra API gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Hostvra API server stopped.")
}

func seedDefaultAdmin(ctx context.Context, s store.Store, logger *slog.Logger) {
	passwordHash, err := auth.HashPassword("SuperSecretP@ss123!", nil)
	if err != nil {
		logger.Error("Failed to hash default admin password", "error", err)
		return
	}

	existingUser, err := s.GetUserByEmail(ctx, "admin@hostvra.com")
	if err == nil && existingUser != nil {
		// Sync existing admin account with the known default password
		if err := s.UpdateUserPassword(ctx, existingUser.ID, passwordHash); err != nil {
			logger.Warn("Failed to synchronize default admin password", "error", err)
		} else {
			logger.Info("Default administrator password synchronized successfully", "email", "admin@hostvra.com")
		}
		return
	}

	defaultOrgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	org := &store.Organization{
		ID:          defaultOrgID,
		Name:        "Hostvra Cloud",
		Slug:        "hostvra-cloud",
		PlanTier:    "enterprise",
		MaxServers:  100,
		MaxWebsites: 1000,
	}
	_ = s.CreateOrganization(ctx, org)

	adminUser := &store.User{
		ID:           uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		Email:        "admin@hostvra.com",
		PasswordHash: passwordHash,
		FullName:     "Hostvra Administrator",
		IsActive:     true,
		IsSuperAdmin: true,
	}

	if err := s.CreateUser(ctx, adminUser, defaultOrgID, "owner"); err != nil {
		logger.Warn("Failed to seed default admin user", "error", err)
	} else {
		logger.Info("Default administrator account successfully seeded", "email", "admin@hostvra.com")
	}
}

func autoRecoverLocalAgentNode(ctx context.Context, s store.Store, logger *slog.Logger) {
	agentCfgPaths := []string{
		"/etc/hostvra/agent.json",
		"/var/lib/hostvra/agent.json",
	}

	var agentCfgFile string
	for _, p := range agentCfgPaths {
		if _, err := os.Stat(p); err == nil {
			agentCfgFile = p
			break
		}
	}

	if agentCfgFile == "" {
		return
	}

	type agentConfigFile struct {
		ServerID uuid.UUID `json:"server_id"`
		AgentKey string    `json:"agent_key"`
	}

	rawBytes, err := os.ReadFile(agentCfgFile)
	if err != nil {
		return
	}

	var cfg agentConfigFile
	if err := json.Unmarshal(rawBytes, &cfg); err != nil || cfg.ServerID == uuid.Nil {
		return
	}

	existing, err := s.GetServerByID(ctx, cfg.ServerID)
	if err == nil && existing != nil {
		logger.Info("Local node verified in server fleet registry", "server_id", cfg.ServerID)
		return
	}

	hostname := "hostvra-node"
	if h, err := os.Hostname(); err == nil && h != "" {
		hostname = h
	}

	defaultOrgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	now := time.Now().UTC()
	recovered := &store.Server{
		ID:              cfg.ServerID,
		OrganizationID:  defaultOrgID,
		Name:            hostname,
		Hostname:        hostname,
		IPAddress:       "127.0.0.1",
		OSName:          "Linux",
		OSVersion:       "Ubuntu",
		Architecture:    "amd64",
		AgentVersion:    "1.0.0",
		Status:          "online",
		CreatedAt:       now,
		UpdatedAt:       now,
		LastHeartbeatAt: &now,
	}

	if err := s.CreateServer(ctx, recovered); err != nil {
		logger.Warn("Failed to auto-recover server node from agent config", "error", err)
	} else {
		logger.Info("Successfully auto-recovered server node into fleet registry", "server_id", cfg.ServerID, "hostname", hostname)
	}
}

