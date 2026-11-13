package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCryptoStore_MemoryLifecycleAndIdempotency(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStoreWithPath(t.TempDir() + "/store.json")

	// 1. Verify seeded networks and currencies
	nets, err := m.ListCryptoNetworks(ctx, true)
	if err != nil {
		t.Fatalf("ListCryptoNetworks error: %v", err)
	}
	if len(nets) < 5 {
		t.Fatalf("expected at least 5 seeded networks, got %d", len(nets))
	}

	tron, err := m.GetCryptoNetworkByID(ctx, "tron")
	if err != nil || tron == nil {
		t.Fatalf("failed to get tron network: %v", err)
	}

	curs, err := m.ListCryptoCurrencies(ctx, "tron", true)
	if err != nil || len(curs) == 0 {
		t.Fatalf("expected currencies for tron, got %v", err)
	}

	// 2. Deposit Address Pool
	depAddr := &CryptoDepositAddress{
		ID:         uuid.New(),
		NetworkID:  "tron",
		CurrencyID: "USDT_TRC20",
		Address:    "TYDzsYUE4jEJTXsyNDoja2d1M7T4ZkYjHn",
		Status:     "available",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if err := m.CreateDepositAddress(ctx, depAddr); err != nil {
		t.Fatalf("CreateDepositAddress error: %v", err)
	}

	invoiceID := uuid.New()
	acquired, err := m.AcquireDepositAddress(ctx, "tron", "USDT_TRC20", invoiceID)
	if err != nil || acquired == nil {
		t.Fatalf("AcquireDepositAddress error: %v", err)
	}
	if acquired.Address != depAddr.Address || acquired.Status != "assigned" {
		t.Fatalf("unexpected acquired address state: %+v", acquired)
	}

	// 3. Invoice Creation & Idempotency
	inv := &CryptoPaymentInvoice{
		ID:             invoiceID,
		InvoiceCode:    "CRYPTO-INV-1001",
		UserID:         uuid.New(),
		FiatAmountUSD:  50.00,
		CryptoAmount:   50.00,
		ExchangeRate:   1.00,
		CurrencyID:     "USDT_TRC20",
		NetworkID:      "tron",
		DepositAddress: acquired.Address,
		Status:         "pending",
		ExpiresAt:      time.Now().UTC().Add(30 * time.Minute),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := m.CreateCryptoInvoice(ctx, inv); err != nil {
		t.Fatalf("CreateCryptoInvoice error: %v", err)
	}

	byCode, err := m.GetCryptoInvoiceByCode(ctx, "CRYPTO-INV-1001")
	if err != nil || byCode == nil {
		t.Fatalf("GetCryptoInvoiceByCode error: %v", err)
	}

	// Update with TX Hash
	txHash := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	byCode.TxHash = txHash
	byCode.Status = "confirming"
	byCode.Confirmations = 5
	if err := m.UpdateCryptoInvoice(ctx, byCode); err != nil {
		t.Fatalf("UpdateCryptoInvoice error: %v", err)
	}

	// Try assigning same TX Hash to second invoice (must fail idempotency check)
	inv2 := &CryptoPaymentInvoice{
		ID:             uuid.New(),
		InvoiceCode:    "CRYPTO-INV-1002",
		UserID:         uuid.New(),
		FiatAmountUSD:  20.00,
		CryptoAmount:   20.00,
		ExchangeRate:   1.00,
		CurrencyID:     "USDT_TRC20",
		NetworkID:      "tron",
		DepositAddress: "TSecondAddress1234567890",
		TxHash:         txHash,
		Status:         "paid",
		ExpiresAt:      time.Now().UTC().Add(30 * time.Minute),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	err = m.CreateCryptoInvoice(ctx, inv2)
	if err == nil {
		t.Fatalf("expected duplicate tx_hash error, got nil")
	}

	// 4. Ledger entry
	ledger := &CryptoInternalLedger{
		ID:        uuid.New(),
		InvoiceID: &inv.ID,
		EntryType: "deposit_detected",
		Network:   "tron",
		Currency:  "USDT",
		Amount:    50.00,
		TxHash:    txHash,
		Notes:     "Incoming deposit detected",
		CreatedAt: time.Now().UTC(),
	}
	if err := m.RecordCryptoLedger(ctx, ledger); err != nil {
		t.Fatalf("RecordCryptoLedger error: %v", err)
	}
	records, err := m.ListCryptoLedgerByInvoice(ctx, inv.ID)
	if err != nil || len(records) != 1 {
		t.Fatalf("expected 1 ledger record, got %d (err: %v)", len(records), err)
	}
}
