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
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestSecurityAudit_PathTraversalAndMultiTenantIsolation(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-at-least-32-bytes-long-12345",
	}
	memStore := store.NewMemoryStore()
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, slogger)
	handler := NewFileHandler(cfg, memStore, auditLogger)

	orgA := uuid.New()
	orgB := uuid.New()

	// Register a website for Org A
	siteA := &store.Website{
		ID:             uuid.New(),
		ServerID:       uuid.New(),
		OrganizationID: orgA,
		PrimaryDomain:  "tenant-a.com",
		DocumentRoot:   "/var/www/tenant-a.com/public_html",
		Status:         "active",
		CreatedAt:      time.Now(),
	}
	_ = memStore.CreateWebsite(context.Background(), siteA)

	// User from Org A (non-admin: developer role)
	devUserClaims := &auth.Claims{
		UserID:         uuid.New(),
		Email:          "dev@tenant-a.com",
		OrganizationID: orgA,
		Role:           "developer",
	}
	devCtx := context.WithValue(context.Background(), auth.UserContextKey, devUserClaims)

	// 1. Dev tries to access restricted root system file
	reqRoot := httptest.NewRequest("GET", "/files/content?path=/etc/shadow", nil).WithContext(devCtx)
	err := handler.checkPathAuthorization(reqRoot, "/etc/shadow")
	if err == nil {
		t.Fatalf("Expected checkPathAuthorization to block /etc/shadow for developer, but allowed")
	}

	// 2. Dev tries to access path outside their tenant domains under /var/www (e.g. Org B)
	reqVictim := httptest.NewRequest("GET", "/files/content?path=/var/www/victim-tenant-b.com/index.php", nil).WithContext(devCtx)
	err = handler.checkPathAuthorization(reqVictim, "/var/www/victim-tenant-b.com/index.php")
	if err == nil {
		t.Fatalf("Expected checkPathAuthorization to block cross-tenant /var/www access, but allowed")
	}

	// 3. Dev tries to browse top-level /var/www
	reqVarWww := httptest.NewRequest("GET", "/files/list?path=/var/www", nil).WithContext(devCtx)
	err = handler.checkPathAuthorization(reqVarWww, "/var/www")
	if err == nil {
		t.Fatalf("Expected checkPathAuthorization to block root /var/www browsing, but allowed")
	}

	// 4. Missing claims fails closed
	reqNoAuth := httptest.NewRequest("GET", "/files/list?path=/var/www", nil)
	err = handler.checkPathAuthorization(reqNoAuth, "/var/www")
	if err == nil {
		t.Fatalf("Expected checkPathAuthorization to fail-closed on missing claims, but allowed")
	}

	// 5. Dev accessing their own domain's document root should succeed
	reqOwnSite := httptest.NewRequest("GET", "/files/list?path=/var/www/tenant-a.com/public_html", nil).WithContext(devCtx)
	err = handler.checkPathAuthorization(reqOwnSite, "/var/www/tenant-a.com/public_html")
	if err != nil {
		t.Fatalf("Expected checkPathAuthorization to permit own website document root, got: %v", err)
	}

	// 6. Verify ListDomains only returns domains belonging to caller org for non-admin
	siteB := &store.Website{
		ID:             uuid.New(),
		ServerID:       uuid.New(),
		OrganizationID: orgB,
		PrimaryDomain:  "tenant-b.com",
		DocumentRoot:   "/var/www/tenant-b.com/public_html",
		Status:         "active",
		CreatedAt:      time.Now(),
	}
	_ = memStore.CreateWebsite(context.Background(), siteB)

	rec := httptest.NewRecorder()
	handler.ListDomains(rec, reqOwnSite)
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected ListDomains to return 200, got %d", rec.Code)
	}
	var envelope struct {
		Success bool                     `json:"success"`
		Data    []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("Failed to decode ListDomains response: %v", err)
	}
	for _, d := range envelope.Data {
		if d["domain"] == "tenant-b.com" {
			t.Fatalf("Security Violation: ListDomains leaked tenant-b.com to developer from org A")
		}
	}
}

func TestSecurityAudit_DatabaseInputValidation(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-at-least-32-bytes-long-12345",
	}
	memStore := store.NewMemoryStore()
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, slogger)
	handler := NewDatabaseHandler(cfg, memStore, auditLogger)

	adminClaims := &auth.Claims{
		UserID: uuid.New(),
		Email:  "admin@hostvra.com",
		Role:   "admin",
	}
	adminCtx := context.WithValue(context.Background(), auth.UserContextKey, adminClaims)

	// Test 1: Injection payload in database name
	maliciousPayload := CreateDatabaseRequest{
		Name: "test_db`; DROP TABLE users; --",
	}
	body, _ := json.Marshal(maliciousPayload)
	req := httptest.NewRequest("POST", "/databases", bytes.NewReader(body)).WithContext(adminCtx)
	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for SQL injection in db name, got %d", rec.Code)
	}

	// Test 2: Valid database name
	validPayload := CreateDatabaseRequest{
		Name:         "valid_company_db",
		CharacterSet: "utf8mb4",
		Collation:    "utf8mb4_unicode_ci",
	}
	body, _ = json.Marshal(validPayload)
	req = httptest.NewRequest("POST", "/databases", bytes.NewReader(body)).WithContext(adminCtx)
	rec = httptest.NewRecorder()
	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for valid database name, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSecurityAudit_TerminalCatastrophicCommandBlocked(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-at-least-32-bytes-long-12345",
	}
	memStore := store.NewMemoryStore()
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, slogger)
	handler := NewTerminalHandler(cfg, memStore, auditLogger)

	adminClaims := &auth.Claims{
		UserID: uuid.New(),
		Email:  "admin@hostvra.com",
		Role:   "admin",
	}
	adminCtx := context.WithValue(context.Background(), auth.UserContextKey, adminClaims)

	// Block catastrophic command
	catastrophicCmd := ExecuteCommandRequest{
		Command: "rm -rf / --no-preserve-root",
	}
	body, _ := json.Marshal(catastrophicCmd)
	req := httptest.NewRequest("POST", "/terminal/execute", bytes.NewReader(body)).WithContext(adminCtx)
	rec := httptest.NewRecorder()
	handler.Execute(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for catastrophic command, got %d", rec.Code)
	}

	// Non-admin role blocked
	devClaims := &auth.Claims{
		UserID: uuid.New(),
		Email:  "dev@tenant.com",
		Role:   "developer",
	}
	devCtx := context.WithValue(context.Background(), auth.UserContextKey, devClaims)
	safeCmd := ExecuteCommandRequest{Command: "uptime"}
	body, _ = json.Marshal(safeCmd)
	reqDev := httptest.NewRequest("POST", "/terminal/execute", bytes.NewReader(body)).WithContext(devCtx)
	recDev := httptest.NewRecorder()
	handler.Execute(recDev, reqDev)

	if recDev.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden for non-admin terminal execution, got %d", recDev.Code)
	}
}
