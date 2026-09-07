package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/installer"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

func TestInstallerHandler_API(t *testing.T) {
	tempDir := t.TempDir()
	docRoot := filepath.Join(tempDir, "www_testapp")

	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)

	h := NewInstallerHandler(cfg, s, auditLogger)

	// Create test website
	orgID := uuid.New()
	siteID := uuid.New()
	serverID := uuid.New()
	_ = s.CreateWebsite(context.Background(), &store.Website{
		ID:             siteID,
		OrganizationID: orgID,
		ServerID:       serverID,
		PrimaryDomain:  "testwp.org",
		DocumentRoot:   docRoot,
		SystemUser:     "u_testwp",
		AppType:        "static",
		Status:         "running",
	})

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
				UserID:         uuid.New(),
				OrganizationID: orgID,
				Role:           "owner",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/installer/templates", h.ListTemplates)
		r.Get("/websites/{id}/app", h.GetWebsiteApp)
		r.Post("/websites/{id}/app/install", h.InstallWebsiteApp)
		r.Post("/websites/{id}/app/uninstall", h.UninstallWebsiteApp)
	})

	// 1. List Templates
	req := httptest.NewRequest("GET", "/api/v1/installer/templates", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list templates failed: %d", rec.Code)
	}

	var tplEnv response.Envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &tplEnv)
	tplBytes, _ := json.Marshal(tplEnv.Data)
	var templates []installer.AppTemplate
	_ = json.Unmarshal(tplBytes, &templates)

	if len(templates) < 4 {
		t.Errorf("expected at least 4 templates, got %d", len(templates))
	}

	// 2. Initial Get App (Clean State)
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/websites/%s/app", siteID.String()), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("get initial app failed: %d", rec.Code)
	}

	// 3. Install WordPress
	installReq := installer.InstallSiteAppRequest{
		AppID:         "wordpress",
		SiteTitle:     "My CloudWP Site",
		AdminUser:     "admin",
		AdminEmail:    "admin@testwp.org",
		AdminPassword: "SuperSecurePassword123!",
		DBType:        "mysql",
	}
	body, _ := json.Marshal(installReq)
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/websites/%s/app/install", siteID.String()), bytes.NewReader(body))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("install wordpress failed: %d, body %s", rec.Code, rec.Body.String())
	}

	var installEnv response.Envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &installEnv)
	if !installEnv.Success {
		t.Fatalf("install returned failure: %s", rec.Body.String())
	}

	// 4. Verify Website Status & AppType Updated
	site, _ := s.GetWebsiteByID(context.Background(), siteID)
	if site.AppType != "wordpress" {
		t.Errorf("expected website app_type wordpress, got %s", site.AppType)
	}

	// 5. Detect Installed WordPress
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/websites/%s/app", siteID.String()), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("get app after install failed: %d", rec.Code)
	}

	var detectEnv response.Envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &detectEnv)
	detectMap, _ := detectEnv.Data.(map[string]interface{})
	if hasApp, ok := detectMap["has_app"].(bool); !ok || !hasApp {
		t.Errorf("expected has_app to be true after install")
	}

	// 6. Uninstall Application
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/websites/%s/app/uninstall", siteID.String()), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("uninstall app failed: %d", rec.Code)
	}

	siteAfter, _ := s.GetWebsiteByID(context.Background(), siteID)
	if siteAfter.AppType != "static" {
		t.Errorf("expected website app_type static after uninstall, got %s", siteAfter.AppType)
	}
}
