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

// TestBillingWebhook_SecurityAndReplay verifies Section 16 & 17 payment webhook safeguards
func TestBillingWebhook_SecurityAndReplay(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)

	h := NewBillingHandler(cfg, s, auditLogger)

	r := chi.NewRouter()
	r.Post("/billing/webhook/{gateway}", h.HandleWebhook)

	ctx := context.Background()

	// 1. Setup payment gateway with secret key
	gatewayName := "stripe"
	_ = s.SaveGatewayConfig(ctx, &store.PaymentGatewayConfig{
		Gateway:   gatewayName,
		SecretKey: "stripe-secret-test-key",
		TestMode:  false,
		Enabled:   true,
	})

	// 2. Create an unpaid invoice
	invID := uuid.New()
	inv := &store.Invoice{
		ID:            invID,
		InvoiceNumber: "INV-TEST-001",
		UserID:        uuid.New(),
		Subtotal:      25.00,
		Total:         25.00,
		Currency:      "USD",
		Status:        store.InvoiceStatusUnpaid,
	}
	_ = s.CreateInvoice(ctx, inv)

	// Helper to send webhook
	sendWebhook := func(payload WebhookPayload, signature string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/billing/webhook/"+gatewayName, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if signature != "" {
			req.Header.Set("X-Signature", signature)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// 3. Test Invalid Signature
	t.Run("InvalidSignature", func(t *testing.T) {
		payload := WebhookPayload{
			InvoiceID:     invID.String(),
			TransactionID: "txn_sig_test_1",
			Amount:        25.00,
			Currency:      "USD",
			Status:        "paid",
		}
		rec := sendWebhook(payload, "invalid-hmac-signature")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized for bad signature, got %d", rec.Code)
		}
	})

	// 4. Test Amount Mismatch
	t.Run("AmountMismatch", func(t *testing.T) {
		// Update gateway to test mode so signature isn't blocking
		_ = s.SaveGatewayConfig(ctx, &store.PaymentGatewayConfig{
			Gateway:   gatewayName,
			TestMode:  true,
			Enabled:   true,
		})

		payload := WebhookPayload{
			InvoiceID:     invID.String(),
			TransactionID: "txn_amount_mismatch",
			Amount:        10.00, // Expected 25.00
			Currency:      "USD",
			Status:        "paid",
		}
		rec := sendWebhook(payload, "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request for amount mismatch, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 5. Test Currency Mismatch
	t.Run("CurrencyMismatch", func(t *testing.T) {
		payload := WebhookPayload{
			InvoiceID:     invID.String(),
			TransactionID: "txn_curr_mismatch",
			Amount:        25.00,
			Currency:      "EUR", // Expected USD
			Status:        "paid",
		}
		rec := sendWebhook(payload, "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request for currency mismatch, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 6. Test Valid Webhook -> Success
	t.Run("ValidWebhookAndReplayDeduplication", func(t *testing.T) {
		payload := WebhookPayload{
			InvoiceID:     invID.String(),
			TransactionID: "txn_success_valid_123",
			Amount:        25.00,
			Currency:      "USD",
			Status:        "paid",
		}

		// First delivery -> 200 OK success
		rec1 := sendWebhook(payload, "")
		if rec1.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK on first webhook, got %d: %s", rec1.Code, rec1.Body.String())
		}

		// Verify invoice marked paid
		updatedInv, err := s.GetInvoiceByID(ctx, invID)
		if err != nil || updatedInv.Status != store.InvoiceStatusPaid {
			t.Fatalf("Expected invoice marked paid, got status: %s", updatedInv.Status)
		}

		// Webhook #2 (Replay) -> Must be recognized as duplicate
		rec2 := sendWebhook(payload, "")
		if rec2.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK on replayed webhook, got %d", rec2.Code)
		}
		if !bytes.Contains(rec2.Body.Bytes(), []byte("idempotent_duplicate")) && !bytes.Contains(rec2.Body.Bytes(), []byte("idempotent_success")) {
			t.Errorf("Expected idempotent response on duplicate webhook, got: %s", rec2.Body.String())
		}

		// Webhook #3 (Replay)
		rec3 := sendWebhook(payload, "")
		if rec3.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK on 3rd webhook replay, got %d", rec3.Code)
		}
	})
}
