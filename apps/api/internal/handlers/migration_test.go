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
	"hostvra/api/internal/migration"
	"hostvra/api/internal/store"
)

func TestMigrationHandler_E2E(t *testing.T) {
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)
	mgr := migration.NewManager()
	mgr.SetStore(memStore)

	handler := NewMigrationHandler(mgr, auditLogger)

	claims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "admin@hostvra.internal",
		Role:           "superadmin",
		IsSuperAdmin:   true,
	}

	// 1. Create Migration Job
	reqPayload := migration.CreateMigrationRequest{
		SourceType: "cpanel",
		SourceHost: "127.0.0.1",
		SourcePort: 22,
		AuthType:   "password",
		Username:   "root",
		Password:   "secret",
		MockMode:   true,
	}
	bodyBytes, _ := json.Marshal(reqPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/migration/jobs", bytes.NewReader(bodyBytes))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	rec := httptest.NewRecorder()

	handler.CreateJob(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 StatusAccepted, got %d: %s", rec.Code, rec.Body.String())
	}

	var createResp struct {
		Data *migration.MigrationJob `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&createResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	jobID := createResp.Data.ID
	if jobID == uuid.Nil {
		t.Fatalf("expected valid UUID, got nil")
	}

	// 2. List Jobs
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/migration/jobs", nil)
	listReq = listReq.WithContext(auth.WithClaims(listReq.Context(), claims))
	listRec := httptest.NewRecorder()

	handler.ListJobs(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", listRec.Code, listRec.Body.String())
	}

	// 3. Get Job
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/migration/jobs/"+jobID.String(), nil)
	getReq = getReq.WithContext(auth.WithClaims(getReq.Context(), claims))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", jobID.String())
	getReq = getReq.WithContext(context.WithValue(getReq.Context(), chi.RouteCtxKey, rctx))
	getRec := httptest.NewRecorder()

	handler.GetJob(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", getRec.Code, getRec.Body.String())
	}

	// 4. Cancel Job
	cancelReq := httptest.NewRequest(http.MethodPost, "/api/v1/migration/jobs/"+jobID.String()+"/cancel", nil)
	cancelReq = cancelReq.WithContext(auth.WithClaims(cancelReq.Context(), claims))
	cancelReq = cancelReq.WithContext(context.WithValue(cancelReq.Context(), chi.RouteCtxKey, rctx))
	cancelRec := httptest.NewRecorder()

	handler.CancelJob(cancelRec, cancelReq)
	if cancelRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on cancel, got %d: %s", cancelRec.Code, cancelRec.Body.String())
	}
}
