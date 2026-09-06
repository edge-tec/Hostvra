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

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/handlers"
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

	// Build Router
	r := chi.NewRouter()

	// Middleware Chain
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS Setup
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Configurable in production
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Server-ID"},
		ExposedHeaders:   []string{"Link", "X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

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
			})

			// Websites & Vhosts
			r.Route("/websites", func(r chi.Router) {
				r.With(rbac.RequirePermission(rbac.PermWebsitesView)).Get("/", websiteHandler.List)
				r.With(rbac.RequirePermission(rbac.PermWebsitesCreate)).Post("/", websiteHandler.Create)
				r.With(rbac.RequirePermission(rbac.PermWebsitesView)).Get("/{id}", websiteHandler.Get)
				r.With(rbac.RequirePermission(rbac.PermWebsitesManage)).Post("/{id}/status", websiteHandler.UpdateStatus)
				r.With(rbac.RequirePermission(rbac.PermWebsitesDelete)).Delete("/{id}", websiteHandler.Delete)
				r.With(rbac.RequirePermission(rbac.PermSSLManage)).Post("/{id}/ssl", websiteHandler.IssueSSL)
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
