package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

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
