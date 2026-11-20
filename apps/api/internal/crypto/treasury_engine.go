package crypto

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"hostvra/api/internal/store"
)

// TreasurySweepEngine manages the independent automated sweep state machine that transfers
// confirmed funds from unique user HD deposit addresses to the admin's Trust Wallet treasury.
// A sweep failure NEVER affects the user's already-credited balance or activated package.
type TreasurySweepEngine struct {
	store   store.Store
	rpc     *MultiChainRpcClient
	signer  *IsolatedSigner
	pricing *PricingService
	mu      sync.Mutex
}

func NewTreasurySweepEngine(st store.Store, rpc *MultiChainRpcClient, signer *IsolatedSigner) *TreasurySweepEngine {
	return &TreasurySweepEngine{
		store:   st,
		rpc:     rpc,
		signer:  signer,
		pricing: GetPricingService(),
	}
}

// EnqueueSweepForInvoice creates an idempotent sweep job for a settled crypto invoice.
// Safe to call multiple times: idempotency key guarantees at most one sweep per invoice/tx.
func (e *TreasurySweepEngine) EnqueueSweepForInvoice(ctx context.Context, inv *store.CryptoPaymentInvoice) (*store.TreasurySweepJob, error) {
	if inv == nil || inv.Status != "paid" {
		return nil, nil
	}

	// Only sweep HD-derived deposit addresses (if deposit address IS already the Trust Wallet, no sweep needed)
	hdAddr, err := e.store.GetHDWalletAddressByDepositAddress(ctx, inv.DepositAddress)
	if err != nil || hdAddr == nil {
		return nil, nil // Direct-to-Trust-Wallet invoice; funds are already in treasury
	}

	// Find active Trust Wallet destination for this network
	destWallet, err := e.resolveTreasuryDestination(ctx, inv.NetworkID)
	if err != nil || destWallet == nil {
		return nil, fmt.Errorf("no active Trust Wallet treasury configured for network %s", inv.NetworkID)
	}
	if strings.EqualFold(hdAddr.DepositAddress, destWallet.Address) {
		return nil, nil
	}

	idempotencyKey := fmt.Sprintf("sweep:%s:%s:%s", inv.NetworkID, inv.InvoiceCode, inv.TxHash)
	if existing, _ := e.store.GetTreasurySweepJobByIdempotencyKey(ctx, idempotencyKey); existing != nil {
		return existing, nil
	}

	settings, _ := e.store.GetCryptoSettings(ctx)
	minSweepUSD := parseSettingFloat(settings["min_sweep_usd"], 0.5)
	amountUSD := inv.FiatAmountUSD
	if amountUSD < minSweepUSD {
		// Still record as queued/below-threshold so admin can manually batch-sweep later
		_ = e.store.UpdateHDWalletAddressStatus(ctx, hdAddr.ID, "funded")
	}

	job := &store.TreasurySweepJob{
		ID:                  uuid.New(),
		InvoiceID:           &inv.ID,
		SourceAddressID:     &hdAddr.ID,
		SourceAddress:       hdAddr.DepositAddress,
		DerivationIndex:     hdAddr.DerivationIndex,
		DestinationWalletID: &destWallet.ID,
		DestinationAddress:  destWallet.Address,
		AssetID:             inv.CurrencyID,
		NetworkID:           inv.NetworkID,
		RequestedAmount:     inv.AmountReceived,
		AmountUSD:           amountUSD,
		Status:              "queued",
		IdempotencyKey:      idempotencyKey,
		MaxRetries:          5,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}

	if err := e.store.CreateTreasurySweepJob(ctx, job); err != nil {
		return nil, err
	}

	_ = e.store.UpdateHDWalletAddressStatus(ctx, hdAddr.ID, "funded")
	_ = e.store.RecordTreasurySweepLog(ctx, &store.TreasurySweepLog{
		SweepJobID:   job.ID,
		EventType:    "sweep_queued",
		EventDetails: fmt.Sprintf("Queued sweep of %.8f %s ($%.2f) from HD address %s (idx=%d) -> Trust Wallet %s", job.RequestedAmount, job.AssetID, job.AmountUSD, job.SourceAddress, job.DerivationIndex, job.DestinationAddress),
	})

	return job, nil
}

// ProcessPendingSweeps executes all eligible queued/retryable sweep jobs while enforcing
// emergency pause, threshold, daily spend limit, destination allowlist, and gas reserve policies.
func (e *TreasurySweepEngine) ProcessPendingSweeps(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	settings, _ := e.store.GetCryptoSettings(ctx)
	if strings.EqualFold(settings["emergency_sweep_pause"], "true") || e.signer.IsEmergencyPaused() {
		return nil
	}
	if strings.EqualFold(settings["auto_sweep_enabled"], "false") {
		return nil
	}

	jobs, err := e.store.ListTreasurySweepJobs(ctx, "queued", "", 50)
	if err != nil {
		return err
	}

	for _, job := range jobs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_ = e.ExecuteSweepJob(ctx, job.ID, false)
		}
	}
	return nil
}

// ExecuteSweepJob signs and broadcasts a single sweep job with full policy enforcement.
func (e *TreasurySweepEngine) ExecuteSweepJob(ctx context.Context, jobID uuid.UUID, forceManual bool) error {
	job, err := e.store.GetTreasurySweepJobByID(ctx, jobID)
	if err != nil {
		return err
	}
	if job.Status == "completed" || job.Status == "broadcast" {
		return nil
	}

	settings, _ := e.store.GetCryptoSettings(ctx)
	if strings.EqualFold(settings["emergency_sweep_pause"], "true") || e.signer.IsEmergencyPaused() {
		return fmt.Errorf("sweep halted: emergency pause is active")
	}

	minSweepUSD := parseSettingFloat(settings["min_sweep_usd"], 0.5)
	maxSingleUSD := parseSettingFloat(settings["max_single_sweep_usd"], 50000.0)
	maxDailyUSD := parseSettingFloat(settings["max_daily_sweep_usd"], 250000.0)

	if !forceManual && job.AmountUSD < minSweepUSD {
		return fmt.Errorf("amount $%.2f below minimum sweep threshold $%.2f", job.AmountUSD, minSweepUSD)
	}
	if job.AmountUSD > maxSingleUSD {
		return e.failJob(ctx, job, fmt.Sprintf("amount $%.2f exceeds max single sweep limit $%.2f", job.AmountUSD, maxSingleUSD))
	}

	dailySwept, _ := e.store.GetDailySweptUSD(ctx, time.Now().UTC().Add(-24*time.Hour))
	if dailySwept+job.AmountUSD > maxDailyUSD {
		return e.failJob(ctx, job, fmt.Sprintf("daily sweep limit exceeded ($%.2f + $%.2f > $%.2f)", dailySwept, job.AmountUSD, maxDailyUSD))
	}

	// Build destination allowlist exclusively from active verified DB treasuries
	allowlist, err := e.buildAllowlistedDestinations(ctx)
	if err != nil || !allowlist[strings.ToLower(strings.TrimSpace(job.DestinationAddress))] {
		return e.failJob(ctx, job, "destination address is not in the active verified Trust Wallet allowlist")
	}

	net, err := e.store.GetCryptoNetworkByID(ctx, job.NetworkID)
	if err != nil || net == nil {
		return e.failJob(ctx, job, "network configuration not found")
	}
	cur, err := e.store.GetCryptoCurrencyByID(ctx, job.AssetID)
	if err != nil || cur == nil {
		return e.failJob(ctx, job, "currency configuration not found")
	}

	job.Status = "signing"
	_ = e.store.UpdateTreasurySweepJob(ctx, job)

	switch job.NetworkID {
	case "bsc", "ethereum", "polygon":
		return e.executeEVMSweep(ctx, job, net, cur, allowlist, maxSingleUSD)
	default:
		// Non-EVM networks (TRON, BTC, SOL, LTC, DOGE): log readiness & require network-specific signer broadcast
		return e.failJob(ctx, job, fmt.Sprintf("network %s sweep queued for external/HSM broadcast (EVM direct broadcast active)", job.NetworkID))
	}
}

func (e *TreasurySweepEngine) executeEVMSweep(
	ctx context.Context,
	job *store.TreasurySweepJob,
	net *store.CryptoNetwork,
	cur *store.CryptoCurrency,
	allowlist map[string]bool,
	maxSingleUSD float64,
) error {
	nonce, err := e.rpc.GetEVMNonce(ctx, net.RPCURL, job.SourceAddress)
	if err != nil {
		return e.retryOrFailJob(ctx, job, fmt.Sprintf("failed to fetch nonce: %v", err))
	}
	gasPriceWei, err := e.rpc.GetEVMGasPrice(ctx, net.RPCURL)
	if err != nil {
		return e.retryOrFailJob(ctx, job, fmt.Sprintf("failed to fetch gas price: %v", err))
	}

	nativeBal, err := e.rpc.GetEVMBalance(ctx, net.RPCURL, job.SourceAddress)
	if err != nil {
		return e.retryOrFailJob(ctx, job, fmt.Sprintf("failed to check source balance: %v", err))
	}

	var gasLimit uint64 = 21000
	if cur.IsToken {
		gasLimit = 65000
	}

	feeWei := new(big.Int).Mul(gasPriceWei, new(big.Int).SetUint64(gasLimit))
	feeNative, _ := new(big.Float).Quo(new(big.Float).SetInt(feeWei), new(big.Float).SetFloat64(1e18)).Float64()
	job.EstimatedNetworkFee = feeNative

	var sweepValueWei *big.Int
	var actualSweepAmount float64

	if cur.IsToken {
		// Token sweep: ensure source address has sufficient native gas
		requiredGas := feeNative * 1.15
		if nativeBal < requiredGas {
			// Controlled EVM Gas Station: fund exact needed gas (strictly capped) from HD index 0 gas reserve
			if fundedTx, gErr := e.tryControlledGasFunding(ctx, job, net, gasPriceWei, requiredGas-nativeBal); gErr == nil && fundedTx != "" {
				return e.retryOrFailJob(ctx, job, fmt.Sprintf("gas station funded %.8f %s (tx=%s); awaiting confirmation before token sweep", requiredGas-nativeBal, net.NativeCurrency, fundedTx))
			}
			return e.retryOrFailJob(ctx, job, fmt.Sprintf("insufficient native gas on deposit address (have %.8f, need %.8f %s)", nativeBal, requiredGas, net.NativeCurrency))
		}
		tokBal, tErr := e.rpc.GetEVMTokenBalance(ctx, net.RPCURL, cur.ContractAddress, job.SourceAddress, cur.Decimals)
		if tErr != nil || tokBal <= 0 {
			return e.retryOrFailJob(ctx, job, "zero or unreadable token balance on deposit address")
		}
		actualSweepAmount = tokBal
		scale := new(big.Float).SetFloat64(math.Pow10(cur.Decimals))
		valF := new(big.Float).Mul(new(big.Float).SetFloat64(tokBal), scale)
		sweepValueWei, _ = valF.Int(nil)
	} else {
		// Native coin sweep: subtract exact gas fee reserve
		sendable := nativeBal - (feeNative * 1.1)
		if sendable <= 0 {
			return e.retryOrFailJob(ctx, job, fmt.Sprintf("balance %.8f too low to cover gas fee %.8f", nativeBal, feeNative))
		}
		actualSweepAmount = sendable
		valF := new(big.Float).Mul(new(big.Float).SetFloat64(sendable), new(big.Float).SetFloat64(1e18))
		sweepValueWei, _ = valF.Int(nil)
	}

	signReq := &EVMSweepSignRequest{
		NetworkID:       job.NetworkID,
		ChainID:         net.ChainID,
		DerivationIndex: uint32(job.DerivationIndex),
		FromAddress:     job.SourceAddress,
		ToAddress:       job.DestinationAddress,
		Nonce:           nonce,
		GasPriceWei:     gasPriceWei,
		GasLimit:        gasLimit,
		ValueWei:        sweepValueWei,
		AmountUSD:       job.AmountUSD,
		MaxSweepUSD:     maxSingleUSD,
	}
	if cur.IsToken {
		signReq.TokenContract = cur.ContractAddress
		signReq.TokenRecipient = job.DestinationAddress
	}

	signed, err := e.signer.SignEVMLegacySweepTx(signReq, allowlist)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("signer policy rejected sweep: %v", err))
	}

	// Broadcast raw signed transaction
	txHash, err := e.rpc.SendEVMRawTransaction(ctx, net.RPCURL, signed.RawTxHex)
	if err != nil {
		return e.retryOrFailJob(ctx, job, fmt.Sprintf("RPC broadcast error: %v", err))
	}

	now := time.Now().UTC()
	job.TransactionHash = txHash
	job.ActualAmount = actualSweepAmount
	job.ActualNetworkFee = feeNative
	job.Status = "completed"
	job.CompletedAt = &now
	job.LastError = ""
	_ = e.store.UpdateTreasurySweepJob(ctx, job)

	if job.SourceAddressID != nil {
		_ = e.store.UpdateHDWalletAddressStatus(ctx, *job.SourceAddressID, "swept")
	}

	_ = e.store.RecordTreasurySweepLog(ctx, &store.TreasurySweepLog{
		SweepJobID:   job.ID,
		EventType:    "sweep_broadcast_completed",
		EventDetails: fmt.Sprintf("Swept %.8f %s (fee=%.8f) to Trust Wallet %s | tx=%s", actualSweepAmount, cur.Symbol, feeNative, job.DestinationAddress, txHash),
	})

	_ = e.store.RecordCryptoLedger(ctx, &store.CryptoInternalLedger{
		ID:          uuid.New(),
		InvoiceID:   job.InvoiceID,
		EntryType:   "treasury_sweep",
		Amount:      actualSweepAmount,
		Currency:    cur.Symbol,
		Network:     job.NetworkID,
		FromAddress: job.SourceAddress,
		ToAddress:   job.DestinationAddress,
		TxHash:      txHash,
		Notes:       fmt.Sprintf("Automated HD Treasury Sweep to Trust Wallet (job=%s, fee=%.8f)", job.ID.String(), feeNative),
		CreatedAt:   now,
	})

	return nil
}

func (e *TreasurySweepEngine) resolveTreasuryDestination(ctx context.Context, networkID string) (*store.CryptoTreasuryWallet, error) {
	list, err := e.store.ListCryptoTreasuryWallets(ctx, networkID)
	if err == nil {
		for _, w := range list {
			if w.IsActive && strings.TrimSpace(w.Address) != "" {
				return w, nil
			}
		}
	}
	// EVM cross-chain fallback (same 0x Trust Wallet address works across BSC/ETH/Polygon)
	if networkID == "bsc" || networkID == "ethereum" || networkID == "polygon" {
		all, _ := e.store.ListCryptoTreasuryWallets(ctx, "")
		for _, w := range all {
			if w.IsActive && (w.NetworkID == "bsc" || w.NetworkID == "ethereum" || w.NetworkID == "polygon") && strings.TrimSpace(w.Address) != "" {
				return w, nil
			}
		}
	}
	return nil, fmt.Errorf("no active treasury destination for %s", networkID)
}

func (e *TreasurySweepEngine) buildAllowlistedDestinations(ctx context.Context) (map[string]bool, error) {
	all, err := e.store.ListCryptoTreasuryWallets(ctx, "")
	if err != nil {
		return nil, err
	}
	allow := make(map[string]bool)
	for _, w := range all {
		if w.IsActive && strings.TrimSpace(w.Address) != "" {
			allow[strings.ToLower(strings.TrimSpace(w.Address))] = true
		}
	}
	return allow, nil
}

func (e *TreasurySweepEngine) retryOrFailJob(ctx context.Context, job *store.TreasurySweepJob, reason string) error {
	job.RetryCount++
	job.LastError = reason
	if job.RetryCount >= job.MaxRetries {
		job.Status = "failed"
	} else {
		job.Status = "queued"
	}
	_ = e.store.UpdateTreasurySweepJob(ctx, job)
	_ = e.store.RecordTreasurySweepLog(ctx, &store.TreasurySweepLog{
		SweepJobID:   job.ID,
		EventType:    "sweep_retry_scheduled",
		EventDetails: fmt.Sprintf("Attempt %d/%d: %s", job.RetryCount, job.MaxRetries, reason),
	})
	return fmt.Errorf("%s", reason)
}

func (e *TreasurySweepEngine) failJob(ctx context.Context, job *store.TreasurySweepJob, reason string) error {
	job.Status = "failed"
	job.LastError = reason
	_ = e.store.UpdateTreasurySweepJob(ctx, job)
	_ = e.store.RecordTreasurySweepLog(ctx, &store.TreasurySweepLog{
		SweepJobID:   job.ID,
		EventType:    "sweep_failed",
		EventDetails: reason,
	})
	return fmt.Errorf("%s", reason)
}

func parseSettingFloat(val string, fallback float64) float64 {
	if f, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil && f >= 0 {
		return f
	}
	return fallback
}

// tryControlledGasFunding sends an exact, strictly capped native gas top-up from the HD index-0
// hot gas-feeder wallet to a customer deposit address so an ERC20/BEP20 USDT sweep can execute.
// Never sends unlimited gas; enforces max_gas_funding_native safety cap and emergency pause.
func (e *TreasurySweepEngine) tryControlledGasFunding(
	ctx context.Context,
	job *store.TreasurySweepJob,
	net *store.CryptoNetwork,
	gasPriceWei *big.Int,
	neededNative float64,
) (string, error) {
	settings, _ := e.store.GetCryptoSettings(ctx)
	if !strings.EqualFold(settings["gas_station_enabled"], "true") {
		return "", fmt.Errorf("gas station disabled")
	}
	if strings.EqualFold(settings["emergency_sweep_pause"], "true") || e.signer.IsEmergencyPaused() {
		return "", fmt.Errorf("emergency pause active")
	}

	maxCapNative := parseSettingFloat(settings["max_gas_funding_native"], 0.005)
	fundAmount := neededNative * 1.15
	if fundAmount <= 0 || fundAmount > maxCapNative {
		return "", fmt.Errorf("requested gas %.8f exceeds hard safety cap %.8f %s", fundAmount, maxCapNative, net.NativeCurrency)
	}

	feederAddr, err := e.signer.DerivePublicAddress(net.ID, 0)
	if err != nil {
		return "", err
	}
	feederBal, err := e.rpc.GetEVMBalance(ctx, net.RPCURL, feederAddr.Address)
	if err != nil || feederBal < fundAmount*1.5 {
		return "", fmt.Errorf("gas feeder wallet %s has insufficient balance (%.8f %s)", feederAddr.Address, feederBal, net.NativeCurrency)
	}

	nonce, err := e.rpc.GetEVMNonce(ctx, net.RPCURL, feederAddr.Address)
	if err != nil {
		return "", err
	}

	valF := new(big.Float).Mul(new(big.Float).SetFloat64(fundAmount), new(big.Float).SetFloat64(1e18))
	valWei, _ := valF.Int(nil)

	// Explicitly allowlist ONLY the target deposit address for this controlled gas top-up
	gasAllowlist := map[string]bool{
		strings.ToLower(strings.TrimSpace(job.SourceAddress)): true,
	}

	signed, err := e.signer.SignEVMLegacySweepTx(&EVMSweepSignRequest{
		NetworkID:       net.ID,
		ChainID:         net.ChainID,
		DerivationIndex: 0,
		FromAddress:     feederAddr.Address,
		ToAddress:       job.SourceAddress,
		Nonce:           nonce,
		GasPriceWei:     gasPriceWei,
		GasLimit:        21000,
		ValueWei:        valWei,
		AmountUSD:       1.0,
		MaxSweepUSD:     50.0,
	}, gasAllowlist)
	if err != nil {
		return "", err
	}

	txHash, err := e.rpc.SendEVMRawTransaction(ctx, net.RPCURL, signed.RawTxHex)
	if err != nil {
		return "", err
	}

	_ = e.store.RecordTreasurySweepLog(ctx, &store.TreasurySweepLog{
		SweepJobID:   job.ID,
		EventType:    "gas_station_funded",
		EventDetails: fmt.Sprintf("Controlled gas funding %.8f %s from feeder %s -> %s (tx=%s)", fundAmount, net.NativeCurrency, feederAddr.Address, job.SourceAddress, txHash),
	})
	return txHash, nil
}
