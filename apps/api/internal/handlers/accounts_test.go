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
	"hostvra/api/internal/rbac"
	"hostvra/api/internal/store"
)

func TestAccountHandler_Flow(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)

	h := NewAccountHandler(cfg, s, auditLogger)

	orgID := uuid.New()
	userID := uuid.New()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
				UserID:         userID,
				OrganizationID: orgID,
				Role:           "owner",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})

	r.Get("/accounts", h.ListAccounts)
	r.Post("/accounts", h.CreateAccount)
	r.Get("/accounts/{id}", h.GetAccount)
	r.Post("/accounts/{id}/suspend", h.SuspendAccount)
	r.Post("/accounts/{id}/unsuspend", h.UnsuspendAccount)
	r.Post("/accounts/{id}/change-plan", h.ChangePlan)
	r.Post("/accounts/{id}/login-token", h.GenerateLoginToken)
	r.Delete("/accounts/{id}", h.DeleteAccount)

	// Seed server and plan
	serverID := uuid.New()
	_ = s.CreateServer(context.Background(), &store.Server{
		ID:             serverID,
		OrganizationID: orgID,
		Name:           "node-primary",
		IPAddress:      "109.199.110.101",
		Status:         "online",
	})

	plan1ID := uuid.New()
	_ = s.CreatePlan(context.Background(), &store.HostingPlan{
		ID:          plan1ID,
		Name:        "Starter Cloud",
		Slug:        "test-starter-cloud",
		DiskSpaceMB: 10240,
		BandwidthMB: 102400,
		MaxWebsites: 1,
		IsActive:    true,
	})

	plan2ID := uuid.New()
	_ = s.CreatePlan(context.Background(), &store.HostingPlan{
		ID:          plan2ID,
		Name:        "Business Pro",
		Slug:        "test-business-pro",
		DiskSpaceMB: 51200,
		BandwidthMB: 512000,
		MaxWebsites: 10,
		IsActive:    true,
	})

	_ = s.CreateHostingAccount(context.Background(), &store.HostingAccount{
		ID:             uuid.New(),
		OrganizationID: orgID,
		UserID:         userID,
		Username:       "seeduser",
		Domain:         "seeduser.com",
		PlanID:         plan1ID,
		PlanName:       "Starter Cloud",
		Status:         store.AccountStatusActive,
		DiskLimitMB:    10240,
		DiskUsedMB:     250,
	})

	// 1. List seed accounts
	req := httptest.NewRequest("GET", "/accounts", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var listResp struct {
		Data []*store.HostingAccount `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to decode accounts: %v", err)
	}
	if len(listResp.Data) == 0 {
		t.Fatalf("expected at least 1 seed account")
	}

	targetAccount := listResp.Data[0]

	// 2. Conflict Prevention: Attempting to create duplicate domain should fail with 409
	dupBody, _ := json.Marshal(CreateAccountRequest{
		Domain:   "seeduser.com",
		Username: "newuser2",
		PlanID:   plan1ID.String(),
	})
	req = httptest.NewRequest("POST", "/accounts", bytes.NewReader(dupBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for duplicate domain, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Conflict Prevention: Attempting to create duplicate username should fail with 409
	dupUserBody, _ := json.Marshal(CreateAccountRequest{
		Domain:   "unique-domain.com",
		Username: "seeduser",
		PlanID:   plan1ID.String(),
	})
	req = httptest.NewRequest("POST", "/accounts", bytes.NewReader(dupUserBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for duplicate username, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. Provision a new client account successfully
	createBody, _ := json.Marshal(CreateAccountRequest{
		Domain:     "enterprise-client.com",
		Username:   "c_enterprise",
		PlanID:     plan1ID.String(),
		ServerID:   serverID.String(),
		PHPVersion: "8.3",
	})
	req = httptest.NewRequest("POST", "/accounts", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for provisioning, got %d: %s", rec.Code, rec.Body.String())
	}

	var createdResp struct {
		Data *store.HostingAccount `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &createdResp)
	if createdResp.Data == nil || createdResp.Data.Domain != "enterprise-client.com" {
		t.Fatalf("expected created account domain enterprise-client.com, got %+v", createdResp.Data)
	}
	newAccountID := createdResp.Data.ID

	// 5. Upgrade package / Change Plan
	changePlanBody, _ := json.Marshal(ChangePlanRequest{
		PlanID: plan2ID.String(),
	})
	req = httptest.NewRequest("POST", "/accounts/"+newAccountID.String()+"/change-plan", bytes.NewReader(changePlanBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for change-plan, got %d: %s", rec.Code, rec.Body.String())
	}

	var planResp struct {
		Data *store.HostingAccount `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &planResp)
	if planResp.Data.PlanName != "Business Pro" || planResp.Data.DiskLimitMB != 51200 {
		t.Fatalf("expected plan updated to Business Pro with 51200MB, got %+v", planResp.Data)
	}

	// 6. Test Suspend Account (and check status sync)
	suspendBody, _ := json.Marshal(SuspendRequest{Reason: "Payment Overdue Test"})
	req = httptest.NewRequest("POST", "/accounts/"+targetAccount.ID.String()+"/suspend", bytes.NewReader(suspendBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for suspend, got %d: %s", rec.Code, rec.Body.String())
	}

	var suspendedResp struct {
		Data *store.HostingAccount `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &suspendedResp)
	if suspendedResp.Data == nil || suspendedResp.Data.Status != store.AccountStatusSuspended {
		t.Fatalf("expected account to be suspended, got %+v", suspendedResp.Data)
	}

	// 7. Test Unsuspend Account
	req = httptest.NewRequest("POST", "/accounts/"+targetAccount.ID.String()+"/unsuspend", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for unsuspend, got %d", rec.Code)
	}

	// 8. Test 1-Click Cryptographic Impersonation Login Token
	req = httptest.NewRequest("POST", "/accounts/"+targetAccount.ID.String()+"/login-token", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for login token, got %d", rec.Code)
	}

	var loginTokenResp struct {
		Data struct {
			Username string          `json:"username"`
			Domain   string          `json:"domain"`
			Tokens   *auth.TokenPair `json:"tokens"`
			LoginURL string          `json:"login_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginTokenResp); err != nil {
		t.Fatalf("failed to decode login token response: %v", err)
	}
	if loginTokenResp.Data.Tokens == nil || loginTokenResp.Data.Tokens.AccessToken == "" {
		t.Fatalf("expected cryptographically signed JWT token pair, got nil")
	}

	// Verify the generated impersonation JWT token
	claims, err := auth.ValidateAccessToken(loginTokenResp.Data.Tokens.AccessToken, cfg.JWTSecret)
	if err != nil {
		t.Fatalf("failed to parse generated impersonation token: %v", err)
	}
	if claims.ImpersonatedBy == nil || *claims.ImpersonatedBy != userID {
		t.Fatalf("expected ImpersonatedBy to be set to admin userID %s, got %+v", userID, claims.ImpersonatedBy)
	}
	if claims.Role != rbac.RoleCustomer {
		t.Fatalf("expected customer role for impersonated session, got %s", claims.Role)
	}

	// 9. Test Terminate Account
	req = httptest.NewRequest("DELETE", "/accounts/"+newAccountID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on termination, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify deleted
	req = httptest.NewRequest("GET", "/accounts/"+newAccountID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after deletion, got %d", rec.Code)
	}
}

func TestAccountHandler_LiveQuotaMeasurement(t *testing.T) {
	tempDir := t.TempDir()
	docRoot := filepath.Join(tempDir, "client_docroot")
	_ = os.MkdirAll(docRoot, 0755)

	// Write 5MB test file
	testFile := filepath.Join(docRoot, "test_payload.bin")
	data := make([]byte, 5*1024*1024)
	_ = os.WriteFile(testFile, data, 0644)

	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)

	h := NewAccountHandler(cfg, s, auditLogger)

	orgID := uuid.New()
	userID := uuid.New()
	accID := uuid.New()

	_ = s.CreateHostingAccount(context.Background(), &store.HostingAccount{
		ID:             accID,
		OrganizationID: orgID,
		UserID:         userID,
		Username:       "storage_user",
		Domain:         "storage-test.com",
		DocumentRoot:   docRoot,
		PlanName:       "Cloud 10G",
		Status:         store.AccountStatusActive,
		DiskLimitMB:    10240,
		DiskUsedMB:     0, // Will be measured dynamically
	})

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
				UserID:         userID,
				OrganizationID: orgID,
				Role:           "owner",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Get("/accounts", h.ListAccounts)

	req := httptest.NewRequest("GET", "/accounts", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var listResp struct {
		Data []*store.HostingAccount `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &listResp)
	if len(listResp.Data) == 0 {
		t.Fatalf("expected account in list")
	}

	measuredMB := listResp.Data[0].DiskUsedMB
	if measuredMB < 5 {
		t.Fatalf("expected live disk measurement >= 5MB, got %d MB", measuredMB)
	}
}
