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

	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestFirewallHandler_LifecycleAndSafety(t *testing.T) {
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}

	h := NewFirewallHandler(cfg, memStore, auditLogger)

	claims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "admin@hostvra.com",
		Role:           "owner",
	}

	// 1. Get Status
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/v1/firewall/status", nil)
	reqStatus = reqStatus.WithContext(context.WithValue(reqStatus.Context(), auth.UserContextKey, claims))
	recStatus := httptest.NewRecorder()
	h.GetStatus(recStatus, reqStatus)

	if recStatus.Code != http.StatusOK {
		t.Fatalf("expected 200 for GetStatus, got %d: %s", recStatus.Code, recStatus.Body.String())
	}

	// 2. List Rules
	reqRules := httptest.NewRequest(http.MethodGet, "/api/v1/firewall/rules", nil)
	reqRules = reqRules.WithContext(context.WithValue(reqRules.Context(), auth.UserContextKey, claims))
	recRules := httptest.NewRecorder()
	h.ListRules(recRules, reqRules)

	if recRules.Code != http.StatusOK {
		t.Fatalf("expected 200 for ListRules, got %d: %s", recRules.Code, recRules.Body.String())
	}

	// 3. Add Rule - SSH Lockout rejection test
	denySSHBody, _ := json.Marshal(AddRuleRequest{
		Port:     "22",
		Protocol: "tcp",
		Action:   "deny",
		Comment:  "Accidental SSH lockout attempt",
	})
	reqDenySSH := httptest.NewRequest(http.MethodPost, "/api/v1/firewall/rules", bytes.NewReader(denySSHBody))
	reqDenySSH = reqDenySSH.WithContext(context.WithValue(reqDenySSH.Context(), auth.UserContextKey, claims))
	recDenySSH := httptest.NewRecorder()
	h.AddRule(recDenySSH, reqDenySSH)

	if recDenySSH.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when attempting to deny SSH, got %d: %s", recDenySSH.Code, recDenySSH.Body.String())
	}

	// 4. Add Rule - Valid port rule
	allowHTTPBody, _ := json.Marshal(AddRuleRequest{
		Port:     "8080",
		Protocol: "tcp",
		Action:   "allow",
		Comment:  "Custom Web Application",
	})
	reqAllowHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/firewall/rules", bytes.NewReader(allowHTTPBody))
	reqAllowHTTP = reqAllowHTTP.WithContext(context.WithValue(reqAllowHTTP.Context(), auth.UserContextKey, claims))
	recAllowHTTP := httptest.NewRecorder()
	h.AddRule(recAllowHTTP, reqAllowHTTP)

	if recAllowHTTP.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for valid rule, got %d: %s", recAllowHTTP.Code, recAllowHTTP.Body.String())
	}

	// 5. Delete Rule with query param
	reqDelete := httptest.NewRequest(http.MethodDelete, "/api/v1/firewall/rules?id=8080", nil)
	reqDelete = reqDelete.WithContext(context.WithValue(reqDelete.Context(), auth.UserContextKey, claims))
	recDelete := httptest.NewRecorder()
	h.DeleteRule(recDelete, reqDelete)

	if recDelete.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for DeleteRule, got %d: %s", recDelete.Code, recDelete.Body.String())
	}

	// 6. Fail2ban Jails list
	reqJails := httptest.NewRequest(http.MethodGet, "/api/v1/firewall/fail2ban/jails", nil)
	reqJails = reqJails.WithContext(context.WithValue(reqJails.Context(), auth.UserContextKey, claims))
	recJails := httptest.NewRecorder()
	h.ListJails(recJails, reqJails)

	if recJails.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for ListJails, got %d: %s", recJails.Code, recJails.Body.String())
	}

	// 7. ICMP Ping Status & Toggle
	reqPingGet := httptest.NewRequest(http.MethodGet, "/api/v1/firewall/ping", nil)
	reqPingGet = reqPingGet.WithContext(context.WithValue(reqPingGet.Context(), auth.UserContextKey, claims))
	recPingGet := httptest.NewRecorder()
	h.GetPingStatus(recPingGet, reqPingGet)
	if recPingGet.Code != http.StatusOK {
		t.Fatalf("expected 200 for GetPingStatus, got %d: %s", recPingGet.Code, recPingGet.Body.String())
	}

	setPingBody, _ := json.Marshal(SetPingRequest{Prohibit: true})
	reqPingPost := httptest.NewRequest(http.MethodPost, "/api/v1/firewall/ping", bytes.NewReader(setPingBody))
	reqPingPost = reqPingPost.WithContext(context.WithValue(reqPingPost.Context(), auth.UserContextKey, claims))
	recPingPost := httptest.NewRecorder()
	h.SetPingStatus(recPingPost, reqPingPost)
	if recPingPost.Code != http.StatusOK {
		t.Fatalf("expected 200 for SetPingStatus, got %d: %s", recPingPost.Code, recPingPost.Body.String())
	}

	// 8. Listening Ports
	reqPorts := httptest.NewRequest(http.MethodGet, "/api/v1/firewall/ports", nil)
	reqPorts = reqPorts.WithContext(context.WithValue(reqPorts.Context(), auth.UserContextKey, claims))
	recPorts := httptest.NewRecorder()
	h.GetListeningPorts(recPorts, reqPorts)
	if recPorts.Code != http.StatusOK {
		t.Fatalf("expected 200 for GetListeningPorts, got %d: %s", recPorts.Code, recPorts.Body.String())
	}
}
