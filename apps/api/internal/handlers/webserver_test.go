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
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func setupWebServerTestEnv(t *testing.T) (*WebServerHandler, store.Store, *auth.Claims, uuid.UUID, uuid.UUID) {
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)

	cfg := &config.Config{
		JWTSecret: "test-secret-at-least-32-bytes-long!",
	}

	handler := NewWebServerHandler(cfg, memStore, auditLogger)

	orgID := uuid.New()
	userID := uuid.New()
	serverID := uuid.New()

	// Seed server
	err := memStore.CreateServer(context.Background(), &store.Server{
		ID:             serverID,
		OrganizationID: orgID,
		Hostname:       "srv1.hostvra.internal",
		IPAddress:      "192.168.1.100",
		OSName:         "ubuntu",
		OSVersion:      "22.04",
		Status:         "online",
		CreatedAt:      time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("failed to seed server: %v", err)
	}

	claims := &auth.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Email:          "admin@hostvra.com",
		Role:           "super_admin",
	}

	return handler, memStore, claims, orgID, serverID
}

func TestListWebServersEndpoint(t *testing.T) {
	handler, _, claims, _, serverID := setupWebServerTestEnv(t)

	r := chi.NewRouter()
	r.Get("/servers/{serverID}/webservers", handler.ListServers)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/servers/%s/webservers", serverID), nil)
	ctx := context.WithValue(req.Context(), auth.UserContextKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var responseBody struct {
		Success bool                       `json:"success"`
		Data    []*store.WebServerInstance `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(responseBody.Data) != 4 {
		t.Errorf("expected 4 web servers (nginx, apache, openlitespeed, litespeed), got %d", len(responseBody.Data))
	}
}

func TestGetPortConflictsEndpoint(t *testing.T) {
	handler, _, claims, _, serverID := setupWebServerTestEnv(t)

	r := chi.NewRouter()
	r.Get("/servers/{serverID}/webservers/conflicts", handler.GetPortConflicts)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/servers/%s/webservers/conflicts", serverID), nil)
	ctx := context.WithValue(req.Context(), auth.UserContextKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestServiceActionEndpoint(t *testing.T) {
	handler, _, claims, _, serverID := setupWebServerTestEnv(t)

	r := chi.NewRouter()
	r.Post("/servers/{serverID}/webservers/{type}/service", handler.ServiceAction)

	// 1. Invalid action should return StatusConflict
	badBody, _ := json.Marshal(WebServerServiceActionRequest{Action: "invalid-action-xyz"})
	reqBad := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/servers/%s/webservers/nginx/service", serverID), bytes.NewReader(badBody))
	ctx := context.WithValue(reqBad.Context(), auth.UserContextKey, claims)
	reqBad = reqBad.WithContext(ctx)

	recBad := httptest.NewRecorder()
	r.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusConflict {
		t.Fatalf("expected status 409 Conflict for invalid action, got %d: %s", recBad.Code, recBad.Body.String())
	}

	// 2. Test syntax action (returns 200 if valid or 409 if nginx not found / config test fails in test env)
	testBody, _ := json.Marshal(WebServerServiceActionRequest{Action: "test"})
	reqTest := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/servers/%s/webservers/nginx/service", serverID), bytes.NewReader(testBody))
	reqTest = reqTest.WithContext(ctx)

	recTest := httptest.NewRecorder()
	r.ServeHTTP(recTest, reqTest)
	if recTest.Code != http.StatusOK && recTest.Code != http.StatusConflict {
		t.Fatalf("expected 200 OK or 409 Conflict, got %d: %s", recTest.Code, recTest.Body.String())
	}
}

func TestWebsiteWebServerIntegration(t *testing.T) {
	handler, memStore, claims, orgID, serverID := setupWebServerTestEnv(t)

	// Seed website
	siteID := uuid.New()
	err := memStore.CreateWebsite(context.Background(), &store.Website{
		ID:             siteID,
		OrganizationID: orgID,
		ServerID:       serverID,
		PrimaryDomain:  "testsite.hostvra.io",
		DocumentRoot:   "/var/www/testsite",
		AppType:        "php",
		WebServerType:  "nginx",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("failed to create website: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/websites/{id}/webserver", handler.GetWebsiteWebServer)
	r.Post("/websites/{id}/webserver/switch", handler.SwitchWebsiteWebServer)

	// 1. Inspect website webserver
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/websites/%s/webserver", siteID), nil)
	ctx := context.WithValue(req.Context(), auth.UserContextKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GetWebsiteWebServer failed with status %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Switch website to Apache
	switchPayload := SwitchWebsiteWebServerRequest{
		TargetServerType: "apache",
	}
	bodyBytes, _ := json.Marshal(switchPayload)
	reqSwitch := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/websites/%s/webserver/switch", siteID), bytes.NewReader(bodyBytes))
	reqSwitch = reqSwitch.WithContext(ctx)

	recSwitch := httptest.NewRecorder()
	r.ServeHTTP(recSwitch, reqSwitch)

	// In test environment, ApplyVHost might write to dev path or fail gracefully if /etc/apache2 not present
	// Either 200 (if writable) or 400 with graceful apply error
	if recSwitch.Code != http.StatusOK && recSwitch.Code != http.StatusBadRequest {
		t.Errorf("expected 200 or 400 for switch, got %d: %s", recSwitch.Code, recSwitch.Body.String())
	}
}
