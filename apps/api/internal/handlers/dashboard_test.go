package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestDashboardOverview(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-key-12345678901234567890",
	}
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)

	h := NewDashboardHandler(cfg, memStore, auditLogger)

	orgID := uuid.New()
	_ = memStore.CreateOrganization(context.Background(), &store.Organization{
		ID:       orgID,
		Name:     "Test Org",
		Slug:     "test-org",
		PlanTier: "pro",
	})

	// Add test website
	_ = memStore.CreateWebsite(context.Background(), &store.Website{
		ID:             uuid.New(),
		OrganizationID: orgID,
		PrimaryDomain:  "test-domain.com",
		DocumentRoot:   "/var/www/test",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/overview", nil)
	// Inject claims into context
	claims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: orgID,
		Email:          "admin@hostvra.com",
		Role:           "owner",
	}
	ctx := context.WithValue(req.Context(), auth.UserContextKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.GetOverview(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Success bool `json:"success"`
		Data    DashboardOverviewResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected success to be true")
	}

	// Verify telemetry has values
	if res.Data.Telemetry.CPU.Cores == 0 {
		t.Errorf("expected CPU cores > 0")
	}
	if res.Data.Telemetry.RAM.TotalMB == 0 {
		t.Errorf("expected RAM Total > 0")
	}
	if res.Data.Counts.WebsitesTotal < 1 {
		t.Errorf("expected WebsitesTotal >= 1, got %d", res.Data.Counts.WebsitesTotal)
	}
}

func TestDashboardRunFix(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-key-12345678901234567890",
	}
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)

	h := NewDashboardHandler(cfg, memStore, auditLogger)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/fix", nil)
	rec := httptest.NewRecorder()

	h.RunFix(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res struct {
		Success bool              `json:"success"`
		Data    SystemFixResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !res.Data.Success {
		t.Errorf("expected Fix success to be true")
	}
	if len(res.Data.Logs) == 0 {
		t.Errorf("expected diagnostic logs")
	}
}
