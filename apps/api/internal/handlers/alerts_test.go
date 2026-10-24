package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/alerts"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestAlertsAndAPIKeysHandlers_E2E(t *testing.T) {
	cfg := &config.Config{}
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)

	alertEngine := alerts.NewEngine()
	alertEngine.SetStore(memStore)

	alertHandler := NewAlertHandler(cfg, alertEngine, auditLogger)
	apiKeyHandler := NewAPIKeyHandler(cfg, memStore, auditLogger)

	claims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "devops@hostvra.internal",
		Role:           "superadmin",
		IsSuperAdmin:   true,
	}

	// 1. Create Alert Rule
	rulePayload := CreateRuleRequest{
		Name:        "Production Memory Warning",
		Type:        alerts.RuleMemory,
		Threshold:   85.0,
		Severity:    alerts.SeverityWarning,
		DurationSec: 60,
	}
	body, _ := json.Marshal(rulePayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/rules", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	rec := httptest.NewRecorder()

	alertHandler.CreateRule(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on create rule, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. List Alert Rules
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/rules", nil)
	listReq = listReq.WithContext(auth.WithClaims(listReq.Context(), claims))
	listRec := httptest.NewRecorder()

	alertHandler.ListRules(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list rules, got %d", listRec.Code)
	}

	// 3. Test Trigger Alert
	testReq := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/test", nil)
	testReq = testReq.WithContext(auth.WithClaims(testReq.Context(), claims))
	testRec := httptest.NewRecorder()

	alertHandler.TestTrigger(testRec, testReq)
	if testRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on test trigger, got %d", testRec.Code)
	}

	// 4. List Incidents
	incReq := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	incReq = incReq.WithContext(auth.WithClaims(incReq.Context(), claims))
	incRec := httptest.NewRecorder()

	alertHandler.ListIncidents(incRec, incReq)
	if incRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list incidents, got %d", incRec.Code)
	}

	// 5. Create Notification Channel
	chPayload := CreateChannelRequest{
		Name:   "DevOps Webhook",
		Type:   alerts.ChannelWebhook,
		Target: "https://webhook.site/test",
	}
	chBody, _ := json.Marshal(chPayload)
	chReq := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/channels", bytes.NewReader(chBody))
	chReq = chReq.WithContext(auth.WithClaims(chReq.Context(), claims))
	chRec := httptest.NewRecorder()

	alertHandler.CreateChannel(chRec, chReq)
	if chRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on create channel, got %d", chRec.Code)
	}

	// 5b. Verify SSRF attack against internal metadata is blocked
	ssrfPayload := CreateChannelRequest{
		Name:   "Evil Webhook",
		Type:   alerts.ChannelWebhook,
		Target: "http://169.254.169.254/latest/meta-data",
	}
	ssrfBody, _ := json.Marshal(ssrfPayload)
	ssrfReq := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/channels", bytes.NewReader(ssrfBody))
	ssrfReq = ssrfReq.WithContext(auth.WithClaims(ssrfReq.Context(), claims))
	ssrfRec := httptest.NewRecorder()
	alertHandler.CreateChannel(ssrfRec, ssrfReq)
	if ssrfRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on SSRF target, got %d", ssrfRec.Code)
	}

	// 6. Create API Key
	keyPayload := CreateKeyRequest{
		Name: "Production Terraform Token",
		Role: "manager",
	}
	keyBody, _ := json.Marshal(keyPayload)
	keyReq := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", bytes.NewReader(keyBody))
	keyReq = keyReq.WithContext(auth.WithClaims(keyReq.Context(), claims))
	keyRec := httptest.NewRecorder()

	apiKeyHandler.Create(keyRec, keyReq)
	if keyRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on create api key, got %d: %s", keyRec.Code, keyRec.Body.String())
	}

	var keyResp struct {
		Data APIKeyRecord `json:"data"`
	}
	_ = json.NewDecoder(keyRec.Body).Decode(&keyResp)
	createdKeyID := keyResp.Data.ID

	// 7. List API Keys (should find persisted key)
	keysListReq := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys", nil)
	keysListReq = keysListReq.WithContext(auth.WithClaims(keysListReq.Context(), claims))
	keysListRec := httptest.NewRecorder()

	apiKeyHandler.List(keysListRec, keysListReq)
	if keysListRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list api keys, got %d", keysListRec.Code)
	}

	// 8. Revoke API Key
	revReq := httptest.NewRequest(http.MethodDelete, "/api/v1/api-keys/"+createdKeyID.String(), nil)
	revReq = revReq.WithContext(auth.WithClaims(revReq.Context(), claims))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyID", createdKeyID.String())
	revReq = revReq.WithContext(context.WithValue(revReq.Context(), chi.RouteCtxKey, rctx))
	revRec := httptest.NewRecorder()

	apiKeyHandler.Revoke(revRec, revReq)
	if revRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on revoke api key, got %d", revRec.Code)
	}
}
