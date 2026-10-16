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
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/email/ratecontrol"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
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
