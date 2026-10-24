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
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/quota"
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
	r.Put("/accounts/{id}/limits", h.UpdateLimits)
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

	// 5b. Test Admin Sets Custom RAM, Storage, CPU, Inodes, and Tasks Limits
	openBaseDirTrue := true
	limitsBody, _ := json.Marshal(UpdateAccountLimitsRequest{
		MemoryLimitMB:    2048,
		CPULimitPercent:  200,
		DiskLimitMB:      25600,
		BandwidthLimitMB: 204800,
		InodesLimit:      500000,
		TasksMax:         250,
		WebsitesLimit:    3,
		DatabasesLimit:   10,
		MailboxesLimit:   20,
		PHPVersion:       "8.3",
		OpenBaseDir:      &openBaseDirTrue,
	})
	req = httptest.NewRequest("PUT", "/accounts/"+newAccountID.String()+"/limits", bytes.NewReader(limitsBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for update-limits, got %d: %s", rec.Code, rec.Body.String())
	}

	var limitsResp struct {
		Data *store.HostingAccount `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &limitsResp)
	if limitsResp.Data == nil || limitsResp.Data.MemoryLimitMB != 2048 || limitsResp.Data.CPULimitPercent != 200 || limitsResp.Data.DiskLimitMB != 25600 || limitsResp.Data.InodesLimit != 500000 || limitsResp.Data.TasksMax != 250 {
		t.Fatalf("expected custom limits applied: RAM 2048MB, CPU 200%%, Disk 25600MB, Inodes 500000, Tasks 250, got %+v", limitsResp.Data)
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

func TestAccountHandler_EnterpriseOperations(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)

	h := NewAccountHandler(cfg, s, auditLogger)

	orgID := uuid.New()
	adminID := uuid.New()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
				UserID:         adminID,
				Email:          "admin@hostvra.local",
				OrganizationID: orgID,
				Role:           rbac.RoleOwner,
				IsSuperAdmin:   true,
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})

	r.Get("/accounts/stats", h.GetWHMStats)
	r.Get("/accounts/export", h.ExportAccounts)
	r.Post("/accounts/impersonate/revoke", h.RevokeImpersonation)
	r.Put("/accounts/{id}/details", h.UpdateDetails)
	r.Post("/accounts/{id}/reset-password", h.ResetPassword)
	r.Post("/accounts/{id}/sync-quota", h.SyncQuota)
	r.Post("/accounts/{id}/terminate", h.TerminateAccount)
	r.Post("/accounts/{id}/restore", h.RestoreAccount)
	r.Post("/accounts/{id}/migrate", h.MigrateAccount)
	r.Post("/accounts/{id}/login-token", h.GenerateLoginToken)

	// Seed account and servers
	node1 := &store.Server{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "Node-East",
		IPAddress:      "192.168.1.10",
		Status:         "online",
	}
	node2 := &store.Server{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "Node-West",
		IPAddress:      "192.168.1.20",
		Status:         "online",
	}
	_ = s.CreateServer(context.Background(), node1)
	_ = s.CreateServer(context.Background(), node2)

	accID := uuid.New()
	testDir := filepath.Join(os.TempDir(), "hostvra-test-whm-ops")
	_ = os.MkdirAll(testDir, 0755)
	_ = os.WriteFile(filepath.Join(testDir, "test.txt"), []byte("enterprise whm test"), 0644)
	defer os.RemoveAll(testDir)

	acc := &store.HostingAccount{
		ID:             accID,
		OrganizationID: orgID,
		Username:       "enterprisex",
		Domain:         "enterprisex.com",
		DocumentRoot:   testDir,
		PlanID:         uuid.New(),
		PlanName:       "Business Pro",
		Status:         store.AccountStatusActive,
		ServerID:       &node1.ID,
		ServerName:     node1.Name,
		IPAddress:      node1.IPAddress,
		DiskLimitMB:    5000,
	}
	_ = s.CreateHostingAccount(context.Background(), acc)

	// 1. Test GetWHMStats
	statsReq := httptest.NewRequest("GET", "/accounts/stats", nil)
	statsRec := httptest.NewRecorder()
	r.ServeHTTP(statsRec, statsReq)
	if statsRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for WHMStats, got %d", statsRec.Code)
	}
	var statsResp struct {
		Data *store.WHMStats `json:"data"`
	}
	_ = json.Unmarshal(statsRec.Body.Bytes(), &statsResp)
	if statsResp.Data.TotalAccounts != 1 || statsResp.Data.ActiveAccounts != 1 {
		t.Fatalf("unexpected stats: %+v", statsResp.Data)
	}

	// 2. Test UpdateDetails
	detBody := bytes.NewBufferString(`{"contact_email":"contact@enterprisex.com","contact_phone":"+123456789","notes":"VIP client account"}`)
	detReq := httptest.NewRequest("PUT", "/accounts/"+accID.String()+"/details", detBody)
	detRec := httptest.NewRecorder()
	r.ServeHTTP(detRec, detReq)
	if detRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for UpdateDetails, got %d", detRec.Code)
	}
	accUpdated, _ := s.GetHostingAccountByID(context.Background(), accID)
	if accUpdated.ContactEmail != "contact@enterprisex.com" || accUpdated.Notes != "VIP client account" {
		t.Fatalf("UpdateDetails failed: %+v", accUpdated)
	}

	// 3. Test ResetPassword
	pwdBody := bytes.NewBufferString(`{"new_password":"SecurePassword!2026"}`)
	pwdReq := httptest.NewRequest("POST", "/accounts/"+accID.String()+"/reset-password", pwdBody)
	pwdRec := httptest.NewRecorder()
	r.ServeHTTP(pwdRec, pwdReq)
	if pwdRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for ResetPassword, got %d: %s", pwdRec.Code, pwdRec.Body.String())
	}

	// 4. Test SyncQuota
	syncReq := httptest.NewRequest("POST", "/accounts/"+accID.String()+"/sync-quota", nil)
	syncRec := httptest.NewRecorder()
	r.ServeHTTP(syncRec, syncReq)
	if syncRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for SyncQuota, got %d", syncRec.Code)
	}
	accSynced, _ := s.GetHostingAccountByID(context.Background(), accID)
	if accSynced.LastSyncAt == nil || accSynced.QuotaStatus != "synchronized" {
		t.Fatalf("SyncQuota failed: %+v", accSynced)
	}

	// 5. Test MigrateAccount
	migBody := bytes.NewBufferString(fmt.Sprintf(`{"target_server_id":%q}`, node2.ID.String()))
	migReq := httptest.NewRequest("POST", "/accounts/"+accID.String()+"/migrate", migBody)
	migRec := httptest.NewRecorder()
	r.ServeHTTP(migRec, migReq)
	if migRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for MigrateAccount, got %d: %s", migRec.Code, migRec.Body.String())
	}
	accMigrated, _ := s.GetHostingAccountByID(context.Background(), accID)
	if accMigrated.ServerName != node2.Name || accMigrated.MigrationStatus != "completed" {
		t.Fatalf("MigrateAccount failed: %+v", accMigrated)
	}

	// 6. Test ExportAccounts
	expReq := httptest.NewRequest("GET", "/accounts/export?format=csv", nil)
	expRec := httptest.NewRecorder()
	r.ServeHTTP(expRec, expReq)
	if expRec.Code != http.StatusOK || expRec.Header().Get("Content-Type") != "text/csv" {
		t.Fatalf("expected 200 text/csv for ExportAccounts, got %d, type: %s", expRec.Code, expRec.Header().Get("Content-Type"))
	}

	// 7. Test GenerateLoginToken & RevokeImpersonation
	tokenReq := httptest.NewRequest("POST", "/accounts/"+accID.String()+"/login-token", nil)
	tokenRec := httptest.NewRecorder()
	r.ServeHTTP(tokenRec, tokenReq)
	if tokenRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GenerateLoginToken, got %d: %s", tokenRec.Code, tokenRec.Body.String())
	}
	var tokenResp struct {
		Data struct {
			SessionID string `json:"session_id"`
			Token     string `json:"token"`
			User      struct {
				ID       uuid.UUID `json:"id"`
				Role     string    `json:"role"`
				FullName string    `json:"full_name"`
			} `json:"user"`
		} `json:"data"`
	}
	_ = json.Unmarshal(tokenRec.Body.Bytes(), &tokenResp)
	if tokenResp.Data.SessionID == "" || tokenResp.Data.Token == "" {
		t.Fatalf("expected session_id and token in response: %s", tokenRec.Body.String())
	}
	if tokenResp.Data.User.Role != rbac.RoleCustomer {
		t.Fatalf("expected impersonated target user to have customer role, got %s", tokenResp.Data.User.Role)
	}
	if tokenResp.Data.User.ID == adminID {
		t.Fatalf("impersonated user ID should never be the admin user ID!")
	}

	// Verify the client's effective plan resolves with client hosting limits, not root admin 999999
	qs := quota.NewService(s)
	clientPlan, err := qs.ResolveEffectivePlan(context.Background(), tokenResp.Data.User.ID)
	if err != nil {
		t.Fatalf("failed to resolve client effective plan: %v", err)
	}
	if clientPlan.IsSuperAdmin || clientPlan.MaxWebsites >= 999999 {
		t.Fatalf("client plan should not have root administrator privileges: %+v", clientPlan)
	}

	revBody := bytes.NewBufferString(fmt.Sprintf(`{"session_id":%q}`, tokenResp.Data.SessionID))
	revReq := httptest.NewRequest("POST", "/accounts/impersonate/revoke", revBody)
	revRec := httptest.NewRecorder()
	r.ServeHTTP(revRec, revReq)
	if revRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for RevokeImpersonation, got %d", revRec.Code)
	}

	// 8. Test TerminateAccount (fails with wrong confirmation, succeeds with exact username)
	badTermBody := bytes.NewBufferString(`{"confirm_username":"wrong"}`)
	badTermReq := httptest.NewRequest("POST", "/accounts/"+accID.String()+"/terminate", badTermBody)
	badTermRec := httptest.NewRecorder()
	r.ServeHTTP(badTermRec, badTermReq)
	if badTermRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad confirmation, got %d", badTermRec.Code)
	}

	termBody := bytes.NewBufferString(`{"confirm_username":"enterprisex"}`)
	termReq := httptest.NewRequest("POST", "/accounts/"+accID.String()+"/terminate", termBody)
	termRec := httptest.NewRecorder()
	r.ServeHTTP(termRec, termReq)
	if termRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for TerminateAccount, got %d", termRec.Code)
	}
	accTerminated, _ := s.GetHostingAccountByID(context.Background(), accID)
	if accTerminated.Status != store.AccountStatusTerminated {
		t.Fatalf("expected terminated status, got %s", accTerminated.Status)
	}

	// 9. Test RestoreAccount
	restReq := httptest.NewRequest("POST", "/accounts/"+accID.String()+"/restore", nil)
	restRec := httptest.NewRecorder()
	r.ServeHTTP(restRec, restReq)
	if restRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for RestoreAccount, got %d", restRec.Code)
	}
	accRestored, _ := s.GetHostingAccountByID(context.Background(), accID)
	if accRestored.Status != store.AccountStatusActive {
		t.Fatalf("expected active status, got %s", accRestored.Status)
	}
}
