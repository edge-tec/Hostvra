package billing

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"hostvra/api/internal/store"
)

func TestDunningWorker_RenewalAutoPayAndOverdueSuspension(t *testing.T) {
	st := store.NewMemoryStore()
	worker := NewDunningWorker(st, time.Hour, 3)
	ctx := context.Background()
	now := time.Now().UTC()

	// 1. User A: Subscription due in 5 days WITH funded Credit Wallet -> should auto-generate renewal & auto-pay
	userA := uuid.New()
	orgA := uuid.New()
	subA := &store.Subscription{
		ID:              uuid.New(),
		UserID:          userA,
		OrganizationID:  orgA,
		PlanID:          uuid.New(),
		PlanName:        "Enterprise Cloud",
		BillingCycle:    "monthly",
		Amount:          24.99,
		Currency:        "USD",
		Status:          store.SubStatusActive,
		NextBillingDate: now.AddDate(0, 0, 5),
		CreatedAt:       now.AddDate(0, -1, 0),
	}
	_ = st.CreateSubscription(ctx, subA)
	_, err := st.CreditWalletAtomic(ctx, userA, 50.00, "topup", "dep-1", "idem-topup-a", "Initial deposit")
	if err != nil {
		t.Fatalf("CreditWalletAtomic failed: %v", err)
	}

	// 2. User B: Subscription overdue by 4 days (> 3-day grace period) -> should auto-suspend
	userB := uuid.New()
	subB := &store.Subscription{
		ID:              uuid.New(),
		UserID:          userB,
		OrganizationID:  uuid.New(),
		PlanID:          uuid.New(),
		PlanName:        "Starter Cloud",
		BillingCycle:    "monthly",
		Amount:          9.99,
		Currency:        "USD",
		Status:          store.SubStatusActive,
		NextBillingDate: now.AddDate(0, 0, -4),
		CreatedAt:       now.AddDate(0, -1, 0),
	}
	_ = st.CreateSubscription(ctx, subB)

	// Run Pass #1
	summary1, err := worker.RunPass(ctx, now)
	if err != nil {
		t.Fatalf("RunPass 1 failed: %v", err)
	}
	if summary1.RenewalsGenerated != 1 || summary1.AutoPaidFromWallet != 1 || summary1.SuspendedOverdue != 1 {
		t.Fatalf("unexpected dunning summary1: %+v", summary1)
	}

	// Verify User A wallet debited accurately (50.00 - 24.99 = 25.01)
	wA, _ := st.GetOrCreateCustomerWallet(ctx, userA)
	if wA.BalanceUSD < 25.00 || wA.BalanceUSD > 25.02 {
		t.Fatalf("expected wallet balance ~25.01, got %.4f", wA.BalanceUSD)
	}

	// Verify User B suspended
	gotB, _ := st.GetSubscriptionByID(ctx, subB.ID)
	if gotB.Status != store.SubStatusSuspended {
		t.Fatalf("expected User B subscription suspended, got %s", gotB.Status)
	}

	// Run Pass #2 immediately -> Idempotency guarantees ZERO duplicate renewals or double-debits!
	summary2, err := worker.RunPass(ctx, now)
	if err != nil {
		t.Fatalf("RunPass 2 failed: %v", err)
	}
	if summary2.RenewalsGenerated != 0 || summary2.AutoPaidFromWallet != 0 || summary2.SuspendedOverdue != 0 {
		t.Fatalf("idempotency violated on pass 2: %+v", summary2)
	}
}
