package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestCronHandler_Lifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-api-cron-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storageFile := filepath.Join(tempDir, "test-crontab")
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}

	h := NewCronHandler(cfg, memStore, auditLogger)
	h.cronMgr.SetStoragePath(storageFile)

	claims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "admin@hostvra.com",
		Role:           "owner",
	}

	// 1. Get Daemon Status
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/v1/cron/status", nil)
	reqStatus = reqStatus.WithContext(context.WithValue(reqStatus.Context(), auth.UserContextKey, claims))
	recStatus := httptest.NewRecorder()
	h.GetStatus(recStatus, reqStatus)

	if recStatus.Code != http.StatusOK {
		t.Fatalf("expected 200 for GetStatus, got %d: %s", recStatus.Code, recStatus.Body.String())
	}

	// 2. Dangerous Command Rejection
	badJobBody, _ := json.Marshal(CreateCronJobRequest{
		Schedule:    "0 * * * *",
		Command:     "rm -rf /",
		SystemUser:  "root",
		Description: "Destructive task",
	})
	reqBadJob := httptest.NewRequest(http.MethodPost, "/api/v1/cron/jobs", bytes.NewReader(badJobBody))
	reqBadJob = reqBadJob.WithContext(context.WithValue(reqBadJob.Context(), auth.UserContextKey, claims))
	recBadJob := httptest.NewRecorder()
	h.CreateJob(recBadJob, reqBadJob)

	if recBadJob.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for dangerous command, got %d: %s", recBadJob.Code, recBadJob.Body.String())
	}

	// 3. Create Valid Job
	goodJobBody, _ := json.Marshal(CreateCronJobRequest{
		Schedule:    "*/5 * * * *",
		Command:     "echo 'hostvra test'",
		SystemUser:  "root",
		Description: "Test Echo Task",
	})
	reqGoodJob := httptest.NewRequest(http.MethodPost, "/api/v1/cron/jobs", bytes.NewReader(goodJobBody))
	reqGoodJob = reqGoodJob.WithContext(context.WithValue(reqGoodJob.Context(), auth.UserContextKey, claims))
	recGoodJob := httptest.NewRecorder()
	h.CreateJob(recGoodJob, reqGoodJob)

	if recGoodJob.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for valid job, got %d: %s", recGoodJob.Code, recGoodJob.Body.String())
	}

	var createdJob map[string]interface{}
	_ = json.NewDecoder(recGoodJob.Body).Decode(&createdJob)
	data, ok := createdJob["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid response format: %+v", createdJob)
	}
	jobID, _ := data["id"].(string)
	if jobID == "" {
		t.Fatalf("expected job ID to be set")
	}

	// 4. List Jobs
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/cron/jobs", nil)
	reqList = reqList.WithContext(context.WithValue(reqList.Context(), auth.UserContextKey, claims))
	recList := httptest.NewRecorder()
	h.ListJobs(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200 for ListJobs, got %d: %s", recList.Code, recList.Body.String())
	}

	// 5. Run Job Immediately
	r := chi.NewRouter()
	r.Post("/api/v1/cron/jobs/{id}/run", h.RunJob)
	reqRun := httptest.NewRequest(http.MethodPost, "/api/v1/cron/jobs/"+jobID+"/run", nil)
	reqRun = reqRun.WithContext(context.WithValue(reqRun.Context(), auth.UserContextKey, claims))
	recRun := httptest.NewRecorder()
	r.ServeHTTP(recRun, reqRun)

	if recRun.Code != http.StatusOK {
		t.Fatalf("expected 200 for RunJob, got %d: %s", recRun.Code, recRun.Body.String())
	}

	// 6. Toggle Job
	rToggle := chi.NewRouter()
	rToggle.Post("/api/v1/cron/jobs/{id}/toggle", h.ToggleJob)
	reqToggle := httptest.NewRequest(http.MethodPost, "/api/v1/cron/jobs/"+jobID+"/toggle", nil)
	reqToggle = reqToggle.WithContext(context.WithValue(reqToggle.Context(), auth.UserContextKey, claims))
	recToggle := httptest.NewRecorder()
	rToggle.ServeHTTP(recToggle, reqToggle)

	if recToggle.Code != http.StatusOK {
		t.Fatalf("expected 200 for ToggleJob, got %d: %s", recToggle.Code, recToggle.Body.String())
	}

	// 7. Delete Job
	rDel := chi.NewRouter()
	rDel.Delete("/api/v1/cron/jobs/{id}", h.DeleteJob)
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/v1/cron/jobs/"+jobID, nil)
	reqDel = reqDel.WithContext(context.WithValue(reqDel.Context(), auth.UserContextKey, claims))
	recDel := httptest.NewRecorder()
	rDel.ServeHTTP(recDel, reqDel)

	if recDel.Code != http.StatusOK {
		t.Fatalf("expected 200 for DeleteJob, got %d: %s", recDel.Code, recDel.Body.String())
	}
}

func TestCronHandler_NonAdminForbiddenFromRunningRootJob(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-cron-perm-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storageFile := filepath.Join(tempDir, "test-crontab")
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}

	h := NewCronHandler(cfg, memStore, auditLogger)
	h.cronMgr.SetStoragePath(storageFile)

	adminClaims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "admin@hostvra.com",
		Role:           "owner",
	}

	// Create root job as admin
	goodJobBody, _ := json.Marshal(CreateCronJobRequest{
		Schedule:    "0 1 * * *",
		Command:     "echo 'root maintenance'",
		SystemUser:  "root",
		Description: "Root system job",
	})
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/cron/jobs", bytes.NewReader(goodJobBody))
	reqCreate = reqCreate.WithContext(context.WithValue(reqCreate.Context(), auth.UserContextKey, adminClaims))
	recCreate := httptest.NewRecorder()
	h.CreateJob(recCreate, reqCreate)

	var createdJob map[string]interface{}
	_ = json.NewDecoder(recCreate.Body).Decode(&createdJob)
	data := createdJob["data"].(map[string]interface{})
	jobID := data["id"].(string)

	// Now try to run this root job as non-admin user
	devClaims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "developer@client.com",
		Role:           "developer", // non-admin
	}

	r := chi.NewRouter()
	r.Post("/api/v1/cron/jobs/{id}/run", h.RunJob)
	reqRun := httptest.NewRequest(http.MethodPost, "/api/v1/cron/jobs/"+jobID+"/run", nil)
	reqRun = reqRun.WithContext(context.WithValue(reqRun.Context(), auth.UserContextKey, devClaims))
	recRun := httptest.NewRecorder()
	r.ServeHTTP(recRun, reqRun)

	if recRun.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-admin running root cron job, got %d: %s", recRun.Code, recRun.Body.String())
	}

	// Non-admin attempting to delete root job must be forbidden
	rDel := chi.NewRouter()
	rDel.Delete("/api/v1/cron/jobs/{id}", h.DeleteJob)
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/v1/cron/jobs/"+jobID, nil)
	reqDel = reqDel.WithContext(context.WithValue(reqDel.Context(), auth.UserContextKey, devClaims))
	recDel := httptest.NewRecorder()
	rDel.ServeHTTP(recDel, reqDel)

	if recDel.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-admin deleting root cron job, got %d: %s", recDel.Code, recDel.Body.String())
	}

	// Non-admin attempting to toggle root job must be forbidden
	rToggle := chi.NewRouter()
	rToggle.Post("/api/v1/cron/jobs/{id}/toggle", h.ToggleJob)
	reqToggle := httptest.NewRequest(http.MethodPost, "/api/v1/cron/jobs/"+jobID+"/toggle", nil)
	reqToggle = reqToggle.WithContext(context.WithValue(reqToggle.Context(), auth.UserContextKey, devClaims))
	recToggle := httptest.NewRecorder()
	rToggle.ServeHTTP(recToggle, reqToggle)

	if recToggle.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-admin toggling root cron job, got %d: %s", recToggle.Code, recToggle.Body.String())
	}
}


