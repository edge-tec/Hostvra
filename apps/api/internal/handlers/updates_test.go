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

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
	"hostvra/api/internal/update"
)

func setupUpdateHandlerTest(t *testing.T) (*UpdateHandler, *auth.Claims) {
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)

	cfg := &config.Config{
		JWTSecret: "test-jwt-secret-at-least-32-chars-long",
		Port:      "8080",
	}

	handler := NewUpdateHandler(cfg, memStore, auditLogger, "1.0.0")

	claims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "owner@hostvra.com",
		Role:           "owner",
	}

	return handler, claims
}

func TestUpdateHandlerStatusAndCheck(t *testing.T) {
	handler, claims := setupUpdateHandlerTest(t)

	r := chi.NewRouter()
	r.Get("/system/updates/status", handler.GetStatus)
	r.Post("/system/updates/check", handler.CheckUpdates)

	// 1. Get Status
	req := httptest.NewRequest(http.MethodGet, "/system/updates/status", nil)
	ctx := context.WithValue(req.Context(), auth.UserContextKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Success bool `json:"success"`
		Data    struct {
			System struct {
				APIVersion      string `json:"api_version"`
				UpdateAvailable bool   `json:"update_available"`
			} `json:"system"`
			LatestRelease struct {
				Version string `json:"version"`
			} `json:"latest_release"`
		} `json:"data"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Data.System.APIVersion != "1.0.0" {
		t.Errorf("expected API version 1.0.0, got %s", res.Data.System.APIVersion)
	}
	if !res.Data.System.UpdateAvailable {
		t.Errorf("expected update available to be true")
	}
	if res.Data.LatestRelease.Version != "1.1.0" {
		t.Errorf("expected latest version 1.1.0, got %s", res.Data.LatestRelease.Version)
	}
}

func TestUpdateHandlerChannelSwitch(t *testing.T) {
	handler, claims := setupUpdateHandlerTest(t)

	r := chi.NewRouter()
	r.Put("/system/updates/channel", handler.SetChannel)

	body, _ := json.Marshal(SetChannelRequest{Channel: "beta"})
	req := httptest.NewRequest(http.MethodPut, "/system/updates/channel", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), auth.UserContextKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if handler.channel != update.ChannelBeta {
		t.Errorf("expected channel to switch to beta, got %s", handler.channel)
	}
}

func TestUpdateHandlerStartJob(t *testing.T) {
	handler, claims := setupUpdateHandlerTest(t)

	r := chi.NewRouter()
	r.Post("/system/updates/start", handler.StartUpdate)
	r.Get("/system/updates/jobs/{id}", handler.GetJobStatus)

	body, _ := json.Marshal(StartUpdateRequest{TargetVersion: "1.1.0"})
	req := httptest.NewRequest(http.MethodPost, "/system/updates/start", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), auth.UserContextKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Success bool             `json:"success"`
		Data    update.UpdateJob `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Data.TargetVersion != "1.1.0" {
		t.Errorf("expected target version 1.1.0, got %s", res.Data.TargetVersion)
	}

	// Query job status
	reqGet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/system/updates/jobs/%s", res.Data.ID), nil)
	reqGet = reqGet.WithContext(ctx)
	recGet := httptest.NewRecorder()
	r.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for job lookup, got %d", recGet.Code)
	}
}

func TestUpdateHandlerRollback(t *testing.T) {
	handler, claims := setupUpdateHandlerTest(t)

	r := chi.NewRouter()
	r.Post("/system/updates/rollback", handler.TriggerRollback)

	req := httptest.NewRequest(http.MethodPost, "/system/updates/rollback", nil)
	ctx := context.WithValue(req.Context(), auth.UserContextKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for rollback, got %d: %s", rec.Code, rec.Body.String())
	}
}
