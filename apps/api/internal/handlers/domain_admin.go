package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/domains"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type DomainAdminHandler struct {
	cfg       *config.Config
	store     store.Store
	audit     *audit.Logger
	domainSvc *domains.Service
}

func NewDomainAdminHandler(cfg *config.Config, s store.Store, a *audit.Logger, svc *domains.Service) *DomainAdminHandler {
	return &DomainAdminHandler{
		cfg:       cfg,
		store:     s,
		audit:     a,
		domainSvc: svc,
	}
}

// ----------------------------------------------------------------------------
// 1. Domains & Orders Overview
// ----------------------------------------------------------------------------

func (h *DomainAdminHandler) ListDomains(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListAllDomains(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, list, &response.Meta{Total: len(list)})
}

func (h *DomainAdminHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.store.ListAllDomainOrders(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, orders, &response.Meta{Total: len(orders)})
}

func (h *DomainAdminHandler) RetryOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid order ID", nil, "")
		return
	}

	order, err := h.store.GetDomainOrderByID(r.Context(), orderID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Order not found", nil, "")
		return
	}

	if order.PaymentStatus != "paid" {
		response.Error(w, http.StatusBadRequest, "PAYMENT_REQUIRED", "Cannot retry provisioning for an unpaid order", nil, "")
		return
	}

	if order.ProvisioningStatus == "completed" {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"message": "Order is already completed",
			"order":   order,
		}, nil)
		return
	}

	// Trigger provisioning worker with decoupled context so HTTP request end doesn't cancel provisioning
	go func() {
		_ = h.domainSvc.Provisioning.ProcessPaidOrder(context.Background(), orderID, nil, nil)
	}()

	h.audit.Log(r.Context(), r, "domains.order.retry", "domain_order", orderID.String(), "success", fmt.Sprintf("Admin retried provisioning for order %s", orderID.String()), nil)

	response.JSON(w, http.StatusAccepted, map[string]interface{}{
		"message":  "Provisioning retry initiated in background",
		"order_id": orderID,
	}, nil)
}

// ----------------------------------------------------------------------------
// 2. Transfers & Renewals
// ----------------------------------------------------------------------------

func (h *DomainAdminHandler) ListTransfers(w http.ResponseWriter, r *http.Request) {
	transfers, err := h.store.ListDomainTransfers(r.Context(), nil)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, transfers, &response.Meta{Total: len(transfers)})
}

func (h *DomainAdminHandler) ListRenewals(w http.ResponseWriter, r *http.Request) {
	renewals, err := h.store.ListDomainRenewals(r.Context(), nil)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, renewals, &response.Meta{Total: len(renewals)})
}

// ----------------------------------------------------------------------------
// 3. TLDs & Pricing Management
// ----------------------------------------------------------------------------

func (h *DomainAdminHandler) ListTLDs(w http.ResponseWriter, r *http.Request) {
	tlds, err := h.store.ListDomainTLDs(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, tlds, &response.Meta{Total: len(tlds)})
}

func (h *DomainAdminHandler) SaveTLD(w http.ResponseWriter, r *http.Request) {
	var req store.DomainTLD
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error(), nil, "")
		return
	}

	cleanTLD := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(req.TLD)), ".")
	if cleanTLD == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "TLD is required", nil, "")
		return
	}
	req.TLD = cleanTLD

	if err := h.store.SaveDomainTLD(r.Context(), &req); err != nil {
		response.Error(w, http.StatusInternalServerError, "SAVE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "domains.admin.tld.save", "domain_tld", cleanTLD, "success", "Saved TLD configuration for "+cleanTLD, nil)

	response.JSON(w, http.StatusOK, req, nil)
}

func (h *DomainAdminHandler) ListPrices(w http.ResponseWriter, r *http.Request) {
	prices, err := h.store.ListDomainPrices(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), nil, "")
		return
	}

	type priceDTO struct {
		store.DomainPrice
		CostPrice     float64 `json:"cost_price"`
		RegisterPrice float64 `json:"register_price"`
		RenewPrice    float64 `json:"renew_price"`
	}

	res := make([]priceDTO, len(prices))
	for i, p := range prices {
		res[i] = priceDTO{
			DomainPrice:   *p,
			CostPrice:     p.RegistrationCost,
			RegisterPrice: p.RegistrationPrice,
			RenewPrice:    p.RenewalPrice,
		}
	}

	response.JSON(w, http.StatusOK, res, &response.Meta{Total: len(res)})
}

func (h *DomainAdminHandler) SavePrice(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", err.Error(), nil, "")
		return
	}

	var raw map[string]interface{}
	_ = json.Unmarshal(bodyBytes, &raw)

	var req store.DomainPrice
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error(), nil, "")
		return
	}

	// Support alternate frontend keys
	if req.RegistrationCost == 0 {
		if cp, ok := raw["cost_price"].(float64); ok {
			req.RegistrationCost = cp
		}
	}
	if req.RegistrationPrice == 0 {
		if rp, ok := raw["register_price"].(float64); ok {
			req.RegistrationPrice = rp
		}
	}
	if req.RenewalPrice == 0 {
		if rp, ok := raw["renew_price"].(float64); ok {
			req.RenewalPrice = rp
		}
	}

	cleanTLD := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(req.TLD)), ".")
	if cleanTLD == "" {
		cleanTLD = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(chi.URLParam(r, "id"))), ".")
	}
	if cleanTLD == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "TLD is required", nil, "")
		return
	}
	req.TLD = cleanTLD

	if err := h.store.SaveDomainPrice(r.Context(), &req); err != nil {
		response.Error(w, http.StatusInternalServerError, "SAVE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "domains.admin.price.save", "domain_price", cleanTLD, "success", fmt.Sprintf("Updated pricing for %s (Selling: $%.2f, Cost: $%.2f)", cleanTLD, req.RegistrationPrice, req.RegistrationCost), nil)

	response.JSON(w, http.StatusOK, req, nil)
}

// ----------------------------------------------------------------------------
// 4. Connectivity Test & Reconciliation
// ----------------------------------------------------------------------------

func (h *DomainAdminHandler) TestRegistrar(w http.ResponseWriter, r *http.Request) {
	result, err := h.domainSvc.Registrar.TestConnection(r.Context())
	if err != nil {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"connected": false,
			"provider":  "resellerclub",
			"error":     err.Error(),
			"message":   "Connection test failed: " + err.Error(),
		}, nil)
		return
	}
	response.JSON(w, http.StatusOK, result, nil)
}

func (h *DomainAdminHandler) Reconcile(w http.ResponseWriter, r *http.Request) {
	syncedCount, err := h.domainSvc.Reconciliation.ReconcileAll(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "RECONCILE_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"synced_domains": syncedCount,
		"message":        fmt.Sprintf("Reconciliation completed. Synchronized %d domain(s).", syncedCount),
	}, nil)
}

// ----------------------------------------------------------------------------
// 5. Accounting & Financial Metrics
// ----------------------------------------------------------------------------

func (h *DomainAdminHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	domainsList, _ := h.store.ListAllDomains(r.Context())
	orders, _ := h.store.ListAllDomainOrders(r.Context())

	var totalRevenue float64
	var totalCost float64
	activeCount := 0
	expiringSoonCount := 0

	now := time.Now().UTC()
	for _, d := range domainsList {
		if d.Status == "active" {
			activeCount++
		}
		if d.ExpiryDate != nil && d.ExpiryDate.After(now) && d.ExpiryDate.Before(now.AddDate(0, 0, 30)) {
			expiringSoonCount++
		}
	}

	for _, o := range orders {
		if o.PaymentStatus == "paid" {
			totalRevenue += o.Amount
			totalCost += o.Cost
		}
	}

	grossProfit := totalRevenue - totalCost
	marginPct := 0.0
	if totalRevenue > 0 {
		marginPct = (grossProfit / totalRevenue) * 100
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"total_domains":   len(domainsList),
		"active_domains":  activeCount,
		"expiring_soon":   expiringSoonCount,
		"total_orders":    len(orders),
		"total_revenue":   totalRevenue,
		"total_cost":      totalCost,
		"gross_profit":    grossProfit,
		"margin_percent":  marginPct,
		"currency":        "USD",
	}, nil)
}

func (h *DomainAdminHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	domainName := r.URL.Query().Get("domain")
	logs, err := h.store.ListDomainAuditLogs(r.Context(), domainName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, logs, &response.Meta{Total: len(logs)})
}
