package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
