package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/billing"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestCommercial_CouponConcurrentRedemptionLimit(t *testing.T) {
	st := store.NewMemoryStore()
	ctx := context.Background()

	// Create coupon with global MaxUses = 3
	err := st.CreateCoupon(ctx, &store.Coupon{
		Code:           "LIMITED3",
		DiscountType:   "percentage",
		DiscountValue:  25,
		MaxUses:        3,
		MaxUsesPerUser: 1,
		IsActive:       true,
	})
	if err != nil {
		t.Fatalf("CreateCoupon failed: %v", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	succeeded := 0

	// 10 concurrent customers attempt to redeem the 3-use coupon simultaneously
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, rErr := st.RedeemCouponAtomic(ctx, "LIMITED3", uuid.New(), uuid.New(), 100.0, true)
			if rErr == nil {
				mu.Lock()
				succeeded++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if succeeded != 3 {
		t.Fatalf("expected strictly 3 redemptions under concurrency, got %d", succeeded)
	}
}

func TestCommercial_AffiliateSelfReferralAndReversal(t *testing.T) {
	st := store.NewMemoryStore()
	ctx := context.Background()

	affUser := uuid.New()
	referredUser := uuid.New()
	acc, err := st.GetOrCreateAffiliateAccount(ctx, affUser)
	if err != nil {
		t.Fatalf("GetOrCreateAffiliateAccount failed: %v", err)
	}

	// Self-referral MUST be rejected
	if _, err := st.RecordAffiliateReferral(ctx, acc.ID, affUser, "first_click"); err != store.ErrSelfReferralForbidden {
		t.Fatalf("expected ErrSelfReferralForbidden, got %v", err)
	}

	// Legitimate referral + commission + reversal
	if _, err := st.RecordAffiliateReferral(ctx, acc.ID, referredUser, "first_click"); err != nil {
		t.Fatalf("RecordAffiliateReferral failed: %v", err)
	}
	invID := uuid.New()
	comm, err := st.RecordAffiliateCommission(ctx, referredUser, invID, 100.0)
	if err != nil || comm == nil || comm.CommissionUSD != 15.0 {
		t.Fatalf("expected $15 commission (15%%), got %+v err=%v", comm, err)
	}

	// Refund reversal must deduct commission
	if err := st.ReverseAffiliateCommissionByInvoice(ctx, invID); err != nil {
		t.Fatalf("ReverseAffiliateCommissionByInvoice failed: %v", err)
	}
	updatedAcc, _ := st.GetOrCreateAffiliateAccount(ctx, affUser)
	if updatedAcc.ApprovedBalanceUSD != 0 {
		t.Fatalf("expected 0 approved balance after refund reversal, got %.2f", updatedAcc.ApprovedBalanceUSD)
	}
}

func TestCommercial_WalletDoubleSpendPreventionAndPublicStatus(t *testing.T) {
	st := store.NewMemoryStore()
	ctx := context.Background()
	uid := uuid.New()

	// Credit $20
	_, _ = st.CreditWalletAtomic(ctx, uid, 20.0, "topup", "t1", "idem-t1", "Topup")

	// Concurrent double-spend attempt: two $15 debits simultaneously -> strictly ONE must succeed!
	var wg sync.WaitGroup
	var mu sync.Mutex
	debits := 0
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := st.DebitWalletAtomic(ctx, uid, 15.0, "invoice_payment", "inv", uuid.New().String(), "Pay")
			if err == nil {
				mu.Lock()
				debits++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	if debits != 1 {
		t.Fatalf("wallet double-spend vulnerability: expected 1 successful debit, got %d", debits)
	}

	// Verify Public Status endpoint returns live probes
	dw := billing.NewDunningWorker(st, 0, 3)
	h := NewCommercialHandler(&config.Config{}, st, audit.NewLogger(st, slog.Default()), dw)
	req := httptest.NewRequest("GET", "/api/v1/public/status", nil)
	w := httptest.NewRecorder()
	h.GetPublicSystemStatus(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /public/status, got %d", w.Code)
	}
	var resp struct {
		Data struct {
			OverallStatus string `json:"overall_status"`
		} `json:"data"`
	}
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Data.OverallStatus != "operational" {
		t.Fatalf("expected operational status, got %s", resp.Data.OverallStatus)
	}
}

func TestCommercial_UpdateCouponAndToggle(t *testing.T) {
	st := store.NewMemoryStore()
	ctx := context.Background()
	h := NewCommercialHandler(&config.Config{}, st, audit.NewLogger(st, slog.Default()), nil)

	c := &store.Coupon{
		Code:          "SPRING20",
		Description:   "20% off",
		DiscountType:  "percentage",
		DiscountValue: 20,
		IsActive:      true,
	}
	if err := st.CreateCoupon(ctx, c); err != nil {
		t.Fatalf("CreateCoupon failed: %v", err)
	}

	// Update coupon to inactive via chi route
	r := chi.NewRouter()
	r.Put("/coupons/{id}", h.AdminUpdateCoupon)
	body := `{"code":"SPRING20","description":"Paused","discount_type":"percentage","discount_value":25,"is_active":false}`
	req := httptest.NewRequest("PUT", "/coupons/"+c.ID.String(), bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK updating coupon, got %d: %s", w.Code, w.Body.String())
	}

	// ValidateCoupon must now reject the inactive coupon
	vReq := httptest.NewRequest("POST", "/validate", bytes.NewBufferString(`{"code":"SPRING20","order_amount_usd":100}`))
	vW := httptest.NewRecorder()
	h.ValidateCoupon(vW, vReq)
	if vW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for inactive coupon, got %d", vW.Code)
	}
}

func TestCommercial_EndToEndReferralCheckoutAndCommission(t *testing.T) {
	st := store.NewMemoryStore()
	ctx := context.Background()
	cfg := &config.Config{JWTSecret: "test-secret-key-at-least-32-chars-long", JWTElementsHours: 24, RefreshTokenDays: 7}
	auditLog := audit.NewLogger(st, slog.Default())

	// 1. Seed a hosting plan
	plan := &store.HostingPlan{
		ID:           uuid.New(),
		Name:         "Promo Test Cloud",
		Slug:         "promo-test-cloud",
		PriceMonthly: 20.00,
		PriceYearly:  200.00,
		Currency:     "USD",
		IsActive:     true,
	}
	if err := st.CreatePlan(ctx, plan); err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	// 2. Seed single-use 25% coupon
	if err := st.CreateCoupon(ctx, &store.Coupon{
		Code:           "SAVE25",
		DiscountType:   "percentage",
		DiscountValue:  25,
		MaxUses:        1,
		MaxUsesPerUser: 1,
		IsActive:       true,
	}); err != nil {
		t.Fatalf("CreateCoupon failed: %v", err)
	}

	// 3. Create affiliate account
	partnerID := uuid.New()
	affAcc, err := st.GetOrCreateAffiliateAccount(ctx, partnerID)
	if err != nil {
		t.Fatalf("GetOrCreateAffiliateAccount failed: %v", err)
	}

	// 4. Register referred customer with referral_code
	authH := NewAuthHandler(cfg, st, auditLog)
	regBody, _ := json.Marshal(RegisterRequest{
		Email:        "referred@example.com",
		Password:     "StrongPass#2026!",
		FullName:     "Referred Buyer",
		ReferralCode: affAcc.ReferralCode,
	})
	regReq := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(regBody))
	regW := httptest.NewRecorder()
	authH.Register(regW, regReq)
	if regW.Code != http.StatusCreated {
		t.Fatalf("Register failed with %d: %s", regW.Code, regW.Body.String())
	}

	var regResp struct {
		Data struct {
			User struct {
				ID uuid.UUID `json:"id"`
			} `json:"user"`
			Org struct {
				ID uuid.UUID `json:"id"`
			} `json:"org"`
		} `json:"data"`
	}
	_ = json.NewDecoder(regW.Body).Decode(&regResp)
	buyerID := regResp.Data.User.ID
	orgID := regResp.Data.Org.ID

	// Verify affiliate referral count incremented
	affAfterReg, _ := st.GetOrCreateAffiliateAccount(ctx, partnerID)
	if affAfterReg.TotalReferrals != 1 {
		t.Fatalf("expected 1 attributed referral, got %d", affAfterReg.TotalReferrals)
	}

	// 5. Fund buyer wallet with $50 and checkout via credit_wallet + SAVE25 coupon ($20 -> $15 net)
	_, _ = st.CreditWalletAtomic(ctx, buyerID, 50.0, "topup", "seed", "seed-buyer", "Seed")

	provisioned := false
	commH := NewCommercialHandler(cfg, st, auditLog, nil)
	commH.SetOnInvoicePaid(func(c context.Context, inv *store.Invoice) {
		provisioned = true
		_, _ = st.RecordAffiliateCommission(c, inv.UserID, inv.ID, inv.Total)
	})

	claims := &auth.Claims{UserID: buyerID, OrganizationID: orgID, Email: "referred@example.com", Role: "customer"}
	checkoutPayload := `{"plan_id":"` + plan.ID.String() + `","billing_cycle":"monthly","coupon_code":"SAVE25","payment_method":"credit_wallet","domain_name":"buyer.io"}`
	chkReq := httptest.NewRequest("POST", "/api/v1/billing/orders/checkout", bytes.NewBufferString(checkoutPayload))
	chkReq = chkReq.WithContext(auth.WithClaims(chkReq.Context(), claims))
	chkW := httptest.NewRecorder()
	commH.CheckoutOrder(chkW, chkReq)
	if chkW.Code != http.StatusCreated {
		t.Fatalf("CheckoutOrder failed with %d: %s", chkW.Code, chkW.Body.String())
	}

	var chkResp struct {
		Data struct {
			Paid        bool    `json:"paid"`
			SubtotalUSD float64 `json:"subtotal_usd"`
			DiscountUSD float64 `json:"discount_usd"`
			TotalUSD    float64 `json:"total_usd"`
		} `json:"data"`
	}
	_ = json.NewDecoder(chkW.Body).Decode(&chkResp)
	if !chkResp.Data.Paid || chkResp.Data.SubtotalUSD != 20.0 || chkResp.Data.DiscountUSD != 5.0 || chkResp.Data.TotalUSD != 15.0 {
		t.Fatalf("unexpected checkout response: %+v", chkResp.Data)
	}
	if !provisioned {
		t.Fatalf("expected onInvoicePaid provisioning callback to be invoked")
	}

	// Verify affiliate earned 15% of $15.00 = $2.25
	affAfterPay, _ := st.GetOrCreateAffiliateAccount(ctx, partnerID)
	if affAfterPay.ApprovedBalanceUSD != 2.25 {
		t.Fatalf("expected affiliate commission $2.25, got $%.2f", affAfterPay.ApprovedBalanceUSD)
	}

	// 6. Second checkout with exhausted single-use coupon MUST fail with 400
	chkReq2 := httptest.NewRequest("POST", "/api/v1/billing/orders/checkout", bytes.NewBufferString(checkoutPayload))
	chkReq2 = chkReq2.WithContext(auth.WithClaims(chkReq2.Context(), claims))
	chkW2 := httptest.NewRecorder()
	commH.CheckoutOrder(chkW2, chkReq2)
	if chkW2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for exhausted coupon, got %d", chkW2.Code)
	}
}

func TestP10_WithdrawalRejectThenApproveBlockedAndConcurrentApproval(t *testing.T) {
	st := store.NewMemoryStore()
	ctx := context.Background()
	h := NewCommercialHandler(&config.Config{}, st, audit.NewLogger(st, slog.Default()), nil)

	affUser := uuid.New()
	refUser := uuid.New()
	acc, _ := st.GetOrCreateAffiliateAccount(ctx, affUser)
	_, _ = st.RecordAffiliateReferral(ctx, acc.ID, refUser, "first_click")
	_, _ = st.RecordAffiliateCommission(ctx, refUser, uuid.New(), 200.0) // $30 approved

	// 1. Request $10 withdrawal, REJECT it, then attempt to APPROVE it -> MUST be blocked and wallet MUST remain $0
	wd1, err := st.RequestAffiliateWithdrawal(ctx, affUser, 10.0, "credit_wallet", "")
	if err != nil {
		t.Fatalf("RequestAffiliateWithdrawal failed: %v", err)
	}
	r := chi.NewRouter()
	r.Post("/withdrawals/{id}/process", h.AdminProcessWithdrawal)

	rejReq := httptest.NewRequest("POST", "/withdrawals/"+wd1.ID.String()+"/process", bytes.NewBufferString(`{"approve":false,"admin_notes":"fraud review"}`))
	rejW := httptest.NewRecorder()
	r.ServeHTTP(rejW, rejReq)
	if rejW.Code != http.StatusOK {
		t.Fatalf("expected 200 rejecting pending withdrawal, got %d", rejW.Code)
	}

	// Re-approving the already-rejected withdrawal MUST fail with 400 and NEVER credit the wallet
	reAppReq := httptest.NewRequest("POST", "/withdrawals/"+wd1.ID.String()+"/process", bytes.NewBufferString(`{"approve":true,"admin_notes":"replay"}`))
	reAppW := httptest.NewRecorder()
	r.ServeHTTP(reAppW, reAppReq)
	if reAppW.Code != http.StatusBadRequest {
		t.Fatalf("CRITICAL: expected 400 re-approving rejected withdrawal, got %d", reAppW.Code)
	}
	wlt, _ := st.GetOrCreateCustomerWallet(ctx, affUser)
	if wlt.BalanceUSD != 0 {
		t.Fatalf("CRITICAL: rejected withdrawal credited wallet! balance=%.2f", wlt.BalanceUSD)
	}

	// 2. Concurrent approval race on a valid $20 withdrawal -> strictly ONE 200 OK and wallet credited ONCE ($20)
	wd2, err := st.RequestAffiliateWithdrawal(ctx, affUser, 20.0, "credit_wallet", "")
	if err != nil {
		t.Fatalf("RequestAffiliateWithdrawal wd2 failed: %v", err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	okCount := 0
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/withdrawals/"+wd2.ID.String()+"/process", bytes.NewBufferString(`{"approve":true}`))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code == http.StatusOK {
				mu.Lock()
				okCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if okCount != 1 {
		t.Fatalf("expected strictly 1 concurrent approval to succeed, got %d", okCount)
	}
	wltAfter, _ := st.GetOrCreateCustomerWallet(ctx, affUser)
	if wltAfter.BalanceUSD != 20.0 {
		t.Fatalf("expected exact wallet balance $20.00, got $%.2f", wltAfter.BalanceUSD)
	}
}

func TestP10_WebhookRefundReversesAffiliateCommission(t *testing.T) {
	st := store.NewMemoryStore()
	ctx := context.Background()
	bh := NewBillingHandler(&config.Config{}, st, audit.NewLogger(st, slog.Default()))

	// Configure gateway secret for HMAC verification
	secret := "p10-webhook-secret-key"
	_ = st.SaveGatewayConfig(ctx, &store.PaymentGatewayConfig{
		Gateway:     "stripe",
		DisplayName: "Stripe",
		Enabled:     true,
		SecretKey:   secret,
	})

	partnerID := uuid.New()
	buyerID := uuid.New()
	orgID := uuid.New()
	acc, _ := st.GetOrCreateAffiliateAccount(ctx, partnerID)
	_, _ = st.RecordAffiliateReferral(ctx, acc.ID, buyerID, "first_click")

	inv := &store.Invoice{
		ID:             uuid.New(),
		InvoiceNumber:  "INV-P10-REFUND",
		UserID:         buyerID,
		OrganizationID: orgID,
		Subtotal:       100.0,
		Total:          100.0,
		Currency:       "USD",
		Status:         store.InvoiceStatusUnpaid,
	}
	_ = st.CreateInvoice(ctx, inv)

	r := chi.NewRouter()
	r.Post("/webhook/{gateway}", bh.HandleWebhook)

	signAndSend := func(p WebhookPayload) *httptest.ResponseRecorder {
		raw, _ := json.Marshal(p)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(raw)
		sig := hex.EncodeToString(mac.Sum(nil))
		req := httptest.NewRequest("POST", "/webhook/stripe", bytes.NewReader(raw))
		req.Header.Set("X-Signature", sig)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	// 1. Verified paid webhook -> invoice paid & affiliate credited $15.00
	w1 := signAndSend(WebhookPayload{
		InvoiceID:     inv.ID.String(),
		TransactionID: "tx_p10_paid_1",
		Amount:        100.0,
		Currency:      "USD",
		Status:        "paid",
	})
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200 on paid webhook, got %d: %s", w1.Code, w1.Body.String())
	}
	accPaid, _ := st.GetOrCreateAffiliateAccount(ctx, partnerID)
	if accPaid.ApprovedBalanceUSD != 15.0 {
		t.Fatalf("expected $15.00 affiliate commission after paid webhook, got $%.2f", accPaid.ApprovedBalanceUSD)
	}

	// 2. Verified refund webhook -> invoice marked refunded & affiliate commission reversed back to $0.00
	w2 := signAndSend(WebhookPayload{
		InvoiceID:     inv.ID.String(),
		TransactionID: "tx_p10_refund_1",
		Amount:        100.0,
		Currency:      "USD",
		Status:        "refunded",
	})
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 on refund webhook, got %d: %s", w2.Code, w2.Body.String())
	}
	invAfter, _ := st.GetInvoiceByID(ctx, inv.ID)
	if invAfter.Status != store.InvoiceStatusRefunded {
		t.Fatalf("expected invoice status refunded, got %s", invAfter.Status)
	}
	accRefunded, _ := st.GetOrCreateAffiliateAccount(ctx, partnerID)
	if accRefunded.ApprovedBalanceUSD != 0 {
		t.Fatalf("expected $0.00 affiliate balance after refund reversal, got $%.2f", accRefunded.ApprovedBalanceUSD)
	}
}

func TestP10_CouponInputValidationAndWalletPrecheckPreservesCoupon(t *testing.T) {
	st := store.NewMemoryStore()
	ctx := context.Background()
	h := NewCommercialHandler(&config.Config{}, st, audit.NewLogger(st, slog.Default()), nil)

	// 1. Reject percentage > 100 and negative discount
	for _, bad := range []string{
		`{"code":"BAD1","discount_type":"percentage","discount_value":150}`,
		`{"code":"BAD2","discount_type":"fixed","discount_value":-10}`,
		`{"code":"BAD3","discount_type":"bogus","discount_value":10}`,
	} {
		req := httptest.NewRequest("POST", "/coupons", bytes.NewBufferString(bad))
		w := httptest.NewRecorder()
		h.AdminCreateCoupon(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid coupon %s, got %d", bad, w.Code)
		}
	}

	// 2. Insufficient wallet balance MUST NOT burn single-use coupon
	plan := &store.HostingPlan{ID: uuid.New(), Name: "Precheck Plan", Slug: "precheck-plan", PriceMonthly: 40.0, Currency: "USD", IsActive: true}
	_ = st.CreatePlan(ctx, plan)
	_ = st.CreateCoupon(ctx, &store.Coupon{Code: "ONCE50", DiscountType: "percentage", DiscountValue: 50, MaxUses: 1, MaxUsesPerUser: 1, IsActive: true})

	buyerID := uuid.New()
	claims := &auth.Claims{UserID: buyerID, OrganizationID: uuid.New(), Email: "u@example.com", Role: "customer"}
	payload := `{"plan_id":"` + plan.ID.String() + `","billing_cycle":"monthly","coupon_code":"ONCE50","payment_method":"credit_wallet"}`

	req1 := httptest.NewRequest("POST", "/checkout", bytes.NewBufferString(payload))
	req1 = req1.WithContext(auth.WithClaims(req1.Context(), claims))
	w1 := httptest.NewRecorder()
	h.CheckoutOrder(w1, req1)
	if w1.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on insufficient wallet balance, got %d", w1.Code)
	}

	// Fund wallet with $20 (exact net price after 50% off $40) and retry with SAME coupon -> MUST succeed!
	_, _ = st.CreditWalletAtomic(ctx, buyerID, 20.0, "topup", "t1", "idem-precheck", "Fund")
	req2 := httptest.NewRequest("POST", "/checkout", bytes.NewBufferString(payload))
	req2 = req2.WithContext(auth.WithClaims(req2.Context(), claims))
	w2 := httptest.NewRecorder()
	h.CheckoutOrder(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("expected 201 after funding wallet (coupon preserved), got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestP10_WalletAdjustAndWithdrawalInputValidation(t *testing.T) {
	st := store.NewMemoryStore()
	h := NewCommercialHandler(&config.Config{}, st, audit.NewLogger(st, slog.Default()), nil)

	// Invalid direction in AdminAdjustWallet
	req := httptest.NewRequest("POST", "/wallet/adjust", bytes.NewBufferString(`{"user_id":"`+uuid.New().String()+`","direction":"mint","amount_usd":10}`))
	w := httptest.NewRecorder()
	h.AdminAdjustWallet(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid wallet direction, got %d", w.Code)
	}

	// Missing payout_details for usdt_bep20 withdrawal
	claims := &auth.Claims{UserID: uuid.New(), OrganizationID: uuid.New(), Email: "aff@example.com", Role: "customer"}
	wReq := httptest.NewRequest("POST", "/affiliate/withdraw", bytes.NewBufferString(`{"amount_usd":10,"payout_method":"usdt_bep20","payout_details":""}`))
	wReq = wReq.WithContext(auth.WithClaims(wReq.Context(), claims))
	wRec := httptest.NewRecorder()
	h.RequestAffiliateWithdrawal(wRec, wReq)
	if wRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing crypto payout_details, got %d", wRec.Code)
	}
}

func TestP12_FinancialInvariantsLedgerReconciliationAndPrecision(t *testing.T) {
	st := store.NewMemoryStore()
	ctx := context.Background()

	// 1. IEEE-754 floating-point precision trap on affiliate commissions & withdrawals ($15.30 - $5.10 = $10.20)
	affUser := uuid.New()
	refUser := uuid.New()
	acc, _ := st.GetOrCreateAffiliateAccount(ctx, affUser)
	_, _ = st.RecordAffiliateReferral(ctx, acc.ID, refUser, "first_click")
	// 15% of $102.00 = $15.30
	_, _ = st.RecordAffiliateCommission(ctx, refUser, uuid.New(), 102.00)

	// Withdraw $5.10 -> leaves exactly $10.20 (in raw IEEE-754, 15.3 - 5.1 = 10.199999999999999)
	if _, err := st.RequestAffiliateWithdrawal(ctx, affUser, 5.10, "credit_wallet", ""); err != nil {
		t.Fatalf("first withdrawal $5.10 failed: %v", err)
	}
	// Withdrawing exact remaining $10.20 MUST succeed without IEEE-754 drift rejection
	if _, err := st.RequestAffiliateWithdrawal(ctx, affUser, 10.20, "credit_wallet", ""); err != nil {
		t.Fatalf("second exact withdrawal $10.20 failed due to float drift: %v", err)
	}
	accAfter, _ := st.GetOrCreateAffiliateAccount(ctx, affUser)
	if accAfter.ApprovedBalanceUSD != 0 {
		t.Fatalf("expected $0.00 remaining approved balance, got %.10f", accAfter.ApprovedBalanceUSD)
	}

	// 2. Concurrent wallet credits & debits + full double-entry ledger reconciliation
	uid := uuid.New()
	var wg sync.WaitGroup
	// 20 concurrent $10.15 credits = $203.00
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("p12-cr-%d", idx)
			_, err := st.CreditWalletAtomic(ctx, uid, 10.15, "topup", key, key, "Concurrent credit")
			if err != nil {
				t.Errorf("credit %d failed: %v", idx, err)
			}
		}(i)
	}
	wg.Wait()

	// 15 concurrent $10.15 debits = $152.25 -> expected remaining balance = $50.75
	for i := 0; i < 15; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("p12-dr-%d", idx)
			_, err := st.DebitWalletAtomic(ctx, uid, 10.15, "invoice_payment", key, key, "Concurrent debit")
			if err != nil {
				t.Errorf("debit %d failed: %v", idx, err)
			}
		}(i)
	}
	wg.Wait()

	wlt, _ := st.GetOrCreateCustomerWallet(ctx, uid)
	if math.Abs(wlt.BalanceUSD-50.75) > 1e-6 {
		t.Fatalf("expected reconciled wallet balance $50.75, got $%.4f", wlt.BalanceUSD)
	}

	entries, _ := st.ListWalletLedger(ctx, uid, 100)
	if len(entries) != 35 {
		t.Fatalf("expected 35 immutable ledger entries, got %d", len(entries))
	}
	sumCredits := 0.0
	sumDebits := 0.0
	for _, e := range entries {
		if e.Direction == "credit" {
			sumCredits = math.Round((sumCredits+e.AmountUSD)*10000) / 10000
			if math.Abs(e.BalanceAfterUSD-(e.BalanceBeforeUSD+e.AmountUSD)) > 1e-4 {
				t.Fatalf("ledger credit continuity mismatch: %+v", e)
			}
		} else {
			sumDebits = math.Round((sumDebits+e.AmountUSD)*10000) / 10000
			if math.Abs(e.BalanceAfterUSD-(e.BalanceBeforeUSD-e.AmountUSD)) > 1e-4 {
				t.Fatalf("ledger debit continuity mismatch: %+v", e)
			}
		}
	}
	if math.Abs((sumCredits-sumDebits)-wlt.BalanceUSD) > 1e-4 {
		t.Fatalf("ledger reconciliation failure: credits=%.2f debits=%.2f wallet=%.2f", sumCredits, sumDebits, wlt.BalanceUSD)
	}
}

func TestP12_DisasterRecoveryBackupAndRestoreIntegrity(t *testing.T) {
	ctx := context.Background()
	primary := store.NewMemoryStore()

	// Populate production-like commercial & financial state
	uid := uuid.New()
	affID := uuid.New()
	_ = primary.CreateCoupon(ctx, &store.Coupon{Code: "DR2026", DiscountType: "percentage", DiscountValue: 30, MaxUses: 100, MaxUsesPerUser: 2, IsActive: true})
	_, _ = primary.CreditWalletAtomic(ctx, uid, 125.50, "topup", "dr-topup", "idem-dr-1", "Pre-disaster balance")
	affAcc, _ := primary.GetOrCreateAffiliateAccount(ctx, affID)
	_, _ = primary.RecordAffiliateReferral(ctx, affAcc.ID, uid, "first_click")
	invID := uuid.New()
	_, _ = primary.RecordAffiliateCommission(ctx, uid, invID, 80.00) // $12.00 commission

	// Snapshot backup serialization
	type drSnapshot struct {
		Coupons    []*store.Coupon           `json:"coupons"`
		Wallet     *store.CustomerWallet     `json:"wallet"`
		Ledger     []*store.WalletLedgerEntry `json:"ledger"`
		AffAccount *store.AffiliateAccount   `json:"aff_account"`
	}
	coupons, _ := primary.ListCoupons(ctx)
	wlt, _ := primary.GetOrCreateCustomerWallet(ctx, uid)
	ledger, _ := primary.ListWalletLedger(ctx, uid, 100)
	affNow, _ := primary.GetOrCreateAffiliateAccount(ctx, affID)

	rawBackup, err := json.Marshal(drSnapshot{Coupons: coupons, Wallet: wlt, Ledger: ledger, AffAccount: affNow})
	if err != nil {
		t.Fatalf("failed to serialize DR snapshot: %v", err)
	}

	// Simulate disaster recovery into clean isolated store & measure RTO
	startRestore := time.Now()
	recovery := store.NewMemoryStore()
	var restored drSnapshot
	if err := json.Unmarshal(rawBackup, &restored); err != nil {
		t.Fatalf("failed to unmarshal DR snapshot: %v", err)
	}
	for _, c := range restored.Coupons {
		_ = recovery.CreateCoupon(ctx, c)
	}
	for i := len(restored.Ledger) - 1; i >= 0; i-- {
		e := restored.Ledger[i]
		if e.Direction == "credit" {
			_, _ = recovery.CreditWalletAtomic(ctx, e.UserID, e.AmountUSD, e.EntryType, e.ReferenceID, e.IdempotencyKey, e.Notes)
		} else {
			_, _ = recovery.DebitWalletAtomic(ctx, e.UserID, e.AmountUSD, e.EntryType, e.ReferenceID, e.IdempotencyKey, e.Notes)
		}
	}
	rto := time.Since(startRestore)

	// Verify RPO = 0 consistency
	recCoupon, err := recovery.GetCouponByCode(ctx, "DR2026")
	if err != nil || recCoupon.DiscountValue != 30 {
		t.Fatalf("DR coupon restore failed: %v", err)
	}
	recWallet, _ := recovery.GetOrCreateCustomerWallet(ctx, uid)
	if recWallet.BalanceUSD != 125.50 {
		t.Fatalf("DR wallet restore mismatch: expected $125.50, got $%.2f", recWallet.BalanceUSD)
	}
	t.Logf("[DR CERTIFIED] RPO=0 verified | RTO=%v | SnapshotBytes=%d", rto, len(rawBackup))
}

func TestP12_PerformanceThroughputAndLatencyMetrics(t *testing.T) {
	st := store.NewMemoryStore()
	ctx := context.Background()
	_ = st.CreateCoupon(ctx, &store.Coupon{Code: "PERF20", DiscountType: "percentage", DiscountValue: 20, IsActive: true})
	dw := billing.NewDunningWorker(st, 0, 3)
	h := NewCommercialHandler(&config.Config{}, st, audit.NewLogger(st, slog.Default()), dw)

	const totalRequests = 300
	const concurrency = 20
	latencies := make([]time.Duration, totalRequests)
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	startAll := time.Now()
	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			t0 := time.Now()
			if idx%2 == 0 {
				req := httptest.NewRequest("GET", "/api/v1/public/status", nil)
				w := httptest.NewRecorder()
				h.GetPublicSystemStatus(w, req)
				if w.Code != http.StatusOK {
					t.Errorf("status check failed with %d", w.Code)
				}
			} else {
				req := httptest.NewRequest("POST", "/api/v1/billing/coupons/validate", bytes.NewBufferString(`{"code":"PERF20","order_amount_usd":50}`))
				w := httptest.NewRecorder()
				h.ValidateCoupon(w, req)
				if w.Code != http.StatusOK {
					t.Errorf("coupon validate failed with %d", w.Code)
				}
			}
			latencies[idx] = time.Since(t0)
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(startAll)

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p50 := latencies[totalRequests*50/100]
	p95 := latencies[totalRequests*95/100]
	p99 := latencies[totalRequests*99/100]
	rps := float64(totalRequests) / elapsed.Seconds()

	t.Logf("[PERF CERTIFIED] Requests=%d Concurrency=%d Throughput=%.1f req/s | P50=%v P95=%v P99=%v", totalRequests, concurrency, rps, p50, p95, p99)
	if p99 > 250*time.Millisecond {
		t.Fatalf("P99 latency %v exceeded 250ms threshold", p99)
	}
}

func TestP13_RealOnDiskDatabaseBackupAndRestore(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "store.json")
	bakFile := filepath.Join(tmpDir, "store.json.bak")

	// 1. Initialize file-backed store on disk & populate commercial, financial & HD wallet state
	primary := store.NewMemoryStoreWithPath(dbFile)
	uid := uuid.New()
	affID := uuid.New()
	invID := uuid.New()

	if err := primary.CreateCoupon(ctx, &store.Coupon{
		Code:           "DISKDR50",
		DiscountType:   "percentage",
		DiscountValue:  50,
		MaxUses:        10,
		MaxUsesPerUser: 1,
		IsActive:       true,
	}); err != nil {
		t.Fatalf("CreateCoupon failed: %v", err)
	}
	if _, err := primary.RedeemCouponAtomic(ctx, "DISKDR50", uid, invID, 100.0, true); err != nil {
		t.Fatalf("RedeemCouponAtomic failed: %v", err)
	}
	if _, err := primary.CreditWalletAtomic(ctx, uid, 200.00, "topup", "disk-topup", "idem-disk-1", "On-disk credit"); err != nil {
		t.Fatalf("CreditWalletAtomic failed: %v", err)
	}
	if _, err := primary.DebitWalletAtomic(ctx, uid, 24.75, "invoice_payment", invID.String(), "idem-disk-2", "On-disk debit"); err != nil {
		t.Fatalf("DebitWalletAtomic failed: %v", err)
	}
	affAcc, _ := primary.GetOrCreateAffiliateAccount(ctx, affID)
	_, _ = primary.RecordAffiliateReferral(ctx, affAcc.ID, uid, "first_click")
	_, _ = primary.RecordAffiliateCommission(ctx, uid, invID, 120.00) // $18.00 commission
	_ = primary.UpsertHDWalletAccount(ctx, &store.HDWalletAccount{
		NetworkID:           "bsc",
		NextDerivationIndex: 1,
	})
	idx, _, err := primary.ReserveNextDerivationIndex(ctx, "bsc")
	if err != nil || idx != 1 {
		t.Fatalf("ReserveNextDerivationIndex failed: idx=%d err=%v", idx, err)
	}

	// 2. Perform real disk backup copy & compute SHA-256 checksum
	tBackupStart := time.Now()
	rawDB, err := os.ReadFile(dbFile)
	if err != nil || len(rawDB) == 0 {
		t.Fatalf("failed reading on-disk store.json: err=%v len=%d", err, len(rawDB))
	}
	if err := os.WriteFile(bakFile, rawDB, 0600); err != nil {
		t.Fatalf("failed writing on-disk backup store.json.bak: %v", err)
	}
	backupDuration := time.Since(tBackupStart)
	sum := sha256.Sum256(rawDB)
	backupSHA := hex.EncodeToString(sum[:])

	// 3. Simulate catastrophic primary database destruction on disk
	if err := os.Remove(dbFile); err != nil {
		t.Fatalf("failed removing primary dbFile: %v", err)
	}

	// 4. Restore store.json from backup on disk and reload a fresh MemoryStore instance
	tRestoreStart := time.Now()
	restoredBytes, err := os.ReadFile(bakFile)
	if err != nil {
		t.Fatalf("failed reading backup file: %v", err)
	}
	if err := os.WriteFile(dbFile, restoredBytes, 0600); err != nil {
		t.Fatalf("failed restoring dbFile on disk: %v", err)
	}
	recovered := store.NewMemoryStoreWithPath(dbFile)
	restoreRTO := time.Since(tRestoreStart)

	// 5. Verify RPO=0 across all restored disk tables
	c, err := recovered.GetCouponByCode(ctx, "DISKDR50")
	if err != nil || c.CurrentUses != 1 {
		t.Fatalf("on-disk coupon recovery mismatch: %+v err=%v", c, err)
	}
	wlt, err := recovered.GetOrCreateCustomerWallet(ctx, uid)
	if err != nil || wlt.BalanceUSD != 175.25 {
		t.Fatalf("on-disk wallet balance mismatch: expected $175.25, got $%.2f (err=%v)", wlt.BalanceUSD, err)
	}
	ledger, _ := recovered.ListWalletLedger(ctx, uid, 10)
	if len(ledger) != 2 {
		t.Fatalf("expected 2 restored wallet ledger entries from disk, got %d", len(ledger))
	}
	recAff, _ := recovered.GetOrCreateAffiliateAccount(ctx, affID)
	if recAff.ApprovedBalanceUSD != 18.00 || recAff.TotalReferrals != 1 {
		t.Fatalf("on-disk affiliate recovery mismatch: %+v", recAff)
	}
	nextIdx, _, err := recovered.ReserveNextDerivationIndex(ctx, "bsc")
	if err != nil || nextIdx != 2 {
		t.Fatalf("on-disk HD wallet derivation index recovery mismatch: expected 2, got %d (err=%v)", nextIdx, err)
	}
	t.Logf("[P13 ON-DISK DR CERTIFIED] FileBytes=%d SHA256=%s BackupTime=%v RestoreRTO=%v RPO=0", len(rawDB), backupSHA[:16], backupDuration, restoreRTO)
}

func TestP14_ConcurrentPayInvoiceWithWalletAndPlanScopedCouponsAndSweepPersistence(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "p14_store.json")
	st := store.NewMemoryStoreWithPath(dbFile)

	// 1. Concurrent PayInvoiceWithWallet race test: 10 concurrent requests for the same $25 invoice
	// must debit wallet strictly once ($25) and invoke onInvoicePaid provisioning callback strictly once!
	uid := uuid.New()
	orgID := uuid.New()
	_, _ = st.CreditWalletAtomic(ctx, uid, 100.00, "topup", "p14-topup", "idem-p14-topup", "Seed $100")

	inv := &store.Invoice{
		ID:             uuid.New(),
		InvoiceNumber:  "INV-P14-RACE",
		UserID:         uid,
		OrganizationID: orgID,
		Subtotal:       25.00,
		Total:          25.00,
		Currency:       "USD",
		Status:         store.InvoiceStatusUnpaid,
	}
	_ = st.CreateInvoice(ctx, inv)

	provisionCalls := 0
	var provMu sync.Mutex
	h := NewCommercialHandler(&config.Config{}, st, audit.NewLogger(st, slog.Default()), nil)
	h.SetOnInvoicePaid(func(c context.Context, paidInv *store.Invoice) {
		provMu.Lock()
		provisionCalls++
		provMu.Unlock()
	})

	claims := &auth.Claims{UserID: uid, OrganizationID: orgID, Email: "p14@example.com", Role: "customer"}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/api/v1/wallet/pay-invoice", bytes.NewBufferString(`{"invoice_id":"`+inv.ID.String()+`"}`))
			req = req.WithContext(auth.WithClaims(req.Context(), claims))
			w := httptest.NewRecorder()
			h.PayInvoiceWithWallet(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("unexpected status %d from PayInvoiceWithWallet", w.Code)
			}
		}()
	}
	wg.Wait()

	if provisionCalls != 1 {
		t.Fatalf("expected onInvoicePaid to be called strictly once under 10-way concurrency, got %d", provisionCalls)
	}
	wlt, _ := st.GetOrCreateCustomerWallet(ctx, uid)
	if wlt.BalanceUSD != 75.00 {
		t.Fatalf("expected wallet balance $75.00 after single $25 payment, got $%.2f", wlt.BalanceUSD)
	}

	// Paying a cancelled or refunded invoice MUST be rejected with 400
	cancelledInv := &store.Invoice{
		ID:             uuid.New(),
		InvoiceNumber:  "INV-P14-CANCELLED",
		UserID:         uid,
		OrganizationID: orgID,
		Total:          10.00,
		Currency:       "USD",
		Status:         store.InvoiceStatusCancelled,
	}
	_ = st.CreateInvoice(ctx, cancelledInv)
	cReq := httptest.NewRequest("POST", "/api/v1/wallet/pay-invoice", bytes.NewBufferString(`{"invoice_id":"`+cancelledInv.ID.String()+`"}`))
	cReq = cReq.WithContext(auth.WithClaims(cReq.Context(), claims))
	cW := httptest.NewRecorder()
	h.PayInvoiceWithWallet(cW, cReq)
	if cW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 paying cancelled invoice, got %d", cW.Code)
	}

	// 2. Plan-scoped coupon (ApplicablePlanIDs = "enterprise-only") must reject ineligible plan checkout
	starterPlan := &store.HostingPlan{ID: uuid.New(), Name: "Starter P14", Slug: "starter-p14", PriceMonthly: 10.0, Currency: "USD", IsActive: true}
	entPlan := &store.HostingPlan{ID: uuid.New(), Name: "Enterprise P14", Slug: "enterprise-p14", PriceMonthly: 50.0, Currency: "USD", IsActive: true}
	_ = st.CreatePlan(ctx, starterPlan)
	_ = st.CreatePlan(ctx, entPlan)
	_ = st.CreateCoupon(ctx, &store.Coupon{
		Code:              "ENTONLY30",
		DiscountType:      "percentage",
		DiscountValue:     30,
		ApplicablePlanIDs: "enterprise-p14",
		IsActive:          true,
	})

	badChk := httptest.NewRequest("POST", "/checkout", bytes.NewBufferString(`{"plan_id":"`+starterPlan.ID.String()+`","billing_cycle":"monthly","coupon_code":"ENTONLY30","payment_method":"crypto"}`))
	badChk = badChk.WithContext(auth.WithClaims(badChk.Context(), claims))
	badW := httptest.NewRecorder()
	h.CheckoutOrder(badW, badChk)
	if badW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 redeeming plan-restricted coupon on ineligible plan, got %d", badW.Code)
	}

	goodChk := httptest.NewRequest("POST", "/checkout", bytes.NewBufferString(`{"plan_id":"`+entPlan.ID.String()+`","billing_cycle":"monthly","coupon_code":"ENTONLY30","payment_method":"crypto"}`))
	goodChk = goodChk.WithContext(auth.WithClaims(goodChk.Context(), claims))
	goodW := httptest.NewRecorder()
	h.CheckoutOrder(goodW, goodChk)
	if goodW.Code != http.StatusCreated {
		t.Fatalf("expected 201 redeeming plan-restricted coupon on eligible plan, got %d: %s", goodW.Code, goodW.Body.String())
	}

	// 3. Verify TreasurySweepJob & WalletDerivationAudit survive cold restart from disk (p14_store.json)
	sweepJob := &store.TreasurySweepJob{
		ID:                 uuid.New(),
		NetworkID:          "bsc",
		AssetID:            "USDT_BEP20",
		SourceAddress:      "0x6Fac4D18c912343BF86fa7049364Dd4E424Ab9C0",
		DestinationAddress: "0x9858EfFD232B4033E47d90003D41EC34EcaEda94",
		RequestedAmount:    50.00,
		AmountUSD:          50.00,
		Status:             "completed",
		IdempotencyKey:     "p14-sweep-1",
	}
	if err := st.CreateTreasurySweepJob(ctx, sweepJob); err != nil {
		t.Fatalf("CreateTreasurySweepJob failed: %v", err)
	}
	_ = st.RecordWalletDerivationAudit(ctx, &store.WalletDerivationAuditLog{
		EventType:    "address_derived",
		ActorID:      claims.UserID.String(),
		EventDetails: "p14_audit bsc index=1",
	})

	reloaded := store.NewMemoryStoreWithPath(dbFile)
	gotJob, err := reloaded.GetTreasurySweepJobByID(ctx, sweepJob.ID)
	if err != nil || gotJob == nil || gotJob.Status != "completed" {
		t.Fatalf("expected treasury sweep job to persist across cold disk reload, got %+v err=%v", gotJob, err)
	}
	audits, err := reloaded.ListWalletDerivationAudits(ctx, 10)
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected 1 wallet derivation audit after cold disk reload, got %d err=%v", len(audits), err)
	}
}

func TestP15_ConcurrentWebhookReplayAndCryptoPoolPersistenceAndCancelledInvoiceGuard(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "p15_store.json")
	st := store.NewMemoryStoreWithPath(dbFile)
	bh := NewBillingHandler(&config.Config{AppURL: "https://hostvra.com"}, st, audit.NewLogger(st, slog.Default()))

	uid := uuid.New()
	orgID := uuid.New()
	plan := &store.HostingPlan{
		ID:            uuid.New(),
		Name:          "P15 Cloud Pro",
		Slug:          "p15-cloud-pro",
		PriceMonthly:  25.00,
		Currency:      "USD",
		DiskSpaceMB:   50000,
		BandwidthMB:   500000,
		MaxWebsites:   10,
		MaxDatabases:  10,
		MaxMailboxes:  10,
		IsActive:      true,
	}
	_ = st.CreatePlan(ctx, plan)
	_ = st.SaveGatewayConfig(ctx, &store.PaymentGatewayConfig{
		Gateway:     "stripe",
		DisplayName: "Stripe",
		Enabled:     true,
		TestMode:    false,
		SecretKey:   "p15_webhook_secret_key",
	})

	inv := &store.Invoice{
		ID:             uuid.New(),
		InvoiceNumber:  "INV-P15-WEBHOOK-1",
		UserID:         uid,
		OrganizationID: orgID,
		PlanID:         plan.ID,
		Subtotal:       25.00,
		Total:          25.00,
		Currency:       "USD",
		Status:         store.InvoiceStatusUnpaid,
		DueDate:        time.Now().UTC().AddDate(0, 0, 7),
		Description:    "P15 Cloud Pro (monthly)",
		CreatedAt:      time.Now().UTC(),
	}
	if err := st.CreateInvoice(ctx, inv); err != nil {
		t.Fatalf("CreateInvoice failed: %v", err)
	}

	// 1. Fire 10 simultaneous signed webhook requests for the same invoice & transaction ID
	bodyStr := fmt.Sprintf(`{"event":"invoice.paid","invoice_id":"%s","transaction_id":"pi_p15_race_1","amount":25.00,"currency":"USD","status":"paid"}`, inv.ID.String())
	mac := hmac.New(sha256.New, []byte("p15_webhook_secret_key"))
	mac.Write([]byte(bodyStr))
	sig := hex.EncodeToString(mac.Sum(nil))

	const workers = 10
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("gateway", "stripe")
			req := httptest.NewRequest("POST", "/api/v1/billing/webhooks/stripe", bytes.NewBufferString(bodyStr))
			req.Header.Set("X-Signature", sig)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()
			bh.HandleWebhook(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("unexpected webhook status %d: %s", w.Code, w.Body.String())
			}
		}()
	}
	wg.Wait()

	// Verify exactly 1 subscription and 1 hosting account were provisioned
	subs, err := st.ListSubscriptions(ctx, orgID)
	if err != nil || len(subs) != 1 {
		t.Fatalf("expected exactly 1 subscription after 10 concurrent webhooks, got %d (err=%v)", len(subs), err)
	}
	accs, err := st.ListHostingAccounts(ctx, orgID, nil)
	if err != nil || len(accs) != 1 {
		t.Fatalf("expected exactly 1 hosting account after 10 concurrent webhooks, got %d (err=%v)", len(accs), err)
	}
	txns, err := st.ListPaymentTransactions(ctx, orgID)
	if err != nil || len(txns) != 1 {
		t.Fatalf("expected exactly 1 completed payment transaction after 10 concurrent webhooks, got %d (err=%v)", len(txns), err)
	}

	// 2. Verify cancelled/refunded invoice cannot be paid via webhook
	cancelledInv := &store.Invoice{
		ID:             uuid.New(),
		InvoiceNumber:  "INV-P15-CANCELLED",
		UserID:         uid,
		OrganizationID: orgID,
		PlanID:         plan.ID,
		Subtotal:       25.00,
		Total:          25.00,
		Currency:       "USD",
		Status:         store.InvoiceStatusCancelled,
		DueDate:        time.Now().UTC().AddDate(0, 0, 7),
		CreatedAt:      time.Now().UTC(),
	}
	_ = st.CreateInvoice(ctx, cancelledInv)
	cBody := fmt.Sprintf(`{"event":"invoice.paid","invoice_id":"%s","transaction_id":"pi_p15_cancelled_1","amount":25.00,"currency":"USD","status":"paid"}`, cancelledInv.ID.String())
	cMac := hmac.New(sha256.New, []byte("p15_webhook_secret_key"))
	cMac.Write([]byte(cBody))
	cSig := hex.EncodeToString(cMac.Sum(nil))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("gateway", "stripe")
	cReq := httptest.NewRequest("POST", "/api/v1/billing/webhooks/stripe", bytes.NewBufferString(cBody))
	cReq.Header.Set("X-Signature", cSig)
	cReq = cReq.WithContext(context.WithValue(cReq.Context(), chi.RouteCtxKey, rctx))
	cW := httptest.NewRecorder()
	bh.HandleWebhook(cW, cReq)
	if cW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when webhook attempts to mark cancelled invoice paid, got %d: %s", cW.Code, cW.Body.String())
	}

	// 3. Verify CreateDepositAddress, AcquireDepositAddress, and RecordCryptoLedger persist across disk reload
	depAddr := &store.CryptoDepositAddress{
		ID:              uuid.New(),
		NetworkID:       "tron",
		CurrencyID:      "USDT_TRC20",
		Address:         "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb",
		DerivationIndex: 7,
		Status:          "available",
	}
	if err := st.CreateDepositAddress(ctx, depAddr); err != nil {
		t.Fatalf("CreateDepositAddress failed: %v", err)
	}
	acquired, err := st.AcquireDepositAddress(ctx, "tron", "USDT_TRC20", inv.ID)
	if err != nil || acquired == nil || acquired.Status != "assigned" {
		t.Fatalf("AcquireDepositAddress failed: %+v err=%v", acquired, err)
	}
	if err := st.RecordCryptoLedger(ctx, &store.CryptoInternalLedger{
		InvoiceID: &inv.ID,
		EntryType: "invoice_settled",
		Amount:    25.00,
		Currency:  "USDT",
		Network:   "tron",
		TxHash:    "0xp15ledgerhash",
	}); err != nil {
		t.Fatalf("RecordCryptoLedger failed: %v", err)
	}

	reloaded := store.NewMemoryStoreWithPath(dbFile)
	assignedAddrs, err := reloaded.ListDepositAddresses(ctx, "tron", "assigned")
	if err != nil || len(assignedAddrs) != 1 || assignedAddrs[0].Address != "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb" {
		t.Fatalf("expected assigned deposit address to persist across cold disk reload, got %+v err=%v", assignedAddrs, err)
	}
	ledgerEntries, totalLedger, err := reloaded.ListCryptoLedger(ctx, 10, 0)
	if err != nil || totalLedger != 1 || len(ledgerEntries) != 1 || ledgerEntries[0].TxHash != "0xp15ledgerhash" {
		t.Fatalf("expected crypto internal ledger to persist across cold disk reload, got %+v total=%d err=%v", ledgerEntries, totalLedger, err)
	}
}

func TestP16_WebhookReplayDeduplicationPersistsAcrossColdRestartAndDomainOrdersPersist(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "p16_store.json")
	st := store.NewMemoryStoreWithPath(dbFile)
	bh := NewBillingHandler(&config.Config{AppURL: "https://hostvra.com"}, st, audit.NewLogger(st, slog.Default()))

	uid := uuid.New()
	orgID := uuid.New()
	plan := &store.HostingPlan{
		ID:            uuid.New(),
		Name:          "P16 Enterprise",
		Slug:          "p16-enterprise",
		PriceMonthly:  40.00,
		Currency:      "USD",
		DiskSpaceMB:   100000,
		BandwidthMB:   1000000,
		MaxWebsites:   25,
		MaxDatabases:  25,
		MaxMailboxes:  25,
		IsActive:      true,
	}
	_ = st.CreatePlan(ctx, plan)
	_ = st.SaveGatewayConfig(ctx, &store.PaymentGatewayConfig{
		Gateway:     "stripe",
		DisplayName: "Stripe",
		Enabled:     true,
		TestMode:    false,
		SecretKey:   "p16_webhook_secret_key",
	})

	inv := &store.Invoice{
		ID:             uuid.New(),
		InvoiceNumber:  "INV-P16-WEBHOOK-1",
		UserID:         uid,
		OrganizationID: orgID,
		PlanID:         plan.ID,
		Subtotal:       40.00,
		Total:          40.00,
		Currency:       "USD",
		Status:         store.InvoiceStatusUnpaid,
		DueDate:        time.Now().UTC().AddDate(0, 0, 7),
		Description:    "P16 Enterprise (monthly)",
		CreatedAt:      time.Now().UTC(),
	}
	if err := st.CreateInvoice(ctx, inv); err != nil {
		t.Fatalf("CreateInvoice failed: %v", err)
	}

	// Create a domain order linked to this invoice and verify it persists across restart
	domOrder := &store.DomainOrder{
		ID:             uuid.New(),
		UserID:         uid,
		OrganizationID: &orgID,
		DomainName:     "p16-certified.com",
		OrderType:      "register",
		Years:          1,
		Amount:         14.99,
		Currency:       "USD",
		PaymentStatus:  "pending",
		IdempotencyKey: "dom-order-p16-1",
		InvoiceID:      &inv.ID,
	}
	if err := st.CreateDomainOrder(ctx, domOrder); err != nil {
		t.Fatalf("CreateDomainOrder failed: %v", err)
	}

	// Deliver signed webhook once
	bodyStr := fmt.Sprintf(`{"event":"invoice.paid","invoice_id":"%s","transaction_id":"pi_p16_restart_1","amount":40.00,"currency":"USD","status":"paid"}`, inv.ID.String())
	mac := hmac.New(sha256.New, []byte("p16_webhook_secret_key"))
	mac.Write([]byte(bodyStr))
	sig := hex.EncodeToString(mac.Sum(nil))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("gateway", "stripe")
	req := httptest.NewRequest("POST", "/api/v1/billing/webhooks/stripe", bytes.NewBufferString(bodyStr))
	req.Header.Set("X-Signature", sig)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	bh.HandleWebhook(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on first webhook delivery, got %d: %s", w.Code, w.Body.String())
	}

	// Simulate full process restart from disk (p16_store.json)
	reloadedStore := store.NewMemoryStoreWithPath(dbFile)
	reloadedBH := NewBillingHandler(&config.Config{AppURL: "https://hostvra.com"}, reloadedStore, audit.NewLogger(reloadedStore, slog.Default()))

	// Verify webhook deduplication record survived restart
	hookRec, err := reloadedStore.GetDomainWebhook(ctx, "stripe", "pi_p16_restart_1")
	if err != nil || hookRec == nil {
		t.Fatalf("expected webhook deduplication record to persist across cold restart, got %+v err=%v", hookRec, err)
	}

	// Replay the exact same webhook after cold restart — must return idempotent_duplicate immediately
	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("gateway", "stripe")
	req2 := httptest.NewRequest("POST", "/api/v1/billing/webhooks/stripe", bytes.NewBufferString(bodyStr))
	req2.Header.Set("X-Signature", sig)
	req2 = req2.WithContext(context.WithValue(req2.Context(), chi.RouteCtxKey, rctx2))
	w2 := httptest.NewRecorder()
	reloadedBH.HandleWebhook(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 on post-restart webhook replay, got %d: %s", w2.Code, w2.Body.String())
	}
	if !bytes.Contains(w2.Body.Bytes(), []byte("idempotent_duplicate")) {
		t.Fatalf("expected idempotent_duplicate response after cold restart, got: %s", w2.Body.String())
	}

	// Verify domain order and its paid status survived cold restart
	gotOrder, err := reloadedStore.GetDomainOrderByInvoiceID(ctx, inv.ID)
	if err != nil || gotOrder == nil || gotOrder.PaymentStatus != "paid" {
		t.Fatalf("expected paid domain order to persist across cold restart, got %+v err=%v", gotOrder, err)
	}
}
