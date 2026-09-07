package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/isolation"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestWebsiteHandler_IsolationAndCgroups(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOSTVRA_WEB_ROOT", filepath.Join(tempDir, "www"))
	t.Setenv("HOSTVRA_PHP_CONFIG_DIR", filepath.Join(tempDir, "php"))
	t.Setenv("HOSTVRA_SYSTEMD_DIR", filepath.Join(tempDir, "systemd"))

	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)

	h := NewWebsiteHandler(cfg, s, auditLogger)

	// Create test org and server
	orgID := uuid.New()
	_ = s.CreateOrganization(context.Background(), &store.Organization{
		ID:   orgID,
		Name: "Test Org",
		Slug: "test-org",
	})

	serverID := uuid.New()
	_ = s.CreateServer(context.Background(), &store.Server{
		ID:             serverID,
		OrganizationID: orgID,
		Name:           "srv-01",
		Hostname:       "srv01.hostvra.io",
		IPAddress:      "127.0.0.1",
		AgentVersion:   "1.0.0",
		Status:         "online",
	})

	r := chi.NewRouter()
	// Middleware injecting test claims
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

	r.Route("/api/v1/websites", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/{id}", h.Get)
		r.Delete("/{id}", h.Delete)
		r.Get("/{id}/isolation", h.GetIsolation)
		r.Put("/{id}/isolation", h.UpdateIsolation)
	})

	// 1. Create Website with user isolation
	createReq := CreateWebsiteRequest{
		ServerID:      serverID.String(),
		PrimaryDomain: "superapp.dev",
		AppType:       "php",
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/websites", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on website creation, got %d: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Success bool          `json:"success"`
		Data    store.Website `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if res.Data.SystemUser == "" || res.Data.SystemUser == "www-data" {
		t.Errorf("expected isolated system user, got %s", res.Data.SystemUser)
	}
	if res.Data.SystemUser != "u_superapp_dev" {
		t.Errorf("expected u_superapp_dev, got %s", res.Data.SystemUser)
	}

	siteID := res.Data.ID.String()

	// 2. Query Isolation and Live cgroups v2 telemetry
	req = httptest.NewRequest("GET", "/api/v1/websites/"+siteID+"/isolation", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on get isolation, got %d: %s", rec.Code, rec.Body.String())
	}

	var isoRes struct {
		Success bool                        `json:"success"`
		Data    isolation.UserIsolationInfo `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&isoRes)
	if isoRes.Data.Username != "u_superapp_dev" {
		t.Errorf("expected username u_superapp_dev, got %s", isoRes.Data.Username)
	}
	if isoRes.Data.Limits.MemoryMaxMB != 512 {
		t.Errorf("expected default MemoryMaxMB 512, got %d", isoRes.Data.Limits.MemoryMaxMB)
	}
	if isoRes.Data.PHPPoolSocket == "" {
		t.Errorf("expected non-empty PHPPoolSocket")
	}

	// 3. Update Resource Limits (cgroups)
	updateLimits := isolation.ResourceLimits{
		MemoryMaxMB: 1024,
		CPUQuota:    200,
		TasksMax:    120,
		OpenBaseDir: true,
	}
	body, _ = json.Marshal(updateLimits)
	req = httptest.NewRequest("PUT", "/api/v1/websites/"+siteID+"/isolation", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on update isolation, got %d: %s", rec.Code, rec.Body.String())
	}

	var updatedIsoRes struct {
		Success bool                        `json:"success"`
		Data    isolation.UserIsolationInfo `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&updatedIsoRes)
	if updatedIsoRes.Data.Limits.MemoryMaxMB != 1024 {
		t.Errorf("expected MemoryMaxMB 1024, got %d", updatedIsoRes.Data.Limits.MemoryMaxMB)
	}
	if updatedIsoRes.Data.Limits.CPUQuota != 200 {
		t.Errorf("expected CPUQuota 200, got %d", updatedIsoRes.Data.Limits.CPUQuota)
	}

	// 4. Delete Website and verify cleanup
	req = httptest.NewRequest("DELETE", "/api/v1/websites/"+siteID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete website, got %d: %s", rec.Code, rec.Body.String())
	}
}
