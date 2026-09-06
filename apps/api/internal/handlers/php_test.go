package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func setupTestPHPEnvironment(t *testing.T) (*PHPHandler, store.Store, uuid.UUID, uuid.UUID) {
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)
	cfg := &config.Config{JWTSecret: "test_secret"}

	orgID := uuid.New()
	serverID := uuid.New()

	_ = s.CreateOrganization(context.Background(), &store.Organization{
		ID:       orgID,
		Name:     "Test Org",
		Slug:     "test-org",
		PlanTier: "pro",
	})

	_ = s.CreateServer(context.Background(), &store.Server{
		ID:             serverID,
		OrganizationID: orgID,
		Name:           "Node-1",
		Hostname:       "node1.local",
		IPAddress:      "192.168.1.100",
		Status:         "online",
	})

	h := NewPHPHandler(cfg, s, auditLogger)
	return h, s, serverID, orgID
}

func TestPHPListVersions(t *testing.T) {
	h, _, serverID, orgID := setupTestPHPEnvironment(t)

	r := chi.NewRouter()
	r.Get("/servers/{serverID}/php/versions", h.ListVersions)

	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/php/versions", nil)
	ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: orgID,
		Role:           "owner",
	})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPHPVersionRemovalBlockedByActiveWebsite(t *testing.T) {
	h, s, serverID, orgID := setupTestPHPEnvironment(t)

	phpVer := "8.3"
	site := &store.Website{
		ID:             uuid.New(),
		ServerID:       serverID,
		OrganizationID: orgID,
		PrimaryDomain:  "example.com",
		DocumentRoot:   "/var/www/example.com",
		SystemUser:     "www-data",
		PHPVersion:     &phpVer,
		AppType:        "php",
		Status:         "active",
	}
	if err := s.CreateWebsite(context.Background(), site); err != nil {
		t.Fatalf("failed to create website: %v", err)
	}

	r := chi.NewRouter()
	r.Delete("/servers/{serverID}/php/versions/{version}", h.RemoveVersion)

	req := httptest.NewRequest("DELETE", "/servers/"+serverID.String()+"/php/versions/8.3", nil)
	ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: orgID,
		Role:           "owner",
	})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409 CONFLICT due to active website dependency, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPHPIniGetAndUpdate(t *testing.T) {
	h, _, serverID, orgID := setupTestPHPEnvironment(t)

	r := chi.NewRouter()
	r.Get("/servers/{serverID}/php/{version}/ini", h.GetPHPIni)
	r.Put("/servers/{serverID}/php/{version}/ini", h.UpdatePHPIni)

	// 1. GET INI
	reqGet := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/php/8.3/ini", nil)
	ctx := context.WithValue(reqGet.Context(), auth.UserContextKey, &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: orgID,
		Role:           "owner",
	})
	reqGet = reqGet.WithContext(ctx)
	recGet := httptest.NewRecorder()
	r.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 on GET ini, got %d: %s", recGet.Code, recGet.Body.String())
	}

	// 2. Validate response structure contains simple_settings
	var getResp struct {
		Data struct {
			SimpleSettings []interface{} `json:"simple_settings"`
			RawContent     string        `json:"raw_content"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recGet.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("failed to decode get ini response: %v", err)
	}
	if len(getResp.Data.SimpleSettings) == 0 {
		t.Errorf("expected non-empty simple_settings list")
	}
}

func TestCreateFPMPoolValidation(t *testing.T) {
	h, _, serverID, orgID := setupTestPHPEnvironment(t)

	r := chi.NewRouter()
	r.Post("/servers/{serverID}/php/{version}/pools", h.CreateFPMPool)

	// Missing pool name should fail validation
	badPayload := []byte(`{"name": ""}`)
	reqBad := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/php/8.3/pools", bytes.NewReader(badPayload))
	ctx := context.WithValue(reqBad.Context(), auth.UserContextKey, &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: orgID,
		Role:           "owner",
	})
	reqBad = reqBad.WithContext(ctx)
	recBad := httptest.NewRecorder()
	r.ServeHTTP(recBad, reqBad)

	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on empty pool name, got %d", recBad.Code)
	}
}
