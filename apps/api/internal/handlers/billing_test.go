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

func TestBillingHandler_Flow(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)

	h := NewBillingHandler(cfg, s, auditLogger)

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

	r.Get("/plans", h.ListPlans)
	r.Post("/plans", h.CreatePlan)
	r.Get("/subscriptions", h.ListSubscriptions)
	r.Post("/subscriptions", h.CreateSubscription)
	r.Get("/invoices", h.ListInvoices)
	r.Post("/invoices/{id}/pay", h.PayInvoice)
	r.Get("/gateways", h.ListGateways)

	// 1. Test List Seed Plans
	req := httptest.NewRequest("GET", "/plans", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var plansResp struct {
		Data []*store.HostingPlan `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &plansResp); err != nil {
		t.Fatalf("failed to decode plans: %v", err)
	}
	if len(plansResp.Data) < 4 {
		t.Fatalf("expected at least 4 seed plans, got %d", len(plansResp.Data))
	}
	starterPlan := plansResp.Data[0]

	// 2. Test Subscribe to Starter Plan
	subBody, _ := json.Marshal(CreateSubscriptionRequest{
		PlanID:        starterPlan.ID.String(),
		BillingCycle:  "monthly",
		PaymentMethod: "stripe",
		AutoRenew:     true,
	})
	req = httptest.NewRequest("POST", "/subscriptions", bytes.NewReader(subBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for subscription, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Test List Invoices
	req = httptest.NewRequest("GET", "/invoices", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for invoices, got %d", rec.Code)
	}

	var invResp struct {
		Data []*store.Invoice `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &invResp); err != nil {
		t.Fatalf("failed to decode invoices: %v", err)
	}
	if len(invResp.Data) == 0 {
		t.Fatalf("expected at least 1 invoice created")
	}

	// 4. Test List Gateways
	req = httptest.NewRequest("GET", "/gateways", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for gateways, got %d", rec.Code)
	}
}
