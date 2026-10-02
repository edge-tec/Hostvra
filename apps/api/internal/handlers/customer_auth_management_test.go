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
	"hostvra/api/internal/email"
	"hostvra/api/internal/handlers"
	"hostvra/api/internal/quota"
	"hostvra/api/internal/rbac"
	"hostvra/api/internal/store"
)

func setupCustomerAuthRouter(memStore *store.MemoryStore, cfg *config.Config, quotaSvc *quota.Service, emailSvc *email.Service) http.Handler {
	r := chi.NewRouter()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auditLogger := audit.NewLogger(memStore, logger)

	authHandler := handlers.NewAuthHandler(cfg, memStore, auditLogger, emailSvc)
	adminUsersHandler := handlers.NewAdminUsersHandler(cfg, memStore, quotaSvc, auditLogger, emailSvc)

	authMiddleware := auth.Middleware(cfg.JWTSecret)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/forgot-password", authHandler.ForgotPassword)
			r.Post("/reset-password", authHandler.ResetPassword)
			r.Get("/verify-email", authHandler.VerifyEmail)
			r.Post("/verify-email", authHandler.VerifyEmail)
			r.Post("/resend-verification", authHandler.ResendVerification)

			r.Group(func(r chi.Router) {
				r.Use(authMiddleware)
				r.Get("/me", authHandler.Me)
				r.Post("/change-password", authHandler.ChangePassword)
				r.Post("/change-email", authHandler.ChangeEmail)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/user/plan", adminUsersHandler.GetUserEffectivePlan)
		})

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(rbac.RequireAdmin())

			r.Route("/admin/users", func(r chi.Router) {
				r.Get("/", adminUsersHandler.ListUsers)
				r.Get("/{id}", adminUsersHandler.GetUserDetails)
				r.Put("/{id}/email", adminUsersHandler.AdminUpdateEmail)
				r.Put("/{id}/password", adminUsersHandler.AdminUpdatePassword)
				r.Post("/{id}/impersonate", adminUsersHandler.ImpersonateUser)
			})
		})
	})

	return r
}

func TestCustomerAuth_LoginEmailPassword(t *testing.T) {
	memStore := store.NewMemoryStore()
	cfg := &config.Config{
		JWTSecret:        "test-super-secret-jwt-key-minimum-32-chars-long!",
		JWTElementsHours: 24 * time.Hour,
		RefreshTokenDays: 7 * 24 * time.Hour,
	}
	quotaSvc := quota.NewService(memStore)
	emailSvc := email.New(cfg, memStore)
	router := setupCustomerAuthRouter(memStore, cfg, quotaSvc, emailSvc)

	// 1. Register customer
	regPayload := map[string]string{
		"email":             "customer@example.com",
		"password":          "SecurePassword123!",
		"full_name":         "John Doe",
		"organization_name": "Acme Corp",
	}
	body, _ := json.Marshal(regPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on register, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Login with valid password
	loginPayload := map[string]string{
		"email":    "customer@example.com",
		"password": "SecurePassword123!",
	}
	body, _ = json.Marshal(loginPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on login, got %d: %s", w.Code, w.Body.String())
	}

	var loginResp struct {
		Success bool `json:"success"`
		Data    struct {
			Tokens struct {
				AccessToken string `json:"access_token"`
			} `json:"tokens"`
			Role string `json:"role"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	if loginResp.Data.Role != "customer" {
		t.Errorf("expected role 'customer', got '%s'", loginResp.Data.Role)
	}

	// 3. Login with invalid password
	loginBad := map[string]string{
		"email":    "customer@example.com",
		"password": "WrongPassword123!",
	}
	body, _ = json.Marshal(loginBad)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for invalid password, got %d", w.Code)
	}
}

func TestCustomerAuth_AdminEmailPasswordUpdate(t *testing.T) {
	memStore := store.NewMemoryStore()
	cfg := &config.Config{
		JWTSecret:        "test-super-secret-jwt-key-minimum-32-chars-long!",
		JWTElementsHours: 24 * time.Hour,
		RefreshTokenDays: 7 * 24 * time.Hour,
	}
	quotaSvc := quota.NewService(memStore)
	emailSvc := email.New(cfg, memStore)
	router := setupCustomerAuthRouter(memStore, cfg, quotaSvc, emailSvc)

	// Create Admin User
	adminID := uuid.New()
	adminOrgID := uuid.New()
	adminHash, _ := auth.HashPassword("AdminPassword123!", nil)
	adminUser := &store.User{
		ID:            adminID,
		Email:         "admin@hostvra.com",
		PasswordHash:  adminHash,
		FullName:      "Super Admin",
		IsActive:      true,
		IsSuperAdmin:  true,
		Role:          "admin",
		DefaultOrgID:  adminOrgID,
	}
	_ = memStore.CreateUser(context.Background(), adminUser, adminOrgID, "admin")
	adminTokens, _, _ := auth.GenerateTokenPair(adminID, adminOrgID, adminUser.Email, "admin", true, cfg.JWTSecret, cfg.JWTElementsHours, cfg.RefreshTokenDays)

	// Create Customer User
	custID := uuid.New()
	custOrgID := uuid.New()
	custHash, _ := auth.HashPassword("CustPass123!", nil)
	custUser := &store.User{
		ID:            custID,
		Email:         "initial.cust@example.com",
		PasswordHash:  custHash,
		FullName:      "Initial Customer",
		IsActive:      true,
		IsSuperAdmin:  false,
		Role:          "customer",
		DefaultOrgID:  custOrgID,
	}
	_ = memStore.CreateUser(context.Background(), custUser, custOrgID, "customer")

	// 1. Admin updates Customer Email
	newEmailPayload := map[string]string{"email": "updated.cust@example.com"}
	body, _ := json.Marshal(newEmailPayload)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/admin/users/%s/email", custID.String()), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminTokens.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on admin update email, got %d: %s", w.Code, w.Body.String())
	}

	// Verify old email can no longer login
	oldLogin := map[string]string{"email": "initial.cust@example.com", "password": "CustPass123!"}
	body, _ = json.Marshal(oldLogin)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 on login with old email, got %d", w.Code)
	}

	// 2. Admin updates Customer Password
	newPassPayload := map[string]string{"password": "BrandNewPassword123!"}
	body, _ = json.Marshal(newPassPayload)
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/admin/users/%s/password", custID.String()), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminTokens.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on admin update password, got %d: %s", w.Code, w.Body.String())
	}

	// 3. Login with updated email + updated password
	newLogin := map[string]string{"email": "updated.cust@example.com", "password": "BrandNewPassword123!"}
	body, _ = json.Marshal(newLogin)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on login with updated email and password, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCustomerAuth_AdminImpersonationSecurity(t *testing.T) {
	memStore := store.NewMemoryStore()
	cfg := &config.Config{
		JWTSecret:        "test-super-secret-jwt-key-minimum-32-chars-long!",
		JWTElementsHours: 24 * time.Hour,
		RefreshTokenDays: 7 * 24 * time.Hour,
	}
	quotaSvc := quota.NewService(memStore)
	emailSvc := email.New(cfg, memStore)
	router := setupCustomerAuthRouter(memStore, cfg, quotaSvc, emailSvc)

	// Create Admin User
	adminID := uuid.New()
	adminOrgID := uuid.New()
	adminHash, _ := auth.HashPassword("AdminPassword123!", nil)
	adminUser := &store.User{
		ID:            adminID,
		Email:         "admin@hostvra.com",
		PasswordHash:  adminHash,
		FullName:      "Super Admin",
		IsActive:      true,
		IsSuperAdmin:  true,
		Role:          "admin",
		DefaultOrgID:  adminOrgID,
	}
	_ = memStore.CreateUser(context.Background(), adminUser, adminOrgID, "admin")
	adminTokens, _, _ := auth.GenerateTokenPair(adminID, adminOrgID, adminUser.Email, "admin", true, cfg.JWTSecret, cfg.JWTElementsHours, cfg.RefreshTokenDays)

	// Create Customer User
	custID := uuid.New()
	custOrgID := uuid.New()
	custHash, _ := auth.HashPassword("CustPass123!", nil)
	custUser := &store.User{
		ID:            custID,
		Email:         "target.cust@example.com",
		PasswordHash:  custHash,
		FullName:      "Target Customer",
		IsActive:      true,
		IsSuperAdmin:  false,
		Role:          "customer",
		DefaultOrgID:  custOrgID,
	}
	_ = memStore.CreateUser(context.Background(), custUser, custOrgID, "customer")

	// 1. Admin calls Impersonate endpoint
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%s/impersonate", custID.String()), nil)
	req.Header.Set("Authorization", "Bearer "+adminTokens.AccessToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on impersonate, got %d: %s", w.Code, w.Body.String())
	}

	var impResp struct {
		Success bool `json:"success"`
		Data    struct {
			Tokens struct {
				AccessToken string `json:"access_token"`
			} `json:"tokens"`
			Role string `json:"role"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &impResp); err != nil {
		t.Fatalf("failed to decode impersonate response: %v", err)
	}

	impersonatedToken := impResp.Data.Tokens.AccessToken

	// 2. Impersonated session can access /auth/me and sees impersonation metadata
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+impersonatedToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on /auth/me for impersonated user, got %d: %s", w.Code, w.Body.String())
	}

	var meResp struct {
		Data struct {
			User struct {
				Email string `json:"email"`
			} `json:"user"`
			ImpersonatedBy string `json:"impersonated_by"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &meResp)
	if meResp.Data.User.Email != "target.cust@example.com" {
		t.Errorf("expected customer email target.cust@example.com, got %s", meResp.Data.User.Email)
	}
	if meResp.Data.ImpersonatedBy != adminID.String() {
		t.Errorf("expected impersonated_by to equal admin ID %s, got %s", adminID.String(), meResp.Data.ImpersonatedBy)
	}

	// 3. CRITICAL SECURITY CHECK: Impersonated session CANNOT access admin endpoints
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+impersonatedToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("CRITICAL SECURITY FLAW: Impersonated customer session accessed /admin/users with code %d! Expected 403 Forbidden.", w.Code)
	}
}

func TestCustomerAuth_ForgotPasswordAndReset(t *testing.T) {
	memStore := store.NewMemoryStore()
	cfg := &config.Config{
		JWTSecret:        "test-super-secret-jwt-key-minimum-32-chars-long!",
		JWTElementsHours: 24 * time.Hour,
		RefreshTokenDays: 7 * 24 * time.Hour,
	}
	quotaSvc := quota.NewService(memStore)
	emailSvc := email.New(cfg, memStore)
	router := setupCustomerAuthRouter(memStore, cfg, quotaSvc, emailSvc)

	// Create user
	userID := uuid.New()
	orgID := uuid.New()
	hash, _ := auth.HashPassword("InitialPassword123!", nil)
	user := &store.User{
		ID:            userID,
		Email:         "forgot.user@example.com",
		PasswordHash:  hash,
		FullName:      "Forgot User",
		IsActive:      true,
		IsSuperAdmin:  false,
		Role:          "customer",
		DefaultOrgID:  orgID,
	}
	_ = memStore.CreateUser(context.Background(), user, orgID, "customer")

	// 1. Request forgot password
	forgotReq := map[string]string{"email": "forgot.user@example.com"}
	body, _ := json.Marshal(forgotReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on forgot password, got %d", w.Code)
	}

	// Manually generate a valid reset token in store to test reset endpoint
	rawToken, tokenHash := email.GenerateSecureToken()
	resetToken := &store.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
	}
	_ = memStore.CreatePasswordResetToken(context.Background(), resetToken)

	// 2. Reset password using valid raw token
	resetReq := map[string]string{
		"token":        rawToken,
		"new_password": "NewResetPassword123!",
	}
	body, _ = json.Marshal(resetReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on reset password, got %d: %s", w.Code, w.Body.String())
	}

	// 3. Old password should fail
	oldLogin := map[string]string{"email": "forgot.user@example.com", "password": "InitialPassword123!"}
	body, _ = json.Marshal(oldLogin)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 on login with old password after reset, got %d", w.Code)
	}

	// 4. New password succeeds
	newLogin := map[string]string{"email": "forgot.user@example.com", "password": "NewResetPassword123!"}
	body, _ = json.Marshal(newLogin)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on login with new reset password, got %d: %s", w.Code, w.Body.String())
	}

	// 5. Re-using the same token should fail
	body, _ = json.Marshal(resetReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request when re-using spent reset token, got %d", w.Code)
	}
}

func TestEmailService_Idempotency(t *testing.T) {
	memStore := store.NewMemoryStore()
	cfg := &config.Config{
		AppURL: "https://panel.hostvra.com",
	}
	emailSvc := email.New(cfg, memStore)

	ctx := context.Background()
	// First send
	isNew1, err := emailSvc.SendIdempotent(ctx, "order_confirmed", "order-12345", "test@example.com", "Order Confirmation", "<p>Thank you</p>")
	if err != nil {
		t.Fatalf("unexpected error on first send: %v", err)
	}
	if !isNew1 {
		t.Errorf("expected first send to be new")
	}

	// Second send with same event_type and event_key
	isNew2, err := emailSvc.SendIdempotent(ctx, "order_confirmed", "order-12345", "test@example.com", "Order Confirmation", "<p>Thank you</p>")
	if err != nil {
		t.Fatalf("unexpected error on duplicate send: %v", err)
	}
	if isNew2 {
		t.Errorf("expected duplicate send to be recognized as already sent (false)")
	}
}
