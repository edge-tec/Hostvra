package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestDatabaseHandler_ExportValidation(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	st := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	aud := audit.NewLogger(st, logger)
	h := NewDatabaseHandler(cfg, st, aud)

	// Test Export without database name -> 400 Bad Request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/export", nil)
	w := httptest.NewRecorder()

	h.Export(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on empty db name, got %d", w.Code)
	}

	// Test GetTables with empty db -> 400 Bad Request
	reqTables := httptest.NewRequest(http.MethodGet, "/api/v1/databases/tables", nil)
	wTables := httptest.NewRecorder()
	h.GetTables(wTables, reqTables)
	if wTables.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on empty db name for tables, got %d", wTables.Code)
	}

	// Test GetRootPassword without admin -> 403 Forbidden
	reqRootAnon := httptest.NewRequest(http.MethodGet, "/api/v1/databases/root-password", nil)
	wRootAnon := httptest.NewRecorder()
	h.GetRootPassword(wRootAnon, reqRootAnon)
	if wRootAnon.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-admin root password request, got %d", wRootAnon.Code)
	}

	// Test GetRootPassword with admin claims -> 200 OK
	adminClaims := &auth.Claims{
		Role:           "admin",
		OrganizationID: uuid.New(),
	}
	ctx := context.WithValue(context.Background(), auth.UserContextKey, adminClaims)
	reqRoot := httptest.NewRequest(http.MethodGet, "/api/v1/databases/root-password", nil).WithContext(ctx)
	wRoot := httptest.NewRecorder()
	h.GetRootPassword(wRoot, reqRoot)
	if wRoot.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for admin root password, got %d", wRoot.Code)
	}
}

func TestDatabaseHandler_CustomerTenantDatabaseAndUserCreation(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	st := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	aud := audit.NewLogger(st, logger)
	h := NewDatabaseHandler(cfg, st, aud)

	// Seed cluster primary server under defaultOrgID
	defaultOrgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	primaryServerID := defaultOrgID
	_ = st.CreateServer(context.Background(), &store.Server{
		ID:             primaryServerID,
		OrganizationID: defaultOrgID,
		Name:           "Primary Node",
		Hostname:       "localhost",
		IPAddress:      "127.0.0.1",
		Status:         "online",
	})

	// Customer tenant claims (no dedicated server in their org)
	customerOrgID := uuid.New()
	customerClaims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: customerOrgID,
		Role:           "customer",
	}
	ctx := context.WithValue(context.Background(), auth.UserContextKey, customerClaims)

	// 1. Customer creates a database with user
	body := strings.NewReader(`{
		"name": "ewywy35",
		"username": "ewywy35",
		"password": "SecurePassword123!",
		"character_set": "utf8mb4",
		"collation": "utf8mb4_unicode_ci",
		"host_allow": "localhost"
	}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/databases", body).WithContext(ctx)
	createRec := httptest.NewRecorder()
	h.Create(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for customer database creation, got %d: %s", createRec.Code, createRec.Body.String())
	}

	// 2. Verify database was saved in store under customer org
	dbs, err := st.ListDatabasesByServer(context.Background(), primaryServerID)
	if err != nil || len(dbs) == 0 {
		t.Fatalf("expected database to be created in store, got len=%d, err=%v", len(dbs), err)
	}
	var found bool
	for _, d := range dbs {
		if d.Name == "ewywy35" && d.OrganizationID == customerOrgID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected database 'ewywy35' under customer org, not found")
	}

	// 3. Verify user was preserved in database_users
	users, err := st.ListDatabaseUsersByServer(context.Background(), primaryServerID)
	if err != nil || len(users) == 0 {
		t.Fatalf("expected user to be created in store, got len=%d, err=%v", len(users), err)
	}
	var foundUser bool
	for _, u := range users {
		if u.Username == "ewywy35" {
			foundUser = true
			break
		}
	}
	if !foundUser {
		t.Fatalf("expected database user 'ewywy35' in database_users, not found")
	}

	// 4. Customer lists databases -> should see their database
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/databases", nil).WithContext(ctx)
	listRec := httptest.NewRecorder()
	h.List(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on list databases, got %d", listRec.Code)
	}
	if !strings.Contains(listRec.Body.String(), "ewywy35") {
		t.Fatalf("expected customer database list to contain 'ewywy35', got: %s", listRec.Body.String())
	}
}
