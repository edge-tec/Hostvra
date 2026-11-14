package handlers

import (
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

// AdminSaveTreasury creates or updates an approved Trust Wallet treasury destination.
func (h *CryptoHandler) AdminSaveTreasury(w http.ResponseWriter, r *http.Request) {
	var tw store.CryptoTreasuryWallet
	if err := json.NewDecoder(r.Body).Decode(&tw); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON payload", nil, "")
		return
	}

	if !crypto.ValidateAddress(tw.NetworkID, tw.Address) {
		response.Error(w, http.StatusBadRequest, "INVALID_ADDRESS", fmt.Sprintf("Invalid wallet address for network '%s'", tw.NetworkID), nil, "")
		return
	}

	if tw.ID == uuid.Nil {
		tw.ID = uuid.New()
		tw.IsActive = true
		now := time.Now().UTC()
		tw.CreatedAt = now
		tw.UpdatedAt = now
		if err := h.store.CreateCryptoTreasuryWallet(r.Context(), &tw); err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create treasury wallet", nil, "")
			return
		}
	} else {
		tw.UpdatedAt = time.Now().UTC()
		if err := h.store.UpdateCryptoTreasuryWallet(r.Context(), &tw); err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update treasury wallet", nil, "")
			return
		}
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
