package main

import (
	"context"
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

	// Build Router
	r := chi.NewRouter()

	// Middleware Chain
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

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

		// Protected Fleet Management Endpoints
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(cfg.JWTSecret))

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

				// Per-Website PHP Integration
				r.With(rbac.RequirePermission(rbac.PermPHPView)).Get("/{id}/php", phpHandler.GetWebsitePHP)
				r.With(rbac.RequirePermission(rbac.PermWebsitesManage)).Post("/{id}/php/switch", phpHandler.SwitchWebsitePHP)
				r.With(rbac.RequirePermission(rbac.PermPHPHealthCheck)).Post("/{id}/php/test", phpHandler.TestWebsitePHP)

				// Per-Website Web Server Integration
				r.With(rbac.RequirePermission(rbac.PermWebServerView)).Get("/{id}/webserver", webServerHandler.GetWebsiteWebServer)
				r.With(rbac.RequirePermission(rbac.PermWebServerSwitch)).Post("/{id}/webserver/switch", webServerHandler.SwitchWebsiteWebServer)
			})

			// Databases & DB Users
			r.Route("/databases", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermDatabasesView)).Get("/", databaseHandler.List)
				r.With(rbac.RequirePermission(rbac.PermDatabasesCreate)).Post("/", databaseHandler.Create)
				r.With(rbac.RequirePermission(rbac.PermDatabasesDelete)).Delete("/{id}", databaseHandler.Delete)
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
				r.With(rbac.RequirePermission(rbac.PermBackupsRestore)).Post("/restore", backupHandler.Restore)
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
	_, err := s.GetUserByEmail(ctx, "admin@hostvra.com")
	if err == nil {
		return // Already seeded
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

	passwordHash, err := auth.HashPassword("SuperSecretP@ss123!", nil)
	if err != nil {
		logger.Error("Failed to hash default admin password", "error", err)
		return
	}

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
