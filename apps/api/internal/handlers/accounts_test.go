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

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
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
	r.Post("/accounts/{id}/suspend", h.SuspendAccount)
	r.Post("/accounts/{id}/unsuspend", h.UnsuspendAccount)
	r.Post("/accounts/{id}/login-token", h.GenerateLoginToken)

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

	// 2. Test Suspend Account
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

	// 3. Test Unsuspend Account
	req = httptest.NewRequest("POST", "/accounts/"+targetAccount.ID.String()+"/unsuspend", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for unsuspend, got %d", rec.Code)
	}

	// 4. Test 1-Click Login Token
	req = httptest.NewRequest("POST", "/accounts/"+targetAccount.ID.String()+"/login-token", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for login token, got %d", rec.Code)
	}

	var tokenResp struct {
		Data map[string]string `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &tokenResp)
	if tokenResp.Data == nil || tokenResp.Data["login_url"] == "" {
		t.Fatalf("expected valid login_url in response")
	}
}
