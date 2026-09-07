package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type AccountHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewAccountHandler(cfg *config.Config, s store.Store, a *audit.Logger) *AccountHandler {
	return &AccountHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
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
		filtered = append(filtered, acc)
	}

	response.JSON(w, http.StatusOK, filtered, &response.Meta{Total: len(filtered)})
}

func (h *AccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	accID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid account ID", nil, "")
		return
	}

	acc, err := h.store.GetHostingAccountByID(r.Context(), accID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Hosting account not found", nil, "")
		return
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

	var serverUUID *uuid.UUID
	var serverName string
	var ipAddress = "127.0.0.1"
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

	account := &store.HostingAccount{
		ID:               accID,
		OrganizationID:   claims.OrganizationID,
		UserID:           claims.UserID,
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
		DiskUsedMB:       250, // Initial boilerplate files
		BandwidthLimitMB: plan.BandwidthMB,
		BandwidthUsedMB:  50,
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

	// Also automatically provision a corresponding Website record for Nginx/Apache routing if server exists
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

	h.audit.Log(r.Context(), r, "accounts.provision", "hosting_account", account.ID.String(), "success", fmt.Sprintf("Provisioned account %s for domain %s", cleanUser, cleanDomain), nil)

	response.JSON(w, http.StatusCreated, account, nil)
}

// ----------------------------------------------------------------------------
// Suspend / Unsuspend / Change Plan
// ----------------------------------------------------------------------------

type SuspendRequest struct {
	Reason string `json:"reason"`
}

func (h *AccountHandler) SuspendAccount(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	accID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid account ID", nil, "")
		return
	}

	acc, err := h.store.GetHostingAccountByID(r.Context(), accID)
	if err != nil {
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
	if err != nil {
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
	if err != nil {
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

	if err := h.store.UpdateHostingAccount(r.Context(), acc); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to change plan", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "accounts.change_plan", "hosting_account", acc.ID.String(), "success", fmt.Sprintf("Changed plan of %s to %s", acc.Username, newPlan.Name), nil)

	response.JSON(w, http.StatusOK, acc, nil)
}

func (h *AccountHandler) GenerateLoginToken(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	accID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid account ID", nil, "")
		return
	}

	acc, err := h.store.GetHostingAccountByID(r.Context(), accID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil, "")
		return
	}

	token := fmt.Sprintf("session_cpanel_%s_%d", acc.Username, time.Now().Unix())
	loginURL := fmt.Sprintf("/dashboard?client_impersonation=%s&token=%s", acc.Username, token)

	response.JSON(w, http.StatusOK, map[string]string{
		"username":  acc.Username,
		"domain":    acc.Domain,
		"token":     token,
		"login_url": loginURL,
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
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil, "")
		return
	}

	if err := h.store.DeleteHostingAccount(r.Context(), accID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", "Failed to terminate account", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "accounts.terminate", "hosting_account", idStr, "success", fmt.Sprintf("Terminated account %s (%s)", acc.Username, acc.Domain), nil)

	response.JSON(w, http.StatusOK, map[string]string{"message": "Account terminated successfully"}, nil)
}
