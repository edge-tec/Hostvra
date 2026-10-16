package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/email/ratecontrol"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/dns"
	"hostvra/api/internal/store"
)

func setupTestEmailHandlerWithStore(t *testing.T) (*EmailHandler, store.Store) {
	cfg := &config.Config{
		JWTSecret: "test-jwt-secret-for-rate-control-tests-32-bytes",
	}
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auditLogger := audit.NewLogger(memStore, logger)

	handler := NewEmailHandler(cfg, memStore, nil, auditLogger)
	return handler, memStore
}

func adminContext() context.Context {
	now := time.Now().UTC()
	claims := &auth.Claims{
		UserID:         uuid.New(),
		Email:          "admin@hostvra.com",
		Role:           "admin",
		IsSuperAdmin:   true,
		OrganizationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return context.WithValue(context.Background(), auth.UserContextKey, claims)
}

func TestGetRateControlStatus(t *testing.T) {
	handler, _ := setupTestEmailHandlerWithStore(t)

	req := httptest.NewRequest("GET", "/api/v1/email/rate-control", nil)
	req = req.WithContext(adminContext())
	w := httptest.NewRecorder()

	handler.GetRateControlStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if res["success"] != true {
		t.Errorf("expected success: true, got: %v", res["success"])
	}

	data, ok := res["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing or invalid data field")
	}

	// Verify gmail_transport configuration is present
	gmailTrans, ok := data["gmail_transport"].(map[string]any)
	if !ok || gmailTrans["name"] != "gmail" {
		t.Errorf("expected gmail transport spec in rate control status: %+v", gmailTrans)
	}
	if gmailTrans["concurrency_limit"] != float64(1) {
		t.Errorf("expected concurrency_limit 1, got %v", gmailTrans["concurrency_limit"])
	}

	// Verify auth_alignment
	authAlign, ok := data["auth_alignment"].(map[string]any)
	if !ok || authAlign["reverse_dns_ptr"] != "mail.onlyflrt.net" {
		t.Errorf("expected FCrDNS PTR mail.onlyflrt.net, got %v", authAlign["reverse_dns_ptr"])
	}
}

func TestRateLimitCooldownAndFlushProtection(t *testing.T) {
	handler, _ := setupTestEmailHandlerWithStore(t)
	ctrl := ratecontrol.GetGlobalController()

	domain := "onlyflrt.net"
	dest := "gmail.com"

	// 1. Simulate Gmail 421 Rate Limit Event
	payload := map[string]string{
		"domain":      domain,
		"destination": dest,
		"status_code": "421",
		"response":    "4.7.28 Gmail has detected an unusual rate of mail originating from your SPF domain [onlyflrt.net]",
	}
	body, _ := json.Marshal(payload)
	recReq := httptest.NewRequest("POST", "/api/v1/email/rate-control/record-event", bytes.NewReader(body))
	recReq = recReq.WithContext(adminContext())
	recW := httptest.NewRecorder()

	handler.RecordSimulatedRateLimitEvent(recW, recReq)
	if recW.Code != http.StatusOK {
		t.Fatalf("expected status 200 recording event, got %d", recW.Code)
	}

	// Verify state is in cooldown
	state := ctrl.GetState(domain, dest)
	if !state.InCooldown {
		t.Fatalf("expected domain to be in cooldown after 421 event")
	}

	// 2. Attempt FlushQueue without ?force=true -> Must be blocked with 429
	flushReq := httptest.NewRequest("POST", "/api/v1/email/queue/flush?domain="+domain+"&destination="+dest, nil)
	flushReq = flushReq.WithContext(adminContext())
	flushW := httptest.NewRecorder()

	handler.FlushQueue(flushW, flushReq)

	if flushW.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 Too Many Requests when flushing during cooldown, got %d: %s", flushW.Code, flushW.Body.String())
	}

	// 3. Reset Cooldown manually via API
	resetReq := httptest.NewRequest("POST", "/api/v1/email/rate-control/cooldown/reset?domain="+domain+"&destination="+dest, nil)
	resetReq = resetReq.WithContext(adminContext())
	resetW := httptest.NewRecorder()

	handler.ResetRateControlCooldown(resetW, resetReq)
	if resetW.Code != http.StatusOK {
		t.Fatalf("expected 200 resetting cooldown, got %d", resetW.Code)
	}

	stateAfter := ctrl.GetState(domain, dest)
	if stateAfter.InCooldown {
		t.Errorf("cooldown should be false after reset")
	}

	// 4. FlushQueue now passes or returns non-429
	flushReq2 := httptest.NewRequest("POST", "/api/v1/email/queue/flush?domain="+domain+"&destination="+dest, nil)
	flushReq2 = flushReq2.WithContext(adminContext())
	flushW2 := httptest.NewRecorder()

	handler.FlushQueue(flushW2, flushReq2)
	if flushW2.Code == http.StatusTooManyRequests {
		t.Errorf("flush should not return 429 after cooldown reset")
	}
}

func TestFlushQueueWithForceOverride(t *testing.T) {
	handler, _ := setupTestEmailHandlerWithStore(t)
	ctrl := ratecontrol.GetGlobalController()

	domain := "force-override-test.com"
	dest := "gmail.com"

	// Trigger cooldown
	ctrl.RecordDeliveryEvent(domain, dest, "421", "4.7.28 unusual rate")

	// Flush with force=true must not return 429
	flushReq := httptest.NewRequest("POST", "/api/v1/email/queue/flush?force=true&domain="+domain+"&destination="+dest, nil)
	flushReq = flushReq.WithContext(adminContext())
	flushW := httptest.NewRecorder()

	handler.FlushQueue(flushW, flushReq)
	if flushW.Code == http.StatusTooManyRequests {
		t.Errorf("expected ?force=true to override cooldown block")
	}
}

func TestDeleteQueueItemAndClearQueue(t *testing.T) {
	handler, _ := setupTestEmailHandlerWithStore(t)

	// 1. Delete with missing ID returns 400 or invokes Clear
	delReq := httptest.NewRequest("DELETE", "/api/v1/email/queue/bad_id_format!", nil)
	delReq = delReq.WithContext(adminContext())
	delW := httptest.NewRecorder()

	handler.DeleteQueueItem(delW, delReq)
	// Expect failure or error since 'bad_id_format!' is invalid regex
	if delW.Code != http.StatusInternalServerError && delW.Code != http.StatusBadRequest {
		t.Logf("DeleteQueueItem returned code: %d, body: %s", delW.Code, delW.Body.String())
	}

	// 2. ClearQueue succeeds cleanly
	clearReq := httptest.NewRequest("DELETE", "/api/v1/email/queue", nil)
	clearReq = clearReq.WithContext(adminContext())
	clearW := httptest.NewRecorder()

	handler.ClearQueue(clearW, clearReq)
	if clearW.Code != http.StatusOK {
		t.Errorf("expected 200 from ClearQueue, got %d: %s", clearW.Code, clearW.Body.String())
	}
}

func TestMailboxRateLimitAPI(t *testing.T) {
	handler, memStore := setupTestEmailHandlerWithStore(t)
	ctx := adminContext()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	serverID := uuid.New()

	// 1. Setup Domain & Mailbox
	dom := &store.EmailDomain{
		ID:             uuid.New(),
		OrganizationID: orgID,
		ServerID:       serverID,
		Domain:         "example-limits.com",
		MailHostname:   "mail.example-limits.com",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	_ = memStore.CreateEmailDomain(ctx, dom)

	mb := &store.EmailMailbox{
		ID:               uuid.New(),
		DomainID:         dom.ID,
		ServerID:         serverID,
		LocalPart:        "support",
		Email:            "support@example-limits.com",
		Name:             "Support Team",
		QuotaBytes:       5368709120,
		RateLimitPerHour: 100,
		RateLimitPerDay:  1000,
		IsActive:         true,
	}
	_ = memStore.CreateEmailMailbox(ctx, mb)

	// 2. GET /mailboxes/{id}/rate-limit
	getReq := httptest.NewRequest("GET", "/api/v1/email/mailboxes/"+mb.ID.String()+"/rate-limit", nil)
	getReq = getReq.WithContext(ctx)
	// chi routing params helper
	rCtx := chi.NewRouteContext()
	rCtx.URLParams.Add("id", mb.ID.String())
	getReq = getReq.WithContext(context.WithValue(getReq.Context(), chi.RouteCtxKey, rCtx))

	getW := httptest.NewRecorder()
	handler.GetMailboxRateLimit(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 from GetMailboxRateLimit, got %d: %s", getW.Code, getW.Body.String())
	}

	var getRes map[string]any
	_ = json.Unmarshal(getW.Body.Bytes(), &getRes)
	data := getRes["data"].(map[string]any)
	if data["hour_limit"] != float64(100) || data["day_limit"] != float64(1000) {
		t.Errorf("unexpected limits: %+v", data)
	}

	// 3. PUT /mailboxes/{id}/rate-limit (update to 250/hr, 2500/day)
	updatePayload := []byte(`{"rate_limit_per_hour": 250, "rate_limit_per_day": 2500}`)
	putReq := httptest.NewRequest("PUT", "/api/v1/email/mailboxes/"+mb.ID.String()+"/rate-limit", bytes.NewReader(updatePayload))
	putReq = putReq.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rCtx))
	putW := httptest.NewRecorder()
	handler.UpdateMailboxRateLimit(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("expected 200 from UpdateMailboxRateLimit, got %d: %s", putW.Code, putW.Body.String())
	}

	var putRes map[string]any
	_ = json.Unmarshal(putW.Body.Bytes(), &putRes)
	putData := putRes["data"].(map[string]any)
	if putData["hour_limit"] != float64(250) || putData["day_limit"] != float64(2500) {
		t.Errorf("expected updated limits 250/2500, got: %+v", putData)
	}

	// 4. Record sending simulation
	ratecontrol.GetGlobalMailboxRateLimiter().RecordMailboxSent(mb.Email)
	ratecontrol.GetGlobalMailboxRateLimiter().RecordMailboxSent(mb.Email)

	// Check usage reflects sends
	chkReq := httptest.NewRequest("GET", "/api/v1/email/mailboxes/"+mb.ID.String()+"/rate-limit", nil)
	chkReq = chkReq.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rCtx))
	chkW := httptest.NewRecorder()
	handler.GetMailboxRateLimit(chkW, chkReq)
	var chkRes map[string]any
	_ = json.Unmarshal(chkW.Body.Bytes(), &chkRes)
	chkData := chkRes["data"].(map[string]any)
	if chkData["hour_sent"] != float64(2) {
		t.Errorf("expected hour_sent=2, got %v", chkData["hour_sent"])
	}

	// 5. POST /mailboxes/{id}/rate-limit/reset
	rstReq := httptest.NewRequest("POST", "/api/v1/email/mailboxes/"+mb.ID.String()+"/rate-limit/reset", nil)
	rstReq = rstReq.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rCtx))
	rstW := httptest.NewRecorder()
	handler.ResetMailboxRateLimit(rstW, rstReq)

	if rstW.Code != http.StatusOK {
		t.Fatalf("expected 200 from ResetMailboxRateLimit, got %d: %s", rstW.Code, rstW.Body.String())
	}

	var rstRes map[string]any
	_ = json.Unmarshal(rstW.Body.Bytes(), &rstRes)
	rstData := rstRes["data"].(map[string]any)
	if rstData["hour_sent"] != float64(0) {
		t.Errorf("expected hour_sent to reset to 0, got %v", rstData["hour_sent"])
	}
}

func TestDNSWizardAPI(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-jwt-secret-for-rate-control-tests-32-bytes",
	}
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auditLogger := audit.NewLogger(memStore, logger)
	dnsService := dns.NewService()

	handler := NewEmailHandler(cfg, memStore, dnsService, auditLogger)
	ctx := adminContext()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	serverID := uuid.New()

	// Mail server with public IP
	ms := &store.MailServer{
		ID:             uuid.New(),
		OrganizationID: orgID,
		NodeServerID:   serverID,
		Name:           "Edge Mail 1",
		Hostname:       "mail.wizard-test.com",
		IPv4Address:    "109.199.110.101",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	_ = memStore.CreateMailServer(ctx, ms)

	dom := &store.EmailDomain{
		ID:             uuid.New(),
		OrganizationID: orgID,
		ServerID:       serverID,
		MailServerID:   &ms.ID,
		Domain:         "wizard-test.com",
		MailHostname:   "mail.wizard-test.com",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	_ = memStore.CreateEmailDomain(ctx, dom)

	// 1. GET /domains/{id}/dns-wizard
	getReq := httptest.NewRequest("GET", "/api/v1/email/domains/"+dom.ID.String()+"/dns-wizard", nil)
	rCtx := chi.NewRouteContext()
	rCtx.URLParams.Add("id", dom.ID.String())
	getReq = getReq.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rCtx))

	getW := httptest.NewRecorder()
	handler.GetDNSWizardData(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 from GetDNSWizardData, got %d: %s", getW.Code, getW.Body.String())
	}

	var res map[string]any
	_ = json.Unmarshal(getW.Body.Bytes(), &res)
	data := res["data"].(map[string]any)

	if data["domain"] != "wizard-test.com" {
		t.Errorf("domain mismatch: %v", data["domain"])
	}
	if data["server_ip"] != "109.199.110.101" {
		t.Errorf("server_ip mismatch: %v", data["server_ip"])
	}

	// Verify BIND Zone Export is generated
	bindExport, ok := data["bind_zone_export"].(string)
	if !ok || !strings.Contains(bindExport, "$ORIGIN wizard-test.com.") || !strings.Contains(bindExport, "v=spf1") {
		t.Errorf("missing or malformed bind_zone_export: %s", bindExport)
	}

	// Verify Cloudflare records array is generated
	cfRecords, ok := data["cloudflare_records"].([]any)
	if !ok || len(cfRecords) < 5 {
		t.Errorf("expected at least 5 cloudflare records, got %v", cfRecords)
	}

	// Verify RBL check result is present
	rblCheck, ok := data["rbl_check"].(map[string]any)
	if !ok || rblCheck["status"] == "" {
		t.Errorf("expected valid rbl_check structure: %+v", data["rbl_check"])
	}

	// 2. POST /domains/{id}/dns-wizard/auto-fix
	postReq := httptest.NewRequest("POST", "/api/v1/email/domains/"+dom.ID.String()+"/dns-wizard/auto-fix", nil)
	postReq = postReq.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rCtx))
	postW := httptest.NewRecorder()

	handler.AutoFixDomainDNS(postW, postReq)
	if postW.Code != http.StatusOK {
		t.Fatalf("expected 200 from AutoFixDomainDNS, got %d: %s", postW.Code, postW.Body.String())
	}

	var autoFixRes map[string]any
	_ = json.Unmarshal(postW.Body.Bytes(), &autoFixRes)
	autoFixData := autoFixRes["data"].(map[string]any)
	if autoFixData["is_dns_verified"] != true {
		t.Errorf("expected is_dns_verified=true after auto fix")
	}
}


