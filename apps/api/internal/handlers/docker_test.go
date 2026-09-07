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

func TestDockerHandler_LifecycleAndStatus(t *testing.T) {
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}

	h := NewDockerHandler(cfg, memStore, auditLogger)

	claims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "admin@hostvra.com",
		Role:           "owner",
	}

	// 1. Get Status (should succeed and report installed=false/true without crashing)
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/v1/docker/status", nil)
	reqStatus = reqStatus.WithContext(context.WithValue(reqStatus.Context(), auth.UserContextKey, claims))
	recStatus := httptest.NewRecorder()
	h.GetStatus(recStatus, reqStatus)

	if recStatus.Code != http.StatusOK {
		t.Fatalf("expected 200 for GetStatus, got %d: %s", recStatus.Code, recStatus.Body.String())
	}

	// 2. List Containers (should succeed even if empty)
	reqContainers := httptest.NewRequest(http.MethodGet, "/api/v1/docker/containers?all=true", nil)
	reqContainers = reqContainers.WithContext(context.WithValue(reqContainers.Context(), auth.UserContextKey, claims))
	recContainers := httptest.NewRecorder()
	h.ListContainers(recContainers, reqContainers)

	if recContainers.Code != http.StatusOK {
		t.Fatalf("expected 200 for ListContainers, got %d: %s", recContainers.Code, recContainers.Body.String())
	}

	// 3. List Images
	reqImages := httptest.NewRequest(http.MethodGet, "/api/v1/docker/images", nil)
	reqImages = reqImages.WithContext(context.WithValue(reqImages.Context(), auth.UserContextKey, claims))
	recImages := httptest.NewRecorder()
	h.ListImages(recImages, reqImages)

	if recImages.Code != http.StatusOK {
		t.Fatalf("expected 200 for ListImages, got %d: %s", recImages.Code, recImages.Body.String())
	}
}

func TestDockerHandler_NonAdminForbiddenFromMountingVolumesAndPrune(t *testing.T) {
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}

	h := NewDockerHandler(cfg, memStore, auditLogger)

	devClaims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "developer@client.com",
		Role:           "developer", // non-admin
	}

	// 1. Run container with volume mount as developer must be forbidden
	runBody := `{"image": "redis:alpine", "volume_mounts": ["/home/victim:/data"]}`
	reqRun := httptest.NewRequest(http.MethodPost, "/api/v1/docker/containers/run", strings.NewReader(runBody))
	reqRun = reqRun.WithContext(context.WithValue(reqRun.Context(), auth.UserContextKey, devClaims))
	recRun := httptest.NewRecorder()
	h.RunContainer(recRun, reqRun)

	if recRun.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for non-admin mounting volumes, got %d: %s", recRun.Code, recRun.Body.String())
	}

	// 2. Prune as developer must be forbidden
	reqPrune := httptest.NewRequest(http.MethodPost, "/api/v1/docker/prune", nil)
	reqPrune = reqPrune.WithContext(context.WithValue(reqPrune.Context(), auth.UserContextKey, devClaims))
	recPrune := httptest.NewRecorder()
	h.PruneSystem(recPrune, reqPrune)

	if recPrune.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for non-admin pruning system, got %d: %s", recPrune.Code, recPrune.Body.String())
	}
}

