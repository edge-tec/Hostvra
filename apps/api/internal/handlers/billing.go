package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/domains"
	"hostvra/api/internal/email"
	"hostvra/api/internal/payment"
	"hostvra/api/internal/quota"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type BillingHandler struct {
	cfg        *config.Config
	store      store.Store
	audit      *audit.Logger
	domainSvc  *domains.Service
	quotaSvc   *quota.Service
	paymentSvc *payment.Service
	emailSvc   *email.Service
}

func NewBillingHandler(cfg *config.Config, s store.Store, a *audit.Logger) *BillingHandler {
	return &BillingHandler{
		cfg:        cfg,
		store:      s,
		audit:      a,
		paymentSvc: payment.NewService(),
		quotaSvc:   quota.NewService(s),
	}
}

func (h *BillingHandler) SetDomainService(svc *domains.Service) {
	h.domainSvc = svc
}

func (h *BillingHandler) SetQuotaService(svc *quota.Service) {
	h.quotaSvc = svc
}

func (h *BillingHandler) SetPaymentService(svc *payment.Service) {
	h.paymentSvc = svc
}

func (h *BillingHandler) SetEmailService(svc *email.Service) {
	h.emailSvc = svc
}

func (h *BillingHandler) notifyInvoicePaid(ctx context.Context, inv *store.Invoice) {
	if h.emailSvc == nil {
		return
	}
	go func() {
		bgCtx := context.Background()
		user, err := h.store.GetUserByID(bgCtx, inv.UserID)
		if err != nil || user == nil {
			return
		}
		planName := inv.Description
		if plan, err := h.store.GetPlanByID(bgCtx, inv.PlanID); err == nil && plan != nil {
			planName = plan.Name
		}
		_ = h.emailSvc.SendPaymentConfirmation(
			bgCtx,
			user.Email,
			user.FullName,
			inv.InvoiceNumber,
			fmt.Sprintf("%.2f", inv.Total),
			inv.Currency,
			planName,
		)
		if inv.SubscriptionID != nil {
			if sub, err := h.store.GetSubscriptionByID(bgCtx, *inv.SubscriptionID); err == nil && sub != nil {
				_ = h.emailSvc.SendOrderConfirmation(
					bgCtx,
					user.Email,
					user.FullName,
					planName,
					sub.BillingCycle,
				)
			}
		}
	}()
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

	claims, hasClaims := auth.GetClaims(r.Context())
	isAdmin := hasClaims && (claims.IsSuperAdmin || claims.Role == "superadmin" || claims.Role == "owner" || claims.Role == "admin")
	if sub.OrganizationID != claims.OrganizationID && !isAdmin {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Subscription not found", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, sub, nil)
}

var slugRegex = regexp.MustCompile(`[^a-z0-9]`)

func sanitizeSlug(input string, maxLen int) string {
	s := strings.ToLower(strings.TrimSpace(input))
	s = slugRegex.ReplaceAllString(s, "")
	if maxLen > 0 && len(s) > maxLen {
		s = s[:maxLen]
	}
	return s
}

func randomAlphanumeric(length int) string {
	const chars = "abcdefghjkmnpqrstuvwxyz23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func (h *BillingHandler) provisionUserForPlan(ctx context.Context, userID, orgID uuid.UUID, plan *store.HostingPlan, sub *store.Subscription) {
	if plan == nil {
		return
	}

	// 1. Synchronize WHM Hosting Accounts if present
	accounts, err := h.store.ListHostingAccounts(ctx, orgID, nil)
	hasExistingAccount := false
	if err == nil {
		for _, acc := range accounts {
			if acc.UserID == userID || (orgID != uuid.Nil && acc.OrganizationID == orgID) {
				hasExistingAccount = true
				acc.PlanID = plan.ID
				acc.PlanName = plan.Name
				acc.DiskLimitMB = plan.DiskSpaceMB
				acc.BandwidthLimitMB = plan.BandwidthMB
				acc.WebsitesLimit = plan.MaxWebsites
				acc.DatabasesLimit = plan.MaxDatabases
				acc.MailboxesLimit = plan.MaxMailboxes
				wasSuspended := acc.Status == store.AccountStatusSuspended || acc.Status == "suspended"
				if wasSuspended {
					acc.Status = store.AccountStatusActive
					acc.SuspendReason = ""
					acc.SuspendedAt = nil
					_ = h.store.UpdateHostingAccount(ctx, acc)

					// Unsuspend corresponding website
					if sites, sErr := h.store.ListAllWebsites(ctx); sErr == nil {
						for _, site := range sites {
							if strings.EqualFold(site.PrimaryDomain, acc.Domain) && (site.Status == "suspended" || site.Status == "disabled") {
								site.Status = "active"
								_ = h.store.UpdateWebsite(ctx, site)
							}
						}
					}

					// Notify account unsuspended
					if h.emailSvc != nil {
						if u, uErr := h.store.GetUserByID(ctx, userID); uErr == nil && u != nil {
							_ = h.emailSvc.SendAccountUnsuspended(ctx, u.Email, u.FullName, acc.Domain)
						}
					}
				} else {
					_ = h.store.UpdateHostingAccount(ctx, acc)
				}
			}
		}
	}

	// 2. If user does NOT have any hosting accounts, execute ZERO-TOUCH AUTO-PROVISIONING!
	if !hasExistingAccount {
		h.autoProvisionAccount(ctx, userID, orgID, plan, sub)
	}
}

func (h *BillingHandler) autoProvisionAccount(ctx context.Context, userID, orgID uuid.UUID, plan *store.HostingPlan, sub *store.Subscription) {
	// 1. Resolve Target Domain
	domain := ""
	if sub != nil && sub.Domain != "" {
		domain = sub.Domain
	}
	if domain == "" {
		if u, err := h.store.GetUserByID(ctx, userID); err == nil && u != nil {
			userSlug := sanitizeSlug(strings.Split(u.Email, "@")[0], 12)
			if userSlug == "" {
				userSlug = sanitizeSlug(u.FullName, 12)
			}
			if userSlug != "" {
				domain = userSlug + ".hostvra.site"
			}
		}
	}
	if domain == "" {
		domain = fmt.Sprintf("site-%s.hostvra.site", randomAlphanumeric(6))
	}
	domain = strings.ToLower(strings.TrimSpace(domain))

	// 2. Resolve Server
	var serverID uuid.UUID
	var serverIP string = "127.0.0.1"
	var serverName string = "Local Cloud Node"

	if sub != nil && sub.ServerID != nil && *sub.ServerID != uuid.Nil {
		if s, err := h.store.GetServerByID(ctx, *sub.ServerID); err == nil && s != nil {
			serverID = s.ID
			if s.IPAddress != "" {
				serverIP = s.IPAddress
			}
			if s.Name != "" {
				serverName = s.Name
			}
		}
	}

	if serverID == uuid.Nil {
		if servers, err := h.store.ListServersByOrg(ctx, orgID); err == nil && len(servers) > 0 {
			serverID = servers[0].ID
			if servers[0].IPAddress != "" {
				serverIP = servers[0].IPAddress
			}
			if servers[0].Name != "" {
				serverName = servers[0].Name
			}
		} else if allServers, err := h.store.ListServersByOrg(ctx, uuid.Nil); err == nil && len(allServers) > 0 {
			serverID = allServers[0].ID
			if allServers[0].IPAddress != "" {
				serverIP = allServers[0].IPAddress
			}
			if allServers[0].Name != "" {
				serverName = allServers[0].Name
			}
		} else {
			defaultServerID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
			if s, err := h.store.GetServerByID(ctx, defaultServerID); err == nil && s != nil {
				serverID = s.ID
			} else {
				_ = h.store.CreateServer(ctx, &store.Server{
					ID:             defaultServerID,
					OrganizationID: orgID,
					Name:           serverName,
					Hostname:       "localhost",
					IPAddress:      serverIP,
					Status:         "online",
					OSName:         "Linux",
					CreatedAt:      time.Now().UTC(),
					UpdatedAt:      time.Now().UTC(),
				})
				serverID = defaultServerID
			}
		}
	}

	// 3. Generate Linux / SFTP System User and Random Password
	cleanSlug := sanitizeSlug(strings.Split(domain, ".")[0], 10)
	if cleanSlug == "" {
		cleanSlug = "user"
	}
	cleanUser := fmt.Sprintf("c_%s%s", cleanSlug, randomAlphanumeric(3))
	if len(cleanUser) > 16 {
		cleanUser = cleanUser[:16]
	}
	cleanPass := randomAlphanumeric(14)
	docRoot := fmt.Sprintf("/home/%s/public_html", cleanUser)
	phpVer := "8.3"

	var subID *uuid.UUID
	if sub != nil {
		subID = &sub.ID
	}

	// 4. Create HostingAccount
	acc := &store.HostingAccount{
		ID:               uuid.New(),
		OrganizationID:   orgID,
		UserID:           userID,
		SubscriptionID:   subID,
		ServerID:         &serverID,
		ServerName:       serverName,
		Domain:           domain,
		Username:         cleanUser,
		DocumentRoot:     docRoot,
		PlanID:           plan.ID,
		PlanName:         plan.Name,
		Status:           store.AccountStatusActive,
		DiskLimitMB:      plan.DiskSpaceMB,
		DiskUsedMB:       120, // initial welcome scaffold
		BandwidthLimitMB: plan.BandwidthMB,
		BandwidthUsedMB:  0,
		WebsitesLimit:    plan.MaxWebsites,
		DatabasesLimit:   plan.MaxDatabases,
		MailboxesLimit:   plan.MaxMailboxes,
		IPAddress:        serverIP,
		PHPVersion:       phpVer,
		SSLActive:        plan.FreeSSL,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	_ = h.store.CreateHostingAccount(ctx, acc)

	// 5. Create Website (Nginx/Apache VHost)
	_ = h.store.CreateWebsite(ctx, &store.Website{
		ID:             uuid.New(),
		OrganizationID: orgID,
		ServerID:       serverID,
		PrimaryDomain:  domain,
		DocumentRoot:   docRoot,
		SystemUser:     cleanUser,
		PHPVersion:     &phpVer,
		WebServerType:  "nginx",
		AppType:        "php",
		Status:         "active",
		SSLEnabled:     plan.FreeSSL,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	})

	// 6. Create Initial Database if allowed
	var dbName, dbUser, dbPass string
	if plan.MaxDatabases > 0 {
		dbName = fmt.Sprintf("db_%s", sanitizeSlug(cleanSlug, 8))
		dbUser = fmt.Sprintf("u_%s", sanitizeSlug(cleanSlug, 8))
		dbPass = randomAlphanumeric(14)

		_ = h.store.CreateDatabase(ctx, &store.Database{
			ID:             uuid.New(),
			OrganizationID: orgID,
			ServerID:       serverID,
			DBType:         "mysql",
			Name:           dbName,
			Username:       dbUser,
			Password:       dbPass,
			CharacterSet:   "utf8mb4",
			Collation:      "utf8mb4_unicode_ci",
			CreatedAt:      time.Now().UTC(),
		})
	}

	// 7. Scaffold welcome index.html safely
	if docRoot != "" && strings.HasPrefix(docRoot, "/home/") {
		_ = os.MkdirAll(docRoot, 0755)
		welcomeFile := filepath.Join(docRoot, "index.html")
		if _, err := os.Stat(welcomeFile); os.IsNotExist(err) {
			welcomeHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Welcome to %s</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; text-align: center; padding: 60px 20px; background: #0f172a; color: #f8fafc; }
        .box { max-width: 600px; margin: 0 auto; background: #1e293b; padding: 40px; border-radius: 16px; border: 1px solid #334155; }
        h1 { color: #38bdf8; font-size: 28px; margin-bottom: 12px; }
        p { color: #94a3b8; font-size: 16px; line-height: 1.6; }
        .badge { display: inline-block; background: #0284c7; color: white; padding: 6px 14px; border-radius: 20px; font-weight: 600; font-size: 13px; margin-top: 10px; }
    </style>
</head>
<body>
    <div class="box">
        <h1>🚀 Welcome to %s</h1>
        <p>Your web hosting account has been successfully provisioned by Hostvra Cloud.</p>
        <div class="badge">PHP %s • SSL Ready • Nginx FastCGI</div>
    </div>
</body>
</html>`, domain, domain, phpVer)
			_ = os.WriteFile(welcomeFile, []byte(welcomeHTML), 0644)
		}
	}

	// 8. Send cPanel-style welcome email with credentials
	if h.emailSvc != nil {
		go func() {
			bgCtx := context.Background()
			u, err := h.store.GetUserByID(bgCtx, userID)
			if err != nil || u == nil {
				return
			}
			ctrlPanel := "https://cp.hostvra.com"
			if h.cfg != nil && h.cfg.AppURL != "" {
				ctrlPanel = h.cfg.AppURL
			}
			provDetails := email.HostingProvisionDetails{
				Domain:       domain,
				PlanName:     plan.Name,
				Username:     cleanUser,
				Password:     cleanPass,
				ControlPanel: ctrlPanel,
				ServerIP:     serverIP,
				Nameservers:  []string{"ns1.hostvra.com", "ns2.hostvra.com"},
				DiskSpaceMB:  plan.DiskSpaceMB,
				BandwidthMB:  plan.BandwidthMB,
				PHPVersion:   phpVer,
				DBName:       dbName,
				DBUser:       dbUser,
				DBPassword:   dbPass,
			}
			_ = h.emailSvc.SendHostingProvisioned(bgCtx, u.Email, u.FullName, provDetails)
		}()
	}
}

type CreateSubscriptionRequest struct {
	PlanID        string `json:"plan_id"`
	BillingCycle  string `json:"billing_cycle"` // monthly, yearly
	PaymentMethod string `json:"payment_method"` // stripe, bkash, nagad
	AutoRenew     bool   `json:"auto_renew"`
	StartTrial    bool   `json:"start_trial"`
	Domain        string `json:"domain"`
	ServerID      string `json:"server_id"`
	SuccessURL    string `json:"success_url"`
	CancelURL     string `json:"cancel_url"`
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

	if !plan.IsActive {
		response.Error(w, http.StatusBadRequest, "PLAN_INACTIVE", "Selected hosting plan is currently not available for purchase", nil, "")
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

		if trialSettings == nil || trialSettings.OneTrialPerCustomer {
			// Anti-abuse: verify user has not previously claimed a trial
			existingSubs, err := h.store.ListSubscriptions(r.Context(), claims.OrganizationID)
			if err == nil {
				for _, es := range existingSubs {
					if es.Status == store.SubStatusTrial || es.TrialStartedAt != nil || es.TrialEndsAt != nil {
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

		var srvUUID *uuid.UUID
		if req.ServerID != "" {
			if parsed, err := uuid.Parse(req.ServerID); err == nil && parsed != uuid.Nil {
				srvUUID = &parsed
			}
		}
		cleanDomain := strings.ToLower(strings.TrimSpace(req.Domain))

		now := time.Now().UTC()
		ends := now.AddDate(0, 0, trialDays)
		subID := uuid.New()
		sub := &store.Subscription{
			ID:              subID,
			UserID:          claims.UserID,
			OrganizationID:  claims.OrganizationID,
			PlanID:          plan.ID,
			PlanName:        plan.Name,
			ServerID:        srvUUID,
			Domain:          cleanDomain,
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
			OrganizationID: claims.OrganizationID,
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

		// Provision user profile and limits directly from plan
		h.provisionUserForPlan(r.Context(), claims.UserID, claims.OrganizationID, plan, sub)

		// Record trial transaction
		_ = h.store.RecordPaymentTransaction(r.Context(), &store.PaymentTransaction{
			ID:             uuid.New(),
			UserID:         claims.UserID,
			OrganizationID: claims.OrganizationID,
			PackageID:      plan.ID,
			SubscriptionID: &subID,
			InvoiceID:      inv.ID,
			Gateway:        "free_trial",
			TransactionID:  inv.TransactionID,
			OrderID:        inv.InvoiceNumber,
			Amount:         0.0,
			Currency:       plan.Currency,
			Status:         "completed",
			PaymentMethod:  "free_trial",
			CreatedAt:      now,
			UpdatedAt:      now,
			ConfirmedAt:    &now,
		})

		h.audit.Log(r.Context(), r, "billing.trial.start", "subscription", sub.ID.String(), "success", fmt.Sprintf("Started %d-day free trial on %s", trialDays, plan.Name), nil)

		h.notifyInvoicePaid(r.Context(), inv)

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

	// ------------------------------------------------------------------------
	// Paid Subscription Flow: NEVER activate without verified payment!
	// ------------------------------------------------------------------------
	gwName := strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	if gwName == "" {
		gwName = "stripe"
	}
	gwConfig, err := h.store.GetGatewayConfig(r.Context(), gwName)
	if err != nil || gwConfig == nil || !gwConfig.Enabled {
		response.Error(w, http.StatusBadRequest, "GATEWAY_DISABLED", fmt.Sprintf("Payment gateway %s is not enabled by administrator", gwName), nil, "")
		return
	}

	var srvUUID *uuid.UUID
	if req.ServerID != "" {
		if parsed, err := uuid.Parse(req.ServerID); err == nil && parsed != uuid.Nil {
			srvUUID = &parsed
		}
	}
	cleanDomain := strings.ToLower(strings.TrimSpace(req.Domain))

	subID := uuid.New()
	now := time.Now().UTC()
	sub := &store.Subscription{
		ID:              subID,
		UserID:          claims.UserID,
		OrganizationID:  claims.OrganizationID,
		PlanID:          plan.ID,
		PlanName:        plan.Name,
		ServerID:        srvUUID,
		Domain:          cleanDomain,
		Status:          store.SubStatusPending, // Subscriptions MUST start as pending until paid!
		BillingCycle:    req.BillingCycle,
		Amount:          amount,
		Currency:        plan.Currency,
		DiskUsedMB:      0,
		BandwidthUsedMB: 0,
		WebsitesCount:   0,
		NextBillingDate: nextBilling,
		AutoRenew:       req.AutoRenew,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := h.store.CreateSubscription(r.Context(), sub); err != nil {
		response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create subscription", err.Error(), "")
		return
	}

	// Create Corresponding Initial Unpaid Invoice
	inv := &store.Invoice{
		ID:             uuid.New(),
		InvoiceNumber:  fmt.Sprintf("INV-%d-%05d", now.Year(), rand.Intn(90000)+10000),
		UserID:         claims.UserID,
		OrganizationID: claims.OrganizationID,
		SubscriptionID: &subID,
		PlanID:         plan.ID,
		Description:    fmt.Sprintf("%s (%s Cycle)", plan.Name, strings.Title(req.BillingCycle)),
		Subtotal:       amount,
		Tax:            0.0,
		Discount:       0.0,
		Total:          amount,
		Currency:       plan.Currency,
		Status:         store.InvoiceStatusUnpaid, // UNPAID until verified!
		PaymentMethod:  gwName,
		TransactionID:  "",
		DueDate:        now.AddDate(0, 0, 7),
		PaidAt:         nil,
		CreatedAt:      now,
	}
	_ = h.store.CreateInvoice(r.Context(), inv)

	// Record Pending Payment Transaction
	_ = h.store.RecordPaymentTransaction(r.Context(), &store.PaymentTransaction{
		ID:             uuid.New(),
		UserID:         claims.UserID,
		OrganizationID: claims.OrganizationID,
		PackageID:      plan.ID,
		SubscriptionID: &subID,
		InvoiceID:      inv.ID,
		Gateway:        gwName,
		TransactionID:  "",
		OrderID:        inv.InvoiceNumber,
		Amount:         amount,
		Currency:       plan.Currency,
		Status:         "pending",
		PaymentMethod:  gwName,
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	// Initiate Payment with Gateway Client
	initResult, initErr := h.paymentSvc.InitiatePayment(r.Context(), gwConfig, inv, plan, req.BillingCycle, req.SuccessURL, req.CancelURL, "")

	checkoutURL := fmt.Sprintf("/billing?invoice_id=%s&pay=true", inv.ID.String())
	var gwRef string
	if initErr == nil && initResult != nil {
		if initResult.CheckoutURL != "" {
			checkoutURL = initResult.CheckoutURL
		}
		gwRef = initResult.GatewayReference
	}

	h.audit.Log(r.Context(), r, "billing.subscription.pending", "subscription", sub.ID.String(), "success", fmt.Sprintf("Created pending order for %s (inv: %s)", plan.Name, inv.InvoiceNumber), nil)

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"subscription":      sub,
		"invoice":           inv,
		"status":            "pending",
		"checkout_url":      checkoutURL,
		"gateway_reference": gwRef,
		"message":           "Order created as pending. Please complete payment to activate your package subscription.",
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

	claims, hasClaims := auth.GetClaims(r.Context())
	isAdmin := hasClaims && (claims.IsSuperAdmin || claims.Role == "superadmin" || claims.Role == "owner" || claims.Role == "admin")
	if sub.OrganizationID != claims.OrganizationID && !isAdmin {
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

	claims, hasClaims := auth.GetClaims(r.Context())
	isAdmin := hasClaims && (claims.IsSuperAdmin || claims.Role == "superadmin" || claims.Role == "owner" || claims.Role == "admin")
	if sub.OrganizationID != claims.OrganizationID && !isAdmin {
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

	claims, hasClaims := auth.GetClaims(r.Context())
	isAdmin := hasClaims && (claims.IsSuperAdmin || claims.Role == "superadmin" || claims.Role == "owner" || claims.Role == "admin")
	if inv.UserID != claims.UserID && (inv.OrganizationID == uuid.Nil || inv.OrganizationID != claims.OrganizationID) && !isAdmin {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Invoice not found", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, inv, nil)
}

type PayInvoiceRequest struct {
	PaymentMethod string `json:"payment_method"` // stripe, bkash, nagad, sslcommerz, paypal
	TransactionID string `json:"transaction_id"`
	PaymentRef    string `json:"payment_ref"`
}

type VerifyPaymentRequest struct {
	PaymentMethod string `json:"payment_method"` // stripe, bkash, nagad
	PaymentRef    string `json:"payment_ref"`    // session_id, paymentID, payment_ref_id
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

	claims, hasClaims := auth.GetClaims(r.Context())
	isAdmin := hasClaims && (claims.IsSuperAdmin || claims.Role == "superadmin" || claims.Role == "admin")
	isOwnerOrMember := hasClaims && (inv.UserID == claims.UserID || (inv.OrganizationID != uuid.Nil && inv.OrganizationID == claims.OrganizationID))
	if !isOwnerOrMember && !isAdmin {
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

	if !isAdmin {
		// Non-admins cannot mark an invoice paid without verified payment confirmation!
		ref := strings.TrimSpace(req.PaymentRef)
		if ref == "" {
			ref = strings.TrimSpace(req.TransactionID)
		}
		if ref == "" {
			response.Error(w, http.StatusPaymentRequired, "PAYMENT_REQUIRED", "Manual payment confirmation requires administrator authorization or a verified gateway payment reference", nil, "")
			return
		}
		verifyReq := VerifyPaymentRequest{
			PaymentMethod: req.PaymentMethod,
			PaymentRef:    ref,
		}
		verifyJSON, _ := json.Marshal(verifyReq)
		r.Body = io.NopCloser(bytes.NewReader(verifyJSON))
		h.VerifyInvoicePayment(w, r)
		return
	}

	// Admin Override Flow: explicitly distinguishable from customer gateway payments
	now := time.Now().UTC()
	payMethod := req.PaymentMethod
	if payMethod == "" {
		payMethod = "admin_override"
	}
	txnID := req.TransactionID
	if txnID == "" {
		txnID = fmt.Sprintf("admin_override_%s_%d", claims.UserID.String()[:8], now.Unix())
	}

	inv.Status = store.InvoiceStatusPaid
	inv.PaymentMethod = payMethod
	inv.TransactionID = txnID
	inv.PaidAt = &now

	if err := h.store.UpdateInvoice(r.Context(), inv); err != nil {
		response.Error(w, http.StatusInternalServerError, "PAY_FAILED", "Failed to update invoice", err.Error(), "")
		return
	}

	// If linked to subscription, make sure subscription is Active and user is provisioned
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

			if plan, err := h.store.GetPlanByID(r.Context(), sub.PlanID); err == nil && plan != nil {
				h.provisionUserForPlan(r.Context(), inv.UserID, inv.OrganizationID, plan, sub)
			}
		}
	}

	// Record transaction
	_ = h.store.RecordPaymentTransaction(r.Context(), &store.PaymentTransaction{
		ID:             uuid.New(),
		UserID:         inv.UserID,
		OrganizationID: inv.OrganizationID,
		PackageID:      inv.PlanID,
		SubscriptionID: inv.SubscriptionID,
		InvoiceID:      inv.ID,
		Gateway:        "admin",
		TransactionID:  txnID,
		OrderID:        inv.InvoiceNumber,
		Amount:         inv.Total,
		Currency:       inv.Currency,
		Status:         "completed",
		PaymentMethod:  payMethod,
		CreatedAt:      now,
		UpdatedAt:      now,
		ConfirmedAt:    &now,
	})

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

	h.notifyInvoicePaid(r.Context(), inv)

	h.audit.Log(r.Context(), r, "billing.invoice.pay_admin_override", "invoice", inv.ID.String(), "success", fmt.Sprintf("Admin marked invoice %s paid (txn: %s)", inv.InvoiceNumber, inv.TransactionID), map[string]interface{}{
		"invoice_number": inv.InvoiceNumber,
		"payment_method": inv.PaymentMethod,
		"transaction_id": inv.TransactionID,
		"is_admin":       true,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "Invoice marked as paid via admin override",
		"invoice": inv,
	}, nil)
}

type InvoiceCheckoutRequest struct {
	PaymentMethod string `json:"payment_method"`
	SuccessURL    string `json:"success_url"`
	CancelURL     string `json:"cancel_url"`
}

func (h *BillingHandler) CheckoutInvoice(w http.ResponseWriter, r *http.Request) {
	claims, hasClaims := auth.GetClaims(r.Context())
	if !hasClaims {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication session", nil, "")
		return
	}

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

	isAdmin := claims.IsSuperAdmin || claims.Role == "superadmin" || claims.Role == "admin"
	isOwnerOrMember := inv.UserID == claims.UserID || (inv.OrganizationID != uuid.Nil && inv.OrganizationID == claims.OrganizationID)
	if !isOwnerOrMember && !isAdmin {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "You do not have permission to pay this invoice", nil, "")
		return
	}

	if inv.Status == store.InvoiceStatusPaid {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"status":  "already_paid",
			"message": "Invoice is already paid",
			"invoice": inv,
		}, nil)
		return
	}

	var req InvoiceCheckoutRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	gwName := strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	if gwName == "" {
		gwName = strings.ToLower(inv.PaymentMethod)
	}
	if gwName == "" {
		gwName = "stripe"
	}

	gwConfig, err := h.store.GetGatewayConfig(r.Context(), gwName)
	if err != nil || gwConfig == nil || !gwConfig.Enabled {
		response.Error(w, http.StatusBadRequest, "GATEWAY_DISABLED", fmt.Sprintf("Payment gateway %s is not enabled by administrator", gwName), nil, "")
		return
	}

	var plan *store.HostingPlan
	if inv.PlanID != uuid.Nil {
		plan, _ = h.store.GetPlanByID(r.Context(), inv.PlanID)
	}

	inv.PaymentMethod = gwName
	_ = h.store.UpdateInvoice(r.Context(), inv)

	now := time.Now().UTC()
	_ = h.store.RecordPaymentTransaction(r.Context(), &store.PaymentTransaction{
		ID:             uuid.New(),
		UserID:         inv.UserID,
		OrganizationID: inv.OrganizationID,
		PackageID:      inv.PlanID,
		SubscriptionID: inv.SubscriptionID,
		InvoiceID:      inv.ID,
		Gateway:        gwName,
		TransactionID:  "",
		OrderID:        inv.InvoiceNumber,
		Amount:         inv.Total,
		Currency:       inv.Currency,
		Status:         "pending",
		PaymentMethod:  gwName,
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	initResult, initErr := h.paymentSvc.InitiatePayment(r.Context(), gwConfig, inv, plan, "monthly", req.SuccessURL, req.CancelURL, "")

	checkoutURL := fmt.Sprintf("/billing?invoice_id=%s&pay=true", inv.ID.String())
	var gwRef string
	if initErr == nil && initResult != nil {
		if initResult.CheckoutURL != "" {
			checkoutURL = initResult.CheckoutURL
		}
		gwRef = initResult.GatewayReference
	}

	h.audit.Log(r.Context(), r, "billing.invoice.checkout_initiated", "invoice", inv.ID.String(), "success", fmt.Sprintf("Initiated checkout for invoice %s via %s", inv.InvoiceNumber, gwName), map[string]interface{}{
		"gateway": gwName,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"invoice_id":        inv.ID,
		"invoice_number":    inv.InvoiceNumber,
		"amount":            inv.Total,
		"currency":          inv.Currency,
		"payment_method":    gwName,
		"checkout_url":      checkoutURL,
		"gateway_reference": gwRef,
	}, nil)
}

func (h *BillingHandler) VerifyInvoicePayment(w http.ResponseWriter, r *http.Request) {
	claims, hasClaims := auth.GetClaims(r.Context())
	if !hasClaims {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication session", nil, "")
		return
	}

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

	isAdmin := claims.IsSuperAdmin || claims.Role == "superadmin" || claims.Role == "owner" || claims.Role == "admin"
	if inv.UserID != claims.UserID && (inv.OrganizationID == uuid.Nil || inv.OrganizationID != claims.OrganizationID) && !isAdmin {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "You do not have permission to verify this invoice", nil, "")
		return
	}

	if inv.Status == store.InvoiceStatusPaid {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"status":         "already_paid",
			"message":        "Invoice is already paid and subscription is active",
			"invoice":        inv,
			"transaction_id": inv.TransactionID,
		}, nil)
		return
	}

	var req VerifyPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	gwName := strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	if gwName == "" {
		gwName = strings.ToLower(inv.PaymentMethod)
	}
	if gwName == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_GATEWAY", "Payment gateway is required", nil, "")
		return
	}

	cfg, err := h.store.GetGatewayConfig(r.Context(), gwName)
	if err != nil || cfg == nil || !cfg.Enabled {
		response.Error(w, http.StatusBadRequest, "GATEWAY_UNAVAILABLE", fmt.Sprintf("Payment gateway %s is not enabled", gwName), nil, "")
		return
	}

	paymentRef := strings.TrimSpace(req.PaymentRef)
	if paymentRef == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_PAYMENT_REF", "Payment transaction reference is required", nil, "")
		return
	}

	// Server-side verification with gateway
	verifyResult, err := h.paymentSvc.VerifyPayment(r.Context(), cfg, inv, paymentRef)
	if err != nil {
		h.audit.Log(r.Context(), r, "billing.payment.verify_failed", "invoice", inv.ID.String(), "failure", err.Error(), map[string]interface{}{
			"gateway":     gwName,
			"payment_ref": paymentRef,
		})

		_ = h.store.RecordPaymentTransaction(r.Context(), &store.PaymentTransaction{
			ID:               uuid.New(),
			UserID:           inv.UserID,
			OrganizationID:   inv.OrganizationID,
			PackageID:        inv.PlanID,
			SubscriptionID:   inv.SubscriptionID,
			InvoiceID:        inv.ID,
			Gateway:          gwName,
			TransactionID:    paymentRef,
			OrderID:          inv.InvoiceNumber,
			Amount:           inv.Total,
			Currency:         inv.Currency,
			Status:           "failed",
			GatewayReference: paymentRef,
			PaymentMethod:    gwName,
			RawResponse:      err.Error(),
		})

		response.Error(w, http.StatusBadRequest, "PAYMENT_UNVERIFIED", "Gateway payment verification failed: "+err.Error(), nil, "")
		return
	}

	// Idempotency: anti-replay check
	if existingTxn, _ := h.store.GetPaymentTransactionByTxnID(r.Context(), verifyResult.Gateway, verifyResult.TransactionID); existingTxn != nil && existingTxn.Status == "completed" && existingTxn.InvoiceID != inv.ID {
		response.Error(w, http.StatusConflict, "DUPLICATE_TRANSACTION", "This payment transaction has already been verified for another invoice", nil, "")
		return
	}

	now := time.Now().UTC()
	inv.Status = store.InvoiceStatusPaid
	inv.PaidAt = &now
	inv.PaymentMethod = verifyResult.PaymentMethod
	inv.TransactionID = verifyResult.TransactionID

	if err := h.store.UpdateInvoice(r.Context(), inv); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update invoice", err.Error(), "")
		return
	}

	var sub *store.Subscription
	if inv.SubscriptionID != nil {
		if s, err := h.store.GetSubscriptionByID(r.Context(), *inv.SubscriptionID); err == nil {
			sub = s
			sub.Status = store.SubStatusActive
			if sub.NextBillingDate.Before(now) {
				if sub.BillingCycle == "yearly" {
					sub.NextBillingDate = now.AddDate(1, 0, 0)
				} else {
					sub.NextBillingDate = now.AddDate(0, 1, 0)
				}
			}
			_ = h.store.UpdateSubscription(r.Context(), sub)

			if plan, err := h.store.GetPlanByID(r.Context(), sub.PlanID); err == nil && plan != nil {
				h.provisionUserForPlan(r.Context(), inv.UserID, inv.OrganizationID, plan, sub)
			}
		}
	}

	_ = h.store.RecordPaymentTransaction(r.Context(), &store.PaymentTransaction{
		ID:               uuid.New(),
		UserID:           inv.UserID,
		OrganizationID:   inv.OrganizationID,
		PackageID:        inv.PlanID,
		SubscriptionID:   inv.SubscriptionID,
		InvoiceID:        inv.ID,
		Gateway:          verifyResult.Gateway,
		TransactionID:    verifyResult.TransactionID,
		OrderID:          inv.InvoiceNumber,
		Amount:           verifyResult.Amount,
		Currency:         verifyResult.Currency,
		Status:           "completed",
		GatewayReference: verifyResult.GatewayReference,
		PaymentMethod:    verifyResult.PaymentMethod,
		RawResponse:      verifyResult.RawResponse,
		CreatedAt:        now,
		UpdatedAt:        now,
		ConfirmedAt:      &now,
	})

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

	h.notifyInvoicePaid(r.Context(), inv)

	h.audit.Log(r.Context(), r, "billing.payment.verified", "invoice", inv.ID.String(), "success", fmt.Sprintf("Verified payment via %s (txn: %s)", verifyResult.Gateway, verifyResult.TransactionID), map[string]interface{}{
		"gateway":        verifyResult.Gateway,
		"transaction_id": verifyResult.TransactionID,
		"amount":         verifyResult.Amount,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"status":         "paid",
		"message":        "Payment verified and subscription activated successfully!",
		"invoice":        inv,
		"subscription":   sub,
		"transaction_id": verifyResult.TransactionID,
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

				// Provision user quotas, auto-provision account if needed, and unsuspend existing accounts
				if plan, err := h.store.GetPlanByID(r.Context(), sub.PlanID); err == nil && plan != nil {
					h.provisionUserForPlan(r.Context(), inv.UserID, inv.OrganizationID, plan, sub)
				}
			}
		}

		// Record Completed Payment Transaction
		_ = h.store.RecordPaymentTransaction(r.Context(), &store.PaymentTransaction{
			ID:               uuid.New(),
			UserID:           inv.UserID,
			OrganizationID:   inv.OrganizationID,
			PackageID:        inv.PlanID,
			SubscriptionID:   inv.SubscriptionID,
			InvoiceID:        inv.ID,
			Gateway:          gatewayName,
			TransactionID:    inv.TransactionID,
			OrderID:          inv.InvoiceNumber,
			Amount:           inv.Total,
			Currency:         inv.Currency,
			Status:           "completed",
			GatewayReference: eventID,
			PaymentMethod:    gatewayName,
			RawResponse:      string(bodyBytes),
			CreatedAt:        now,
			UpdatedAt:        now,
			ConfirmedAt:      &now,
		})

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

		h.notifyInvoicePaid(r.Context(), inv)

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
		OrganizationID: sub.OrganizationID,
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
	Domain        string `json:"domain"`
	ServerID      string `json:"server_id"`
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
			Domain:        req.Domain,
			ServerID:      req.ServerID,
		}
		jsonBody, _ := json.Marshal(subReq)
		r.Body = io.NopCloser(strings.NewReader(string(jsonBody)))
		h.CreateSubscription(w, r)
		return
	}

	// Validate payment gateway is enabled
	gwName := strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	if gwName == "" {
		gwName = "stripe"
	}
	gwConfig, err := h.store.GetGatewayConfig(r.Context(), gwName)
	if err != nil || gwConfig == nil || !gwConfig.Enabled {
		response.Error(w, http.StatusBadRequest, "GATEWAY_DISABLED", fmt.Sprintf("Payment gateway %s is not enabled by administrator", gwName), nil, "")
		return
	}

	// Create Pending Subscription & Invoice
	var srvUUID *uuid.UUID
	if req.ServerID != "" {
		if parsed, err := uuid.Parse(req.ServerID); err == nil && parsed != uuid.Nil {
			srvUUID = &parsed
		}
	}
	cleanDomain := strings.ToLower(strings.TrimSpace(req.Domain))

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
		ServerID:        srvUUID,
		Domain:          cleanDomain,
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
		OrganizationID: claims.OrganizationID,
		SubscriptionID: &subID,
		PlanID:         plan.ID,
		Description:    fmt.Sprintf("%s (%s Subscription)", plan.Name, strings.Title(req.BillingCycle)),
		Subtotal:       amount,
		Tax:            0.0,
		Discount:       0.0,
		Total:          amount,
		Currency:       plan.Currency,
		Status:         store.InvoiceStatusUnpaid,
		PaymentMethod:  gwName,
		DueDate:        now.AddDate(0, 0, 3),
		CreatedAt:      now,
	}
	_ = h.store.CreateInvoice(r.Context(), inv)

	// Record Pending Payment Transaction
	_ = h.store.RecordPaymentTransaction(r.Context(), &store.PaymentTransaction{
		ID:             uuid.New(),
		UserID:         claims.UserID,
		OrganizationID: claims.OrganizationID,
		PackageID:      plan.ID,
		SubscriptionID: &subID,
		InvoiceID:      inv.ID,
		Gateway:        gwName,
		TransactionID:  "",
		OrderID:        inv.InvoiceNumber,
		Amount:         amount,
		Currency:       plan.Currency,
		Status:         "pending",
		PaymentMethod:  gwName,
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	// Initiate Payment with Gateway Client
	initResult, initErr := h.paymentSvc.InitiatePayment(r.Context(), gwConfig, inv, plan, req.BillingCycle, req.SuccessURL, req.CancelURL, "")

	checkoutURL := fmt.Sprintf("/billing?invoice_id=%s&pay=true", inv.ID.String())
	var gwRef string
	if initErr == nil && initResult != nil {
		if initResult.CheckoutURL != "" {
			checkoutURL = initResult.CheckoutURL
		}
		gwRef = initResult.GatewayReference
	}

	h.audit.Log(r.Context(), r, "billing.checkout.create", "invoice", inv.ID.String(), "success", fmt.Sprintf("Created checkout session for %s", plan.Name), nil)

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"subscription_id":   sub.ID,
		"invoice_id":        inv.ID,
		"invoice_number":    inv.InvoiceNumber,
		"amount":            amount,
		"currency":          plan.Currency,
		"plan_name":         plan.Name,
		"billing_cycle":     req.BillingCycle,
		"payment_method":    gwName,
		"status":            "pending",
		"checkout_url":      checkoutURL,
		"gateway_reference": gwRef,
	}, nil)
}

func (h *BillingHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	claims, hasClaims := auth.GetClaims(r.Context())
	if !hasClaims {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication session", nil, "")
		return
	}

	isAdmin := claims.IsSuperAdmin || claims.Role == "superadmin" || claims.Role == "admin"
	var orgID uuid.UUID
	if !isAdmin {
		orgID = claims.OrganizationID
	}

	txns, err := h.store.ListPaymentTransactions(r.Context(), orgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve transactions", err.Error(), "")
		return
	}

	response.JSON(w, http.StatusOK, txns, &response.Meta{Total: len(txns)})
}

// ----------------------------------------------------------------------------
// Automated Billing Lifecycle Engine (Auto-Suspension & Overdue Tracking)
// ----------------------------------------------------------------------------

// ProcessBillingLifecycle evaluates all unpaid invoices, overdue grace periods, and expired free trials
func (h *BillingHandler) ProcessBillingLifecycle(ctx context.Context) (map[string]int, error) {
	stats := map[string]int{
		"overdue_invoices":   0,
		"suspended_accounts": 0,
		"expired_trials":     0,
	}

	now := time.Now().UTC()

	// 1. Process Invoices: Check for unpaid invoices past due date
	invoices, err := h.store.ListInvoices(ctx, uuid.Nil)
	if err == nil {
		for _, inv := range invoices {
			// Mark overdue if due date has passed
			if inv.Status == store.InvoiceStatusUnpaid && inv.DueDate.Before(now) {
				inv.Status = store.InvoiceStatusOverdue
				_ = h.store.UpdateInvoice(ctx, inv)
				stats["overdue_invoices"]++
			}

			// If invoice is overdue past 2-day grace period (48h) and linked to a subscription, suspend services
			if inv.Status == store.InvoiceStatusOverdue && now.Sub(inv.DueDate) > 48*time.Hour && inv.SubscriptionID != nil {
				if sub, sErr := h.store.GetSubscriptionByID(ctx, *inv.SubscriptionID); sErr == nil && sub != nil && sub.Status == store.SubStatusActive {
					sub.Status = store.SubStatusSuspended
					_ = h.store.UpdateSubscription(ctx, sub)

					// Suspend hosting accounts
					if accounts, aErr := h.store.ListHostingAccounts(ctx, sub.OrganizationID, nil); aErr == nil {
						for _, acc := range accounts {
							if acc.UserID == inv.UserID && acc.Status == store.AccountStatusActive {
								acc.Status = store.AccountStatusSuspended
								acc.SuspendReason = fmt.Sprintf("Overdue invoice %s past grace period", inv.InvoiceNumber)
								suspendedAt := now
								acc.SuspendedAt = &suspendedAt
								_ = h.store.UpdateHostingAccount(ctx, acc)
								stats["suspended_accounts"]++

								// Suspend website vhost
								if sites, siteErr := h.store.ListAllWebsites(ctx); siteErr == nil {
									for _, site := range sites {
										if strings.EqualFold(site.PrimaryDomain, acc.Domain) && site.Status == "active" {
											site.Status = "suspended"
											_ = h.store.UpdateWebsite(ctx, site)
										}
									}
								}

								// Send email notice
								if h.emailSvc != nil {
									if u, uErr := h.store.GetUserByID(ctx, inv.UserID); uErr == nil && u != nil {
										_ = h.emailSvc.SendAccountSuspended(ctx, u.Email, u.FullName, acc.Domain, "Overdue unpaid invoice past grace period", &inv.ID)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// 2. Process Free Trials: Expire trials that ended
	subs, sErr := h.store.ListSubscriptions(ctx, uuid.Nil)
	if sErr == nil {
		for _, sub := range subs {
			if sub.Status == store.SubStatusTrial && sub.TrialEndsAt != nil && sub.TrialEndsAt.Before(now) {
				sub.Status = store.SubStatusExpired
				_ = h.store.UpdateSubscription(ctx, sub)
				stats["expired_trials"]++

				// Suspend linked hosting accounts
				if accounts, aErr := h.store.ListHostingAccounts(ctx, sub.OrganizationID, nil); aErr == nil {
					for _, acc := range accounts {
						if acc.UserID == sub.UserID && acc.Status == store.AccountStatusActive {
							acc.Status = store.AccountStatusSuspended
							acc.SuspendReason = "Free trial expired. Please upgrade to a paid plan."
							suspendedAt := now
							acc.SuspendedAt = &suspendedAt
							_ = h.store.UpdateHostingAccount(ctx, acc)

							// Suspend website vhost
							if sites, siteErr := h.store.ListAllWebsites(ctx); siteErr == nil {
								for _, site := range sites {
									if strings.EqualFold(site.PrimaryDomain, acc.Domain) && site.Status == "active" {
										site.Status = "suspended"
										_ = h.store.UpdateWebsite(ctx, site)
									}
								}
							}

							// Send suspension notification
							if h.emailSvc != nil {
								if u, uErr := h.store.GetUserByID(ctx, sub.UserID); uErr == nil && u != nil {
									_ = h.emailSvc.SendAccountSuspended(ctx, u.Email, u.FullName, acc.Domain, "Free trial period expired", nil)
								}
							}
						}
					}
				}
			}
		}
	}

	return stats, nil
}

// RunLifecycle triggers a manual run of the billing lifecycle engine
func (h *BillingHandler) RunLifecycle(w http.ResponseWriter, r *http.Request) {
	stats, err := h.ProcessBillingLifecycle(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIFECYCLE_FAILED", "Failed to process billing lifecycle", err.Error(), "")
		return
	}

	auditDetails := map[string]interface{}{
		"overdue_invoices":   stats["overdue_invoices"],
		"suspended_accounts": stats["suspended_accounts"],
		"expired_trials":     stats["expired_trials"],
	}
	h.audit.Log(r.Context(), r, "billing.lifecycle.run", "billing", "system", "success", "Ran automated billing lifecycle scan", auditDetails)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Billing lifecycle processed successfully",
		"stats":   stats,
	}, nil)
}

// StartLifecycleWorker runs the automated billing lifecycle in a background goroutine
func (h *BillingHandler) StartLifecycleWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 1 * time.Hour
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats, err := h.ProcessBillingLifecycle(context.Background())
				if err != nil {
					slog.Error("Billing lifecycle worker error", "error", err)
				} else if stats["overdue_invoices"] > 0 || stats["suspended_accounts"] > 0 || stats["expired_trials"] > 0 {
					slog.Info("Billing lifecycle worker cycle completed",
						"overdue_invoices", stats["overdue_invoices"],
						"suspended_accounts", stats["suspended_accounts"],
						"expired_trials", stats["expired_trials"])
				}
			}
		}
	}()
}




