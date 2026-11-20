package crypto

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"hostvra/api/internal/store"
)

// Standard BIP-39 test vector mnemonic (NEVER use in production)
const testVectorMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

func TestHDWallet_DeterministicMultiChainDerivationAndWatchOnlyXPub(t *testing.T) {
	seed, err := SeedFromMnemonic(testVectorMnemonic, "")
	if err != nil {
		t.Fatalf("SeedFromMnemonic failed: %v", err)
	}

	networks := []string{"ethereum", "bsc", "polygon", "tron", "bitcoin", "solana", "litecoin", "dogecoin"}
	for _, netID := range networks {
		res0, err := DeriveAddressFromSeed(seed, netID, 0)
		if err != nil {
			t.Fatalf("DeriveAddressFromSeed(%s, 0) failed: %v", netID, err)
		}
		res1, err := DeriveAddressFromSeed(seed, netID, 1)
		if err != nil {
			t.Fatalf("DeriveAddressFromSeed(%s, 1) failed: %v", netID, err)
		}

		if !ValidateAddress(netID, res0.Address) {
			t.Errorf("derived address %s failed ValidateAddress for %s", res0.Address, netID)
		}
		if res0.Address == res1.Address {
			t.Errorf("network %s produced identical addresses for index 0 and 1: %s", netID, res0.Address)
		}

		// Verify watch-only xpub derivation matches seed derivation identically
		spec := NetworkDerivationSpecs[netID]
		if spec.SupportsXPub {
			xpub, _, err := DeriveAccountXPub(seed, netID)
			if err != nil {
				t.Fatalf("DeriveAccountXPub(%s) failed: %v", netID, err)
			}
			watchOnlyRes0, err := DeriveAddressFromXPub(xpub, netID, 0)
			if err != nil {
				t.Fatalf("DeriveAddressFromXPub(%s, 0) failed: %v", netID, err)
			}
			if watchOnlyRes0.Address != res0.Address {
				t.Errorf("watch-only xpub mismatch on %s: got %s, want %s", netID, watchOnlyRes0.Address, res0.Address)
			}
		}

		t.Logf("[%s] %s path=%s addr0=%s addr1=%s", netID, spec.Scheme, res0.DerivationPath, res0.Address, res1.Address)
	}

	// Verify well-known BIP-44 Ethereum address for "abandon ... about" m/44'/60'/0'/0/0
	eth0, _ := DeriveAddressFromSeed(seed, "ethereum", 0)
	if !strings.EqualFold(eth0.Address, "0x9858EfFD232B4033E47d90003D41EC34EcaEda94") {
		t.Errorf("BIP-44 Ethereum vector mismatch: got %s, want 0x9858EfFD232B4033E47d90003D41EC34EcaEda94", eth0.Address)
	}
}

func TestHDWallet_ConcurrentUniqueAllocationAndIdempotency(t *testing.T) {
	st := store.NewMemoryStore()
	signer, err := NewTestIsolatedSigner(testVectorMnemonic)
	if err != nil {
		t.Fatalf("NewTestIsolatedSigner failed: %v", err)
	}
	hdSvc := NewHDWalletService(st, signer)
	ctx := context.Background()

	if _, err := hdSvc.InitializeWalletAccounts(ctx, "test_admin"); err != nil {
		t.Fatalf("InitializeWalletAccounts failed: %v", err)
	}

	const userCount = 20
	var wg sync.WaitGroup
	results := make([]*store.HDWalletAddress, userCount)
	errs := make([]error, userCount)

	for i := 0; i < userCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			uid := uuid.New()
			invID := uuid.New()
			addr, allocErr := hdSvc.AllocateUserAddress(ctx, uid, invID, "bsc", "BNB", time.Now().Add(30*time.Minute))
			results[idx] = addr
			errs[idx] = allocErr
		}(i)
	}
	wg.Wait()

	seenAddrs := make(map[string]bool)
	seenIndices := make(map[int]bool)
	for i := 0; i < userCount; i++ {
		if errs[i] != nil {
			t.Fatalf("concurrent allocation %d failed: %v", i, errs[i])
		}
		if seenAddrs[results[i].DepositAddress] {
			t.Fatalf("duplicate deposit address allocated: %s", results[i].DepositAddress)
		}
		if seenIndices[results[i].DerivationIndex] {
			t.Fatalf("duplicate derivation index allocated: %d", results[i].DerivationIndex)
		}
		seenAddrs[results[i].DepositAddress] = true
		seenIndices[results[i].DerivationIndex] = true
	}

	// Verify idempotency: same invoiceID returns exact same address
	uid := uuid.New()
	invID := uuid.New()
	first, err := hdSvc.AllocateUserAddress(ctx, uid, invID, "tron", "USDT_TRC20", time.Now().Add(30*time.Minute))
	if err != nil {
		t.Fatalf("first tron allocation failed: %v", err)
	}
	second, err := hdSvc.AllocateUserAddress(ctx, uid, invID, "tron", "USDT_TRC20", time.Now().Add(30*time.Minute))
	if err != nil {
		t.Fatalf("second tron allocation failed: %v", err)
	}
	if first.DepositAddress != second.DepositAddress || first.DerivationIndex != second.DerivationIndex {
		t.Fatalf("idempotency violated: first=%s(#%d) second=%s(#%d)", first.DepositAddress, first.DerivationIndex, second.DepositAddress, second.DerivationIndex)
	}

	// Verify new invoice for the SAME user generates a BRAND-NEW unique HD address (index >= 1, never index 0)
	invID2 := uuid.New()
	third, err := hdSvc.AllocateUserAddress(ctx, uid, invID2, "tron", "USDT_TRC20", time.Now().Add(30*time.Minute))
	if err != nil {
		t.Fatalf("third tron allocation failed: %v", err)
	}
	if third.DepositAddress == first.DepositAddress || third.DerivationIndex == first.DerivationIndex {
		t.Fatalf("expected brand-new address for new invoice, got duplicate: %s (#%d)", third.DepositAddress, third.DerivationIndex)
	}
	if first.DerivationIndex < 1 || third.DerivationIndex < 2 {
		t.Fatalf("expected derivation indices >= 1 (skipping merchant index 0), got first=%d third=%d", first.DerivationIndex, third.DerivationIndex)
	}
}

func TestIsolatedSigner_PolicyEnforcementAndNoSecretLeakage(t *testing.T) {
	signer, err := NewTestIsolatedSigner(testVectorMnemonic)
	if err != nil {
		t.Fatalf("NewTestIsolatedSigner failed: %v", err)
	}

	// 1. Verify SignerStatus JSON never leaks mnemonic or seed
	statusBytes, _ := json.Marshal(signer.Status())
	statusStr := string(statusBytes)
	if strings.Contains(statusStr, "abandon") || strings.Contains(statusStr, "xprv") {
		t.Fatalf("SECURITY VIOLATION: signer status leaked secret material: %s", statusStr)
	}

	derived0, _ := signer.DerivePublicAddress("bsc", 0)
	trustedTrustWallet := "0x80beBA68E5477712B3342fE43267C947cffAe54E"
	untrustedAttacker := "0x000000000000000000000000000000000000dEaD"
	allowlist := map[string]bool{
		strings.ToLower(trustedTrustWallet): true,
	}

	// 2. Reject unallowlisted destination
	_, err = signer.SignEVMLegacySweepTx(&EVMSweepSignRequest{
		NetworkID:       "bsc",
		ChainID:         56,
		DerivationIndex: 0,
		FromAddress:     derived0.Address,
		ToAddress:       untrustedAttacker,
		Nonce:           0,
		GasPriceWei:     big.NewInt(3_000_000_000),
		GasLimit:        21000,
		ValueWei:        big.NewInt(1_000_000_000_000_000),
		AmountUSD:       10,
		MaxSweepUSD:     50000,
	}, allowlist)
	if err == nil {
		t.Fatalf("expected signing rejection for non-allowlisted destination")
	}

	// 3. Reject when emergency pause is active
	signer.SetEmergencyPause(true)
	_, err = signer.SignEVMLegacySweepTx(&EVMSweepSignRequest{
		NetworkID:       "bsc",
		ChainID:         56,
		DerivationIndex: 0,
		FromAddress:     derived0.Address,
		ToAddress:       trustedTrustWallet,
		Nonce:           0,
		GasPriceWei:     big.NewInt(3_000_000_000),
		GasLimit:        21000,
		ValueWei:        big.NewInt(1_000_000_000_000_000),
		AmountUSD:       10,
		MaxSweepUSD:     50000,
	}, allowlist)
	if err == nil {
		t.Fatalf("expected signing rejection during emergency pause")
	}
	signer.SetEmergencyPause(false)

	// 4. Reject exceeding MaxSweepUSD
	_, err = signer.SignEVMLegacySweepTx(&EVMSweepSignRequest{
		NetworkID:       "bsc",
		ChainID:         56,
		DerivationIndex: 0,
		FromAddress:     derived0.Address,
		ToAddress:       trustedTrustWallet,
		Nonce:           0,
		GasPriceWei:     big.NewInt(3_000_000_000),
		GasLimit:        21000,
		ValueWei:        big.NewInt(1_000_000_000_000_000),
		AmountUSD:       60000,
		MaxSweepUSD:     50000,
	}, allowlist)
	if err == nil {
		t.Fatalf("expected signing rejection when exceeding MaxSweepUSD")
	}

	// 5. Valid native BNB sweep signs cleanly with real EIP-155 RLP raw tx & 32-byte txHash
	signed, err := signer.SignEVMLegacySweepTx(&EVMSweepSignRequest{
		NetworkID:       "bsc",
		ChainID:         56,
		DerivationIndex: 0,
		FromAddress:     derived0.Address,
		ToAddress:       trustedTrustWallet,
		Nonce:           0,
		GasPriceWei:     big.NewInt(3_000_000_000),
		GasLimit:        21000,
		ValueWei:        big.NewInt(1_000_000_000_000_000),
		AmountUSD:       10,
		MaxSweepUSD:     50000,
	}, allowlist)
	if err != nil {
		t.Fatalf("valid native sweep signing failed: %v", err)
	}
	if !strings.HasPrefix(signed.RawTxHex, "0x") || !IsHexTxHash(signed.TxHash) {
		t.Fatalf("invalid signed tx output: raw=%s hash=%s", signed.RawTxHex, signed.TxHash)
	}

	// 6. Valid BEP20 USDT sweep signs cleanly
	signedTok, err := signer.SignEVMLegacySweepTx(&EVMSweepSignRequest{
		NetworkID:       "bsc",
		ChainID:         56,
		DerivationIndex: 0,
		FromAddress:     derived0.Address,
		TokenContract:   "0x55d398326f99059fF775485246999027B3197955",
		TokenRecipient:  trustedTrustWallet,
		Nonce:           1,
		GasPriceWei:     big.NewInt(3_000_000_000),
		GasLimit:        65000,
		ValueWei:        big.NewInt(50_000_000_000_000_000),
		AmountUSD:       50,
		MaxSweepUSD:     50000,
	}, allowlist)
	if err != nil {
		t.Fatalf("valid BEP20 token sweep signing failed: %v", err)
	}
	if !IsHexTxHash(signedTok.TxHash) {
		t.Fatalf("invalid signed token tx hash: %s", signedTok.TxHash)
	}
}

func TestTreasurySweepEngine_IdempotentEnqueue(t *testing.T) {
	st := store.NewMemoryStore()
	signer, _ := NewTestIsolatedSigner(testVectorMnemonic)
	hdSvc := NewHDWalletService(st, signer)
	engine := NewTreasurySweepEngine(st, NewMultiChainRpcClient(), signer)
	ctx := context.Background()

	// Configure Trust Wallet destination
	_ = st.CreateCryptoTreasuryWallet(ctx, &store.CryptoTreasuryWallet{
		ID:        uuid.New(),
		NetworkID: "bsc",
		Name:      "Trust Wallet BSC",
		Address:   "0x80beBA68E5477712B3342fE43267C947cffAe54E",
		IsActive:  true,
	})

	uid := uuid.New()
	invID := uuid.New()
	addr, err := hdSvc.AllocateUserAddress(ctx, uid, invID, "bsc", "BNB", time.Now().Add(30*time.Minute))
	if err != nil {
		t.Fatalf("AllocateUserAddress failed: %v", err)
	}

	inv := &store.CryptoPaymentInvoice{
		ID:             invID,
		InvoiceCode:    "HV-CRYPTO-TEST-001",
		UserID:         uid,
		NetworkID:      "bsc",
		CurrencyID:     "BNB",
		DepositAddress: addr.DepositAddress,
		FiatAmountUSD:  25.0,
		AmountReceived: 0.04,
		TxHash:         "0x1111111111111111111111111111111111111111111111111111111111111111",
		Status:         "paid",
	}

	job1, err := engine.EnqueueSweepForInvoice(ctx, inv)
	if err != nil || job1 == nil {
		t.Fatalf("first EnqueueSweepForInvoice failed: %v", err)
	}
	job2, err := engine.EnqueueSweepForInvoice(ctx, inv)
	if err != nil || job2 == nil {
		t.Fatalf("second EnqueueSweepForInvoice failed: %v", err)
	}
	if job1.ID != job2.ID {
		t.Fatalf("sweep idempotency violated: created duplicate jobs %s and %s", job1.ID, job2.ID)
	}
}
