package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/crypto"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type CryptoHandler struct {
	cfg       *config.Config
	store     store.Store
	audit     *audit.Logger
	cryptoSvc *crypto.CryptoService
}

func NewCryptoHandler(cfg *config.Config, s store.Store, a *audit.Logger, cryptoSvc *crypto.CryptoService) *CryptoHandler {
	return &CryptoHandler{
		cfg:       cfg,
		store:     s,
		audit:     a,
		cryptoSvc: cryptoSvc,
	}
}

// ----------------------------------------------------------------------------
// Client / Public Endpoints
// ----------------------------------------------------------------------------

// ListCurrencies returns all active cryptocurrency networks and tokens with live USD pricing.
func (h *CryptoHandler) ListCurrencies(w http.ResponseWriter, r *http.Request) {
	nets, err := h.store.ListCryptoNetworks(r.Context(), true)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list crypto networks", nil, "")
		return
	}

	activeTreasuries, _ := h.store.ListCryptoTreasuryWallets(r.Context(), "")
	activeNetMap := make(map[string]bool)
	hasAnyEVM := false
	for _, tw := range activeTreasuries {
		if tw.IsActive && strings.TrimSpace(tw.Address) != "" {
			activeNetMap[tw.NetworkID] = true
			if tw.NetworkID == "bsc" || tw.NetworkID == "ethereum" || tw.NetworkID == "polygon" {
				hasAnyEVM = true
			}
		}
	}
	if hasAnyEVM {
		activeNetMap["bsc"] = true
		activeNetMap["ethereum"] = true
		activeNetMap["polygon"] = true
	}

	type CurrencyOption struct {
		ID               string  `json:"id"`
		NetworkID        string  `json:"network_id"`
		NetworkName      string  `json:"network_name"`
		Symbol           string  `json:"symbol"`
		Name             string  `json:"name"`
		ContractAddress  string  `json:"contract_address,omitempty"`
		Decimals         int     `json:"decimals"`
		MinConfirmations int     `json:"min_confirmations"`
		MinAmountUSD     float64 `json:"min_amount_usd"`
		Icon             string  `json:"icon"`
		PriceUSD         float64 `json:"price_usd"`
		HasTreasury      bool    `json:"has_treasury"`
	}

	pricing := crypto.GetPricingService()
	options := make([]CurrencyOption, 0)

	for _, net := range nets {
		currencies, cErr := h.store.ListCryptoCurrencies(r.Context(), net.ID, true)
		if cErr != nil {
			continue
		}
		for _, cur := range currencies {
			price, _ := pricing.GetPriceUSD(r.Context(), cur.Symbol)
			options = append(options, CurrencyOption{
				ID:               cur.ID,
				NetworkID:        net.ID,
				NetworkName:      net.Name,
				Symbol:           cur.Symbol,
				Name:             cur.Name,
				ContractAddress:  cur.ContractAddress,
				Decimals:         cur.Decimals,
				MinConfirmations: cur.MinConfirmations,
				MinAmountUSD:     cur.MinAmountUSD,
				Icon:             cur.Icon,
				PriceUSD:         price,
				HasTreasury:      activeNetMap[net.ID],
			})
		}
	}

	response.JSON(w, http.StatusOK, options, nil)
}

type CreateInvoiceReq struct {
	PayableType   string  `json:"payable_type"` // invoice, balance_topup
	PayableID     string  `json:"payable_id"`
	FiatAmountUSD float64 `json:"fiat_amount_usd"`
	NetworkID     string  `json:"network_id"`
	CurrencyID    string  `json:"currency_id"`
}

// CreateInvoice generates a dedicated crypto deposit invoice.
func (h *CryptoHandler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	userID := uuid.Nil
	orgID := uuid.Nil
	if claims != nil {
		userID = claims.UserID
		orgID = claims.OrganizationID
	}

	var req CreateInvoiceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON payload", nil, "")
		return
	}

	// If linked to an existing billing invoice, load and verify amount
	if req.PayableType == "invoice" && req.PayableID != "" {
		if invUUID, err := uuid.Parse(req.PayableID); err == nil {
			if billingInv, err := h.store.GetInvoiceByID(r.Context(), invUUID); err == nil && billingInv != nil {
				req.FiatAmountUSD = billingInv.Total
				if claims == nil {
					userID = billingInv.UserID
					orgID = billingInv.OrganizationID
				}
			}
		}
	}

	svcReq := &crypto.CreateCryptoInvoiceRequest{
		UserID:         userID,
		OrganizationID: orgID,
		PayableType:    req.PayableType,
		PayableID:      req.PayableID,
		FiatAmountUSD:  req.FiatAmountUSD,
		NetworkID:      req.NetworkID,
		CurrencyID:     req.CurrencyID,
	}

	inv, err := h.cryptoSvc.CreateInvoice(r.Context(), svcReq)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "CREATION_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "crypto.invoice.create", "crypto_invoice", inv.ID.String(), "success",
		fmt.Sprintf("Created crypto invoice %s for $%.2f USD (%s on %s)", inv.InvoiceCode, inv.FiatAmountUSD, inv.CurrencyID, inv.NetworkID),
		map[string]interface{}{
			"invoice_code":    inv.InvoiceCode,
			"deposit_address": inv.DepositAddress,
			"crypto_amount":   inv.CryptoAmount,
		},
	)

	response.JSON(w, http.StatusCreated, inv, nil)
}

// GetInvoice returns live status, confirmations, and countdown for a crypto invoice.
func (h *CryptoHandler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_CODE", "Invoice code is required", nil, "")
		return
	}

	inv, err := h.cryptoSvc.GetInvoiceByCode(r.Context(), code)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Crypto payment invoice not found", nil, "")
		return
	}

	// Trigger quick on-chain check if still pending/confirming
	if inv.Status == "pending" || inv.Status == "confirming" {
		if checked, checkErr := h.cryptoSvc.VerifyInvoiceOnChain(r.Context(), inv.InvoiceCode, inv.TxHash); checkErr == nil && checked != nil {
			inv = checked
		}
	}

	response.JSON(w, http.StatusOK, inv, nil)
}

type VerifyPaymentReq struct {
	TxHash string `json:"tx_hash,omitempty"`
}

// VerifyInvoice manually triggers an on-chain blockchain scan or binds a customer-submitted transaction hash.
func (h *CryptoHandler) VerifyInvoice(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_CODE", "Invoice code is required", nil, "")
		return
	}

	var req VerifyPaymentReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	updated, err := h.cryptoSvc.VerifyInvoiceOnChain(r.Context(), code, strings.TrimSpace(req.TxHash))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "VERIFICATION_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, updated, nil)
}

// ----------------------------------------------------------------------------
// Super Admin Endpoints (PermAdmin / PermOwner)
// ----------------------------------------------------------------------------

// AdminListInvoices displays all crypto payment invoices with filters.
func (h *CryptoHandler) AdminListInvoices(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	networkID := r.URL.Query().Get("network_id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	invoices, total, err := h.store.ListCryptoInvoices(r.Context(), uuid.Nil, uuid.Nil, status, networkID, limit, offset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve crypto invoices", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"invoices": invoices,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	}, nil)
}

// AdminListTreasuries returns all configured Trust Wallet treasury destinations.
func (h *CryptoHandler) AdminListTreasuries(w http.ResponseWriter, r *http.Request) {
	networkID := r.URL.Query().Get("network_id")
	treasuries, err := h.store.ListCryptoTreasuryWallets(r.Context(), networkID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve treasury wallets", nil, "")
		return
	}
	response.JSON(w, http.StatusOK, treasuries, nil)
}

func (h *CryptoHandler) upsertTreasury(ctx context.Context, networkID, name, address string) error {
	networkID = strings.ToLower(strings.TrimSpace(networkID))
	address = strings.TrimSpace(address)
	if address == "" {
		return nil
	}
	if !crypto.ValidateAddress(networkID, address) {
		return fmt.Errorf("invalid address for network '%s'", networkID)
	}

	existing, _ := h.store.ListCryptoTreasuryWallets(ctx, networkID)
	now := time.Now().UTC()
	if len(existing) > 0 {
		target := existing[0]
		target.Name = name
		target.Address = address
		target.IsActive = true
		target.UpdatedAt = now
		return h.store.UpdateCryptoTreasuryWallet(ctx, target)
	}

	tw := &store.CryptoTreasuryWallet{
		ID:        uuid.New(),
		NetworkID: networkID,
		Name:      name,
		Address:   address,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return h.store.CreateCryptoTreasuryWallet(ctx, tw)
}

// AdminQuickSetupTreasuries configures all Trust Wallet networks in 1 click.
func (h *CryptoHandler) AdminQuickSetupTreasuries(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TronAddress     string `json:"tron_address"`
		EVMAddress      string `json:"evm_address"`
		BitcoinAddress  string `json:"btc_address"`
		SolanaAddress   string `json:"sol_address"`
		LitecoinAddress string `json:"ltc_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON payload", nil, "")
		return
	}

	configuredCount := 0
	if strings.TrimSpace(req.TronAddress) != "" {
		if err := h.upsertTreasury(r.Context(), "tron", "Trust Wallet TRON (TRC20)", req.TronAddress); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_TRON_ADDRESS", "Invalid TRON address format (must start with T and be 34 characters)", nil, "")
			return
		}
		configuredCount++
	}

	if strings.TrimSpace(req.EVMAddress) != "" {
		if !crypto.ValidateAddress("ethereum", strings.TrimSpace(req.EVMAddress)) {
			response.Error(w, http.StatusBadRequest, "INVALID_EVM_ADDRESS", "Invalid Multi-Chain EVM address format (must start with 0x followed by 40 hex characters)", nil, "")
			return
		}
		_ = h.upsertTreasury(r.Context(), "bsc", "Trust Wallet BNB Smart Chain (BEP20)", req.EVMAddress)
		_ = h.upsertTreasury(r.Context(), "ethereum", "Trust Wallet Ethereum (ERC20)", req.EVMAddress)
		_ = h.upsertTreasury(r.Context(), "polygon", "Trust Wallet Polygon (PoS)", req.EVMAddress)
		configuredCount += 3
	}

	if strings.TrimSpace(req.BitcoinAddress) != "" {
		if err := h.upsertTreasury(r.Context(), "bitcoin", "Trust Wallet Bitcoin", req.BitcoinAddress); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_BTC_ADDRESS", "Invalid Bitcoin address format", nil, "")
			return
		}
		configuredCount++
	}

	if strings.TrimSpace(req.SolanaAddress) != "" {
		if err := h.upsertTreasury(r.Context(), "solana", "Trust Wallet Solana", req.SolanaAddress); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_SOL_ADDRESS", "Invalid Solana address format", nil, "")
			return
		}
		configuredCount++
	}

	if strings.TrimSpace(req.LitecoinAddress) != "" {
		if err := h.upsertTreasury(r.Context(), "litecoin", "Trust Wallet Litecoin", req.LitecoinAddress); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_LTC_ADDRESS", "Invalid Litecoin address format", nil, "")
			return
		}
		configuredCount++
	}

	if configuredCount == 0 {
		response.Error(w, http.StatusBadRequest, "NO_ADDRESSES", "Please provide at least one valid wallet address (e.g. TRON or EVM)", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "crypto.treasury.quick_setup", "treasuries", "bulk", "success",
		fmt.Sprintf("Quick setup completed across %d networks", configuredCount), nil)

	treasuries, _ := h.store.ListCryptoTreasuryWallets(r.Context(), "")
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"configured": configuredCount,
		"treasuries": treasuries,
	}, nil)
}

// AdminDeleteTreasury removes an approved treasury destination.
func (h *CryptoHandler) AdminDeleteTreasury(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid treasury ID", nil, "")
		return
	}

	if err := h.store.DeleteCryptoTreasuryWallet(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete treasury wallet", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "crypto.treasury.delete", "treasury_wallet", id.String(), "success", "Deleted crypto treasury", nil)
	response.JSON(w, http.StatusOK, map[string]bool{"success": true}, nil)
}

// AdminSaveTreasury creates or updates an approved Trust Wallet treasury destination.
func (h *CryptoHandler) AdminSaveTreasury(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		store.CryptoTreasuryWallet
		ApplyToAllEVM bool `json:"apply_to_all_evm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON payload", nil, "")
		return
	}

	tw := payload.CryptoTreasuryWallet
	tw.NetworkID = strings.ToLower(strings.TrimSpace(tw.NetworkID))
	tw.Address = strings.TrimSpace(tw.Address)

	if !crypto.ValidateAddress(tw.NetworkID, tw.Address) {
		response.Error(w, http.StatusBadRequest, "INVALID_ADDRESS", fmt.Sprintf("Invalid wallet address for network '%s'", tw.NetworkID), nil, "")
		return
	}

	if tw.ID == uuid.Nil {
		existing, _ := h.store.ListCryptoTreasuryWallets(r.Context(), tw.NetworkID)
		if len(existing) > 0 {
			target := existing[0]
			target.Name = tw.Name
			target.Address = tw.Address
			target.IsActive = true
			target.UpdatedAt = time.Now().UTC()
			_ = h.store.UpdateCryptoTreasuryWallet(r.Context(), target)
			tw = *target
		} else {
			tw.ID = uuid.New()
			tw.IsActive = true
			now := time.Now().UTC()
			tw.CreatedAt = now
			tw.UpdatedAt = now
			if err := h.store.CreateCryptoTreasuryWallet(r.Context(), &tw); err != nil {
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create treasury wallet", nil, "")
				return
			}
		}
	} else {
		tw.UpdatedAt = time.Now().UTC()
		if err := h.store.UpdateCryptoTreasuryWallet(r.Context(), &tw); err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update treasury wallet", nil, "")
			return
		}
	}

	if payload.ApplyToAllEVM && (tw.NetworkID == "bsc" || tw.NetworkID == "ethereum" || tw.NetworkID == "polygon") {
		_ = h.upsertTreasury(r.Context(), "bsc", "Trust Wallet BNB Smart Chain (BEP20)", tw.Address)
		_ = h.upsertTreasury(r.Context(), "ethereum", "Trust Wallet Ethereum (ERC20)", tw.Address)
		_ = h.upsertTreasury(r.Context(), "polygon", "Trust Wallet Polygon (PoS)", tw.Address)
	}

	h.audit.Log(r.Context(), r, "crypto.treasury.save", "treasury_wallet", tw.ID.String(), "success",
		fmt.Sprintf("Saved treasury wallet '%s' (%s) for network %s", tw.Name, tw.Address, tw.NetworkID), nil)

	response.JSON(w, http.StatusOK, tw, nil)
}

// AdminListLedger returns immutable double-entry ledger records.
func (h *CryptoHandler) AdminListLedger(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 50
	}

	entries, total, err := h.store.ListCryptoLedger(r.Context(), limit, offset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve ledger entries", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	}, nil)
}

// AdminGetSettings returns configurable crypto payment settings.
func (h *CryptoHandler) AdminGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.store.GetCryptoSettings(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve settings", nil, "")
		return
	}
	response.JSON(w, http.StatusOK, settings, nil)
}

// AdminUpdateSettings updates crypto payment settings.
func (h *CryptoHandler) AdminUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var payload map[string]string
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON payload", nil, "")
		return
	}

	for k, v := range payload {
		_ = h.store.SetCryptoSetting(r.Context(), k, v)
	}

	h.audit.Log(r.Context(), r, "crypto.settings.update", "crypto_settings", "global", "success", "Updated crypto payment settings", nil)
	response.JSON(w, http.StatusOK, map[string]string{"message": "Settings updated successfully"}, nil)
}
