package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	"hostvra/api/internal/audit"
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
	h := NewCommercialHandler(&config.Config{}, st, audit.NewLogger(st, nil), dw)
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
