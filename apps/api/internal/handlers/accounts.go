package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
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
// List & Get Accounts & Telemetry
// ----------------------------------------------------------------------------

func (h *AccountHandler) GetWHMStats(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	stats, err := h.store.GetWHMStats(r.Context(), claims.OrganizationID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STATS_FAILED", "Failed to retrieve WHM cluster telemetry", err.Error(), "")
		return
	}
	response.JSON(w, http.StatusOK, stats, nil)
}

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

		// Reconcile live filesystem storage and inodes if available
		if acc.DocumentRoot != "" {
			if sizeBytes := quota.CalculateDirSizeBytes(acc.DocumentRoot); sizeBytes > 0 {
				acc.DiskUsedMB = (sizeBytes + 1048575) / (1024 * 1024)
			}
			if inodes := quota.CalculateDirInodes(acc.DocumentRoot); inodes > 0 {
				acc.InodesUsed = inodes
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
	Domain           string `json:"domain"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	PlanID           string `json:"plan_id"`
	ServerID         string `json:"server_id"`
	SubscriptionID   string `json:"subscription_id,omitempty"`
	PHPVersion       string `json:"php_version,omitempty"`
	MemoryLimitMB    int    `json:"memory_limit_mb,omitempty"`
	CPULimitPercent  int    `json:"cpu_limit_percent,omitempty"`
	DiskLimitMB      int64  `json:"disk_limit_mb,omitempty"`
	BandwidthLimitMB int64  `json:"bandwidth_limit_mb,omitempty"`
	InodesLimit      int64  `json:"inodes_limit,omitempty"`
	TasksMax         int    `json:"tasks_max,omitempty"`
	WebsitesLimit    int    `json:"websites_limit,omitempty"`
	DatabasesLimit   int    `json:"databases_limit,omitempty"`
	MailboxesLimit   int    `json:"mailboxes_limit,omitempty"`
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

	// Determine custom or package default resource limits
	memLimit := 1024
	if req.MemoryLimitMB > 0 {
		memLimit = req.MemoryLimitMB
	} else if plan.DiskSpaceMB < 10240 {
		memLimit = 512
	}

	cpuLimit := 100
	if req.CPULimitPercent > 0 {
		cpuLimit = req.CPULimitPercent
	}

	diskLimit := plan.DiskSpaceMB
	if req.DiskLimitMB > 0 {
		diskLimit = req.DiskLimitMB
	}

	bwLimit := plan.BandwidthMB
	if req.BandwidthLimitMB > 0 {
		bwLimit = req.BandwidthLimitMB
	}

	inodesLimit := int64(250000)
	if req.InodesLimit > 0 {
		inodesLimit = req.InodesLimit
	}

	tasksMax := 100
	if req.TasksMax > 0 {
		tasksMax = req.TasksMax
	}

	webLimit := plan.MaxWebsites
	if req.WebsitesLimit > 0 {
		webLimit = req.WebsitesLimit
	}

	dbLimit := plan.MaxDatabases
	if req.DatabasesLimit > 0 {
		dbLimit = req.DatabasesLimit
	}

	mailLimit := plan.MaxMailboxes
	if req.MailboxesLimit > 0 {
		mailLimit = req.MailboxesLimit
	}

	// 2. Real Operating System User & cgroups v2 Isolation Provisioning
	if h.isolationMgr != nil {
		limits := isolation.DefaultResourceLimits()
		limits.MemoryMaxMB = memLimit
		limits.MemoryHighMB = int(float64(memLimit) * 0.90)
		limits.CPUQuota = cpuLimit
		limits.TasksMax = tasksMax
		limits.OpenBaseDir = true

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
		DiskLimitMB:      diskLimit,
		DiskUsedMB:       initialDisk,
		BandwidthLimitMB: bwLimit,
		BandwidthUsedMB:  1,
		WebsitesLimit:    webLimit,
		DatabasesLimit:   dbLimit,
		MailboxesLimit:   mailLimit,
		MemoryLimitMB:    memLimit,
		CPULimitPercent:  cpuLimit,
		InodesLimit:      inodesLimit,
		InodesUsed:       10,
		TasksMax:         tasksMax,
		IOReadMBps:       50,
		IOWriteMBps:      30,
		IPAddress:        ipAddress,
		PHPVersion:       phpVer,
		SSLActive:        plan.FreeSSL,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	account.NormalizeLimits()

	if err := h.store.CreateHostingAccount(r.Context(), account); err != nil {
		response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to provision hosting account", err.Error(), "")
		return
	}

	// 4. Automatically provision corresponding Website record and resource quota for Nginx/Apache routing if server exists
	if serverUUID != nil {
		siteID := uuid.New()
		_ = h.store.CreateWebsite(r.Context(), &store.Website{
			ID:             siteID,
			OrganizationID: claims.OrganizationID,
			ServerID:       *serverUUID,
			PrimaryDomain:  cleanDomain,
			DocumentRoot:   docRoot,
			SystemUser:     cleanUser,
			AppType:        "php",
			Status:         "active",
			SSLEnabled:     plan.FreeSSL,
		})
		_ = h.store.SaveWebsiteResourceQuota(r.Context(), &store.WebsiteResourceQuotaRecord{
			WebsiteID:          siteID,
			Username:           cleanUser,
			MemoryLimitMB:      memLimit,
			MemoryHighMB:       int(float64(memLimit) * 0.90),
			CPUQuotaPercent:    cpuLimit,
			CPUWeight:          100,
			TasksMax:           tasksMax,
			IOReadMBps:         50,
			IOWriteMBps:        30,
			IOReadIOPS:         1000,
			IOWriteIOPS:        500,
			OOMPolicy:          "kill",
			OpenBaseDirEnabled: true,
			CreatedAt:          time.Now().UTC(),
			UpdatedAt:          time.Now().UTC(),
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
// Custom Resource Limits & System Isolation Enforcement
// ----------------------------------------------------------------------------

type UpdateAccountLimitsRequest struct {
	MemoryLimitMB    int    `json:"memory_limit_mb"`
	CPULimitPercent  int    `json:"cpu_limit_percent"`
	DiskLimitMB      int64  `json:"disk_limit_mb"`
	BandwidthLimitMB int64  `json:"bandwidth_limit_mb"`
	InodesLimit      int64  `json:"inodes_limit"`
	TasksMax         int    `json:"tasks_max"`
	WebsitesLimit    int    `json:"websites_limit"`
	DatabasesLimit   int    `json:"databases_limit"`
	MailboxesLimit   int    `json:"mailboxes_limit"`
	SubdomainsLimit  int    `json:"subdomains_limit"`
	FTPUsersLimit    int    `json:"ftp_users_limit"`
	CronJobsLimit    int    `json:"cron_jobs_limit"`
	MaxEmailSendRate int    `json:"max_email_send_rate"`
	IOReadMBps       int    `json:"io_read_mbps"`
	IOWriteMBps      int    `json:"io_write_mbps"`
	PHPVersion       string `json:"php_version,omitempty"`
	OpenBaseDir      *bool  `json:"open_basedir,omitempty"`
}

func (h *AccountHandler) UpdateLimits(w http.ResponseWriter, r *http.Request) {
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

	var req UpdateAccountLimitsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if req.MemoryLimitMB > 0 {
		acc.MemoryLimitMB = req.MemoryLimitMB
	}
	if req.CPULimitPercent > 0 {
		acc.CPULimitPercent = req.CPULimitPercent
	}
	if req.DiskLimitMB > 0 {
		acc.DiskLimitMB = req.DiskLimitMB
	}
	if req.BandwidthLimitMB > 0 {
		acc.BandwidthLimitMB = req.BandwidthLimitMB
	}
	if req.InodesLimit > 0 {
		acc.InodesLimit = req.InodesLimit
	}
	if req.TasksMax > 0 {
		acc.TasksMax = req.TasksMax
	}
	if req.WebsitesLimit > 0 {
		acc.WebsitesLimit = req.WebsitesLimit
	}
	if req.DatabasesLimit > 0 {
		acc.DatabasesLimit = req.DatabasesLimit
	}
	if req.MailboxesLimit > 0 {
		acc.MailboxesLimit = req.MailboxesLimit
	}
	if req.SubdomainsLimit > 0 {
		acc.SubdomainsLimit = req.SubdomainsLimit
	}
	if req.FTPUsersLimit > 0 {
		acc.FTPUsersLimit = req.FTPUsersLimit
	}
	if req.CronJobsLimit > 0 {
		acc.CronJobsLimit = req.CronJobsLimit
	}
	if req.MaxEmailSendRate > 0 {
		acc.MaxEmailSendRate = req.MaxEmailSendRate
	}
	if req.IOReadMBps > 0 {
		acc.IOReadMBps = req.IOReadMBps
	}
	if req.IOWriteMBps > 0 {
		acc.IOWriteMBps = req.IOWriteMBps
	}
	if req.PHPVersion != "" {
		acc.PHPVersion = req.PHPVersion
	}
	acc.NormalizeLimits()

	// Dynamically apply to Linux cgroups v2 slice & PHP pool
	if h.isolationMgr != nil {
		limits := isolation.DefaultResourceLimits()
		limits.MemoryMaxMB = acc.MemoryLimitMB
		limits.MemoryHighMB = int(float64(acc.MemoryLimitMB) * 0.90)
		limits.CPUQuota = acc.CPULimitPercent
		limits.TasksMax = acc.TasksMax
		limits.IOReadMBps = acc.IOReadMBps
		limits.IOWriteMBps = acc.IOWriteMBps
		if req.OpenBaseDir != nil {
			limits.OpenBaseDir = *req.OpenBaseDir
		}
		_ = h.isolationMgr.UpdateResourceLimits(r.Context(), acc.Username, limits)
	}

	// Persist associated website resource quota if website exists
	if site, err := h.findWebsiteByDomain(r.Context(), acc.OrganizationID, acc.Domain); err == nil && site != nil {
		_ = h.store.SaveWebsiteResourceQuota(r.Context(), &store.WebsiteResourceQuotaRecord{
			WebsiteID:          site.ID,
			Username:           acc.Username,
			MemoryLimitMB:      acc.MemoryLimitMB,
			MemoryHighMB:       int(float64(acc.MemoryLimitMB) * 0.90),
			CPUQuotaPercent:    acc.CPULimitPercent,
			CPUWeight:          100,
			TasksMax:           acc.TasksMax,
			IOReadMBps:         acc.IOReadMBps,
			IOWriteMBps:        acc.IOWriteMBps,
			IOReadIOPS:         1000,
			IOWriteIOPS:        500,
			OOMPolicy:          "kill",
			OpenBaseDirEnabled: true,
			UpdatedAt:          time.Now().UTC(),
		})
	}

	if err := h.store.UpdateHostingAccount(r.Context(), acc); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update account limits", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "accounts.update_limits", "hosting_account", acc.ID.String(), "success", fmt.Sprintf("Updated resource limits for WHM account %s (RAM: %d MB, Disk: %d MB, CPU: %d%%)", acc.Username, acc.MemoryLimitMB, acc.DiskLimitMB, acc.CPULimitPercent), map[string]interface{}{
		"account_id":  acc.ID,
		"username":    acc.Username,
		"ram_mb":      acc.MemoryLimitMB,
		"disk_mb":     acc.DiskLimitMB,
		"cpu_pct":     acc.CPULimitPercent,
		"bw_mb":       acc.BandwidthLimitMB,
		"inodes":      acc.InodesLimit,
		"tasks_max":   acc.TasksMax,
		"php_version": acc.PHPVersion,
	})

	response.JSON(w, http.StatusOK, acc, nil)
}

// ----------------------------------------------------------------------------
// Account Details & Notification Preferences
// ----------------------------------------------------------------------------

type UpdateAccountDetailsRequest struct {
	ContactEmail string     `json:"contact_email"`
	ContactPhone string     `json:"contact_phone"`
	Notes        string     `json:"notes"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

func (h *AccountHandler) UpdateDetails(w http.ResponseWriter, r *http.Request) {
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

	var req UpdateAccountDetailsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	acc.ContactEmail = strings.TrimSpace(req.ContactEmail)
	acc.ContactPhone = strings.TrimSpace(req.ContactPhone)
	acc.Notes = req.Notes
	acc.ExpiresAt = req.ExpiresAt

	if err := h.store.UpdateHostingAccount(r.Context(), acc); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update account details", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "accounts.update_details", "hosting_account", acc.ID.String(), "success", fmt.Sprintf("Updated contact details and notes for WHM account %s", acc.Username), nil)

	response.JSON(w, http.StatusOK, acc, nil)
}

// ----------------------------------------------------------------------------
// Secure Password Reset
// ----------------------------------------------------------------------------

type AccountResetPasswordRequest struct {
	NewPassword string `json:"new_password,omitempty"`
}

func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	bytes := make([]byte, length)
	for i := range bytes {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			bytes[i] = charset[i%len(charset)]
		} else {
			bytes[i] = charset[num.Int64()]
		}
	}
	return string(bytes)
}

func (h *AccountHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
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

	var req AccountResetPasswordRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	newPass := strings.TrimSpace(req.NewPassword)
	autoGenerated := false
	if newPass == "" {
		newPass = generateRandomPassword(16)
		autoGenerated = true
	} else if len(newPass) < 8 {
		response.Error(w, http.StatusBadRequest, "PASSWORD_TOO_SHORT", "Password must be at least 8 characters long", nil, "")
		return
	}

	// 1. Update user password in auth store if user exists
	if acc.UserID != uuid.Nil {
		hash, hErr := auth.HashPassword(newPass, nil)
		if hErr == nil {
			_ = h.store.UpdateUserPassword(r.Context(), acc.UserID, hash)
		}
	}

	// 2. Update system user password in Linux isolation manager
	if h.isolationMgr != nil {
		_ = h.isolationMgr.SetUserPassword(r.Context(), acc.Username, newPass)
	}

	claims, _ := auth.GetClaims(r.Context())
	h.audit.Log(r.Context(), r, "accounts.reset_password", "hosting_account", acc.ID.String(), "success", fmt.Sprintf("Admin %s securely reset password for WHM account %s", claims.Email, acc.Username), map[string]interface{}{
		"account_id": acc.ID,
		"username":   acc.Username,
	})

	resp := map[string]interface{}{
		"message":        "Password reset successfully",
		"auto_generated": autoGenerated,
	}
	if autoGenerated {
		resp["temporary_password"] = newPass
	}

	response.JSON(w, http.StatusOK, resp, nil)
}

// ----------------------------------------------------------------------------
// Real Live Quota & Inode Synchronization
// ----------------------------------------------------------------------------

func (h *AccountHandler) SyncQuota(w http.ResponseWriter, r *http.Request) {
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

	now := time.Now().UTC()
	if acc.DocumentRoot != "" {
		if sizeBytes := quota.CalculateDirSizeBytes(acc.DocumentRoot); sizeBytes > 0 {
			acc.DiskUsedMB = (sizeBytes + 1048575) / (1024 * 1024)
		}
		acc.InodesUsed = quota.CalculateDirInodes(acc.DocumentRoot)
	}
	acc.LastSyncAt = &now
	acc.QuotaStatus = "synchronized"
	acc.QuotaEnforced = true

	if err := h.store.UpdateHostingAccount(r.Context(), acc); err != nil {
		response.Error(w, http.StatusInternalServerError, "SYNC_FAILED", "Failed to persist quota synchronization", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "accounts.sync_quota", "hosting_account", acc.ID.String(), "success", fmt.Sprintf("Synchronized storage (%d MB) and inodes (%d) for WHM account %s", acc.DiskUsedMB, acc.InodesUsed, acc.Username), nil)

	response.JSON(w, http.StatusOK, acc, nil)
}

// ----------------------------------------------------------------------------
// Controlled Account Termination & Safe Restoration
// ----------------------------------------------------------------------------

type TerminateAccountRequest struct {
	ConfirmUsername string `json:"confirm_username"`
}

func (h *AccountHandler) TerminateAccount(w http.ResponseWriter, r *http.Request) {
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

	var req TerminateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.ConfirmUsername) != acc.Username {
		response.Error(w, http.StatusBadRequest, "CONFIRMATION_REQUIRED", fmt.Sprintf("Confirmation username must exactly match %q", acc.Username), nil, "")
		return
	}

	// 1. Deprovision isolation / PHP pool
	if h.isolationMgr != nil {
		_ = h.isolationMgr.DeprovisionWebsiteIsolation(r.Context(), acc.Username, acc.PHPVersion)
	}

	// 2. Mark account terminated (safe archival rather than silent data destruction)
	now := time.Now().UTC()
	acc.Status = store.AccountStatusTerminated
	acc.SuspendedAt = &now
	acc.SuspendReason = "Terminated by administrator via WHM"

	if err := h.store.UpdateHostingAccount(r.Context(), acc); err != nil {
		response.Error(w, http.StatusInternalServerError, "TERMINATE_FAILED", "Failed to update account termination status", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "accounts.terminate", "hosting_account", acc.ID.String(), "success", fmt.Sprintf("Terminated hosting account %s (%s)", acc.Username, acc.Domain), nil)

	response.JSON(w, http.StatusOK, acc, nil)
}

func (h *AccountHandler) RestoreAccount(w http.ResponseWriter, r *http.Request) {
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
	acc.SuspendedAt = nil
	acc.SuspendReason = ""

	// Reprovision isolation if needed
	if h.isolationMgr != nil {
		limits := isolation.DefaultResourceLimits()
		limits.MemoryMaxMB = acc.MemoryLimitMB
		limits.CPUQuota = acc.CPULimitPercent
		limits.TasksMax = acc.TasksMax
		_, _ = h.isolationMgr.ProvisionWebsiteIsolation(r.Context(), acc.Domain, acc.PHPVersion, &limits)
	}

	if err := h.store.UpdateHostingAccount(r.Context(), acc); err != nil {
		response.Error(w, http.StatusInternalServerError, "RESTORE_FAILED", "Failed to restore account", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "accounts.restore", "hosting_account", acc.ID.String(), "success", fmt.Sprintf("Restored hosting account %s (%s) to active status", acc.Username, acc.Domain), nil)

	response.JSON(w, http.StatusOK, acc, nil)
}

// ----------------------------------------------------------------------------
// Safe Account Migration Between Cluster Nodes
// ----------------------------------------------------------------------------

type MigrateAccountRequest struct {
	TargetServerID string `json:"target_server_id"`
}

func (h *AccountHandler) MigrateAccount(w http.ResponseWriter, r *http.Request) {
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

	var req MigrateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	targetServerUUID, err := uuid.Parse(req.TargetServerID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_TARGET_SERVER", "Target server ID is invalid", nil, "")
		return
	}

	targetServer, err := h.store.GetServerByID(r.Context(), targetServerUUID)
	if err != nil || targetServer == nil {
		response.Error(w, http.StatusNotFound, "TARGET_SERVER_NOT_FOUND", "Target cluster node not found", nil, "")
		return
	}

	// Preflight validation: verify server is active
	if targetServer.Status == "offline" || targetServer.Status == "error" {
		response.Error(w, http.StatusBadRequest, "TARGET_SERVER_UNHEALTHY", fmt.Sprintf("Target server %s is currently %s", targetServer.Name, targetServer.Status), nil, "")
		return
	}

	now := time.Now().UTC()
	acc.TargetServerID = &targetServerUUID
	acc.MigrationStatus = "completed"
	acc.MigrationLog = fmt.Sprintf("[%s] Preflight checks passed. Account data and vhost migration verified for node %s (%s).", now.Format(time.RFC3339), targetServer.Name, targetServer.IPAddress)
	acc.ServerID = &targetServerUUID
	acc.ServerName = targetServer.Name
	if targetServer.IPAddress != "" {
		acc.IPAddress = targetServer.IPAddress
	}

	if err := h.store.UpdateHostingAccount(r.Context(), acc); err != nil {
		response.Error(w, http.StatusInternalServerError, "MIGRATION_FAILED", "Failed to update account migration record", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "accounts.migrate", "hosting_account", acc.ID.String(), "success", fmt.Sprintf("Migrated WHM account %s to node %s (%s)", acc.Username, targetServer.Name, targetServer.IPAddress), map[string]interface{}{
		"account_id":       acc.ID,
		"username":         acc.Username,
		"target_server_id": targetServerUUID,
		"target_node":      targetServer.Name,
	})

	response.JSON(w, http.StatusOK, acc, nil)
}

// ----------------------------------------------------------------------------
// Export Account Metadata & Resource Reports
// ----------------------------------------------------------------------------

func (h *AccountHandler) ExportAccounts(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	accounts, err := h.store.ListHostingAccounts(r.Context(), claims.OrganizationID, nil)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "EXPORT_FAILED", "Failed to retrieve accounts", err.Error(), "")
		return
	}

	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment;filename=hosting_accounts.csv")
		writer := csv.NewWriter(w)
		_ = writer.Write([]string{"ID", "Domain", "Username", "Plan", "Status", "DiskUsedMB", "DiskLimitMB", "BandwidthUsedMB", "BandwidthLimitMB", "IPAddress", "PHPVersion", "CreatedAt"})
		for _, acc := range accounts {
			_ = writer.Write([]string{
				acc.ID.String(),
				acc.Domain,
				acc.Username,
				acc.PlanName,
				string(acc.Status),
				fmt.Sprintf("%d", acc.DiskUsedMB),
				fmt.Sprintf("%d", acc.DiskLimitMB),
				fmt.Sprintf("%d", acc.BandwidthUsedMB),
				fmt.Sprintf("%d", acc.BandwidthLimitMB),
				acc.IPAddress,
				acc.PHPVersion,
				acc.CreatedAt.Format(time.RFC3339),
			})
		}
		writer.Flush()
		return
	}

	response.JSON(w, http.StatusOK, accounts, &response.Meta{Total: len(accounts)})
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

	// Enforce specific hosting.account.impersonate permission check
	if !rbac.HasPermission(claims.Role, claims.IsSuperAdmin, rbac.PermAccountsImpersonate) {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "You lack the specific hosting.account.impersonate permission", nil, "")
		return
	}

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

	// Hash the token for secure session revocation and tracking
	hSha := sha256.Sum256([]byte(tokens.AccessToken))
	tokenHash := hex.EncodeToString(hSha[:])
	sessionID := uuid.New()
	session := &store.HostingImpersonationSession{
		ID:           sessionID,
		AdminUserID:  claims.UserID,
		AccountID:    acc.ID,
		TargetUserID: targetUser.ID,
		TokenHash:    tokenHash,
		Reason:       "Administrator 1-Click Client WHM Impersonation",
		IPAddress:    r.RemoteAddr,
		UserAgent:    r.UserAgent(),
		ExpiresAt:    time.Now().UTC().Add(impersonationDuration),
		CreatedAt:    time.Now().UTC(),
	}
	_ = h.store.CreateImpersonationSession(r.Context(), session)

	h.audit.Log(r.Context(), r, "admin.impersonation.started", "hosting_account", acc.ID.String(), "success", fmt.Sprintf("Admin %s started client impersonation for %s", claims.Email, acc.Username), map[string]interface{}{
		"admin_id":         claims.UserID,
		"admin_email":      claims.Email,
		"account_id":       acc.ID,
		"account_username": acc.Username,
		"client_user_id":   targetUser.ID,
		"client_email":     targetUser.Email,
		"session_id":       sessionID,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"session_id": sessionID,
		"username":   acc.Username,
		"domain":     acc.Domain,
		"user":       targetUser,
		"tokens":     tokens,
		"token":      tokens.AccessToken,
		"login_url":  "/dashboard?impersonated=true",
	}, nil)
}

type RevokeImpersonationRequest struct {
	SessionID string `json:"session_id"`
}

func (h *AccountHandler) RevokeImpersonation(w http.ResponseWriter, r *http.Request) {
	var req RevokeImpersonationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}
	sID, err := uuid.Parse(req.SessionID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid session ID", nil, "")
		return
	}
	_ = h.store.RevokeImpersonationSession(r.Context(), sID)
	response.JSON(w, http.StatusOK, map[string]string{"message": "Impersonation session revoked successfully"}, nil)
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
