package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/domains"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type BillingHandler struct {
	cfg       *config.Config
	store     store.Store
	audit     *audit.Logger
	domainSvc *domains.Service
}

func NewBillingHandler(cfg *config.Config, s store.Store, a *audit.Logger) *BillingHandler {
	return &BillingHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
}

func (h *BillingHandler) SetDomainService(svc *domains.Service) {
	h.domainSvc = svc
}

// ----------------------------------------------------------------------------
// Plans API
// ----------------------------------------------------------------------------

func (h *BillingHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.store.ListPlans(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve plans", err.Error(), "")
		return
	}

	includeInactive := r.URL.Query().Get("all") == "true"
	var filtered []*store.HostingPlan
	for _, p := range plans {
		if includeInactive || p.IsActive {
			filtered = append(filtered, p)
		}
	}

	response.JSON(w, http.StatusOK, filtered, &response.Meta{Total: len(filtered)})
}

func (h *BillingHandler) GetPlan(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "id")
	if parsedID, err := uuid.Parse(idOrSlug); err == nil {
		plan, err := h.store.GetPlanByID(r.Context(), parsedID)
		if err != nil {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "Hosting plan not found", nil, "")
			return
		}
		response.JSON(w, http.StatusOK, plan, nil)
		return
	}

	plan, err := h.store.GetPlanBySlug(r.Context(), idOrSlug)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Hosting plan not found", nil, "")
		return
	}
	response.JSON(w, http.StatusOK, plan, nil)
}

type CreatePlanRequest struct {
	Name          string         `json:"name"`
	Slug          string         `json:"slug"`
	Description   string         `json:"description"`
	Tier          store.PlanTier `json:"tier"`
	PriceMonthly  float64        `json:"price_monthly"`
	PriceYearly   float64        `json:"price_yearly"`
	Currency      string         `json:"currency"`
	SetupFee      float64        `json:"setup_fee"`
	TrialAllowed  bool           `json:"trial_allowed"`
	TrialDays     int            `json:"trial_days"`
	IsFeatured    bool           `json:"is_featured"`
	CPULimit      float64        `json:"cpu_limit"`
	RAMLimitMB    int            `json:"ram_limit_mb"`
	DiskSpaceMB   int64          `json:"disk_space_mb"`
	BandwidthMB   int64          `json:"bandwidth_mb"`
	MaxWebsites   int            `json:"max_websites"`
	MaxDatabases  int            `json:"max_databases"`
	MaxMailboxes  int            `json:"max_mailboxes"`
	MaxFTP        int            `json:"max_ftp"`
	MaxCron       int            `json:"max_cron"`
	MaxSubdomains int            `json:"max_subdomains"`
	DedicatedIP   bool           `json:"dedicated_ip"`
	FreeSSL       bool           `json:"free_ssl"`
	Features      []string       `json:"features"`
	IsActive      bool           `json:"is_active"`
	SortOrder     int            `json:"sort_order"`
}

func (h *BillingHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var req CreatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Plan name is required", nil, "")
		return
	}
	if req.Slug == "" {
		req.Slug = strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	}
	if req.Tier == "" {
		req.Tier = store.PlanTierStarter
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.TrialDays <= 0 && req.TrialAllowed {
		req.TrialDays = 14
	}

	plan := &store.HostingPlan{
		ID:            uuid.New(),
		Name:          req.Name,
		Slug:          req.Slug,
		Description:   req.Description,
		Tier:          req.Tier,
		PriceMonthly:  req.PriceMonthly,
		PriceYearly:   req.PriceYearly,
		Currency:      req.Currency,
		SetupFee:      req.SetupFee,
		TrialAllowed:  req.TrialAllowed,
		TrialDays:     req.TrialDays,
		IsFeatured:    req.IsFeatured,
		CPULimit:      req.CPULimit,
		RAMLimitMB:    req.RAMLimitMB,
		DiskSpaceMB:   req.DiskSpaceMB,
		BandwidthMB:   req.BandwidthMB,
		MaxWebsites:   req.MaxWebsites,
		MaxDatabases:  req.MaxDatabases,
		MaxMailboxes:  req.MaxMailboxes,
		MaxFTP:        req.MaxFTP,
		MaxCron:       req.MaxCron,
		MaxSubdomains: req.MaxSubdomains,
		DedicatedIP:   req.DedicatedIP,
		FreeSSL:       req.FreeSSL,
		Features:      req.Features,
		IsActive:      req.IsActive,
		SortOrder:     req.SortOrder,
	}

	if err := h.store.CreatePlan(r.Context(), plan); err != nil {
		response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create plan", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "billing.plan.create", "hosting_plan", plan.ID.String(), "success", "Created hosting plan "+plan.Name, nil)

	response.JSON(w, http.StatusCreated, plan, nil)
}

func (h *BillingHandler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	planID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid plan ID", nil, "")
		return
	}

	plan, err := h.store.GetPlanByID(r.Context(), planID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Plan not found", nil, "")
		return
	}

	var req CreatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if req.Name != "" {
		plan.Name = req.Name
	}
	if req.Slug != "" {
		plan.Slug = req.Slug
	}
	plan.Description = req.Description
	if req.Tier != "" {
		plan.Tier = req.Tier
	}
	plan.PriceMonthly = req.PriceMonthly
	plan.PriceYearly = req.PriceYearly
	if req.Currency != "" {
		plan.Currency = req.Currency
	}
	plan.SetupFee = req.SetupFee
	plan.TrialAllowed = req.TrialAllowed
	if req.TrialDays > 0 {
		plan.TrialDays = req.TrialDays
	}
	plan.IsFeatured = req.IsFeatured
	if req.CPULimit > 0 {
		plan.CPULimit = req.CPULimit
	}
	if req.RAMLimitMB > 0 {
		plan.RAMLimitMB = req.RAMLimitMB
	}
	plan.DiskSpaceMB = req.DiskSpaceMB
	plan.BandwidthMB = req.BandwidthMB
	plan.MaxWebsites = req.MaxWebsites
	plan.MaxDatabases = req.MaxDatabases
	plan.MaxMailboxes = req.MaxMailboxes
	plan.MaxFTP = req.MaxFTP
	plan.MaxCron = req.MaxCron
	plan.MaxSubdomains = req.MaxSubdomains
	plan.DedicatedIP = req.DedicatedIP
	plan.FreeSSL = req.FreeSSL
	if req.Features != nil {
		plan.Features = req.Features
	}
	plan.IsActive = req.IsActive
	plan.SortOrder = req.SortOrder

	if err := h.store.UpdatePlan(r.Context(), plan); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update plan", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "billing.plan.update", "hosting_plan", plan.ID.String(), "success", "Updated plan "+plan.Name, nil)

	response.JSON(w, http.StatusOK, plan, nil)
}

func (h *BillingHandler) DeletePlan(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	planID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid plan ID", nil, "")
		return
	}

	if err := h.store.DeletePlan(r.Context(), planID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete plan", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "billing.plan.delete", "hosting_plan", idStr, "success", "Deleted plan "+idStr, nil)

	response.JSON(w, http.StatusOK, map[string]string{"message": "Plan deleted successfully"}, nil)
}

// ----------------------------------------------------------------------------
// Subscriptions API
// ----------------------------------------------------------------------------

func (h *BillingHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	subs, err := h.store.ListSubscriptions(r.Context(), claims.OrganizationID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve subscriptions", err.Error(), "")
		return
	}
	response.JSON(w, http.StatusOK, subs, &response.Meta{Total: len(subs)})
}

func (h *BillingHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	subID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid subscription ID", nil, "")
		return
	}

	sub, err := h.store.GetSubscriptionByID(r.Context(), subID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Subscription not found", nil, "")
		return
	}
	response.JSON(w, http.StatusOK, sub, nil)
}

type CreateSubscriptionRequest struct {
	PlanID        string `json:"plan_id"`
	BillingCycle  string `json:"billing_cycle"` // monthly, yearly
	PaymentMethod string `json:"payment_method"` // stripe, bkash, nagad, sslcommerz, paypal
	AutoRenew     bool   `json:"auto_renew"`
	StartTrial    bool   `json:"start_trial"`
}

func (h *BillingHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var req CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	planUUID, err := uuid.Parse(req.PlanID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PLAN_ID", "Invalid plan ID", nil, "")
		return
	}

	plan, err := h.store.GetPlanByID(r.Context(), planUUID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "PLAN_NOT_FOUND", "Selected hosting plan does not exist", nil, "")
		return
	}

	if req.BillingCycle == "" {
		req.BillingCycle = "monthly"
	}

	amount := plan.PriceMonthly
	nextBilling := time.Now().UTC().AddDate(0, 1, 0)
	if req.BillingCycle == "yearly" {
		amount = plan.PriceYearly
		nextBilling = time.Now().UTC().AddDate(1, 0, 0)
	}

	// ------------------------------------------------------------------------
	// Free Trial Activation Flow
	// ------------------------------------------------------------------------
	if req.StartTrial {
		trialSettings, _ := h.store.GetTrialSettings(r.Context())
		if trialSettings != nil && !trialSettings.Enabled {
			response.Error(w, http.StatusBadRequest, "TRIALS_DISABLED", "Free trials are currently disabled by administrator", nil, "")
			return
		}
		if !plan.TrialAllowed {
			response.Error(w, http.StatusBadRequest, "PLAN_TRIAL_NOT_ALLOWED", "Selected plan is not eligible for free trial", nil, "")
			return
		}

		if trialSettings != nil && trialSettings.OneTrialPerCustomer {
			// Anti-abuse: verify user has not previously claimed a trial
			existingSubs, err := h.store.ListSubscriptions(r.Context(), claims.OrganizationID)
			if err == nil {
				for _, es := range existingSubs {
					if es.Status == store.SubStatusTrial || es.TrialEndsAt != nil {
						response.Error(w, http.StatusBadRequest, "TRIAL_ALREADY_USED", "A free trial has already been claimed for this account", nil, "")
						return
					}
				}
			}
		}

		trialDays := plan.TrialDays
		if trialDays <= 0 && trialSettings != nil && trialSettings.DefaultDays > 0 {
			trialDays = trialSettings.DefaultDays
		}
		if trialDays <= 0 {
			trialDays = 14
		}

		now := time.Now().UTC()
		ends := now.AddDate(0, 0, trialDays)
		subID := uuid.New()
		sub := &store.Subscription{
			ID:              subID,
			UserID:          claims.UserID,
			OrganizationID:  claims.OrganizationID,
			PlanID:          plan.ID,
			PlanName:        plan.Name,
			Status:          store.SubStatusTrial,
			BillingCycle:    req.BillingCycle,
			Amount:          amount,
			Currency:        plan.Currency,
			DiskUsedMB:      100,
			BandwidthUsedMB: 50,
			WebsitesCount:   0,
			NextBillingDate: ends,
			TrialStartedAt:  &now,
			TrialEndsAt:     &ends,
			AutoRenew:       req.AutoRenew,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if err := h.store.CreateSubscription(r.Context(), sub); err != nil {
			response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to activate free trial", err.Error(), "")
			return
		}

		// Create $0.00 Trial Invoice marked Paid
		inv := &store.Invoice{
			ID:             uuid.New(),
			InvoiceNumber:  fmt.Sprintf("INV-TRL-%d-%05d", now.Year(), rand.Intn(90000)+10000),
			UserID:         claims.UserID,
			SubscriptionID: &subID,
			PlanID:         plan.ID,
			Description:    fmt.Sprintf("%s (%d-Day Free Trial)", plan.Name, trialDays),
			Subtotal:       0.0,
			Tax:            0.0,
			Discount:       amount,
			Total:          0.0,
			Currency:       plan.Currency,
			Status:         store.InvoiceStatusPaid,
			PaymentMethod:  "free_trial",
			TransactionID:  fmt.Sprintf("trial_%s_%d", subID.String()[:8], now.Unix()),
			DueDate:        ends,
			PaidAt:         &now,
			CreatedAt:      now,
		}
		_ = h.store.CreateInvoice(r.Context(), inv)

		h.audit.Log(r.Context(), r, "billing.trial.start", "subscription", sub.ID.String(), "success", fmt.Sprintf("Started %d-day free trial on %s", trialDays, plan.Name), nil)

		response.JSON(w, http.StatusCreated, map[string]interface{}{
			"subscription":    sub,
			"invoice":         inv,
			"trial_activated": true,
			"trial_days":      trialDays,
			"expires_at":      ends,
			"message":         fmt.Sprintf("Congratulations! Your %d-day free trial for %s has been activated.", trialDays, plan.Name),
		}, nil)
		return
	}

	subID := uuid.New()
	sub := &store.Subscription{
		ID:              subID,
		UserID:          claims.UserID,
		OrganizationID:  claims.OrganizationID,
		PlanID:          plan.ID,
		PlanName:        plan.Name,
		Status:          store.SubStatusActive,
		BillingCycle:    req.BillingCycle,
		Amount:          amount,
		Currency:        plan.Currency,
		DiskUsedMB:      100, // starting baseline
		BandwidthUsedMB: 50,
		WebsitesCount:   0,
		NextBillingDate: nextBilling,
		AutoRenew:       req.AutoRenew,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if err := h.store.CreateSubscription(r.Context(), sub); err != nil {
		response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create subscription", err.Error(), "")
		return
	}

	// Create Corresponding Initial Invoice
	now := time.Now().UTC()
	paidAt := now
	inv := &store.Invoice{
		ID:             uuid.New(),
		InvoiceNumber:  fmt.Sprintf("INV-%d-%05d", now.Year(), rand.Intn(90000)+10000),
		UserID:         claims.UserID,
		SubscriptionID: &subID,
		PlanID:         plan.ID,
		Description:    fmt.Sprintf("%s (%s Cycle)", plan.Name, strings.Title(req.BillingCycle)),
		Subtotal:       amount,
		Tax:            0.0,
		Discount:       0.0,
		Total:          amount,
		Currency:       plan.Currency,
		Status:         store.InvoiceStatusPaid,
		PaymentMethod:  req.PaymentMethod,
		TransactionID:  fmt.Sprintf("txn_%s_%d", req.PaymentMethod, time.Now().UnixNano()),
		DueDate:        now.AddDate(0, 0, 7),
		PaidAt:         &paidAt,
		CreatedAt:      now,
	}
	_ = h.store.CreateInvoice(r.Context(), inv)

	h.audit.Log(r.Context(), r, "billing.subscription.create", "subscription", sub.ID.String(), "success", "Subscribed to plan "+plan.Name, nil)

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"subscription": sub,
		"invoice":      inv,
		"message":      "Hosting package subscription activated successfully!",
	}, nil)
}

func (h *BillingHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	subID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid subscription ID", nil, "")
		return
	}

	sub, err := h.store.GetSubscriptionByID(r.Context(), subID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Subscription not found", nil, "")
		return
	}

	sub.Status = store.SubStatusCancelled
	sub.AutoRenew = false
	if err := h.store.UpdateSubscription(r.Context(), sub); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to cancel subscription", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "billing.subscription.cancel", "subscription", sub.ID.String(), "success", "Cancelled subscription "+sub.PlanName, nil)

	response.JSON(w, http.StatusOK, sub, nil)
}

func (h *BillingHandler) RenewSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	subID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid subscription ID", nil, "")
		return
	}

	sub, err := h.store.GetSubscriptionByID(r.Context(), subID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Subscription not found", nil, "")
		return
	}

	// Extend next billing date
	if sub.BillingCycle == "yearly" {
		sub.NextBillingDate = sub.NextBillingDate.AddDate(1, 0, 0)
	} else {
		sub.NextBillingDate = sub.NextBillingDate.AddDate(0, 1, 0)
	}
	sub.Status = store.SubStatusActive

	if err := h.store.UpdateSubscription(r.Context(), sub); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to renew subscription", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "billing.subscription.renew", "subscription", sub.ID.String(), "success", "Renewed subscription "+sub.PlanName, nil)

	response.JSON(w, http.StatusOK, sub, nil)
}

// ----------------------------------------------------------------------------
// Invoices API
// ----------------------------------------------------------------------------

func (h *BillingHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	invoices, err := h.store.ListInvoices(r.Context(), claims.OrganizationID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve invoices", err.Error(), "")
		return
	}
	response.JSON(w, http.StatusOK, invoices, &response.Meta{Total: len(invoices)})
}

func (h *BillingHandler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	invID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid invoice ID", nil, "")
		return
	}

	inv, err := h.store.GetInvoiceByID(r.Context(), invID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Invoice not found", nil, "")
		return
	}
	response.JSON(w, http.StatusOK, inv, nil)
}

type PayInvoiceRequest struct {
	PaymentMethod string `json:"payment_method"` // stripe, bkash, nagad, sslcommerz, paypal
	TransactionID string `json:"transaction_id"`
}

func (h *BillingHandler) PayInvoice(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	invID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid invoice ID", nil, "")
		return
	}

	inv, err := h.store.GetInvoiceByID(r.Context(), invID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Invoice not found", nil, "")
		return
	}

	if inv.Status == store.InvoiceStatusPaid {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"message": "Invoice has already been paid",
			"invoice": inv,
		}, nil)
		return
	}

	claims, hasClaims := auth.GetClaims(r.Context())
	isAdmin := hasClaims && (claims.IsSuperAdmin || claims.Role == "owner" || claims.Role == "admin")

	var req PayInvoiceRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if !isAdmin {
		// Non-admins must provide an external verified transaction ID from the payment gateway
		cleanTxn := strings.TrimSpace(req.TransactionID)
		if cleanTxn == "" {
			response.Error(w, http.StatusPaymentRequired, "PAYMENT_REQUIRED", "Manual payment confirmation requires administrator authorization or a verified gateway transaction ID", nil, "")
			return
		}
		if req.PaymentMethod == "" {
			req.PaymentMethod = "gateway"
		}
	} else {
		if req.PaymentMethod == "" {
			req.PaymentMethod = "manual_admin"
		}
		if req.TransactionID == "" {
			req.TransactionID = fmt.Sprintf("txn_admin_%d", time.Now().Unix())
		}
	}

	now := time.Now().UTC()
	inv.Status = store.InvoiceStatusPaid
	inv.PaymentMethod = req.PaymentMethod
	inv.TransactionID = req.TransactionID
	inv.PaidAt = &now

	if err := h.store.UpdateInvoice(r.Context(), inv); err != nil {
		response.Error(w, http.StatusInternalServerError, "PAY_FAILED", "Failed to update invoice", err.Error(), "")
		return
	}

	// If linked to subscription, make sure subscription is Active
	if inv.SubscriptionID != nil {
		if sub, err := h.store.GetSubscriptionByID(r.Context(), *inv.SubscriptionID); err == nil {
			sub.Status = store.SubStatusActive
			_ = h.store.UpdateSubscription(r.Context(), sub)
		}
	}

	// If linked to domain order, provision it
	if h.domainSvc != nil {
		if domainOrder, err := h.store.GetDomainOrderByInvoiceID(r.Context(), inv.ID); err == nil && domainOrder != nil {
			if domainOrder.PaymentStatus != "paid" {
				domainOrder.PaymentStatus = "paid"
				_ = h.store.UpdateDomainOrder(r.Context(), domainOrder)
				go func(orderID uuid.UUID) {
					_ = h.domainSvc.Provisioning.ProcessPaidOrder(context.Background(), orderID, nil, nil)
				}(domainOrder.ID)
			}
		}
	}

	h.audit.Log(r.Context(), r, "billing.invoice.pay", "invoice", inv.ID.String(), "success", fmt.Sprintf("Paid invoice %s via %s (txn: %s)", inv.InvoiceNumber, inv.PaymentMethod, inv.TransactionID), map[string]interface{}{
		"invoice_number": inv.InvoiceNumber,
		"payment_method": inv.PaymentMethod,
		"transaction_id": inv.TransactionID,
		"is_admin":       isAdmin,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "Invoice paid successfully",
		"invoice": inv,
	}, nil)
}

// ----------------------------------------------------------------------------
// Payment Gateways API
// ----------------------------------------------------------------------------

func (h *BillingHandler) ListGateways(w http.ResponseWriter, r *http.Request) {
	gateways, err := h.store.ListGateways(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve gateways", err.Error(), "")
		return
	}

	// Mask secret keys for security
	var safeList []*store.PaymentGatewayConfig
	for _, g := range gateways {
		copyG := *g
		if copyG.SecretKey != "" {
			if len(copyG.SecretKey) > 8 {
				copyG.SecretKey = copyG.SecretKey[:4] + "••••••••" + copyG.SecretKey[len(copyG.SecretKey)-4:]
			} else {
				copyG.SecretKey = "••••••••"
			}
		}
		safeList = append(safeList, &copyG)
	}

	response.JSON(w, http.StatusOK, safeList, nil)
}

func (h *BillingHandler) UpdateGateway(w http.ResponseWriter, r *http.Request) {
	gatewayName := chi.URLParam(r, "gateway")
	if gatewayName == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PARAM", "Gateway name is required", nil, "")
		return
	}

	var req store.PaymentGatewayConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	existing, err := h.store.GetGatewayConfig(r.Context(), gatewayName)
	if err != nil {
		existing = &store.PaymentGatewayConfig{
			Gateway: gatewayName,
		}
	}

	if req.DisplayName != "" {
		existing.DisplayName = req.DisplayName
	}
	existing.Enabled = req.Enabled
	existing.TestMode = req.TestMode
	if req.ApiKey != "" {
		existing.ApiKey = req.ApiKey
	}
	if req.SecretKey != "" && !strings.Contains(req.SecretKey, "••••") {
		existing.SecretKey = req.SecretKey
	}
	if req.MerchantID != "" {
		existing.MerchantID = req.MerchantID
	}

	if err := h.store.SaveGatewayConfig(r.Context(), existing); err != nil {
		response.Error(w, http.StatusInternalServerError, "SAVE_FAILED", "Failed to update gateway", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "billing.gateway.update", "payment_gateway", gatewayName, "success", "Updated gateway "+gatewayName, nil)

	response.JSON(w, http.StatusOK, existing, nil)
}

// ----------------------------------------------------------------------------
// Payment Webhook Receiver API
// ----------------------------------------------------------------------------

type WebhookPayload struct {
	Event         string  `json:"event"`
	InvoiceID     string  `json:"invoice_id"`
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"` // paid, success, failed, refunded
	Signature     string  `json:"signature"`
}

func (h *BillingHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	gatewayName := strings.ToLower(chi.URLParam(r, "gateway"))
	if gatewayName == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_GATEWAY", "Gateway parameter required", nil, "")
		return
	}

	cfg, err := h.store.GetGatewayConfig(r.Context(), gatewayName)
	if err != nil || cfg == nil {
		response.Error(w, http.StatusNotFound, "GATEWAY_NOT_FOUND", "Gateway not found", nil, "")
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Failed to read request body", nil, "")
		return
	}

	var payload WebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", nil, "")
		return
	}

	// Signature verification (HMAC verification, Stripe protocol, or provider token check)
	reqSig := r.Header.Get("X-Signature")
	if reqSig == "" {
		reqSig = r.Header.Get("Stripe-Signature")
	}
	if reqSig == "" {
		reqSig = payload.Signature
	}

	if cfg.SecretKey != "" && !cfg.TestMode {
		if reqSig == "" {
			h.audit.Log(r.Context(), r, "billing.webhook.reject", "webhook", gatewayName, "failure", "Missing webhook signature", nil)
			response.Error(w, http.StatusUnauthorized, "MISSING_SIGNATURE", "Webhook signature is required", nil, "")
			return
		}

		mac := hmac.New(sha256.New, []byte(cfg.SecretKey))
		mac.Write(bodyBytes)
		expectedSig := hex.EncodeToString(mac.Sum(nil))

		var sigValid bool
		if hmac.Equal([]byte(reqSig), []byte(expectedSig)) {
			sigValid = true
		} else if strings.Contains(reqSig, "v1=") {
			// Official Stripe webhook signature scheme: t=timestamp,v1=signature
			var v1Sig, timestamp string
			for _, part := range strings.Split(reqSig, ",") {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "v1=") {
					v1Sig = strings.TrimPrefix(part, "v1=")
				} else if strings.HasPrefix(part, "t=") {
					timestamp = strings.TrimPrefix(part, "t=")
				}
			}

			if v1Sig != "" {
				if hmac.Equal([]byte(v1Sig), []byte(expectedSig)) {
					sigValid = true
				} else if timestamp != "" {
					stripeMac := hmac.New(sha256.New, []byte(cfg.SecretKey))
					stripeMac.Write([]byte(timestamp + "."))
					stripeMac.Write(bodyBytes)
					stripeExpected := hex.EncodeToString(stripeMac.Sum(nil))
					if hmac.Equal([]byte(v1Sig), []byte(stripeExpected)) {
						sigValid = true
					}
				}
			}
		}

		if !sigValid {
			h.audit.Log(r.Context(), r, "billing.webhook.reject", "webhook", gatewayName, "failure", "Invalid HMAC signature", nil)
			response.Error(w, http.StatusUnauthorized, "INVALID_SIGNATURE", "Webhook signature verification failed", nil, "")
			return
		}
	}

	// Webhook replay attack check
	eventID := payload.TransactionID
	if eventID == "" {
		hSum := sha256.Sum256(bodyBytes)
		eventID = hex.EncodeToString(hSum[:])
	}
	if existingWebhook, _ := h.store.GetDomainWebhook(r.Context(), gatewayName, eventID); existingWebhook != nil {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"status":  "idempotent_duplicate",
			"message": "Webhook event was already processed",
		}, nil)
		return
	}

	if payload.InvoiceID == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_INVOICE", "invoice_id is required", nil, "")
		return
	}

	invUUID, err := uuid.Parse(payload.InvoiceID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_INVOICE_ID", "Invalid invoice ID format", nil, "")
		return
	}

	inv, err := h.store.GetInvoiceByID(r.Context(), invUUID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Invoice not found", nil, "")
		return
	}

	// Verify Payment Amount
	if payload.Amount > 0 && math.Abs(payload.Amount-inv.Total) > 0.01 {
		response.Error(w, http.StatusBadRequest, "INVALID_AMOUNT", fmt.Sprintf("Payment amount mismatch: expected %.2f, received %.2f", inv.Total, payload.Amount), nil, "")
		return
	}

	// Verify Payment Currency
	if payload.Currency != "" && !strings.EqualFold(payload.Currency, inv.Currency) {
		response.Error(w, http.StatusBadRequest, "INVALID_CURRENCY", fmt.Sprintf("Payment currency mismatch: expected %s, received %s", inv.Currency, payload.Currency), nil, "")
		return
	}

	// Idempotency: If invoice is already paid with same transaction ID, return 200 OK immediately
	if inv.Status == store.InvoiceStatusPaid {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"status":  "idempotent_success",
			"message": "Invoice was already paid",
			"invoice": inv,
		}, nil)
		return
	}

	now := time.Now().UTC()
	statusLower := strings.ToLower(payload.Status)
	if statusLower == "paid" || statusLower == "success" || statusLower == "completed" {
		inv.Status = store.InvoiceStatusPaid
		inv.PaidAt = &now
		inv.PaymentMethod = gatewayName
		if payload.TransactionID != "" {
			inv.TransactionID = payload.TransactionID
		} else {
			inv.TransactionID = fmt.Sprintf("txn_%s_%d", gatewayName, now.Unix())
		}

		if err := h.store.UpdateInvoice(r.Context(), inv); err != nil {
			response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update invoice", err.Error(), "")
			return
		}

		// Activate associated subscription if present and advance billing period
		if inv.SubscriptionID != nil {
			if sub, err := h.store.GetSubscriptionByID(r.Context(), *inv.SubscriptionID); err == nil {
				sub.Status = store.SubStatusActive
				if sub.NextBillingDate.Before(now) {
					if sub.BillingCycle == "yearly" {
						sub.NextBillingDate = now.AddDate(1, 0, 0)
					} else {
						sub.NextBillingDate = now.AddDate(0, 1, 0)
					}
				}
				_ = h.store.UpdateSubscription(r.Context(), sub)

				// Automatically unsuspend user's hosting accounts if suspended for billing
				if accounts, err := h.store.ListHostingAccounts(r.Context(), sub.OrganizationID, nil); err == nil {
					for _, acc := range accounts {
						if acc.UserID == inv.UserID && acc.Status == "suspended" {
							acc.Status = "active"
							_ = h.store.UpdateHostingAccount(r.Context(), acc)
						}
					}
				}
			}
		}

		// Record Webhook to prevent replays
		_ = h.store.RecordDomainWebhook(r.Context(), &store.DomainWebhook{
			ID:              uuid.New(),
			Provider:        gatewayName,
			EventType:       "payment",
			ExternalEventID: eventID,
			Payload:         string(bodyBytes),
			Status:          "processed",
			ProcessedAt:     now,
			CreatedAt:       now,
		})

		// Provision associated domain order if present (Direct O(1) lookup via index)
		if domainOrder, err := h.store.GetDomainOrderByInvoiceID(r.Context(), inv.ID); err == nil && domainOrder != nil {
			if domainOrder.PaymentStatus != "paid" {
				domainOrder.PaymentStatus = "paid"
				_ = h.store.UpdateDomainOrder(r.Context(), domainOrder)
				if h.domainSvc != nil {
					go func(orderID uuid.UUID) {
						_ = h.domainSvc.Provisioning.ProcessPaidOrder(context.Background(), orderID, nil, nil)
					}(domainOrder.ID)
				}
			}
		} else {
			// Fallback scanning
			orders, _ := h.store.ListAllDomainOrders(r.Context())
			for _, o := range orders {
				if o.InvoiceID != nil && *o.InvoiceID == inv.ID && o.PaymentStatus != "paid" {
					o.PaymentStatus = "paid"
					_ = h.store.UpdateDomainOrder(r.Context(), o)
					if h.domainSvc != nil {
						go func(orderID uuid.UUID) {
							_ = h.domainSvc.Provisioning.ProcessPaidOrder(context.Background(), orderID, nil, nil)
						}(o.ID)
					}
				}
			}
		}

		h.audit.Log(r.Context(), r, "billing.webhook.paid", "invoice", inv.ID.String(), "success", fmt.Sprintf("Processed %s webhook payment for %s", gatewayName, inv.InvoiceNumber), map[string]interface{}{
			"gateway":        gatewayName,
			"transaction_id": inv.TransactionID,
			"amount":         inv.Total,
		})

		response.JSON(w, http.StatusOK, map[string]interface{}{
			"status":         "success",
			"message":        "Payment verified and invoice marked paid",
			"invoice_number": inv.InvoiceNumber,
			"transaction_id": inv.TransactionID,
		}, nil)
		return
	} else if statusLower == "failed" || statusLower == "cancelled" {
		inv.Status = store.InvoiceStatusUnpaid
		_ = h.store.UpdateInvoice(r.Context(), inv)
		h.audit.Log(r.Context(), r, "billing.webhook.failed", "invoice", inv.ID.String(), "failure", fmt.Sprintf("Payment failed via %s", gatewayName), nil)
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"status":  "payment_failed",
			"message": "Payment marked as failed",
		}, nil)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ignored",
		"message": "Unhandled event status: " + payload.Status,
	}, nil)
}

// ----------------------------------------------------------------------------
// Trial Management & Settings API
// ----------------------------------------------------------------------------

func (h *BillingHandler) GetTrialSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.store.GetTrialSettings(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve trial settings", err.Error(), "")
		return
	}
	response.JSON(w, http.StatusOK, settings, nil)
}

func (h *BillingHandler) UpdateTrialSettings(w http.ResponseWriter, r *http.Request) {
	var req store.TrialSettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if req.DefaultDays <= 0 {
		req.DefaultDays = 14
	}

	if err := h.store.SaveTrialSettings(r.Context(), &req); err != nil {
		response.Error(w, http.StatusInternalServerError, "SAVE_FAILED", "Failed to save trial settings", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "billing.trial_settings.update", "trial_settings", "global", "success", "Updated trial settings", nil)

	response.JSON(w, http.StatusOK, req, nil)
}

func (h *BillingHandler) ListTrials(w http.ResponseWriter, r *http.Request) {
	trials, err := h.store.ListTrials(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list trials", err.Error(), "")
		return
	}

	now := time.Now().UTC()
	activeCount := 0
	expiredCount := 0
	convertedCount := 0

	for _, t := range trials {
		if t.Status == store.SubStatusActive {
			convertedCount++
		} else if t.Status == store.SubStatusTrial {
			if t.TrialEndsAt != nil && t.TrialEndsAt.Before(now) {
				expiredCount++
			} else {
				activeCount++
			}
		} else if t.Status == store.SubStatusExpired {
			expiredCount++
		}
	}

	conversionRate := 0.0
	if len(trials) > 0 {
		conversionRate = (float64(convertedCount) / float64(len(trials))) * 100.0
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"trials": trials,
		"metrics": map[string]interface{}{
			"total_trials":     len(trials),
			"active_trials":    activeCount,
			"expired_trials":   expiredCount,
			"converted_trials": convertedCount,
			"conversion_rate":  conversionRate,
		},
	}, &response.Meta{Total: len(trials)})
}

type ExtendTrialRequest struct {
	Days int `json:"days"`
}

func (h *BillingHandler) ExtendTrial(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	subID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid subscription ID", nil, "")
		return
	}

	sub, err := h.store.GetSubscriptionByID(r.Context(), subID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Subscription not found", nil, "")
		return
	}

	var req ExtendTrialRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Days <= 0 {
		req.Days = 7
	}

	baseTime := time.Now().UTC()
	if sub.TrialEndsAt != nil && sub.TrialEndsAt.After(baseTime) {
		baseTime = *sub.TrialEndsAt
	}
	newEnds := baseTime.AddDate(0, 0, req.Days)
	sub.TrialEndsAt = &newEnds
	sub.NextBillingDate = newEnds
	sub.Status = store.SubStatusTrial

	if err := h.store.UpdateSubscription(r.Context(), sub); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to extend trial", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "billing.trial.extend", "subscription", sub.ID.String(), "success", fmt.Sprintf("Extended trial by %d days", req.Days), nil)

	response.JSON(w, http.StatusOK, sub, nil)
}

func (h *BillingHandler) EndTrial(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	subID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid subscription ID", nil, "")
		return
	}

	sub, err := h.store.GetSubscriptionByID(r.Context(), subID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Subscription not found", nil, "")
		return
	}

	now := time.Now().UTC()
	sub.TrialEndsAt = &now
	sub.Status = store.SubStatusExpired

	if err := h.store.UpdateSubscription(r.Context(), sub); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to end trial", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "billing.trial.end", "subscription", sub.ID.String(), "success", "Ended trial early", nil)

	response.JSON(w, http.StatusOK, sub, nil)
}

func (h *BillingHandler) ConvertTrial(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	subID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid subscription ID", nil, "")
		return
	}

	sub, err := h.store.GetSubscriptionByID(r.Context(), subID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Subscription not found", nil, "")
		return
	}

	plan, err := h.store.GetPlanByID(r.Context(), sub.PlanID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "PLAN_NOT_FOUND", "Associated plan not found", nil, "")
		return
	}

	now := time.Now().UTC()
	sub.Status = store.SubStatusActive
	if sub.BillingCycle == "yearly" {
		sub.NextBillingDate = now.AddDate(1, 0, 0)
	} else {
		sub.NextBillingDate = now.AddDate(0, 1, 0)
	}

	if err := h.store.UpdateSubscription(r.Context(), sub); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to convert trial", err.Error(), "")
		return
	}

	// Create first paid invoice
	amount := plan.PriceMonthly
	if sub.BillingCycle == "yearly" {
		amount = plan.PriceYearly
	}
	inv := &store.Invoice{
		ID:             uuid.New(),
		InvoiceNumber:  fmt.Sprintf("INV-%d-%05d", now.Year(), rand.Intn(90000)+10000),
		UserID:         sub.UserID,
		SubscriptionID: &sub.ID,
		PlanID:         plan.ID,
		Description:    fmt.Sprintf("%s (%s Cycle Conversion from Trial)", plan.Name, strings.Title(sub.BillingCycle)),
		Subtotal:       amount,
		Tax:            0.0,
		Discount:       0.0,
		Total:          amount,
		Currency:       plan.Currency,
		Status:         store.InvoiceStatusPaid,
		PaymentMethod:  "admin_conversion",
		TransactionID:  fmt.Sprintf("conv_%s_%d", sub.ID.String()[:8], now.Unix()),
		DueDate:        now.AddDate(0, 0, 7),
		PaidAt:         &now,
		CreatedAt:      now,
	}
	_ = h.store.CreateInvoice(r.Context(), inv)

	h.audit.Log(r.Context(), r, "billing.trial.convert", "subscription", sub.ID.String(), "success", "Converted trial to paid subscription", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"subscription": sub,
		"invoice":      inv,
		"message":      "Trial successfully converted to active paid subscription",
	}, nil)
}

// ----------------------------------------------------------------------------
// Unified Checkout Session API
// ----------------------------------------------------------------------------

type CheckoutSessionRequest struct {
	PlanID        string `json:"plan_id"`
	BillingCycle  string `json:"billing_cycle"` // monthly, yearly
	PaymentMethod string `json:"payment_method"` // stripe, bkash, nagad, paypal
	StartTrial    bool   `json:"start_trial"`
	SuccessURL    string `json:"success_url"`
	CancelURL     string `json:"cancel_url"`
}

func (h *BillingHandler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var req CheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	planUUID, err := uuid.Parse(req.PlanID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PLAN_ID", "Invalid plan ID", nil, "")
		return
	}

	plan, err := h.store.GetPlanByID(r.Context(), planUUID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "PLAN_NOT_FOUND", "Plan not found", nil, "")
		return
	}

	if req.BillingCycle == "" {
		req.BillingCycle = "monthly"
	}

	amount := plan.PriceMonthly
	if req.BillingCycle == "yearly" {
		amount = plan.PriceYearly
	}

	// If Free Trial requested
	if req.StartTrial {
		subReq := CreateSubscriptionRequest{
			PlanID:        req.PlanID,
			BillingCycle:  req.BillingCycle,
			PaymentMethod: "free_trial",
			AutoRenew:     true,
			StartTrial:    true,
		}
		jsonBody, _ := json.Marshal(subReq)
		r.Body = io.NopCloser(strings.NewReader(string(jsonBody)))
		h.CreateSubscription(w, r)
		return
	}

	// Create Pending Subscription & Invoice
	subID := uuid.New()
	now := time.Now().UTC()
	nextBilling := now.AddDate(0, 1, 0)
	if req.BillingCycle == "yearly" {
		nextBilling = now.AddDate(1, 0, 0)
	}

	sub := &store.Subscription{
		ID:              subID,
		UserID:          claims.UserID,
		OrganizationID:  claims.OrganizationID,
		PlanID:          plan.ID,
		PlanName:        plan.Name,
		Status:          store.SubStatusPending,
		BillingCycle:    req.BillingCycle,
		Amount:          amount,
		Currency:        plan.Currency,
		DiskUsedMB:      0,
		BandwidthUsedMB: 0,
		WebsitesCount:   0,
		NextBillingDate: nextBilling,
		AutoRenew:       true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_ = h.store.CreateSubscription(r.Context(), sub)

	inv := &store.Invoice{
		ID:             uuid.New(),
		InvoiceNumber:  fmt.Sprintf("INV-%d-%05d", now.Year(), rand.Intn(90000)+10000),
		UserID:         claims.UserID,
		SubscriptionID: &subID,
		PlanID:         plan.ID,
		Description:    fmt.Sprintf("%s (%s Subscription)", plan.Name, strings.Title(req.BillingCycle)),
		Subtotal:       amount,
		Tax:            0.0,
		Discount:       0.0,
		Total:          amount,
		Currency:       plan.Currency,
		Status:         store.InvoiceStatusUnpaid,
		PaymentMethod:  req.PaymentMethod,
		DueDate:        now.AddDate(0, 0, 3),
		CreatedAt:      now,
	}
	_ = h.store.CreateInvoice(r.Context(), inv)

	h.audit.Log(r.Context(), r, "billing.checkout.create", "invoice", inv.ID.String(), "success", fmt.Sprintf("Created checkout session for %s", plan.Name), nil)

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"subscription_id": sub.ID,
		"invoice_id":      inv.ID,
		"invoice_number":  inv.InvoiceNumber,
		"amount":          amount,
		"currency":        plan.Currency,
		"plan_name":       plan.Name,
		"billing_cycle":   req.BillingCycle,
		"payment_method":  req.PaymentMethod,
		"status":          "pending",
		"checkout_url":    fmt.Sprintf("/billing?invoice_id=%s&pay=true", inv.ID.String()),
	}, nil)
}


