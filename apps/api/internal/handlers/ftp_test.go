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

func TestFTPHandler_Lifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-api-ftp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	passwdFile := filepath.Join(tempDir, "pureftpd.passwd")
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}

	h := NewFTPHandler(cfg, memStore, auditLogger)
	h.ftpMgr.SetStoragePath(passwdFile)

	claims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "admin@hostvra.com",
		Role:           "owner",
	}

	// 1. Get Status
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/v1/ftp/status", nil)
	reqStatus = reqStatus.WithContext(context.WithValue(reqStatus.Context(), auth.UserContextKey, claims))
	recStatus := httptest.NewRecorder()
	h.GetStatus(recStatus, reqStatus)

	if recStatus.Code != http.StatusOK {
		t.Fatalf("expected 200 for GetStatus, got %d: %s", recStatus.Code, recStatus.Body.String())
	}

	// 2. Create User
	siteHome := filepath.Join(tempDir, "site1")
	userBody, _ := json.Marshal(CreateFTPUserRequest{
		Username:          "site1_user",
		Password:          "SecretPass123!",
		HomeDir:           siteHome,
		QuotaMB:           2048,
		UploadBandwidth:   500,
		DownloadBandwidth: 1000,
	})
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/ftp/users", bytes.NewReader(userBody))
	reqCreate = reqCreate.WithContext(context.WithValue(reqCreate.Context(), auth.UserContextKey, claims))
	recCreate := httptest.NewRecorder()
	h.CreateUser(recCreate, reqCreate)

	if recCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 for CreateUser, got %d: %s", recCreate.Code, recCreate.Body.String())
	}

	// 3. List Users
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/ftp/users", nil)
	reqList = reqList.WithContext(context.WithValue(reqList.Context(), auth.UserContextKey, claims))
	recList := httptest.NewRecorder()
	h.ListUsers(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200 for ListUsers, got %d: %s", recList.Code, recList.Body.String())
	}

	// 4. Change Password
	rPass := chi.NewRouter()
	rPass.Put("/api/v1/ftp/users/{username}/password", h.ChangePassword)
	passBody, _ := json.Marshal(UpdateFTPPasswordRequest{Password: "NewSecretPassword123!"})
	reqPass := httptest.NewRequest(http.MethodPut, "/api/v1/ftp/users/site1_user/password", bytes.NewReader(passBody))
	reqPass = reqPass.WithContext(context.WithValue(reqPass.Context(), auth.UserContextKey, claims))
	recPass := httptest.NewRecorder()
	rPass.ServeHTTP(recPass, reqPass)

	if recPass.Code != http.StatusOK {
		t.Fatalf("expected 200 for ChangePassword, got %d: %s", recPass.Code, recPass.Body.String())
	}

	// 5. Toggle User
	rToggle := chi.NewRouter()
	rToggle.Post("/api/v1/ftp/users/{username}/toggle", h.ToggleUser)
	reqToggle := httptest.NewRequest(http.MethodPost, "/api/v1/ftp/users/site1_user/toggle", nil)
	reqToggle = reqToggle.WithContext(context.WithValue(reqToggle.Context(), auth.UserContextKey, claims))
	recToggle := httptest.NewRecorder()
	rToggle.ServeHTTP(recToggle, reqToggle)

	if recToggle.Code != http.StatusOK {
		t.Fatalf("expected 200 for ToggleUser, got %d: %s", recToggle.Code, recToggle.Body.String())
	}

	// 6. Delete User
	rDel := chi.NewRouter()
	rDel.Delete("/api/v1/ftp/users/{username}", h.DeleteUser)
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/v1/ftp/users/site1_user", nil)
	reqDel = reqDel.WithContext(context.WithValue(reqDel.Context(), auth.UserContextKey, claims))
	recDel := httptest.NewRecorder()
	rDel.ServeHTTP(recDel, reqDel)

	if recDel.Code != http.StatusOK {
		t.Fatalf("expected 200 for DeleteUser, got %d: %s", recDel.Code, recDel.Body.String())
	}
}
