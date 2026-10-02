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

	// Setup enabled gateway for testing
	_ = s.SaveGatewayConfig(context.Background(), &store.PaymentGatewayConfig{
		Gateway:   "stripe",
		SecretKey: "mock_test_secret",
		ApiKey:    "pk_test_123",
		TestMode:  true,
		Enabled:   true,
	})

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

	// 2. Test Subscribe to Starter Plan (Paid Subscription Starts as PENDING with Unpaid Invoice)
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

	var subResp struct {
		Data struct {
			Subscription *store.Subscription `json:"subscription"`
			Invoice      *store.Invoice      `json:"invoice"`
			CheckoutURL  string              `json:"checkout_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &subResp); err != nil {
		t.Fatalf("failed to parse subscription response: %v", err)
	}
	if subResp.Data.Subscription.Status != store.SubStatusPending {
		t.Fatalf("expected subscription to be pending, got %s", subResp.Data.Subscription.Status)
	}
	if subResp.Data.Invoice == nil || subResp.Data.Invoice.Status != store.InvoiceStatusUnpaid {
		t.Fatalf("expected unpaid invoice for paid subscription")
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

// TestBilling_TrialLifecycle tests free trial start, package limits application, and anti-abuse duplicate check
func TestBilling_TrialLifecycle(t *testing.T) {
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
	r.Post("/subscriptions", h.CreateSubscription)

	plans, _ := s.ListPlans(context.Background())
	var trialPlan *store.HostingPlan
	for _, p := range plans {
		if p.TrialDays > 0 {
			trialPlan = p
			break
		}
	}
	if trialPlan == nil {
		t.Fatalf("no plan with trial days found")
	}

	// 1. Eligible user starts trial
	trialReqBody, _ := json.Marshal(CreateSubscriptionRequest{
		PlanID:     trialPlan.ID.String(),
		StartTrial: true,
	})
	req := httptest.NewRequest("POST", "/subscriptions", bytes.NewReader(trialReqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for trial creation, got %d: %s", rec.Code, rec.Body.String())
	}

	var subResp struct {
		Data struct {
			Subscription *store.Subscription `json:"subscription"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &subResp)
	if subResp.Data.Subscription.Status != store.SubStatusTrial && subResp.Data.Subscription.Status != store.SubStatusActive {
		t.Fatalf("expected trial subscription to be active or trial, got %s", subResp.Data.Subscription.Status)
	}
	if subResp.Data.Subscription.TrialEndsAt == nil {
		t.Fatalf("expected trial ends at to be set")
	}

	// 2. Anti-abuse: Ineligible user cannot start duplicate trial
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/subscriptions", bytes.NewReader(trialReqBody))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for duplicate trial abuse, got %d: %s", rec2.Code, rec2.Body.String())
	}
	if !bytes.Contains(rec2.Body.Bytes(), []byte("TRIAL_ALREADY_USED")) {
		t.Fatalf("expected trial abuse error message, got: %s", rec2.Body.String())
	}
}

// TestBilling_ServerSidePaymentVerification tests server-side verification of payment gateway transactions
func TestBilling_ServerSidePaymentVerification(t *testing.T) {
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
	r.Post("/invoices/{id}/verify", h.VerifyInvoicePayment)

	// Configure Stripe in test mode
	_ = s.SaveGatewayConfig(context.Background(), &store.PaymentGatewayConfig{
		Gateway:   "stripe",
		SecretKey: "mock_test_secret",
		ApiKey:    "pk_test_123",
		TestMode:  true,
		Enabled:   true,
	})

	// Create plan & subscription
	plan := &store.HostingPlan{
		ID:           uuid.New(),
		Name:         "Pro",
		PriceMonthly: 20.00,
	}
	_ = s.CreatePlan(context.Background(), plan)

	sub := &store.Subscription{
		ID:             uuid.New(),
		UserID:         userID,
		OrganizationID: orgID,
		PlanID:         plan.ID,
		Status:         store.SubStatusPending,
	}
	_ = s.CreateSubscription(context.Background(), sub)

	inv := &store.Invoice{
		ID:             uuid.New(),
		InvoiceNumber:  "INV-PRO-001",
		UserID:         userID,
		OrganizationID: orgID,
		SubscriptionID: &sub.ID,
		PlanID:         plan.ID,
		Subtotal:       20.00,
		Total:          20.00,
		Currency:       "USD",
		Status:         store.InvoiceStatusUnpaid,
	}
	_ = s.CreateInvoice(context.Background(), inv)

	// 1. Test verification with failed payment reference -> must fail
	failedReq, _ := json.Marshal(VerifyPaymentRequest{
		PaymentMethod: "stripe",
		PaymentRef:    "cs_fail_test",
	})
	req := httptest.NewRequest("POST", "/invoices/"+inv.ID.String()+"/verify", bytes.NewReader(failedReq))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for failed payment reference, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Test verification with valid test transaction
	validReq, _ := json.Marshal(VerifyPaymentRequest{
		PaymentMethod: "stripe",
		PaymentRef:    "cs_test_mock_123",
	})
	req = httptest.NewRequest("POST", "/invoices/"+inv.ID.String()+"/verify", bytes.NewReader(validReq))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid verification, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify invoice marked paid and subscription active
	updatedInv, _ := s.GetInvoiceByID(context.Background(), inv.ID)
	if updatedInv.Status != store.InvoiceStatusPaid {
		t.Fatalf("expected invoice status paid, got %s", updatedInv.Status)
	}

	updatedSub, _ := s.GetSubscriptionByID(context.Background(), sub.ID)
	if updatedSub.Status != store.SubStatusActive {
		t.Fatalf("expected subscription status active, got %s", updatedSub.Status)
	}

	// 3. Test Idempotency: Re-verifying the same invoice returns OK without duplicate side effects
	reqDup := httptest.NewRequest("POST", "/invoices/"+inv.ID.String()+"/verify", bytes.NewReader(validReq))
	reqDup.Header.Set("Content-Type", "application/json")
	recDup := httptest.NewRecorder()
	r.ServeHTTP(recDup, reqDup)

	if recDup.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for idempotent repeat verification, got %d", recDup.Code)
	}

	// 4. Test Multi-tenant Isolation: User from Org B cannot verify invoice of Org A
	orgB := uuid.New()
	userB := uuid.New()
	rB := chi.NewRouter()
	rB.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
				UserID:         userB,
				OrganizationID: orgB,
				Role:           "member",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	rB.Post("/invoices/{id}/verify", h.VerifyInvoicePayment)

	reqB := httptest.NewRequest("POST", "/invoices/"+inv.ID.String()+"/verify", bytes.NewReader(validReq))
	reqB.Header.Set("Content-Type", "application/json")
	recB := httptest.NewRecorder()
	rB.ServeHTTP(recB, reqB)

	if recB.Code != http.StatusForbidden && recB.Code != http.StatusNotFound {
		t.Fatalf("expected 403 or 404 for cross-tenant invoice verification, got %d: %s", recB.Code, recB.Body.String())
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

	// 3b. Test Missing Signature in non-test mode (Must be rejected with 401)
	t.Run("MissingSignatureInProductionMode", func(t *testing.T) {
		payload := WebhookPayload{
			InvoiceID:     invID.String(),
			TransactionID: "txn_missing_sig",
			Amount:        25.00,
			Currency:      "USD",
			Status:        "paid",
		}
		rec := sendWebhook(payload, "")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized for missing signature, got %d", rec.Code)
		}
	})

	// 3c. Test Stripe Official Signature Scheme (t=timestamp,v1=signature)
	t.Run("StripeOfficialSignatureScheme", func(t *testing.T) {
		payload := WebhookPayload{
			InvoiceID:     invID.String(),
			TransactionID: "txn_stripe_v1_sig",
			Amount:        25.00,
			Currency:      "USD",
			Status:        "paid",
		}
		body, _ := json.Marshal(payload)
		ts := "1725800000"
		mac := hmac.New(sha256.New, []byte("stripe-secret-test-key"))
		mac.Write([]byte(ts + "."))
		mac.Write(body)
		v1Sig := hex.EncodeToString(mac.Sum(nil))
		stripeHeader := "t=" + ts + ",v1=" + v1Sig

		req := httptest.NewRequest("POST", "/billing/webhook/"+gatewayName, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Stripe-Signature", stripeHeader)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK for valid Stripe official signature, got %d: %s", rec.Code, rec.Body.String())
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
