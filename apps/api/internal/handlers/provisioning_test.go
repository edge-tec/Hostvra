package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/rbac"
	"hostvra/api/internal/store"
)

func setupProvisioningTestEnv() (*config.Config, store.Store, *ProvisioningHandler, *ResellerHandler, *AccountHandler, uuid.UUID, uuid.UUID) {
	cfg := &config.Config{JWTSecret: "test-jwt-secret-32-bytes-long-12345"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)

	provHandler := NewProvisioningHandler(cfg, s, auditLogger)
	resellerHandler := NewResellerHandler(cfg, s, auditLogger)
	accountHandler := NewAccountHandler(cfg, s, auditLogger)

	orgID := uuid.New()
	adminUserID := uuid.New()

	// Seed Organization & Admin User
	_ = s.CreateUser(context.Background(), &store.User{
		ID:            adminUserID,
		Email:         "admin@hostvra.local",
		FullName:      "Admin User",
		IsActive:      true,
		Role:          rbac.RoleOwner,
		DefaultOrgID:  orgID,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}, orgID, rbac.RoleOwner)

	// Seed Server
	serverID := uuid.New()
	_ = s.CreateServer(context.Background(), &store.Server{
		ID:             serverID,
		OrganizationID: orgID,
		Name:           "Production Node 1",
		Hostname:       "node1.hostvra.local",
		IPAddress:      "198.51.100.10",
		Status:         "online",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	})

	// Seed Starter Plan
	_ = s.CreatePlan(context.Background(), &store.HostingPlan{
		ID:           uuid.New(),
		Name:         "Starter Cloud",
		Slug:         "starter-cloud",
		DiskSpaceMB:  2048,
		BandwidthMB:  20480,
		MaxWebsites:  1,
		MaxDatabases: 2,
		MaxMailboxes: 5,
		FreeSSL:      true,
		IsActive:     true,
	})

	// Seed Business Plan
	_ = s.CreatePlan(context.Background(), &store.HostingPlan{
		ID:           uuid.New(),
		Name:         "Business Cloud",
		Slug:         "business-cloud",
		DiskSpaceMB:  10240,
		BandwidthMB:  102400,
		MaxWebsites:  5,
		MaxDatabases: 10,
		MaxMailboxes: 25,
		FreeSSL:      true,
		IsActive:     true,
	})

	return cfg, s, provHandler, resellerHandler, accountHandler, orgID, adminUserID
}

func calculateHMAC(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestProvisioning_WebhookAuth(t *testing.T) {
	_, s, provHandler, _, _, orgID, _ := setupProvisioningTestEnv()

	// Create an integration with known secret
	rawSecret := "secret-integration-key-64-bytes-0123456789abcdef0123456789abcdef"
	hash := sha256.Sum256([]byte(rawSecret))
	secretHash := hex.EncodeToString(hash[:])

	integration := &store.ProvisioningIntegration{
		ID:            uuid.New(),
		OrganizationID: orgID,
		Provider:      "whmcs",
		Name:          "WHMCS Production",
		SecretKeyHash: secretHash,
		RawSecretKey:  rawSecret,
		IsActive:      true,
	}
	_ = s.CreateProvisioningIntegration(context.Background(), integration)

	r := chi.NewRouter()
	r.Post("/provisioning/webhook", provHandler.HandleWebhook)

	pingBody := `{"action":"ping"}`

	// 1. Unauthorized when no signature or token is sent
	req := httptest.NewRequest("POST", "/provisioning/webhook", bytes.NewBufferString(pingBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized without auth, got %d", rec.Code)
	}

	// 2. Authorized with valid HMAC-SHA256 signature
	sig := calculateHMAC(rawSecret, pingBody)
	req = httptest.NewRequest("POST", "/provisioning/webhook", bytes.NewBufferString(pingBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hostvra-Signature", sig)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with valid HMAC, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Authorized with raw secret token
	req = httptest.NewRequest("POST", "/provisioning/webhook", bytes.NewBufferString(pingBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hostvra-Token", rawSecret)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with valid token, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProvisioning_WebhookLifecycle(t *testing.T) {
	_, s, provHandler, _, _, orgID, _ := setupProvisioningTestEnv()

	rawSecret := "lifecycle-test-secret-whmcs-0123456789abcdef"
	hash := sha256.Sum256([]byte(rawSecret))
	_ = s.CreateProvisioningIntegration(context.Background(), &store.ProvisioningIntegration{
		ID:            uuid.New(),
		OrganizationID: orgID,
		Provider:      "whmcs",
		Name:          "WHMCS Primary",
		SecretKeyHash: hex.EncodeToString(hash[:]),
		RawSecretKey:  rawSecret,
		IsActive:      true,
	})

	r := chi.NewRouter()
	r.Post("/provisioning/webhook", provHandler.HandleWebhook)

	sendWebhook := func(payload string) *httptest.ResponseRecorder {
		sig := calculateHMAC(rawSecret, payload)
		req := httptest.NewRequest("POST", "/provisioning/webhook", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-WHMCS-Signature", sig)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// 1. Create Account via Webhook
	createPayload := `{
		"action": "create_account",
		"domain": "whmcsclient.com",
		"username": "whmcsclient",
		"email": "client@whmcsclient.com",
		"plan": "Starter Cloud"
	}`
	rec := sendWebhook(createPayload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for create_account, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify account exists in store
	acc, err := s.GetHostingAccountByDomain(context.Background(), "whmcsclient.com")
	if err != nil || acc == nil {
		t.Fatalf("expected hosting account in store, got err: %v", err)
	}
	if acc.Username != "whmcsclient" || acc.Status != store.AccountStatusActive {
		t.Fatalf("expected active account for whmcsclient, got %+v", acc)
	}

	// 2. Suspend Account via Webhook
	suspendPayload := `{
		"action": "suspend",
		"domain": "whmcsclient.com",
		"reason": "Invoice #1049 overdue"
	}`
	rec = sendWebhook(suspendPayload)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for suspend, got %d: %s", rec.Code, rec.Body.String())
	}

	acc, _ = s.GetHostingAccountByDomain(context.Background(), "whmcsclient.com")
	if acc.Status != store.AccountStatusSuspended || acc.SuspendReason != "Invoice #1049 overdue" {
		t.Fatalf("expected account suspended with reason, got %+v", acc)
	}

	// 3. Unsuspend Account via Webhook
	unsuspendPayload := `{
		"action": "unsuspend",
		"domain": "whmcsclient.com"
	}`
	rec = sendWebhook(unsuspendPayload)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for unsuspend, got %d: %s", rec.Code, rec.Body.String())
	}

	acc, _ = s.GetHostingAccountByDomain(context.Background(), "whmcsclient.com")
	if acc.Status != store.AccountStatusActive || acc.SuspendReason != "" {
		t.Fatalf("expected active account, got %+v", acc)
	}

	// 4. Change Package via Webhook
	changePayload := `{
		"action": "change_package",
		"domain": "whmcsclient.com",
		"plan": "Business Cloud"
	}`
	rec = sendWebhook(changePayload)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for change_package, got %d: %s", rec.Code, rec.Body.String())
	}

	acc, _ = s.GetHostingAccountByDomain(context.Background(), "whmcsclient.com")
	if acc.PlanName != "Business Cloud" || acc.DiskLimitMB <= 2048 {
		t.Fatalf("expected upgraded plan limits, got %+v", acc)
	}

	// 5. Terminate Account via Webhook
	terminatePayload := `{
		"action": "terminate",
		"domain": "whmcsclient.com"
	}`
	rec = sendWebhook(terminatePayload)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for terminate, got %d: %s", rec.Code, rec.Body.String())
	}

	acc, err = s.GetHostingAccountByDomain(context.Background(), "whmcsclient.com")
	if err == nil && acc != nil {
		t.Fatalf("expected account to be deleted from store, but found: %+v", acc)
	}
}

func TestProvisioning_WebhookLoggingAndReplay(t *testing.T) {
	_, s, provHandler, _, _, orgID, _ := setupProvisioningTestEnv()

	rawSecret := "logging-replay-secret-0123456789abcdef"
	hash := sha256.Sum256([]byte(rawSecret))
	integrationID := uuid.New()
	_ = s.CreateProvisioningIntegration(context.Background(), &store.ProvisioningIntegration{
		ID:            integrationID,
		OrganizationID: orgID,
		Provider:      "blesta",
		Name:          "Blesta Billing",
		SecretKeyHash: hex.EncodeToString(hash[:]),
		RawSecretKey:  rawSecret,
		IsActive:      true,
	})

	r := chi.NewRouter()
	r.Post("/provisioning/webhook", provHandler.HandleWebhook)
	r.Post("/provisioning/webhook/{provider}", provHandler.HandleWebhook)
	r.Post("/provisioning/logs/{id}/replay", provHandler.ReplayWebhookLog)

	payload := `{
		"action": "create_account",
		"domain": "replaytest.com",
		"username": "replayuser",
		"email": "user@replaytest.com",
		"plan": "Starter Cloud"
	}`
	sig := calculateHMAC(rawSecret, payload)

	req := httptest.NewRequest("POST", "/provisioning/webhook?provider=blesta", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Blesta-Signature", sig)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify log was recorded
	logs, err := s.ListProvisioningWebhookLogs(context.Background(), orgID, 10)
	if err != nil || len(logs) == 0 {
		t.Fatalf("expected webhook logs, got %v", err)
	}

	loggedItem := logs[0]
	if loggedItem.Domain != "replaytest.com" || loggedItem.Status != "success" {
		t.Fatalf("unexpected log item: %+v", loggedItem)
	}

	// Delete the created account to test replay
	acc, _ := s.GetHostingAccountByDomain(context.Background(), "replaytest.com")
	if acc != nil {
		_ = s.DeleteHostingAccount(context.Background(), acc.ID)
	}

	// Replay webhook action
	replayReq := httptest.NewRequest("POST", "/provisioning/logs/"+loggedItem.ID.String()+"/replay", nil)
	replayRec := httptest.NewRecorder()
	r.ServeHTTP(replayRec, replayReq)

	if replayRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for replay, got %d: %s", replayRec.Code, replayRec.Body.String())
	}

	// Verify account was re-created by replay
	recreatedAcc, err := s.GetHostingAccountByDomain(context.Background(), "replaytest.com")
	if err != nil || recreatedAcc == nil {
		t.Fatalf("expected account to be re-provisioned via replay")
	}
}

func TestReseller_QuotaPoolingAndOverselling(t *testing.T) {
	_, s, _, resellerHandler, accountHandler, orgID, _ := setupProvisioningTestEnv()

	// Create Reseller User
	resellerUserID := uuid.New()
	_ = s.CreateUser(context.Background(), &store.User{
		ID:           resellerUserID,
		Email:        "reseller@partner.com",
		FullName:     "Host Partner",
		IsActive:     true,
		Role:         rbac.RoleReseller,
		DefaultOrgID: orgID,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}, orgID, rbac.RoleReseller)

	// Create Reseller Profile: max 2 accounts, 50000 MB disk, no overselling
	profile := &store.ResellerProfile{
		ID:                   uuid.New(),
		OrganizationID:       orgID,
		UserID:               resellerUserID,
		MaxAccounts:          2,
		AllocatedDiskMB:      50000,
		AllocatedBandwidthMB: 500000,
		AllowOverselling:     false,
		BrandName:            "PartnerHost",
		CustomNameservers:    []string{"ns1.partnerhost.net", "ns2.partnerhost.net"},
		IsActive:             true,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}
	_ = s.CreateResellerProfile(context.Background(), profile)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
				UserID:         resellerUserID,
				OrganizationID: orgID,
				Role:           rbac.RoleReseller,
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Post("/accounts", accountHandler.CreateAccount)
	r.Get("/reseller/usage", resellerHandler.GetUsage)

	plans, _ := s.ListPlans(context.Background())
	starterPlanID := plans[0].ID.String()

	createSubAccount := func(domain, username string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(CreateAccountRequest{
			Domain:   domain,
			Username: username,
			PlanID:   starterPlanID,
		})
		req := httptest.NewRequest("POST", "/accounts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// 1. First account succeeds
	rec1 := createSubAccount("site1.com", "user1")
	if rec1.Code != http.StatusCreated {
		t.Fatalf("expected 201 for first account, got %d: %s", rec1.Code, rec1.Body.String())
	}

	// Tag user as belonging to reseller to test quota aggregation
	u1, _ := s.GetHostingAccountByDomain(context.Background(), "site1.com")
	if u1 != nil {
		user1, _ := s.GetUserByID(context.Background(), u1.UserID)
		if user1 != nil {
			user1.ResellerID = &resellerUserID
		}
	}

	// 2. Second account succeeds
	rec2 := createSubAccount("site2.com", "user2")
	if rec2.Code != http.StatusCreated {
		t.Fatalf("expected 201 for second account, got %d: %s", rec2.Code, rec2.Body.String())
	}
	u2, _ := s.GetHostingAccountByDomain(context.Background(), "site2.com")
	if u2 != nil {
		user2, _ := s.GetUserByID(context.Background(), u2.UserID)
		if user2 != nil {
			user2.ResellerID = &resellerUserID
		}
	}

	// 3. Third account must be rejected (quota exceeded: max 2 accounts)
	rec3 := createSubAccount("site3.com", "user3")
	if rec3.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when max accounts exceeded, got %d: %s", rec3.Code, rec3.Body.String())
	}

	// 4. Verify Usage API returns accurate stats
	reqUsage := httptest.NewRequest("GET", "/reseller/usage", nil)
	recUsage := httptest.NewRecorder()
	r.ServeHTTP(recUsage, reqUsage)

	if recUsage.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for reseller usage, got %d", recUsage.Code)
	}

	var usageResp struct {
		Data *store.ResellerUsageSummary `json:"data"`
	}
	_ = json.Unmarshal(recUsage.Body.Bytes(), &usageResp)
	if usageResp.Data == nil || usageResp.Data.MaxAccounts != 2 || usageResp.Data.UsedAccounts != 2 {
		t.Fatalf("expected 2/2 used accounts, got: %+v", usageResp.Data)
	}
}

func TestReseller_WhiteLabelBranding(t *testing.T) {
	_, s, _, resellerHandler, _, orgID, _ := setupProvisioningTestEnv()

	resellerUserID := uuid.New()
	_ = s.CreateUser(context.Background(), &store.User{
		ID:           resellerUserID,
		Email:        "brandmaster@agency.com",
		FullName:     "Agency Admin",
		IsActive:     true,
		Role:         rbac.RoleReseller,
		DefaultOrgID: orgID,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}, orgID, rbac.RoleReseller)

	_ = s.CreateResellerProfile(context.Background(), &store.ResellerProfile{
		ID:                   uuid.New(),
		OrganizationID:       orgID,
		UserID:               resellerUserID,
		MaxAccounts:          10,
		AllocatedDiskMB:      50000,
		AllocatedBandwidthMB: 200000,
		AllowOverselling:     true,
		BrandName:            "Agency Cloud",
		BrandLogoURL:         "https://agency.com/logo.png",
		CustomNameservers:    []string{"ns1.agencycloud.io", "ns2.agencycloud.io"},
		CustomEmailSender:    "support@agencycloud.io",
		IsActive:             true,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	})

	r := chi.NewRouter()
	r.Get("/branding", resellerHandler.GetBranding)

	// 1. Default branding without reseller query
	req := httptest.NewRequest("GET", "/branding", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	var defaultBrand struct {
		Data BrandingResponse `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &defaultBrand)
	if defaultBrand.Data.BrandName != "Hostvra" {
		t.Fatalf("expected default brand 'Hostvra', got %s", defaultBrand.Data.BrandName)
	}

	// 2. Custom white-label branding by reseller_id
	req = httptest.NewRequest("GET", "/branding?reseller_id="+resellerUserID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	var agencyBrand struct {
		Data BrandingResponse `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &agencyBrand)
	if agencyBrand.Data.BrandName != "Agency Cloud" || agencyBrand.Data.CustomNameservers[0] != "ns1.agencycloud.io" {
		t.Fatalf("expected Agency Cloud branding, got: %+v", agencyBrand.Data)
	}
}
