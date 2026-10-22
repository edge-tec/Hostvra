package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/isolation"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/quota"
	"hostvra/api/internal/rbac"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type AccountHandler struct {
	cfg          *config.Config
	store        store.Store
	audit        *audit.Logger
	isolationMgr *isolation.Manager
	quotaSvc     *quota.Service
}

func NewAccountHandler(cfg *config.Config, s store.Store, a *audit.Logger) *AccountHandler {
	isoMgr, _ := isolation.NewManager(isolation.Config{
		WebRootDir:   os.Getenv("HOSTVRA_WEB_ROOT"),
		PHPConfigDir: os.Getenv("HOSTVRA_PHP_CONFIG_DIR"),
		SystemdDir:   os.Getenv("HOSTVRA_SYSTEMD_DIR"),
	})
	return &AccountHandler{
		cfg:          cfg,
		store:        s,
		audit:        a,
		isolationMgr: isoMgr,
	}
}

func (h *AccountHandler) SetIsolationManager(mgr *isolation.Manager) {
	h.isolationMgr = mgr
}

func (h *AccountHandler) SetQuotaService(svc *quota.Service) {
	h.quotaSvc = svc
}

// ----------------------------------------------------------------------------
// List & Get Accounts
// ----------------------------------------------------------------------------

func (h *AccountHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var serverIDPtr *uuid.UUID
	if sID := r.URL.Query().Get("server_id"); sID != "" {
		if parsed, err := uuid.Parse(sID); err == nil {
			serverIDPtr = &parsed
		}
	}

	accounts, err := h.store.ListHostingAccounts(r.Context(), claims.OrganizationID, serverIDPtr)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve accounts", err.Error(), "")
		return
	}

	// Filter by search query if provided
	search := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("search")))
	statusFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))

	var filtered []*store.HostingAccount
	for _, acc := range accounts {
		if statusFilter != "" && strings.ToLower(string(acc.Status)) != statusFilter {
			continue
		}
		if search != "" {
			if !strings.Contains(strings.ToLower(acc.Domain), search) &&
				!strings.Contains(strings.ToLower(acc.Username), search) &&
				!strings.Contains(strings.ToLower(acc.PlanName), search) {
				continue
			}
		}

		// Reconcile live filesystem storage if available
		if acc.DocumentRoot != "" {
			if sizeBytes := quota.CalculateDirSizeBytes(acc.DocumentRoot); sizeBytes > 0 {
				acc.DiskUsedMB = (sizeBytes + 1048575) / (1024 * 1024)
			}
		}

		filtered = append(filtered, acc)
	}

	response.JSON(w, http.StatusOK, filtered, &response.Meta{Total: len(filtered)})
}

func (h *AccountHandler) authorizeAccount(r *http.Request, acc *store.HostingAccount) bool {
	claims, hasClaims := auth.GetClaims(r.Context())
	if !hasClaims || claims == nil {
		return false
	}
	if claims.IsSuperAdmin || claims.Role == "superadmin" || claims.Role == "owner" || claims.Role == "admin" {
		return true
	}
	return acc.OrganizationID == claims.OrganizationID
}

func (h *AccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	accID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid account ID", nil, "")
		return
	}

	acc, err := h.store.GetHostingAccountByID(r.Context(), accID)
	if err != nil || !h.authorizeAccount(r, acc) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Hosting account not found", nil, "")
		return
	}

	if acc.DocumentRoot != "" {
		if sizeBytes := quota.CalculateDirSizeBytes(acc.DocumentRoot); sizeBytes > 0 {
			acc.DiskUsedMB = (sizeBytes + 1048575) / (1024 * 1024)
		}
	}

	response.JSON(w, http.StatusOK, acc, nil)
}

// ----------------------------------------------------------------------------
// Create Account (Provisioning)
// ----------------------------------------------------------------------------

type CreateAccountRequest struct {
	Domain         string `json:"domain"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	PlanID         string `json:"plan_id"`
	ServerID       string `json:"server_id"`
	SubscriptionID string `json:"subscription_id,omitempty"`
	PHPVersion     string `json:"php_version,omitempty"`
}

func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	cleanDomain := strings.ToLower(strings.TrimSpace(req.Domain))
	cleanUser := strings.ToLower(strings.TrimSpace(req.Username))

	if cleanDomain == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Primary domain is required", nil, "")
		return
	}
	if cleanUser == "" {
		// Auto-generate username from domain
		cleanUser = "c_" + strings.ReplaceAll(strings.Split(cleanDomain, ".")[0], "-", "")
		if len(cleanUser) > 16 {
			cleanUser = cleanUser[:16]
		}
	}

	// 1. Conflict Prevention: Check domain and username uniqueness
	if existing, _ := h.store.GetHostingAccountByDomain(r.Context(), cleanDomain); existing != nil {
		response.Error(w, http.StatusConflict, "DOMAIN_EXISTS", fmt.Sprintf("A hosting account for domain %s already exists", cleanDomain), nil, "")
		return
	}
	if existing, _ := h.store.GetHostingAccountByUsername(r.Context(), cleanUser); existing != nil {
		response.Error(w, http.StatusConflict, "USERNAME_EXISTS", fmt.Sprintf("A hosting account with username %s already exists", cleanUser), nil, "")
		return
	}

	planUUID, err := uuid.Parse(req.PlanID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PLAN_ID", "Valid Plan ID is required", nil, "")
		return
	}

	plan, err := h.store.GetPlanByID(r.Context(), planUUID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "PLAN_NOT_FOUND", "Selected hosting plan not found", nil, "")
		return
	}

	// Reseller quota enforcement
	if claims.Role == rbac.RoleReseller {
		resellerProfile, pErr := h.store.GetResellerProfileByUserID(r.Context(), claims.UserID)
		if pErr == nil && resellerProfile != nil {
			if !resellerProfile.IsActive {
				response.Error(w, http.StatusForbidden, "RESELLER_INACTIVE", "Reseller profile is inactive or suspended", nil, "")
				return
			}
			usage, uErr := h.store.GetResellerUsage(r.Context(), claims.UserID)
			if uErr == nil && usage != nil {
				if usage.MaxAccounts > 0 && usage.UsedAccounts >= usage.MaxAccounts {
					response.Error(w, http.StatusForbidden, "QUOTA_EXCEEDED", fmt.Sprintf("Reseller account limit (%d) reached", usage.MaxAccounts), nil, "")
					return
				}
				if !resellerProfile.AllowOverselling {
					if resellerProfile.AllocatedDiskMB > 0 && (usage.UsedDiskMB+plan.DiskSpaceMB) > resellerProfile.AllocatedDiskMB {
						response.Error(w, http.StatusForbidden, "QUOTA_EXCEEDED", fmt.Sprintf("Reseller disk allocation pool (%d MB) exceeded", resellerProfile.AllocatedDiskMB), nil, "")
						return
					}
					if resellerProfile.AllocatedBandwidthMB > 0 && (usage.UsedBandwidthMB+plan.BandwidthMB) > resellerProfile.AllocatedBandwidthMB {
						response.Error(w, http.StatusForbidden, "QUOTA_EXCEEDED", fmt.Sprintf("Reseller bandwidth allocation pool (%d MB) exceeded", resellerProfile.AllocatedBandwidthMB), nil, "")
						return
					}
				}
			}
		}
	}

	var serverUUID *uuid.UUID
	var serverName = "Primary Cluster Node"
	var ipAddress = "109.199.110.101"
	if req.ServerID != "" {
		if sID, err := uuid.Parse(req.ServerID); err == nil {
			serverUUID = &sID
			if srv, err := h.store.GetServerByID(r.Context(), sID); err == nil {
				serverName = srv.Name
				if srv.IPAddress != "" {
					ipAddress = srv.IPAddress
				}
			}
		}
	} else {
		if servers, err := h.store.ListServersByOrg(r.Context(), claims.OrganizationID); err == nil && len(servers) > 0 {
			serverUUID = &servers[0].ID
			serverName = servers[0].Name
			if servers[0].IPAddress != "" {
				ipAddress = servers[0].IPAddress
			}
		}
	}

	var subUUID *uuid.UUID
	if req.SubscriptionID != "" {
		if sID, err := uuid.Parse(req.SubscriptionID); err == nil {
			subUUID = &sID
		}
	}

	phpVer := req.PHPVersion
	if phpVer == "" {
		phpVer = "8.3"
	}

	accID := uuid.New()
	docRoot := fmt.Sprintf("/home/%s/public_html", cleanUser)

	// 2. Real Operating System User & cgroups v2 Isolation Provisioning
	if h.isolationMgr != nil {
		limits := isolation.DefaultResourceLimits()
		if plan.DiskSpaceMB > 0 {
			if plan.DiskSpaceMB >= 10240 {
				limits.MemoryMaxMB = 1024
				limits.MemoryHighMB = 920
			}
		}
		if isoInfo, err := h.isolationMgr.ProvisionWebsiteIsolation(r.Context(), cleanDomain, phpVer, &limits); err == nil && isoInfo != nil {
			if isoInfo.DocumentRoot != "" {
				docRoot = isoInfo.DocumentRoot
			}
			if isoInfo.Username != "" {
				cleanUser = isoInfo.Username
			}
		}
	}

	// 3. Ensure Client User Record Exists for Tenant Isolation and 1-Click Login
	var clientUser *store.User
	if claims.Role == rbac.RoleCustomer {
		clientUser, _ = h.store.GetUserByID(r.Context(), claims.UserID)
	}
	if clientUser == nil {
		clientEmail := fmt.Sprintf("%s@%s", cleanUser, cleanDomain)
		clientUser, _ = h.store.GetUserByEmail(r.Context(), clientEmail)
		if clientUser == nil {
			newUserID := uuid.New()
			clientUser = &store.User{
				ID:           newUserID,
				DefaultOrgID: claims.OrganizationID,
				Email:        clientEmail,
				FullName:     fmt.Sprintf("Client %s", cleanUser),
				Role:         rbac.RoleCustomer,
				IsActive:     true,
				CreatedAt:    time.Now().UTC(),
				UpdatedAt:    time.Now().UTC(),
			}
			_ = h.store.CreateUser(r.Context(), clientUser, claims.OrganizationID, rbac.RoleCustomer)
		}
	}
	targetUserID := claims.UserID
	if clientUser != nil {
		targetUserID = clientUser.ID
	}

	// Measure initial disk space from authoritative filesystem
	initialDisk := int64(25)
	if diskB := quota.CalculateDirSizeBytes(docRoot); diskB > 0 {
		initialDisk = (diskB + 1048575) / (1024 * 1024)
	}

	account := &store.HostingAccount{
		ID:               accID,
		OrganizationID:   claims.OrganizationID,
		UserID:           targetUserID,
		SubscriptionID:   subUUID,
		ServerID:         serverUUID,
		ServerName:       serverName,
		Domain:           cleanDomain,
		Username:         cleanUser,
		DocumentRoot:     docRoot,
		PlanID:           plan.ID,
		PlanName:         plan.Name,
		Status:           store.AccountStatusActive,
		DiskLimitMB:      plan.DiskSpaceMB,
		DiskUsedMB:       initialDisk,
		BandwidthLimitMB: plan.BandwidthMB,
		BandwidthUsedMB:  1,
		WebsitesLimit:    plan.MaxWebsites,
		DatabasesLimit:   plan.MaxDatabases,
		MailboxesLimit:   plan.MaxMailboxes,
		IPAddress:        ipAddress,
		PHPVersion:       phpVer,
		SSLActive:        plan.FreeSSL,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	if err := h.store.CreateHostingAccount(r.Context(), account); err != nil {
		response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to provision hosting account", err.Error(), "")
		return
	}

	// 4. Automatically provision corresponding Website record for Nginx/Apache routing if server exists
	if serverUUID != nil {
		_ = h.store.CreateWebsite(r.Context(), &store.Website{
			ID:             uuid.New(),
			OrganizationID: claims.OrganizationID,
			ServerID:       *serverUUID,
			PrimaryDomain:  cleanDomain,
			DocumentRoot:   docRoot,
			SystemUser:     cleanUser,
			AppType:        "php",
			Status:         "active",
			SSLEnabled:     plan.FreeSSL,
		})
	}

	h.audit.Log(r.Context(), r, "accounts.provision", "hosting_account", account.ID.String(), "success", fmt.Sprintf("Provisioned isolated WHM account %s for domain %s", cleanUser, cleanDomain), map[string]interface{}{
		"account_id":  account.ID,
		"username":    cleanUser,
		"domain":      cleanDomain,
		"doc_root":    docRoot,
		"plan_id":     plan.ID,
		"plan_name":   plan.Name,
		"ip_address":  ipAddress,
		"php_version": phpVer,
	})

	response.JSON(w, http.StatusCreated, account, nil)
}

// ----------------------------------------------------------------------------
// Suspend / Unsuspend / Change Plan
// ----------------------------------------------------------------------------

type SuspendRequest struct {
	Reason string `json:"reason"`
}

func (h *AccountHandler) findWebsiteByDomain(ctx context.Context, orgID uuid.UUID, domain string) (*store.Website, error) {
	sites, err := h.store.ListWebsitesByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for _, site := range sites {
		if strings.EqualFold(site.PrimaryDomain, domain) {
			return site, nil
		}
	}
	return nil, nil
}

func (h *AccountHandler) SuspendAccount(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	accID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid account ID", nil, "")
		return
	}

	acc, err := h.store.GetHostingAccountByID(r.Context(), accID)
	if err != nil || !h.authorizeAccount(r, acc) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil, "")
		return
	}

	var req SuspendRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Reason == "" {
		req.Reason = "Administrative Suspension"
	}

	now := time.Now().UTC()
	acc.Status = store.AccountStatusSuspended
	acc.SuspendReason = req.Reason
	acc.SuspendedAt = &now

	if err := h.store.UpdateHostingAccount(r.Context(), acc); err != nil {
		response.Error(w, http.StatusInternalServerError, "SUSPEND_FAILED", "Failed to suspend account", err.Error(), "")
		return
	}

	// Update corresponding website status
	if site, err := h.findWebsiteByDomain(r.Context(), acc.OrganizationID, acc.Domain); err == nil && site != nil {
		site.Status = "suspended"
		_ = h.store.UpdateWebsite(r.Context(), site)
	}

	h.audit.Log(r.Context(), r, "accounts.suspend", "hosting_account", acc.ID.String(), "success", fmt.Sprintf("Suspended account %s (%s)", acc.Username, req.Reason), nil)

	response.JSON(w, http.StatusOK, acc, nil)
}

func (h *AccountHandler) UnsuspendAccount(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	accID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid account ID", nil, "")
		return
	}

	acc, err := h.store.GetHostingAccountByID(r.Context(), accID)
	if err != nil || !h.authorizeAccount(r, acc) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil, "")
		return
	}

	acc.Status = store.AccountStatusActive
	acc.SuspendReason = ""
	acc.SuspendedAt = nil

	if err := h.store.UpdateHostingAccount(r.Context(), acc); err != nil {
		response.Error(w, http.StatusInternalServerError, "UNSUSPEND_FAILED", "Failed to unsuspend account", err.Error(), "")
		return
	}

	// Restore corresponding website status
	if site, err := h.findWebsiteByDomain(r.Context(), acc.OrganizationID, acc.Domain); err == nil && site != nil {
		site.Status = "active"
		_ = h.store.UpdateWebsite(r.Context(), site)
	}

	h.audit.Log(r.Context(), r, "accounts.unsuspend", "hosting_account", acc.ID.String(), "success", fmt.Sprintf("Unsuspended account %s", acc.Username), nil)

	response.JSON(w, http.StatusOK, acc, nil)
}

type ChangePlanRequest struct {
	PlanID string `json:"plan_id"`
}

func (h *AccountHandler) ChangePlan(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	accID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid account ID", nil, "")
		return
	}

	acc, err := h.store.GetHostingAccountByID(r.Context(), accID)
	if err != nil || !h.authorizeAccount(r, acc) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil, "")
		return
	}

	var req ChangePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	planUUID, err := uuid.Parse(req.PlanID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PLAN_ID", "Valid Plan ID is required", nil, "")
		return
	}

	newPlan, err := h.store.GetPlanByID(r.Context(), planUUID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "PLAN_NOT_FOUND", "New plan not found", nil, "")
		return
	}

	acc.PlanID = newPlan.ID
	acc.PlanName = newPlan.Name
	acc.DiskLimitMB = newPlan.DiskSpaceMB
	acc.BandwidthLimitMB = newPlan.BandwidthMB
	acc.WebsitesLimit = newPlan.MaxWebsites
	acc.DatabasesLimit = newPlan.MaxDatabases
	acc.MailboxesLimit = newPlan.MaxMailboxes

	// Dynamically update cgroups v2 resource limits
	if h.isolationMgr != nil {
		limits := isolation.DefaultResourceLimits()
		if newPlan.DiskSpaceMB >= 10240 {
			limits.MemoryMaxMB = 1024
			limits.MemoryHighMB = 920
		}
		_ = h.isolationMgr.UpdateResourceLimits(r.Context(), acc.Username, limits)
	}

	if err := h.store.UpdateHostingAccount(r.Context(), acc); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to change plan", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "accounts.change_plan", "hosting_account", acc.ID.String(), "success", fmt.Sprintf("Changed plan of %s to %s", acc.Username, newPlan.Name), map[string]interface{}{
		"account_id": acc.ID,
		"new_plan":   newPlan.Name,
		"disk_limit": newPlan.DiskSpaceMB,
		"bw_limit":   newPlan.BandwidthMB,
	})

	response.JSON(w, http.StatusOK, acc, nil)
}

// ----------------------------------------------------------------------------
// 1-Click Cryptographically Secure Client Login / Impersonation
// ----------------------------------------------------------------------------

func (h *AccountHandler) GenerateLoginToken(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	accID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid account ID", nil, "")
		return
	}

	acc, err := h.store.GetHostingAccountByID(r.Context(), accID)
	if err != nil || !h.authorizeAccount(r, acc) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())

	// Resolve or create customer User record for this hosting account
	var targetUser *store.User
	if acc.UserID != uuid.Nil {
		targetUser, _ = h.store.GetUserByID(r.Context(), acc.UserID)
	}
	if targetUser == nil {
		clientEmail := fmt.Sprintf("%s@%s", acc.Username, acc.Domain)
		targetUser, _ = h.store.GetUserByEmail(r.Context(), clientEmail)
		if targetUser == nil {
			newUserID := uuid.New()
			targetUser = &store.User{
				ID:           newUserID,
				DefaultOrgID: acc.OrganizationID,
				Email:        clientEmail,
				FullName:     fmt.Sprintf("Client %s", acc.Username),
				Role:         rbac.RoleCustomer,
				IsActive:     true,
				CreatedAt:    time.Now().UTC(),
				UpdatedAt:    time.Now().UTC(),
			}
			_ = h.store.CreateUser(r.Context(), targetUser, acc.OrganizationID, rbac.RoleCustomer)
			acc.UserID = targetUser.ID
			_ = h.store.UpdateHostingAccount(r.Context(), acc)
		}
	}

	// Generate short-lived (1 hour) cryptographically signed JWT impersonation tokens
	impersonationDuration := 1 * time.Hour
	tokens, err := auth.GenerateImpersonationTokenPair(
		claims.UserID, targetUser.ID, targetUser.DefaultOrgID, targetUser.Email,
		h.cfg.JWTSecret, impersonationDuration,
	)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TOKEN_FAILED", "Failed to generate cryptographically secure client session", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "admin.impersonation.started", "hosting_account", acc.ID.String(), "success", fmt.Sprintf("Admin %s started client impersonation for %s", claims.Email, acc.Username), map[string]interface{}{
		"admin_id":         claims.UserID,
		"admin_email":      claims.Email,
		"account_id":       acc.ID,
		"account_username": acc.Username,
		"client_user_id":   targetUser.ID,
		"client_email":     targetUser.Email,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"username":   acc.Username,
		"domain":     acc.Domain,
		"user":       targetUser,
		"tokens":     tokens,
		"token":      tokens.AccessToken,
		"login_url":  "/dashboard?impersonated=true",
	}, nil)
}

func (h *AccountHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	accID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid account ID", nil, "")
		return
	}

	acc, err := h.store.GetHostingAccountByID(r.Context(), accID)
	if err != nil || !h.authorizeAccount(r, acc) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil, "")
		return
	}

	// 1. Deprovision user isolation, PHP pool, and cgroup slice
	if h.isolationMgr != nil {
		_ = h.isolationMgr.DeprovisionWebsiteIsolation(r.Context(), acc.Username, acc.PHPVersion)
	}

	// 2. Clean up associated Website record
	if site, err := h.findWebsiteByDomain(r.Context(), acc.OrganizationID, acc.Domain); err == nil && site != nil {
		_ = h.store.DeleteWebsite(r.Context(), site.ID)
	}

	// 3. Delete hosting account record
	if err := h.store.DeleteHostingAccount(r.Context(), accID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", "Failed to terminate account", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "accounts.terminate", "hosting_account", idStr, "success", fmt.Sprintf("Terminated account %s (%s)", acc.Username, acc.Domain), nil)

	response.JSON(w, http.StatusOK, map[string]string{"message": "Account terminated successfully"}, nil)
}
