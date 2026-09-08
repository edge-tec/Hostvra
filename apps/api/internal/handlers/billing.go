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
	Name         string         `json:"name"`
	Slug         string         `json:"slug"`
	Description  string         `json:"description"`
	Tier         store.PlanTier `json:"tier"`
	PriceMonthly float64        `json:"price_monthly"`
	PriceYearly  float64        `json:"price_yearly"`
	Currency     string         `json:"currency"`
	DiskSpaceMB  int64          `json:"disk_space_mb"`
	BandwidthMB  int64          `json:"bandwidth_mb"`
	MaxWebsites  int            `json:"max_websites"`
	MaxDatabases int            `json:"max_databases"`
	MaxMailboxes int            `json:"max_mailboxes"`
	MaxFTP       int            `json:"max_ftp"`
	DedicatedIP  bool           `json:"dedicated_ip"`
	FreeSSL      bool           `json:"free_ssl"`
	Features     []string       `json:"features"`
	IsActive     bool           `json:"is_active"`
	SortOrder    int            `json:"sort_order"`
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

	plan := &store.HostingPlan{
		ID:           uuid.New(),
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  req.Description,
		Tier:         req.Tier,
		PriceMonthly: req.PriceMonthly,
		PriceYearly:  req.PriceYearly,
		Currency:     req.Currency,
		DiskSpaceMB:  req.DiskSpaceMB,
		BandwidthMB:  req.BandwidthMB,
		MaxWebsites:  req.MaxWebsites,
		MaxDatabases: req.MaxDatabases,
		MaxMailboxes: req.MaxMailboxes,
		MaxFTP:       req.MaxFTP,
		DedicatedIP:  req.DedicatedIP,
		FreeSSL:      req.FreeSSL,
		Features:     req.Features,
		IsActive:     req.IsActive,
		SortOrder:    req.SortOrder,
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
	plan.DiskSpaceMB = req.DiskSpaceMB
	plan.BandwidthMB = req.BandwidthMB
	plan.MaxWebsites = req.MaxWebsites
	plan.MaxDatabases = req.MaxDatabases
	plan.MaxMailboxes = req.MaxMailboxes
	plan.MaxFTP = req.MaxFTP
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
	PlanID       string `json:"plan_id"`
	BillingCycle string `json:"billing_cycle"` // monthly, yearly
	PaymentMethod string `json:"payment_method"` // stripe, bkash, nagad, sslcommerz, paypal
	AutoRenew    bool   `json:"auto_renew"`
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

	var req PayInvoiceRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.PaymentMethod == "" {
		req.PaymentMethod = "stripe"
	}
	if req.TransactionID == "" {
		req.TransactionID = fmt.Sprintf("txn_%s_%d", req.PaymentMethod, time.Now().Unix())
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

	h.audit.Log(r.Context(), r, "billing.invoice.pay", "invoice", inv.ID.String(), "success", fmt.Sprintf("Paid invoice %s via %s", inv.InvoiceNumber, inv.PaymentMethod), nil)

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

		// Activate associated subscription if present
		if inv.SubscriptionID != nil {
			if sub, err := h.store.GetSubscriptionByID(r.Context(), *inv.SubscriptionID); err == nil {
				sub.Status = store.SubStatusActive
				_ = h.store.UpdateSubscription(r.Context(), sub)
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

