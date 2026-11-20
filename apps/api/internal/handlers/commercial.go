package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/billing"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type CommercialHandler struct {
	cfg           *config.Config
	store         store.Store
	audit         *audit.Logger
	dunningWorker *billing.DunningWorker
	onInvoicePaid func(ctx context.Context, inv *store.Invoice)
	payMu         sync.Mutex
	startedAt     time.Time
}

func NewCommercialHandler(cfg *config.Config, s store.Store, a *audit.Logger, dw *billing.DunningWorker) *CommercialHandler {
	return &CommercialHandler{
		cfg:           cfg,
		store:         s,
		audit:         a,
		dunningWorker: dw,
		startedAt:     time.Now().UTC(),
	}
}

func (h *CommercialHandler) SetOnInvoicePaid(fn func(ctx context.Context, inv *store.Invoice)) {
	h.onInvoicePaid = fn
}

// ============================================================================
// 1. PUBLIC REAL-TIME SYSTEM STATUS (/api/v1/public/status)
// ============================================================================

func (h *CommercialHandler) GetPublicSystemStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now().UTC()

	type ServiceProbe struct {
		Name      string `json:"name"`
		Category  string `json:"category"`
		Status    string `json:"status"` // operational, degraded, outage
		LatencyMs int64  `json:"latency_ms"`
		Details   string `json:"details"`
	}

	probes := make([]ServiceProbe, 0, 5)

	// 1. API Control Plane
	uptimeSecs := int64(now.Sub(h.startedAt).Seconds())
	probes = append(probes, ServiceProbe{
		Name:      "Hostvra Control Plane API",
		Category:  "Core Infrastructure",
		Status:    "operational",
		LatencyMs: 1,
		Details:   fmt.Sprintf("Live process uptime: %ds (Go %s)", uptimeSecs, runtime.Version()),
	})

	// 2. Database / Persistence Store live query latency
	dbStart := time.Now()
	_, dbErr := h.store.ListPlans(ctx)
	dbLatency := time.Since(dbStart).Milliseconds()
	dbStatus := "operational"
	dbDetails := "Database queries responding normally"
	if dbErr != nil {
		dbStatus = "outage"
		dbDetails = "Database probe failed"
	}
	probes = append(probes, ServiceProbe{
		Name:      "Primary Database & Billing Ledger",
		Category:  "Core Infrastructure",
		Status:    dbStatus,
		LatencyMs: dbLatency,
		Details:   dbDetails,
	})

	// 3. Crypto Payment & HD Treasury Engine
	cStart := time.Now()
	nets, cErr := h.store.ListCryptoNetworks(ctx, true)
	cLatency := time.Since(cStart).Milliseconds()
	cStatus := "operational"
	cDetails := fmt.Sprintf("%d active blockchain networks monitored", len(nets))
	if cErr != nil {
		cStatus = "degraded"
		cDetails = "Unable to query crypto network registry"
	}
	probes = append(probes, ServiceProbe{
		Name:      "In-House Crypto Gateway & HD Wallet",
		Category:  "Payments",
		Status:    cStatus,
		LatencyMs: cLatency,
		Details:   cDetails,
	})

	// 4. Automated Subscription Dunning Worker
	dStatus := "operational"
	dDetails := "Automated renewal & suspension worker active"
	if h.dunningWorker == nil {
		dStatus = "degraded"
		dDetails = "Dunning worker not initialized"
	}
	probes = append(probes, ServiceProbe{
		Name:      "Automated Billing & Renewal Engine",
		Category:  "Billing",
		Status:    dStatus,
		LatencyMs: 0,
		Details:   dDetails,
	})

	overall := "operational"
	for _, p := range probes {
		if p.Status == "outage" {
			overall = "outage"
			break
		}
		if p.Status == "degraded" {
			overall = "degraded"
		}
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"overall_status": overall,
		"checked_at":     now,
		"uptime_seconds": uptimeSecs,
		"memory_mb":      memStats.Alloc / (1024 * 1024),
		"goroutines":     runtime.NumGoroutine(),
		"services":       probes,
	}, nil)
}

// ============================================================================
// 2. COUPON & PROMOTION ENGINE
// ============================================================================

func isCouponApplicableToPlan(applicableCSV string, plan *store.HostingPlan) bool {
	clean := strings.TrimSpace(applicableCSV)
	if clean == "" || plan == nil {
		return true
	}
	for _, token := range strings.Split(clean, ",") {
		t := strings.TrimSpace(token)
		if t == "" {
			continue
		}
		if strings.EqualFold(t, plan.ID.String()) || strings.EqualFold(t, plan.Slug) || strings.EqualFold(t, plan.Name) {
			return true
		}
	}
	return false
}

func (h *CommercialHandler) ValidateCoupon(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code           string  `json:"code"`
		OrderAmountUSD float64 `json:"order_amount_usd"`
		PlanID         string  `json:"plan_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON payload", nil, "")
		return
	}
	if math.IsNaN(req.OrderAmountUSD) || math.IsInf(req.OrderAmountUSD, 0) || req.OrderAmountUSD < 0 {
		response.Error(w, http.StatusBadRequest, "INVALID_AMOUNT", "order_amount_usd must be a non-negative finite number", nil, "")
		return
	}

	c, err := h.store.GetCouponByCode(r.Context(), req.Code)
	if err != nil {
		response.Error(w, http.StatusNotFound, "COUPON_NOT_FOUND", "Coupon code not found", nil, "")
		return
	}

	now := time.Now().UTC()
	if !c.IsActive || (c.ExpiresAt != nil && c.ExpiresAt.Before(now)) {
		response.Error(w, http.StatusBadRequest, "COUPON_EXPIRED", "This coupon is expired or inactive", nil, "")
		return
	}
	if c.MaxUses > 0 && c.CurrentUses >= c.MaxUses {
		response.Error(w, http.StatusBadRequest, "COUPON_EXHAUSTED", "Coupon redemption limit reached", nil, "")
		return
	}
	if req.OrderAmountUSD < c.MinOrderUSD {
		response.Error(w, http.StatusBadRequest, "MIN_ORDER_NOT_MET", fmt.Sprintf("Minimum order amount for %s is $%.2f", c.Code, c.MinOrderUSD), nil, "")
		return
	}
	if strings.TrimSpace(c.ApplicablePlanIDs) != "" && strings.TrimSpace(req.PlanID) != "" {
		var plan *store.HostingPlan
		if pid, pErr := uuid.Parse(strings.TrimSpace(req.PlanID)); pErr == nil {
			plan, _ = h.store.GetPlanByID(r.Context(), pid)
		} else {
			plan, _ = h.store.GetPlanBySlug(r.Context(), strings.TrimSpace(req.PlanID))
		}
		if plan != nil && !isCouponApplicableToPlan(c.ApplicablePlanIDs, plan) {
			response.Error(w, http.StatusBadRequest, "COUPON_PLAN_INELIGIBLE", fmt.Sprintf("Coupon %s is not applicable to the %s plan", c.Code, plan.Name), nil, "")
			return
		}
	}

	var discount float64
	if strings.EqualFold(c.DiscountType, "fixed") {
		discount = math.Min(c.DiscountValue, req.OrderAmountUSD)
	} else {
		discount = math.Round((req.OrderAmountUSD*(c.DiscountValue/100.0))*100) / 100
	}
	finalTotal := math.Max(0, math.Round((req.OrderAmountUSD-discount)*100)/100)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"valid":          true,
		"code":           c.Code,
		"description":    c.Description,
		"discount_type":  c.DiscountType,
		"discount_value": c.DiscountValue,
		"discount_usd":   discount,
		"final_usd":      finalTotal,
	}, nil)
}

func (h *CommercialHandler) AdminListCoupons(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListCoupons(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list coupons", nil, "")
		return
	}
	response.JSON(w, http.StatusOK, list, nil)
}

func validateCouponFields(c *store.Coupon) error {
	if c.DiscountType == "" {
		c.DiscountType = "percentage"
	}
	c.DiscountType = strings.ToLower(strings.TrimSpace(c.DiscountType))
	if c.DiscountType != "percentage" && c.DiscountType != "fixed" {
		return fmt.Errorf("discount_type must be 'percentage' or 'fixed'")
	}
	if math.IsNaN(c.DiscountValue) || math.IsInf(c.DiscountValue, 0) || c.DiscountValue <= 0 {
		return fmt.Errorf("discount_value must be a positive finite number")
	}
	if c.DiscountType == "percentage" && c.DiscountValue > 100 {
		return fmt.Errorf("percentage discount_value cannot exceed 100")
	}
	if math.IsNaN(c.MinOrderUSD) || math.IsInf(c.MinOrderUSD, 0) || c.MinOrderUSD < 0 {
		return fmt.Errorf("min_order_usd cannot be negative")
	}
	if c.MaxUses < 0 {
		return fmt.Errorf("max_uses cannot be negative")
	}
	if c.MaxUsesPerUser <= 0 {
		c.MaxUsesPerUser = 1
	}
	return nil
}

func (h *CommercialHandler) AdminCreateCoupon(w http.ResponseWriter, r *http.Request) {
	var c store.Coupon
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid coupon payload", nil, "")
		return
	}
	if err := validateCouponFields(&c); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_COUPON", err.Error(), nil, "")
		return
	}
	c.IsActive = true
	if err := h.store.CreateCoupon(r.Context(), &c); err != nil {
		response.Error(w, http.StatusBadRequest, "COUPON_CREATE_FAILED", err.Error(), nil, "")
		return
	}
	h.audit.Log(r.Context(), r, "commercial.coupon.create", "coupon", c.Code, "success", fmt.Sprintf("Created coupon %s", c.Code), nil)
	response.JSON(w, http.StatusCreated, c, nil)
}

func (h *CommercialHandler) AdminDeleteCoupon(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid coupon ID", nil, "")
		return
	}
	if err := h.store.DeleteCoupon(r.Context(), id); err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}
	h.audit.Log(r.Context(), r, "commercial.coupon.delete", "coupon", id.String(), "success", "Deleted coupon", nil)
	response.JSON(w, http.StatusOK, map[string]bool{"deleted": true}, nil)
}

func (h *CommercialHandler) AdminUpdateCoupon(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid coupon ID", nil, "")
		return
	}
	var c store.Coupon
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid coupon payload", nil, "")
		return
	}
	c.ID = id
	if err := validateCouponFields(&c); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_COUPON", err.Error(), nil, "")
		return
	}
	if err := h.store.UpdateCoupon(r.Context(), &c); err != nil {
		response.Error(w, http.StatusBadRequest, "COUPON_UPDATE_FAILED", err.Error(), nil, "")
		return
	}
	h.audit.Log(r.Context(), r, "commercial.coupon.update", "coupon", c.Code, "success", fmt.Sprintf("Updated coupon %s (active=%v)", c.Code, c.IsActive), nil)
	response.JSON(w, http.StatusOK, c, nil)
}

// CheckoutOrder creates an authoritative server-side invoice for a hosting plan order,
// atomically validates & redeems any supplied coupon code, and either settles immediately
// via customer credit wallet (triggering provisioning) or returns the invoice for Crypto/Card payment.
func (h *CommercialHandler) CheckoutOrder(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}
	var req struct {
		PlanID        string `json:"plan_id"`
		BillingCycle  string `json:"billing_cycle"` // monthly, yearly
		DomainName    string `json:"domain_name,omitempty"`
		CouponCode    string `json:"coupon_code,omitempty"`
		PaymentMethod string `json:"payment_method"` // crypto, card, credit_wallet
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid checkout payload", nil, "")
		return
	}

	// 1. Authoritative server-side plan lookup (never trust client price)
	var plan *store.HostingPlan
	if pid, err := uuid.Parse(strings.TrimSpace(req.PlanID)); err == nil {
		plan, _ = h.store.GetPlanByID(r.Context(), pid)
	} else if req.PlanID != "" {
		plan, _ = h.store.GetPlanBySlug(r.Context(), strings.TrimSpace(req.PlanID))
	}
	if plan == nil || !plan.IsActive {
		response.Error(w, http.StatusNotFound, "PLAN_NOT_FOUND", "Selected hosting plan not found or inactive", nil, "")
		return
	}

	cycle := strings.ToLower(strings.TrimSpace(req.BillingCycle))
	if cycle != "yearly" {
		cycle = "monthly"
	}
	subtotal := plan.PriceMonthly
	if cycle == "yearly" {
		if plan.PriceYearly > 0 {
			subtotal = plan.PriceYearly
		} else {
			subtotal = math.Round(plan.PriceMonthly*10*100) / 100
		}
	}
	if subtotal < 0 {
		subtotal = 0
	}

	payMethod := strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	if payMethod == "" {
		payMethod = "crypto"
	}

	invoiceID := uuid.New()
	now := time.Now().UTC()

	// Pre-validate credit wallet balance BEFORE burning any single-use coupon or creating an orphan invoice
	if payMethod == "credit_wallet" {
		estTotal := subtotal
		if code := strings.ToUpper(strings.TrimSpace(req.CouponCode)); code != "" {
			if c, err := h.store.GetCouponByCode(r.Context(), code); err == nil && c != nil && c.IsActive {
				if strings.EqualFold(c.DiscountType, "fixed") {
					estTotal = math.Max(0, subtotal-math.Min(c.DiscountValue, subtotal))
				} else {
					estTotal = math.Max(0, math.Round((subtotal-(subtotal*(c.DiscountValue/100.0)))*100)/100)
				}
			}
		}
		if estTotal > 0 {
			wlt, err := h.store.GetOrCreateCustomerWallet(r.Context(), claims.UserID)
			if err != nil || wlt.BalanceUSD+1e-9 < estTotal {
				response.Error(w, http.StatusBadRequest, "INSUFFICIENT_BALANCE", "Insufficient credit wallet balance for this order", nil, "")
				return
			}
		}
	}

	// 2. Authoritative server-side coupon revalidation & atomic redemption
	discountUSD := 0.0
	appliedCoupon := ""
	if code := strings.ToUpper(strings.TrimSpace(req.CouponCode)); code != "" {
		if c, cErr := h.store.GetCouponByCode(r.Context(), code); cErr == nil && c != nil {
			if !isCouponApplicableToPlan(c.ApplicablePlanIDs, plan) {
				response.Error(w, http.StatusBadRequest, "COUPON_PLAN_INELIGIBLE", fmt.Sprintf("Coupon %s is not applicable to the %s plan", c.Code, plan.Name), nil, "")
				return
			}
		}
		isFirstOrder := true
		if existingInvs, err := h.store.ListInvoices(r.Context(), claims.OrganizationID); err == nil {
			for _, ei := range existingInvs {
				if ei.UserID == claims.UserID && ei.Status == store.InvoiceStatusPaid {
					isFirstOrder = false
					break
				}
			}
		}
		red, err := h.store.RedeemCouponAtomic(r.Context(), code, claims.UserID, invoiceID, subtotal, isFirstOrder)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "COUPON_INVALID", err.Error(), nil, "")
			return
		}
		discountUSD = red.DiscountUSD
		appliedCoupon = code
	}

	finalTotal := math.Max(0, math.Round((subtotal-discountUSD)*100)/100)

	desc := fmt.Sprintf("%s (%s)", plan.Name, cycle)
	if dom := strings.TrimSpace(req.DomainName); dom != "" {
		desc += fmt.Sprintf(" — Domain: %s", dom)
	}
	if appliedCoupon != "" {
		desc += fmt.Sprintf(" [Promo %s: -$%.2f]", appliedCoupon, discountUSD)
	}

	currency := plan.Currency
	if currency == "" {
		currency = "USD"
	}

	inv := &store.Invoice{
		ID:             invoiceID,
		InvoiceNumber:  fmt.Sprintf("INV-%d-%s", now.Unix(), strings.ToUpper(invoiceID.String()[:6])),
		UserID:         claims.UserID,
		OrganizationID: claims.OrganizationID,
		PlanID:         plan.ID,
		Subtotal:       subtotal,
		Tax:            0,
		Total:          finalTotal,
		Currency:       currency,
		Status:         store.InvoiceStatusUnpaid,
		DueDate:        now.AddDate(0, 0, 7),
		Description:    desc,
		CreatedAt:      now,
	}
	if err := h.store.CreateInvoice(r.Context(), inv); err != nil {
		response.Error(w, http.StatusInternalServerError, "INVOICE_CREATE_FAILED", "Failed to create order invoice", err.Error(), "")
		return
	}

	// 3. Instant settlement if paid by Credit Wallet or 100%-discounted free order
	if payMethod == "credit_wallet" || finalTotal == 0 {
		txnID := "promo_100pct_" + invoiceID.String()[:8]
		methodLabel := "promo_free"
		if finalTotal > 0 {
			idemKey := fmt.Sprintf("wallet_pay:%s", inv.ID.String())
			entry, err := h.store.DebitWalletAtomic(r.Context(), claims.UserID, finalTotal, "invoice_payment", inv.ID.String(), idemKey, fmt.Sprintf("Order checkout %s", inv.InvoiceNumber))
			if err != nil {
				response.Error(w, http.StatusBadRequest, "INSUFFICIENT_BALANCE", err.Error(), nil, "")
				return
			}
			txnID = entry.ID.String()
			methodLabel = "credit_wallet"
		}
		inv.Status = store.InvoiceStatusPaid
		inv.PaymentMethod = methodLabel
		inv.TransactionID = txnID
		inv.PaidAt = &now
		_ = h.store.UpdateInvoice(r.Context(), inv)

		if h.onInvoicePaid != nil {
			h.onInvoicePaid(r.Context(), inv)
		} else {
			_, _ = h.store.RecordAffiliateCommission(r.Context(), claims.UserID, inv.ID, inv.Total)
		}

		response.JSON(w, http.StatusCreated, map[string]interface{}{
			"invoice":        inv,
			"paid":           true,
			"subtotal_usd":   subtotal,
			"discount_usd":   discountUSD,
			"total_usd":      finalTotal,
			"coupon_applied": appliedCoupon,
			"next_url":       "/billing?tab=invoices&order_paid=true",
		}, nil)
		return
	}

	nextURL := fmt.Sprintf("/billing?tab=invoices&pay_invoice=%s", inv.ID.String())
	if payMethod == "crypto" {
		nextURL = fmt.Sprintf("/billing/crypto/%s", inv.ID.String())
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"invoice":        inv,
		"paid":           false,
		"subtotal_usd":   subtotal,
		"discount_usd":   discountUSD,
		"total_usd":      finalTotal,
		"coupon_applied": appliedCoupon,
		"next_url":       nextURL,
	}, nil)
}

// ============================================================================
// 3. CUSTOMER CREDIT WALLET & AUTO-PAY
// ============================================================================

func (h *CommercialHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}
	wallet, err := h.store.GetOrCreateCustomerWallet(r.Context(), claims.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "WALLET_ERROR", err.Error(), nil, "")
		return
	}
	ledger, _ := h.store.ListWalletLedger(r.Context(), claims.UserID, 50)
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"wallet": wallet,
		"ledger": ledger,
	}, nil)
}

func (h *CommercialHandler) ToggleWalletAutoPay(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}
	var req struct {
		AutoPayEnabled bool `json:"auto_pay_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid payload", nil, "")
		return
	}
	wallet, err := h.store.SetWalletAutoPay(r.Context(), claims.UserID, req.AutoPayEnabled)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "WALLET_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, wallet, nil)
}

func (h *CommercialHandler) PayInvoiceWithWallet(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}
	var req struct {
		InvoiceID string `json:"invoice_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid payload", nil, "")
		return
	}
	invID, err := uuid.Parse(req.InvoiceID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid invoice_id", nil, "")
		return
	}

	// Serialize wallet invoice settlement to prevent concurrent double-provisioning on the same invoice
	h.payMu.Lock()
	defer h.payMu.Unlock()

	inv, err := h.store.GetInvoiceByID(r.Context(), invID)
	if err != nil || inv == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Invoice not found", nil, "")
		return
	}
	if inv.UserID != claims.UserID {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cannot pay another user's invoice", nil, "")
		return
	}
	if inv.Status == store.InvoiceStatusPaid {
		response.JSON(w, http.StatusOK, inv, nil)
		return
	}
	if inv.Status == store.InvoiceStatusCancelled || inv.Status == store.InvoiceStatusRefunded {
		response.Error(w, http.StatusBadRequest, "INVALID_INVOICE_STATUS", fmt.Sprintf("Cannot pay a %s invoice", inv.Status), nil, "")
		return
	}

	var entry *store.WalletLedgerEntry
	txnID := "wallet_free_" + inv.ID.String()[:8]
	if inv.Total > 0 {
		idemKey := fmt.Sprintf("wallet_pay:%s", inv.ID.String())
		entry, err = h.store.DebitWalletAtomic(r.Context(), claims.UserID, inv.Total, "invoice_payment", inv.ID.String(), idemKey, fmt.Sprintf("Paid invoice %s", inv.InvoiceNumber))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INSUFFICIENT_BALANCE", err.Error(), nil, "")
			return
		}
		txnID = entry.ID.String()
	}

	now := time.Now().UTC()
	inv.Status = store.InvoiceStatusPaid
	inv.PaymentMethod = "credit_wallet"
	inv.TransactionID = txnID
	inv.PaidAt = &now
	_ = h.store.UpdateInvoice(r.Context(), inv)

	// Activate/provision hosting & record affiliate commission if user was referred
	if h.onInvoicePaid != nil {
		h.onInvoicePaid(r.Context(), inv)
	} else {
		_, _ = h.store.RecordAffiliateCommission(r.Context(), claims.UserID, inv.ID, inv.Total)
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"invoice":      inv,
		"ledger_entry": entry,
	}, nil)
}

func (h *CommercialHandler) AdminAdjustWallet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID    string  `json:"user_id"`
		Direction string  `json:"direction"` // credit, debit
		AmountUSD float64 `json:"amount_usd"`
		Notes     string  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid payload", nil, "")
		return
	}
	uid, err := uuid.Parse(req.UserID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user_id", nil, "")
		return
	}
	if math.IsNaN(req.AmountUSD) || math.IsInf(req.AmountUSD, 0) || req.AmountUSD <= 0 {
		response.Error(w, http.StatusBadRequest, "INVALID_AMOUNT", "amount_usd must be a positive finite number", nil, "")
		return
	}
	dir := strings.ToLower(strings.TrimSpace(req.Direction))
	if dir != "credit" && dir != "debit" {
		response.Error(w, http.StatusBadRequest, "INVALID_DIRECTION", "direction must be 'credit' or 'debit'", nil, "")
		return
	}
	idemKey := fmt.Sprintf("admin_adj:%s:%d", uid.String(), time.Now().UnixNano())
	var entry *store.WalletLedgerEntry
	if dir == "debit" {
		entry, err = h.store.DebitWalletAtomic(r.Context(), uid, req.AmountUSD, "admin_adjustment", "admin", idemKey, req.Notes)
	} else {
		entry, err = h.store.CreditWalletAtomic(r.Context(), uid, req.AmountUSD, "admin_adjustment", "admin", idemKey, req.Notes)
	}
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ADJUSTMENT_FAILED", err.Error(), nil, "")
		return
	}
	h.audit.Log(r.Context(), r, "commercial.wallet.admin_adjust", "customer_wallet", uid.String(), "success", fmt.Sprintf("Admin %s $%.2f: %s", req.Direction, req.AmountUSD, req.Notes), nil)
	response.JSON(w, http.StatusOK, entry, nil)
}

// ============================================================================
// 4. AFFILIATE & REFERRAL SYSTEM
// ============================================================================

func (h *CommercialHandler) GetAffiliateDashboard(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}
	acc, err := h.store.GetOrCreateAffiliateAccount(r.Context(), claims.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "AFFILIATE_ERROR", err.Error(), nil, "")
		return
	}
	withdrawals, _ := h.store.ListAffiliateWithdrawals(r.Context(), claims.UserID)
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"account":       acc,
		"referral_link": fmt.Sprintf("https://hostvra.com/register?ref=%s", acc.ReferralCode),
		"withdrawals":   withdrawals,
	}, nil)
}

func (h *CommercialHandler) RequestAffiliateWithdrawal(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}
	var req struct {
		AmountUSD     float64 `json:"amount_usd"`
		PayoutMethod  string  `json:"payout_method"`
		PayoutDetails string  `json:"payout_details"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid payload", nil, "")
		return
	}
	if math.IsNaN(req.AmountUSD) || math.IsInf(req.AmountUSD, 0) || req.AmountUSD <= 0 {
		response.Error(w, http.StatusBadRequest, "INVALID_AMOUNT", "amount_usd must be a positive finite number", nil, "")
		return
	}
	method := strings.ToLower(strings.TrimSpace(req.PayoutMethod))
	if method == "" {
		method = "credit_wallet"
	}
	switch method {
	case "credit_wallet", "usdt_bep20", "usdt_trc20", "bank_transfer", "paypal":
	default:
		response.Error(w, http.StatusBadRequest, "INVALID_PAYOUT_METHOD", "Unsupported payout method", nil, "")
		return
	}
	if method != "credit_wallet" && strings.TrimSpace(req.PayoutDetails) == "" {
		response.Error(w, http.StatusBadRequest, "PAYOUT_DETAILS_REQUIRED", "Receiving wallet address or payout details are required", nil, "")
		return
	}
	wd, err := h.store.RequestAffiliateWithdrawal(r.Context(), claims.UserID, req.AmountUSD, method, strings.TrimSpace(req.PayoutDetails))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "WITHDRAWAL_FAILED", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusCreated, wd, nil)
}

func (h *CommercialHandler) AdminListAffiliates(w http.ResponseWriter, r *http.Request) {
	accounts, _ := h.store.ListAffiliateAccounts(r.Context())
	withdrawals, _ := h.store.ListAffiliateWithdrawals(r.Context(), uuid.Nil)
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"accounts":    accounts,
		"withdrawals": withdrawals,
	}, nil)
}

func (h *CommercialHandler) AdminProcessWithdrawal(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid withdrawal ID", nil, "")
		return
	}
	var req struct {
		Approve    bool   `json:"approve"`
		AdminNotes string `json:"admin_notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid payload", nil, "")
		return
	}
	wd, err := h.store.ApproveAffiliateWithdrawal(r.Context(), id, req.Approve, req.AdminNotes)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "PROCESS_FAILED", err.Error(), nil, "")
		return
	}
	// If approved and transitioned to paid for credit_wallet, atomically credit customer wallet
	if req.Approve && wd.Status == "paid" && strings.EqualFold(wd.PayoutMethod, "credit_wallet") {
		_, _ = h.store.CreditWalletAtomic(r.Context(), wd.UserID, wd.AmountUSD, "affiliate_payout", wd.ID.String(), "aff_payout:"+wd.ID.String(), "Affiliate commission payout to credit wallet")
	}
	h.audit.Log(r.Context(), r, "commercial.affiliate.withdrawal", "affiliate_withdrawal", id.String(), "success", fmt.Sprintf("Withdrawal approved=%v", req.Approve), nil)
	response.JSON(w, http.StatusOK, wd, nil)
}

// ============================================================================
// 5. ADMIN DUNNING TRIGGER
// ============================================================================

func (h *CommercialHandler) AdminRunDunningPass(w http.ResponseWriter, r *http.Request) {
	if h.dunningWorker == nil {
		response.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Billing dunning worker not initialized", nil, "")
		return
	}
	summary, err := h.dunningWorker.RunPass(r.Context(), time.Now().UTC())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DUNNING_ERROR", err.Error(), nil, "")
		return
	}
	h.audit.Log(r.Context(), r, "billing.dunning.manual_run", "billing", "dunning", "success", fmt.Sprintf("Dunning pass: renewals=%d autopaid=%d suspended=%d", summary.RenewalsGenerated, summary.AutoPaidFromWallet, summary.SuspendedOverdue), nil)
	response.JSON(w, http.StatusOK, summary, nil)
}
