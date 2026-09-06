package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestAdminLoginSuccess(t *testing.T) {
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auditLogger := audit.NewLogger(memStore, logger)

	cfg := &config.Config{
		JWTSecret:         "super-secret-jwt-key-for-test-purposes-only-32chars!",
		JWTElementsHours:  24,
		RefreshTokenDays: 7,
	}

	// Seed default admin
	defaultOrgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	org := &store.Organization{
		ID:          defaultOrgID,
		Name:        "Hostvra Cloud",
		Slug:        "hostvra-cloud",
		PlanTier:    "enterprise",
		MaxServers:  100,
		MaxWebsites: 1000,
	}
	if err := memStore.CreateOrganization(context.Background(), org); err != nil {
		t.Fatalf("failed to create default org: %v", err)
	}

	passwordHash, err := auth.HashPassword("SuperSecretP@ss123!", nil)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	adminUser := &store.User{
		ID:           uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		Email:        "admin@hostvra.com",
		PasswordHash: passwordHash,
		FullName:     "Hostvra Administrator",
		IsActive:     true,
		IsSuperAdmin: true,
	}
	if err := memStore.CreateUser(context.Background(), adminUser, defaultOrgID, "owner"); err != nil {
		t.Fatalf("failed to seed admin user: %v", err)
	}

	authHandler := NewAuthHandler(cfg, memStore, auditLogger)

	loginReq := LoginRequest{
		Email:    "admin@hostvra.com",
		Password: "SuperSecretP@ss123!",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	authHandler.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Tokens struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
			} `json:"tokens"`
			User struct {
				Email        string `json:"email"`
				IsSuperAdmin bool   `json:"is_superadmin"`
			} `json:"user"`
		} `json:"data"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}

	if !resp.Success {
		t.Fatalf("expected success to be true, got %v", resp.Success)
	}
	if resp.Data.User.Email != "admin@hostvra.com" {
		t.Fatalf("expected email admin@hostvra.com, got %s", resp.Data.User.Email)
	}
	if !resp.Data.User.IsSuperAdmin {
		t.Fatalf("expected is_super_admin to be true")
	}
	if resp.Data.Tokens.AccessToken == "" {
		t.Fatalf("expected non-empty access token")
	}
}
