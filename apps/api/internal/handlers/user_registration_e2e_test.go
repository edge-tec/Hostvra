package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/handlers"
	"hostvra/api/internal/quota"
	"hostvra/api/internal/rbac"
	"hostvra/api/internal/store"
)

func setupTestRouter(memStore *store.MemoryStore, cfg *config.Config, quotaSvc *quota.Service) http.Handler {
	r := chi.NewRouter()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auditLogger := audit.NewLogger(memStore, logger)

	authHandler := handlers.NewAuthHandler(cfg, memStore, auditLogger)
	adminUsersHandler := handlers.NewAdminUsersHandler(memStore, quotaSvc, auditLogger)

	authMiddleware := auth.Middleware(cfg.JWTSecret)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		// User self-service plan
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/user/plan", adminUsersHandler.GetUserEffectivePlan)
		})

		// Admin only endpoints
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(rbac.RequireAdmin())

			r.Route("/admin/users", func(r chi.Router) {
				r.Get("/", adminUsersHandler.ListUsers)
				r.Get("/{id}", adminUsersHandler.GetUserDetails)
				r.Put("/{id}/plan", adminUsersHandler.UpdateUserPlan)
				r.Put("/{id}/overrides", adminUsersHandler.UpdateUserOverrides)
				r.Delete("/{id}/overrides", adminUsersHandler.DeleteUserOverrides)
				r.Put("/{id}/status", adminUsersHandler.UpdateUserStatus)
			})
		})
	})

	return r
}

func TestRegistrationCreatesCustomerAndBlocksAdminAccess(t *testing.T) {
	memStore := store.NewMemoryStore()
	cfg := &config.Config{
		JWTSecret:        "test-super-secret-jwt-key-minimum-32-chars-long!",
		JWTElementsHours: 24 * time.Hour,
		RefreshTokenDays: 7 * 24 * time.Hour,
	}
	quotaSvc := quota.NewService(memStore)
	router := setupTestRouter(memStore, cfg, quotaSvc)

	// 1. Register a brand new user
	regEmail := fmt.Sprintf("customer_%d@example.com", time.Now().UnixNano())
	regReq := handlers.RegisterRequest{
		Email:            regEmail,
		Password:         "SecureCustomerP@ss2026!",
		FullName:         "Jane Hosting Customer",
		OrganizationName: "Jane Enterprises",
	}
	body, _ := json.Marshal(regReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on registration, got %d: %s", w.Code, w.Body.String())
	}

	var regResp struct {
		Success bool `json:"success"`
		Data    struct {
			Tokens struct {
				AccessToken string `json:"access_token"`
			} `json:"tokens"`
			User struct {
				ID           string `json:"id"`
				Email        string `json:"email"`
				Role         string `json:"role"`
				IsSuperAdmin bool   `json:"is_superadmin"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &regResp); err != nil {
		t.Fatalf("failed to decode registration response: %v", err)
	}

	// VERIFY: The new user MUST have role 'customer', NOT 'owner' and NOT 'admin'
	if regResp.Data.User.Role != "customer" {
		t.Errorf("SECURITY FAILURE: Expected new user role to be 'customer', got '%s'", regResp.Data.User.Role)
	}
	if regResp.Data.User.IsSuperAdmin {
		t.Errorf("SECURITY FAILURE: Expected new user is_superadmin to be false")
	}

	customerToken := regResp.Data.Tokens.AccessToken
	if customerToken == "" {
		t.Fatalf("expected access token in registration response")
	}

	// 2. ATTEMPT PRIVILEGE ESCALATION: Customer tries to access /api/v1/admin/users
	adminReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	adminReq.Header.Set("Authorization", "Bearer "+customerToken)
	adminW := httptest.NewRecorder()

	router.ServeHTTP(adminW, adminReq)

	// MUST be 403 Forbidden!
	if adminW.Code != http.StatusForbidden {
		t.Fatalf("SECURITY FAILURE: Expected customer accessing /admin/users to get 403 Forbidden, got %d: %s", adminW.Code, adminW.Body.String())
	}

	// 3. Customer checks their own self plan (/api/v1/user/plan)
	planReq := httptest.NewRequest(http.MethodGet, "/api/v1/user/plan", nil)
	planReq.Header.Set("Authorization", "Bearer "+customerToken)
	planW := httptest.NewRecorder()

	router.ServeHTTP(planW, planReq)

	if planW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for user self plan, got %d: %s", planW.Code, planW.Body.String())
	}

	var planResp struct {
		Success bool                    `json:"success"`
		Data    store.EffectiveUserPlan `json:"data"`
	}
	if err := json.Unmarshal(planW.Body.Bytes(), &planResp); err != nil {
		t.Fatalf("failed to parse plan response: %v", err)
	}

	if planResp.Data.Role != "customer" {
		t.Errorf("expected plan role 'customer', got '%s'", planResp.Data.Role)
	}
	if planResp.Data.PlanSlug != "starter-cloud" {
		t.Errorf("expected default plan slug 'starter-cloud', got '%s'", planResp.Data.PlanSlug)
	}

	// 4. Seed an actual Admin user and verify Admin CAN access /api/v1/admin/users
	adminID := uuid.New()
	adminOrgID := uuid.New()
	adminPassHash, _ := auth.HashPassword("AdminSuperSecretP@ss123!", nil)
	adminUser := &store.User{
		ID:           adminID,
		Email:        "admin@hostvra.com",
		PasswordHash: adminPassHash,
		FullName:     "Hostvra Admin",
		IsActive:     true,
		IsSuperAdmin: true,
	}
	_ = memStore.CreateUser(context.Background(), adminUser, adminOrgID, "admin")

	tokens, _, err := auth.GenerateTokenPair(adminID, adminOrgID, adminUser.Email, "admin", true, cfg.JWTSecret, time.Hour, time.Hour*24)
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}
	adminToken := tokens.AccessToken

	adminSuccessReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	adminSuccessReq.Header.Set("Authorization", "Bearer "+adminToken)
	adminSuccessW := httptest.NewRecorder()

	router.ServeHTTP(adminSuccessW, adminSuccessReq)

	if adminSuccessW.Code != http.StatusOK {
		t.Fatalf("expected admin to get 200 OK, got %d: %s", adminSuccessW.Code, adminSuccessW.Body.String())
	}

	// 5. Admin updates Customer's Plan Overrides
	custUUID := uuid.MustParse(regResp.Data.User.ID)
	newWebsites := 10
	newTerminal := true
	overrideBody, _ := json.Marshal(store.UserPlanOverride{
		MaxWebsites:        &newWebsites,
		PermissionTerminal: &newTerminal,
		Notes:              "VIP Client upgrade",
	})
	putReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/admin/users/%s/overrides", custUUID), bytes.NewReader(overrideBody))
	putReq.Header.Set("Authorization", "Bearer "+adminToken)
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()

	router.ServeHTTP(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK updating overrides, got %d: %s", putW.Code, putW.Body.String())
	}

	// Customer re-checks their plan - it should now reflect the admin override!
	planW2 := httptest.NewRecorder()
	router.ServeHTTP(planW2, planReq)

	if planW2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", planW2.Code)
	}

	var planResp2 struct {
		Success bool                    `json:"success"`
		Data    store.EffectiveUserPlan `json:"data"`
	}
	_ = json.Unmarshal(planW2.Body.Bytes(), &planResp2)
	if planResp2.Data.MaxWebsites != 10 {
		t.Errorf("expected overridden max websites 10, got %d", planResp2.Data.MaxWebsites)
	}
	if !planResp2.Data.Permissions["terminal"] {
		t.Errorf("expected overridden permission terminal to be true")
	}

	// 6. Admin deletes override - reverts to package default
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/admin/users/%s/overrides", custUUID), nil)
	delReq.Header.Set("Authorization", "Bearer "+adminToken)
	delW := httptest.NewRecorder()

	router.ServeHTTP(delW, delReq)

	if delW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on delete overrides, got %d", delW.Code)
	}

	// Verify reverted
	planW3 := httptest.NewRecorder()
	router.ServeHTTP(planW3, planReq)
	var planResp3 struct {
		Success bool                    `json:"success"`
		Data    store.EffectiveUserPlan `json:"data"`
	}
	_ = json.Unmarshal(planW3.Body.Bytes(), &planResp3)
	if planResp3.Data.MaxWebsites != 1 {
		t.Errorf("expected reverted max websites 1, got %d", planResp3.Data.MaxWebsites)
	}
	if planResp3.Data.Permissions["terminal"] {
		t.Errorf("expected reverted permission terminal to be false")
	}
}
