package crypto

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"hostvra/api/internal/store"
)

func TestCrypto_AddressValidator(t *testing.T) {
	// Valid TRON
	if !ValidateAddress("tron", "TYDzsYUE4jEJTXsyNDoja2d1M7T4ZkYjHn") {
		t.Errorf("expected valid TRON address")
	}
	// Invalid TRON
	if ValidateAddress("tron", "0x55d398326f99059fF775485246999027B3197955") {
		t.Errorf("expected invalid TRON address for EVM input")
	}

	// Valid EVM
	if !ValidateAddress("ethereum", "0xdAC17F958D2ee523a2206206994597C13D831ec7") {
		t.Errorf("expected valid Ethereum address")
	}
	if !ValidateAddress("bsc", "0x55d398326f99059fF775485246999027B3197955") {
		t.Errorf("expected valid BSC address")
	}

	// Valid BTC
	if !ValidateAddress("bitcoin", "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa") {
		t.Errorf("expected valid Bitcoin address")
	}
	if !ValidateAddress("bitcoin", "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq") {
		t.Errorf("expected valid Bitcoin bech32 address")
	}

	// TX Hash validator
	if !IsHexTxHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") {
		t.Errorf("expected valid hex tx hash")
	}
}

func TestCrypto_PricingService(t *testing.T) {
	ps := NewPricingService(60 * time.Second)
	ctx := context.Background()

	// USDT must always be $1.00 USD
	rate, err := ps.GetPriceUSD(ctx, "USDT")
	if err != nil || rate != 1.0 {
		t.Fatalf("expected USDT rate to be 1.0, got %f (err: %v)", rate, err)
	}

	// Conversion $50 USD to USDT with 6 decimals
	cryptoAmount, usedRate, err := ps.ConvertFiatToCrypto(ctx, 50.0, "USDT", 6)
	if err != nil || cryptoAmount != 50.0 || usedRate != 1.0 {
		t.Fatalf("expected 50.0 USDT, got %f (rate: %f)", cryptoAmount, usedRate)
	}
}

func TestCrypto_ServiceLifecycle(t *testing.T) {
	ctx := context.Background()
	m := store.NewMemoryStoreWithPath(t.TempDir() + "/store.json")

	// Create deposit address in pool
	depAddr := &store.CryptoDepositAddress{
		ID:         uuid.New(),
		NetworkID:  "tron",
		CurrencyID: "USDT_TRC20",
		Address:    "TYDzsYUE4jEJTXsyNDoja2d1M7T4ZkYjHn",
		Status:     "available",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	_ = m.CreateDepositAddress(ctx, depAddr)

	settledCalled := false
	svc := NewCryptoService(m, func(ctx context.Context, invoice *store.CryptoPaymentInvoice) error {
		settledCalled = true
		return nil
	})

	// Create invoice
	req := &CreateCryptoInvoiceRequest{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		PayableType:    "invoice",
		PayableID:      "INV-1001",
		FiatAmountUSD:  25.0,
		NetworkID:      "tron",
		CurrencyID:     "USDT_TRC20",
	}

	inv, err := svc.CreateInvoice(ctx, req)
	if err != nil {
		t.Fatalf("CreateInvoice error: %v", err)
	}
	if inv.DepositAddress != depAddr.Address || inv.CryptoAmount != 25.0 {
		t.Fatalf("unexpected invoice properties: %+v", inv)
	}

	// Verify invoice retrieval
	fetched, err := svc.GetInvoiceByCode(ctx, inv.InvoiceCode)
	if err != nil || fetched.InvoiceCode != inv.InvoiceCode {
		t.Fatalf("GetInvoiceByCode failed: %v", err)
	}

	// Start & Stop Background Scanner cleanly
	scanner := NewBackgroundScanner(svc, 100*time.Millisecond)
	scanCtx, cancel := context.WithCancel(ctx)
	scanner.Start(scanCtx)
	time.Sleep(150 * time.Millisecond)
	cancel()
	scanner.Stop()

	if settledCalled {
		t.Logf("settled called early")
	}
}
