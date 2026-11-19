package crypto

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"hostvra/api/internal/store"
)

type SettleCallbackFunc func(ctx context.Context, invoice *store.CryptoPaymentInvoice) error

type CreateCryptoInvoiceRequest struct {
	UserID         uuid.UUID `json:"user_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	PayableType    string    `json:"payable_type"` // invoice, balance_topup, domain_order
	PayableID      string    `json:"payable_id"`
	FiatAmountUSD  float64   `json:"fiat_amount_usd"`
	NetworkID      string    `json:"network_id"`
	CurrencyID     string    `json:"currency_id"`
}

type CryptoService struct {
	store          store.Store
	pricing        *PricingService
	rpc            *MultiChainRpcClient
	signer         *IsolatedSigner
	hdWallet       *HDWalletService
	treasury       *TreasurySweepEngine
	settleCallback SettleCallbackFunc
	mu             sync.Mutex
}

func NewCryptoService(st store.Store, settleCb SettleCallbackFunc) *CryptoService {
	rpc := NewMultiChainRpcClient()
	signer := NewIsolatedSigner()
	hdWallet := NewHDWalletService(st, signer)
	treasury := NewTreasurySweepEngine(st, rpc, signer)
	return &CryptoService{
		store:          st,
		pricing:        GetPricingService(),
		rpc:            rpc,
		signer:         signer,
		hdWallet:       hdWallet,
		treasury:       treasury,
		settleCallback: settleCb,
	}
}

func (s *CryptoService) Signer() *IsolatedSigner         { return s.signer }
func (s *CryptoService) HDWallet() *HDWalletService      { return s.hdWallet }
func (s *CryptoService) Treasury() *TreasurySweepEngine  { return s.treasury }

// SetSettleCallback sets or overrides the billing settlement callback.
func (s *CryptoService) SetSettleCallback(cb SettleCallbackFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settleCallback = cb
}

// CreateInvoice generates a new active crypto invoice with a dedicated deposit address and locked exchange rate.
func (s *CryptoService) CreateInvoice(ctx context.Context, req *CreateCryptoInvoiceRequest) (*store.CryptoPaymentInvoice, error) {
	if req.FiatAmountUSD <= 0 {
		return nil, fmt.Errorf("fiat amount must be greater than zero")
	}

	net, err := s.store.GetCryptoNetworkByID(ctx, req.NetworkID)
	if err != nil || net == nil || !net.IsActive {
		return nil, fmt.Errorf("cryptocurrency network '%s' is not active or supported", req.NetworkID)
	}

	cur, err := s.store.GetCryptoCurrencyByID(ctx, req.CurrencyID)
	if err != nil || cur == nil || !cur.IsActive {
		return nil, fmt.Errorf("cryptocurrency '%s' is not active or supported", req.CurrencyID)
	}

	if req.FiatAmountUSD < cur.MinAmountUSD {
		return nil, fmt.Errorf("minimum payment amount for %s is $%.2f USD", cur.Symbol, cur.MinAmountUSD)
	}

	// Calculate crypto token amount based on live exchange rate
	cryptoAmount, rate, err := s.pricing.ConvertFiatToCrypto(ctx, req.FiatAmountUSD, cur.Symbol, cur.Decimals)
	if err != nil {
		return nil, fmt.Errorf("pricing calculation error: %w", err)
	}

	invoiceID := uuid.New()

	now := time.Now().UTC()
	expiryMinutes := 30
	settings, _ := s.store.GetCryptoSettings(ctx)
	if settings["invoice_expiry_minutes"] != "" {
		if exp, pErr := fmt.Sscanf(settings["invoice_expiry_minutes"], "%d", &expiryMinutes); pErr != nil || exp <= 0 {
			expiryMinutes = 30
		}
	}
	expiresAt := now.Add(time.Duration(expiryMinutes) * time.Minute)

	// Acquire unique HD deposit address (Phase 5) when HD wallet is enabled & initialized,
	// falling back to configured treasury pool for backward compatibility.
	var depositAddressStr string
	var poolAddrID uuid.UUID
	useHD := !strings.EqualFold(settings["hd_wallet_enabled"], "false") && s.signer.Status().Initialized

	// Verify merchant has an active Trust Wallet destination configured before accepting deposits
	if _, tErr := s.treasury.resolveTreasuryDestination(ctx, net.ID); tErr != nil {
		// Allow existing pool addresses if any are manually seeded
		depositAddr, pErr := s.store.AcquireDepositAddress(ctx, net.ID, cur.ID, invoiceID)
		if pErr != nil {
			return nil, fmt.Errorf("no treasury destination address configured for %s (%s). Super Admin must add an approved Trust Wallet address in Admin Dashboard -> Crypto Gateway (/admin/crypto)", net.Name, cur.Symbol)
		}
		depositAddressStr = depositAddr.Address
		poolAddrID = depositAddr.ID
	} else if useHD {
		hdAddr, hdErr := s.hdWallet.AllocateUserAddress(ctx, req.UserID, invoiceID, net.ID, cur.ID, expiresAt)
		if hdErr == nil && hdAddr != nil {
			depositAddressStr = hdAddr.DepositAddress
		}
	}

	if depositAddressStr == "" {
		depositAddr, pErr := s.store.AcquireDepositAddress(ctx, net.ID, cur.ID, invoiceID)
		if pErr != nil {
			return nil, fmt.Errorf("no treasury destination address configured for %s (%s). Super Admin must add an approved Trust Wallet address in Admin Dashboard -> Crypto Gateway (/admin/crypto)", net.Name, cur.Symbol)
		}
		depositAddressStr = depositAddr.Address
		poolAddrID = depositAddr.ID
	}

	// QR Code URI scheme
	qrUri := ""
	switch net.ID {
	case "tron":
		qrUri = fmt.Sprintf("tron:%s?amount=%.6f", depositAddressStr, cryptoAmount)
	case "ethereum", "bsc", "polygon":
		if cur.IsToken {
			qrUri = fmt.Sprintf("ethereum:%s@%d/transfer?address=%s&uint256=%.0f", cur.ContractAddress, net.ChainID, depositAddressStr, cryptoAmount*math.Pow10(cur.Decimals))
		} else {
			qrUri = fmt.Sprintf("ethereum:%s?value=%.18f", depositAddressStr, cryptoAmount)
		}
	case "bitcoin":
		qrUri = fmt.Sprintf("bitcoin:%s?amount=%.8f", depositAddressStr, cryptoAmount)
	case "solana":
		qrUri = fmt.Sprintf("solana:%s?amount=%.9f", depositAddressStr, cryptoAmount)
	case "litecoin":
		qrUri = fmt.Sprintf("litecoin:%s?amount=%.8f", depositAddressStr, cryptoAmount)
	case "dogecoin":
		qrUri = fmt.Sprintf("dogecoin:%s?amount=%.8f", depositAddressStr, cryptoAmount)
	default:
		qrUri = depositAddressStr
	}

	invCode := fmt.Sprintf("HV-CRYPTO-%d-%05d", now.Year(), rand.Intn(90000)+10000)

	invoice := &store.CryptoPaymentInvoice{
		ID:               invoiceID,
		InvoiceCode:      invCode,
		UserID:           req.UserID,
		OrganizationID:   req.OrganizationID,
		PayableType:      req.PayableType,
		PayableID:        req.PayableID,
		FiatAmountUSD:    req.FiatAmountUSD,
		CryptoAmount:     cryptoAmount,
		ExchangeRate:     rate,
		CurrencyID:       cur.ID,
		NetworkID:        net.ID,
		DepositAddress:   depositAddressStr,
		Confirmations:    0,
		MinConfirmations: cur.MinConfirmations,
		AmountReceived:   0,
		Status:           "pending",
		QRUri:            qrUri,
		ExpiresAt:        expiresAt,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.store.CreateCryptoInvoice(ctx, invoice); err != nil {
		if poolAddrID != uuid.Nil {
			_ = s.store.ReleaseDepositAddress(ctx, poolAddrID)
		}
		return nil, fmt.Errorf("failed to persist crypto payment invoice: %w", err)
	}

	return invoice, nil
}

// GetInvoiceByCode retrieves a crypto payment invoice with calculated time remaining.
func (s *CryptoService) GetInvoiceByCode(ctx context.Context, code string) (*store.CryptoPaymentInvoice, error) {
	inv, err := s.store.GetCryptoInvoiceByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// VerifyInvoiceOnChain performs real-time on-chain verification for a specific invoice.
func (s *CryptoService) VerifyInvoiceOnChain(ctx context.Context, invoiceCode string, manualTxHash string) (*store.CryptoPaymentInvoice, error) {
	inv, err := s.store.GetCryptoInvoiceByCode(ctx, invoiceCode)
	if err != nil {
		return nil, err
	}

	if inv.Status == "paid" {
		return inv, nil
	}

	now := time.Now().UTC()
	if inv.ExpiresAt.Before(now) && inv.Status == "pending" {
		inv.Status = "expired"
		_ = s.store.UpdateCryptoInvoice(ctx, inv)
		return inv, nil
	}

	net, err := s.store.GetCryptoNetworkByID(ctx, inv.NetworkID)
	if err != nil || net == nil {
		return nil, fmt.Errorf("network not found: %s", inv.NetworkID)
	}

	cur, err := s.store.GetCryptoCurrencyByID(ctx, inv.CurrencyID)
	if err != nil || cur == nil {
		return nil, fmt.Errorf("currency not found: %s", inv.CurrencyID)
	}

	var bestMatch *DetectedPayment
	manualTxHash = strings.TrimSpace(manualTxHash)

	// 1. If manualTxHash is provided, verify directly on-chain first
	if manualTxHash != "" {
		switch inv.NetworkID {
		case "bsc", "ethereum", "polygon":
			match, err := s.rpc.VerifyEVMTransaction(ctx, net.RPCURL, manualTxHash, inv.DepositAddress, cur.ContractAddress, cur.Decimals, cur.IsToken, inv.CryptoAmount)
			if err == nil && match != nil {
				bestMatch = match
			}
		case "tron":
			match, err := s.rpc.VerifyTronTransaction(ctx, net.RPCURL, manualTxHash, inv.DepositAddress, cur.ContractAddress, cur.IsToken, inv.CryptoAmount)
			if err == nil && match != nil {
				bestMatch = match
			}
		case "solana":
			match, err := s.rpc.VerifySolanaTransaction(ctx, net.RPCURL, manualTxHash, inv.DepositAddress, inv.CryptoAmount)
			if err == nil && match != nil {
				bestMatch = match
			}
		case "bitcoin", "litecoin", "dogecoin":
			var detected []*DetectedPayment
			if inv.NetworkID == "bitcoin" {
				detected, _ = s.rpc.ScanBitcoinDeposits(ctx, net.RPCURL, inv.DepositAddress, 1)
			} else if inv.NetworkID == "litecoin" {
				detected, _ = s.rpc.ScanLitecoinDeposits(ctx, net.RPCURL, inv.DepositAddress, 1)
			} else {
				detected, _ = s.rpc.ScanDogecoinDeposits(ctx, net.RPCURL, inv.DepositAddress, 1)
			}
			for _, d := range detected {
				if strings.EqualFold(d.TxHash, manualTxHash) && d.Amount >= inv.CryptoAmount*0.97 {
					bestMatch = d
					break
				}
			}
		}
	}

	// 2. Automatic on-chain detection (polling or fallback)
	if bestMatch == nil {
		minExpected := inv.CryptoAmount * 0.97 // 3% tolerance for price fluctuation / gas deduction

		switch inv.NetworkID {
		case "tron":
			if cur.IsToken {
				detectedList, _ := s.rpc.ScanTronTRC20Deposits(ctx, net.RPCURL, cur.ContractAddress, inv.DepositAddress, cur.MinConfirmations)
				for _, d := range detectedList {
					if d.Amount >= minExpected {
						bestMatch = d
						break
					}
				}
			} else {
				if tronBal, err := s.rpc.GetTronBalance(ctx, net.RPCURL, inv.DepositAddress); err == nil && tronBal >= minExpected {
					bestMatch = &DetectedPayment{
						TxHash:        fmt.Sprintf("tron_trx_%s_%s", inv.InvoiceCode, inv.DepositAddress[len(inv.DepositAddress)-6:]),
						ToAddress:     inv.DepositAddress,
						Amount:        tronBal,
						Confirmations: 3,
						Confirmed:     true,
						Timestamp:     now,
					}
				}
			}

		case "bsc", "ethereum", "polygon":
			if cur.IsToken {
				// Token transfer: scan logs first
				detectedList, _ := s.rpc.ScanEVMTokenDeposits(ctx, net.RPCURL, cur.ContractAddress, inv.DepositAddress, cur.Decimals, 0)
				for _, d := range detectedList {
					if d.Amount >= minExpected {
						bestMatch = d
						break
					}
				}
				// Fallback to token balance check
				if bestMatch == nil {
					if tokBal, err := s.rpc.GetEVMTokenBalance(ctx, net.RPCURL, cur.ContractAddress, inv.DepositAddress, cur.Decimals); err == nil && tokBal >= minExpected {
						bestMatch = &DetectedPayment{
							TxHash:          fmt.Sprintf("%s_tok_%s", inv.NetworkID, inv.InvoiceCode),
							ToAddress:       inv.DepositAddress,
							ContractAddress: cur.ContractAddress,
							Amount:          tokBal,
							Confirmations:   3,
							Confirmed:       true,
							Timestamp:       now,
						}
					}
				}
			} else {
				// Native coin transfer (BNB, ETH, POL):
				// First: scan recent blocks
				if match, err := s.rpc.ScanEVMLatestBlocks(ctx, net.RPCURL, inv.DepositAddress, 25, inv.CryptoAmount); err == nil && match != nil {
					bestMatch = match
				}
				// Second: check on-chain balance directly
				if bestMatch == nil {
					if evmBal, err := s.rpc.GetEVMBalance(ctx, net.RPCURL, inv.DepositAddress); err == nil && evmBal >= minExpected {
						bestMatch = &DetectedPayment{
							TxHash:        fmt.Sprintf("%s_native_%s", inv.NetworkID, inv.InvoiceCode),
							ToAddress:     inv.DepositAddress,
							Amount:        evmBal,
							Confirmations: 3,
							Confirmed:     true,
							Timestamp:     now,
						}
					}
				}
			}

		case "bitcoin":
			detectedList, _ := s.rpc.ScanBitcoinDeposits(ctx, net.RPCURL, inv.DepositAddress, cur.MinConfirmations)
			for _, d := range detectedList {
				if d.Amount >= minExpected {
					bestMatch = d
					break
				}
			}

		case "litecoin":
			detectedList, _ := s.rpc.ScanLitecoinDeposits(ctx, net.RPCURL, inv.DepositAddress, cur.MinConfirmations)
			for _, d := range detectedList {
				if d.Amount >= minExpected {
					bestMatch = d
					break
				}
			}

		case "dogecoin":
			detectedList, _ := s.rpc.ScanDogecoinDeposits(ctx, net.RPCURL, inv.DepositAddress, cur.MinConfirmations)
			for _, d := range detectedList {
				if d.Amount >= minExpected {
					bestMatch = d
					break
				}
			}

		case "solana":
			if solBal, err := s.rpc.GetSolanaBalance(ctx, net.RPCURL, inv.DepositAddress); err == nil && solBal >= minExpected {
				bestMatch = &DetectedPayment{
					TxHash:        fmt.Sprintf("sol_native_%s", inv.InvoiceCode),
					ToAddress:     inv.DepositAddress,
					Amount:        solBal,
					Confirmations: 1,
					Confirmed:     true,
					Timestamp:     now,
				}
			}
		}
	}

	// 3. If manualTxHash was provided and payment not yet confirmed, bind tx_hash and set status to detecting
	if bestMatch == nil && manualTxHash != "" {
		if inv.TxHash != manualTxHash {
			inv.TxHash = manualTxHash
			if inv.Status == "pending" {
				inv.Status = "detecting"
			}
			_ = s.store.UpdateCryptoInvoice(ctx, inv)
		}
	}

	if bestMatch != nil {
		// Verify idempotency: ensure tx_hash isn't already credited to another invoice
		if !strings.Contains(bestMatch.TxHash, inv.InvoiceCode) {
			if existing, _ := s.store.GetCryptoInvoiceByTxHash(ctx, bestMatch.TxHash); existing != nil && existing.ID != inv.ID {
				return nil, fmt.Errorf("transaction hash '%s' has already been used for another invoice", bestMatch.TxHash)
			}
		}

		inv.TxHash = bestMatch.TxHash
		inv.AmountReceived = bestMatch.Amount
		inv.Confirmations = bestMatch.Confirmations
		if inv.DetectedAt == nil {
			inv.DetectedAt = &now
		}

		if bestMatch.Confirmations >= inv.MinConfirmations || bestMatch.Confirmed {
			inv.Status = "paid"
			inv.PaidAt = &now

			if err := s.store.UpdateCryptoInvoice(ctx, inv); err != nil {
				return nil, err
			}

			// Record immutable double-entry ledger entry
			ledger := &store.CryptoInternalLedger{
				ID:            uuid.New(),
				InvoiceID:     &inv.ID,
				EntryType:     "invoice_settled",
				Amount:        inv.AmountReceived,
				Currency:      cur.Symbol,
				Network:       net.ID,
				FromAddress:   bestMatch.FromAddress,
				ToAddress:     inv.DepositAddress,
				TxHash:        inv.TxHash,
				BlockNumber:   bestMatch.BlockNumber,
				Confirmations: bestMatch.Confirmations,
				Notes:         fmt.Sprintf("Payment verified on-chain and settled for %s", inv.InvoiceCode),
				CreatedAt:     now,
			}
			_ = s.store.RecordCryptoLedger(ctx, ledger)

			// Execute billing settlement callback (package activation & subscription renewal)
			if s.settleCallback != nil {
				_ = s.settleCallback(ctx, inv)
			}

			// Decoupled Phase 5 Automated Treasury Sweep enqueue (non-blocking: sweep failure never affects user package)
			if s.treasury != nil {
				_, _ = s.treasury.EnqueueSweepForInvoice(ctx, inv)
			}
		} else {
			inv.Status = "confirming"
			_ = s.store.UpdateCryptoInvoice(ctx, inv)
		}
	}

	return inv, nil
}

// ScanAllPendingInvoices is executed by background workers to continuously update pending invoices.
func (s *CryptoService) ScanAllPendingInvoices(ctx context.Context) error {
	pending, err := s.store.ListPendingCryptoInvoices(ctx)
	if err != nil {
		return err
	}

	for _, inv := range pending {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, _ = s.VerifyInvoiceOnChain(ctx, inv.InvoiceCode, inv.TxHash)
		}
	}
	return nil
}
