package handlers

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestBillingHandler_SettleCryptoPayment_AutoRenewsSubscriptionAndNotifies(t *testing.T) {
	st := store.NewMemoryStoreWithPath(t.TempDir() + "/store.json")
	cfg := &config.Config{JWTSecret: "test-secret-32-chars-long-12345678"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(st, logger)

	h := NewBillingHandler(cfg, st, auditLogger)

	ctx := context.Background()
	orgID := uuid.New()
	userID := uuid.New()

	// 1. Create User
	user := &store.User{
		ID:       userID,
		Email:    "client@agency.com",
		FullName: "Agency Client",
	}
	_ = st.CreateUser(ctx, user, orgID, "user")

	// 2. Create Plan
	planID := uuid.New()
	plan := &store.HostingPlan{
		ID:           planID,
		Name:         "Enterprise Turbo SSD",
		PriceMonthly: 25.0,
		DiskSpaceMB:  20480,
		BandwidthMB:  100000,
		MaxWebsites:  10,
		MaxDatabases: 10,
		MaxMailboxes: 20,
	}
	_ = st.CreatePlan(ctx, plan)

	// 3. Create Subscription (which was due today)
	subID := uuid.New()
	now := time.Now().UTC()
	sub := &store.Subscription{
		ID:              subID,
		UserID:          userID,
		OrganizationID:  orgID,
		PlanID:          planID,
		PlanName:        plan.Name,
		BillingCycle:    "monthly",
		Amount:          25.0,
		Currency:        "USD",
		Status:          store.SubStatusActive,
		NextBillingDate: now, // Expiring now
		CreatedAt:       now.AddDate(0, -1, 0),
		UpdatedAt:       now,
	}
	_ = st.CreateSubscription(ctx, sub)

	// 4. Create Unpaid Billing Invoice
	invID := uuid.New()
	billingInv := &store.Invoice{
		ID:             invID,
		InvoiceNumber:  "INV-2026-0001",
		UserID:         userID,
		OrganizationID: orgID,
		SubscriptionID: &subID,
		PlanID:         planID,
		Description:    plan.Name,
		Subtotal:       25.0,
		Total:          25.0,
		Currency:       "USD",
		Status:         store.InvoiceStatusUnpaid,
		DueDate:        now,
		CreatedAt:      now,
	}
	_ = st.CreateInvoice(ctx, billingInv)

	// 5. Crypto Invoice Paid on Blockchain
	cryptoInv := &store.CryptoPaymentInvoice{
		ID:             uuid.New(),
		InvoiceCode:    "HV-CRYPTO-1001",
		UserID:         userID,
		OrganizationID: orgID,
		PayableType:    "invoice",
		PayableID:      invID.String(),
		FiatAmountUSD:  25.0,
		CryptoAmount:   25.0,
		CurrencyID:     "USDT_TRC20",
		NetworkID:      "tron",
		DepositAddress: "TYDzsYUE4jEJTXsyNDoja2d1M7T4ZkYjHn",
		TxHash:         "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		Status:         "paid",
	}

	// 6. Trigger SettleCryptoPayment
	err := h.SettleCryptoPayment(ctx, cryptoInv)
	if err != nil {
		t.Fatalf("SettleCryptoPayment failed: %v", err)
	}

	// 7. Verify Invoice is marked Paid
	updatedInv, err := st.GetInvoiceByID(ctx, invID)
	if err != nil || updatedInv.Status != store.InvoiceStatusPaid {
		t.Fatalf("expected invoice to be Paid, got %v (err: %v)", updatedInv.Status, err)
	}
	if updatedInv.TransactionID != cryptoInv.TxHash {
		t.Fatalf("expected txn ID %s, got %s", cryptoInv.TxHash, updatedInv.TransactionID)
	}

	// 8. Verify Subscription is Renewed and NextBillingDate extended by 1 month!
	updatedSub, err := st.GetSubscriptionByID(ctx, subID)
	if err != nil {
		t.Fatalf("failed to fetch updated subscription: %v", err)
	}
	if updatedSub.Status != store.SubStatusActive {
		t.Fatalf("expected sub status active, got %s", updatedSub.Status)
	}
	if !updatedSub.NextBillingDate.After(now) {
		t.Fatalf("expected NextBillingDate to be extended into the future, got %v (now: %v)", updatedSub.NextBillingDate, now)
	}
}
