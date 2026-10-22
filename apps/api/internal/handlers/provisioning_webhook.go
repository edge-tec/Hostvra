package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/rbac"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type ProvisioningHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewProvisioningHandler(cfg *config.Config, s store.Store, a *audit.Logger) *ProvisioningHandler {
	return &ProvisioningHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
}

// ----------------------------------------------------------------------------
// Provisioning Webhook Request Payloads & Normalization
// ----------------------------------------------------------------------------

type ProvisioningWebhookPayload struct {
	Action         string            `json:"action"` // create_account, suspend, unsuspend, terminate, change_package, ping
	Provider       string            `json:"provider"`
	Domain         string            `json:"domain"`
	Username       string            `json:"username"`
	Password       string            `json:"password"`
	Email          string            `json:"email"`
	Plan           string            `json:"plan"`
	PlanID         string            `json:"plan_id"`
	ServerID       string            `json:"server_id"`
	ResellerID     string            `json:"reseller_id"`
	Reason         string            `json:"reason"`
	ExtraParams    map[string]string `json:"extra_params,omitempty"`
}

// ----------------------------------------------------------------------------
// Webhook Endpoint (WHMCS, Blesta, Custom)
// ----------------------------------------------------------------------------

func (h *ProvisioningHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	providerParam := chi.URLParam(r, "provider")
	if providerParam == "" {
		providerParam = r.URL.Query().Get("provider")
	}
	if providerParam == "" {
		providerParam = "whmcs"
	}
	providerParam = strings.ToLower(strings.TrimSpace(providerParam))

	// Read and buffer body for HMAC verification and parsing
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.logAndRespondError(w, r, nil, providerParam, "unknown", "", "", http.StatusBadRequest, "Failed to read request body", err.Error())
		return
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Verify HMAC-SHA256 signature or Secret Token
	matchedIntegration, authErr := h.verifyWebhookAuth(r, bodyBytes, providerParam)
	if authErr != nil {
		h.logAndRespondError(w, r, matchedIntegration, providerParam, "auth", "", "", http.StatusUnauthorized, "Webhook authentication failed", authErr.Error())
		return
	}

	// Parse payload (supports JSON and application/x-www-form-urlencoded)
	payload, parseErr := h.parseWebhookPayload(r, bodyBytes)
	if parseErr != nil {
		h.logAndRespondError(w, r, matchedIntegration, providerParam, "parse", "", "", http.StatusBadRequest, "Invalid payload format", parseErr.Error())
		return
	}

	// Dispatch action
	var result interface{}
	var execErr error
	var httpStatus = http.StatusOK

	switch strings.ToLower(payload.Action) {
	case "ping", "test_connection", "test":
		result = map[string]interface{}{
			"status":    "success",
			"message":   "Hostvra Provisioning API connection verified",
			"provider":  providerParam,
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0",
		}

	case "create_account", "create", "provision":
		result, execErr = h.executeCreateAccount(r, matchedIntegration, payload)
		if execErr == nil {
			httpStatus = http.StatusCreated
		}

	case "suspend":
		result, execErr = h.executeSuspendAccount(r, payload)

	case "unsuspend":
		result, execErr = h.executeUnsuspendAccount(r, payload)

	case "terminate", "delete", "destroy":
		result, execErr = h.executeTerminateAccount(r, payload)

	case "change_package", "change_plan", "upgrade", "downgrade":
		result, execErr = h.executeChangePackage(r, payload)

	default:
		execErr = fmt.Errorf("unsupported provisioning action: %s", payload.Action)
		httpStatus = http.StatusBadRequest
	}

	if execErr != nil {
		status := httpStatus
		if status == http.StatusOK {
			status = http.StatusBadRequest
		}
		h.logAndRespondError(w, r, matchedIntegration, providerParam, payload.Action, payload.Username, payload.Domain, status, "Provisioning action failed", execErr.Error())
		return
	}

	// Log success and return response
	respBytes, _ := json.Marshal(result)
	var integrationID *uuid.UUID
	var orgID uuid.UUID
	if matchedIntegration != nil {
		integrationID = &matchedIntegration.ID
		orgID = matchedIntegration.OrganizationID
	}

	_ = h.store.LogProvisioningWebhook(r.Context(), &store.ProvisioningWebhookLog{
		ID:             uuid.New(),
		OrganizationID: orgID,
		IntegrationID:  integrationID,
		Provider:       providerParam,
		Action:         payload.Action,
		Username:       payload.Username,
		Domain:         payload.Domain,
		Payload:        payload,
		Status:         "success",
		HTTPStatus:     httpStatus,
		ResponseBody:   string(respBytes),
		IPAddress:      r.RemoteAddr,
		CreatedAt:      time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_, _ = w.Write(respBytes)
}

// ----------------------------------------------------------------------------
// Action Executors
// ----------------------------------------------------------------------------

func (h *ProvisioningHandler) executeCreateAccount(r *http.Request, integration *store.ProvisioningIntegration, p *ProvisioningWebhookPayload) (interface{}, error) {
	ctx := r.Context()
	cleanDomain := strings.ToLower(strings.TrimSpace(p.Domain))
	cleanUser := strings.ToLower(strings.TrimSpace(p.Username))

	if cleanDomain == "" {
		return nil, fmt.Errorf("domain is required for account creation")
	}

	if cleanUser == "" {
		cleanUser = "c_" + strings.ReplaceAll(strings.Split(cleanDomain, ".")[0], "-", "")
		if len(cleanUser) > 16 {
			cleanUser = cleanUser[:16]
		}
	}

	var orgID uuid.UUID
	if integration != nil {
		orgID = integration.OrganizationID
	}
	if orgID == uuid.Nil {
		// Fallback to primary organization from users
		if users, err := h.store.ListUsers(ctx); err == nil && len(users) > 0 {
			orgID = users[0].DefaultOrgID
		}
	}

	// Check if account already exists
	existingAccounts, _ := h.store.ListHostingAccounts(ctx, orgID, nil)
	for _, acc := range existingAccounts {
		if strings.EqualFold(acc.Domain, cleanDomain) || strings.EqualFold(acc.Username, cleanUser) {
			return nil, fmt.Errorf("hosting account with domain %s or username %s already exists", cleanDomain, cleanUser)
		}
	}

	// Resolve Plan
	var plan *store.HostingPlan
	if p.PlanID != "" {
		if planUUID, err := uuid.Parse(p.PlanID); err == nil {
			plan, _ = h.store.GetPlanByID(ctx, planUUID)
		}
	}
	if plan == nil && p.Plan != "" {
		plans, _ := h.store.ListPlans(ctx)
		for _, pl := range plans {
			if strings.EqualFold(pl.Name, p.Plan) || strings.EqualFold(pl.Slug, p.Plan) {
				plan = pl
				break
			}
		}
	}
	if plan == nil && integration != nil && integration.DefaultPlanID != nil {
		plan, _ = h.store.GetPlanByID(ctx, *integration.DefaultPlanID)
	}
	if plan == nil {
		plans, _ := h.store.ListPlans(ctx)
		if len(plans) > 0 {
			plan = plans[0]
		}
	}
	if plan == nil {
		return nil, fmt.Errorf("no valid hosting plan found to provision account")
	}

	// Resolve Server
	var serverID *uuid.UUID
	var serverName = "Default Node"
	var ipAddress = "127.0.0.1"

	if p.ServerID != "" {
		if sUUID, err := uuid.Parse(p.ServerID); err == nil {
			serverID = &sUUID
		}
	}
	if serverID == nil && integration != nil && integration.DefaultServerID != nil {
		serverID = integration.DefaultServerID
	}
	if serverID == nil {
		servers, _ := h.store.ListServersByOrg(ctx, orgID)
		for _, s := range servers {
			if s.Status == "online" && !s.Draining && !s.IsMaintenance {
				serverID = &s.ID
				serverName = s.Name
				if s.IPAddress != "" {
					ipAddress = s.IPAddress
				}
				break
			}
		}
	} else {
		if s, err := h.store.GetServerByID(ctx, *serverID); err == nil {
			serverName = s.Name
			if s.IPAddress != "" {
				ipAddress = s.IPAddress
			}
		}
	}

	// Enforce Reseller Quotas if provisioned under a Reseller
	var resellerUUID *uuid.UUID
	if p.ResellerID != "" {
		if rUUID, err := uuid.Parse(p.ResellerID); err == nil {
			resellerUUID = &rUUID
		}
	}

	if resellerUUID != nil {
		resellerProfile, err := h.store.GetResellerProfileByUserID(ctx, *resellerUUID)
		if err != nil {
			// Try by profile ID
			resellerProfile, _ = h.store.GetResellerProfileByID(ctx, *resellerUUID)
		}

		if resellerProfile != nil {
			if !resellerProfile.IsActive {
				return nil, fmt.Errorf("reseller account is inactive or suspended")
			}

			usage, err := h.store.GetResellerUsage(ctx, resellerProfile.UserID)
			if err == nil && usage != nil {
				if usage.MaxAccounts > 0 && usage.UsedAccounts >= usage.MaxAccounts {
					return nil, fmt.Errorf("reseller max accounts quota (%d) exceeded", usage.MaxAccounts)
				}
				if !resellerProfile.AllowOverselling {
					if resellerProfile.AllocatedDiskMB > 0 && (usage.UsedDiskMB+plan.DiskSpaceMB) > resellerProfile.AllocatedDiskMB {
						return nil, fmt.Errorf("reseller allocated disk space pool (%d MB) exceeded", resellerProfile.AllocatedDiskMB)
					}
					if resellerProfile.AllocatedBandwidthMB > 0 && (usage.UsedBandwidthMB+plan.BandwidthMB) > resellerProfile.AllocatedBandwidthMB {
						return nil, fmt.Errorf("reseller allocated bandwidth pool (%d MB) exceeded", resellerProfile.AllocatedBandwidthMB)
					}
				}
			}
		}
	}

	// Create or resolve client user
	clientEmail := strings.ToLower(strings.TrimSpace(p.Email))
	if clientEmail == "" {
		clientEmail = fmt.Sprintf("%s@%s", cleanUser, cleanDomain)
	}

	var clientUser *store.User
	users, _ := h.store.ListUsers(ctx)
	for _, u := range users {
		if strings.EqualFold(u.Email, clientEmail) {
			clientUser = u
			break
		}
	}

	if clientUser == nil {
		clientUser = &store.User{
			ID:               uuid.New(),
			Email:            clientEmail,
			FullName:         cleanUser,
			IsActive:         true,
			EmailVerified:    true,
			Role:             rbac.RoleCustomer,
			DefaultOrgID:     orgID,
			ResellerID:       resellerUUID,
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		}
		_ = h.store.CreateUser(ctx, clientUser, orgID, rbac.RoleCustomer)
	}

	docRoot := fmt.Sprintf("/home/%s/public_html", cleanUser)
	accID := uuid.New()

	account := &store.HostingAccount{
		ID:               accID,
		OrganizationID:   orgID,
		UserID:           clientUser.ID,
		ServerID:         serverID,
		ServerName:       serverName,
		Domain:           cleanDomain,
		Username:         cleanUser,
		DocumentRoot:     docRoot,
		PlanID:           plan.ID,
		PlanName:         plan.Name,
		Status:           store.AccountStatusActive,
		DiskLimitMB:      plan.DiskSpaceMB,
		DiskUsedMB:       250, // base initial files
		BandwidthLimitMB: plan.BandwidthMB,
		BandwidthUsedMB:  10,
		WebsitesLimit:    plan.MaxWebsites,
		DatabasesLimit:   plan.MaxDatabases,
		MailboxesLimit:   plan.MaxMailboxes,
		IPAddress:        ipAddress,
		PHPVersion:       "8.3",
		SSLActive:        plan.FreeSSL,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	if err := h.store.CreateHostingAccount(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to save hosting account: %w", err)
	}

	// Auto-provision corresponding website record for routing
	if serverID != nil {
		_ = h.store.CreateWebsite(ctx, &store.Website{
			ID:             uuid.New(),
			OrganizationID: orgID,
			ServerID:       *serverID,
			PrimaryDomain:  cleanDomain,
			DocumentRoot:   docRoot,
			SystemUser:     cleanUser,
			AppType:        "php",
			Status:         "active",
			SSLEnabled:     plan.FreeSSL,
		})
	}

	return map[string]interface{}{
		"status":          "success",
		"message":         "Account successfully provisioned",
		"account_id":      account.ID,
		"domain":          account.Domain,
		"username":        account.Username,
		"server_id":       account.ServerID,
		"server_name":     account.ServerName,
		"ip_address":      account.IPAddress,
		"plan_name":       account.PlanName,
		"disk_limit_mb":   account.DiskLimitMB,
		"bandwidth_limit": account.BandwidthLimitMB,
		"client_email":    clientEmail,
		"created_at":      account.CreatedAt,
	}, nil
}

func (h *ProvisioningHandler) executeSuspendAccount(r *http.Request, p *ProvisioningWebhookPayload) (interface{}, error) {
	ctx := r.Context()
	acc, err := h.findAccountByTarget(ctx, p.Domain, p.Username)
	if err != nil {
		return nil, err
	}

	reason := p.Reason
	if reason == "" {
		reason = "Automated Billing Suspension"
	}

	acc.Status = store.AccountStatusSuspended
	acc.SuspendReason = reason
	now := time.Now().UTC()
	acc.SuspendedAt = &now
	acc.UpdatedAt = now

	if err := h.store.UpdateHostingAccount(ctx, acc); err != nil {
		return nil, fmt.Errorf("failed to suspend account: %w", err)
	}

	return map[string]interface{}{
		"status":     "success",
		"message":    fmt.Sprintf("Account %s (%s) suspended successfully", acc.Username, acc.Domain),
		"account_id": acc.ID,
		"reason":     reason,
	}, nil
}

func (h *ProvisioningHandler) executeUnsuspendAccount(r *http.Request, p *ProvisioningWebhookPayload) (interface{}, error) {
	ctx := r.Context()
	acc, err := h.findAccountByTarget(ctx, p.Domain, p.Username)
	if err != nil {
		return nil, err
	}

	acc.Status = store.AccountStatusActive
	acc.SuspendReason = ""
	acc.SuspendedAt = nil
	acc.UpdatedAt = time.Now().UTC()

	if err := h.store.UpdateHostingAccount(ctx, acc); err != nil {
		return nil, fmt.Errorf("failed to unsuspend account: %w", err)
	}

	return map[string]interface{}{
		"status":     "success",
		"message":    fmt.Sprintf("Account %s (%s) unsuspended successfully", acc.Username, acc.Domain),
		"account_id": acc.ID,
	}, nil
}

func (h *ProvisioningHandler) executeTerminateAccount(r *http.Request, p *ProvisioningWebhookPayload) (interface{}, error) {
	ctx := r.Context()
	acc, err := h.findAccountByTarget(ctx, p.Domain, p.Username)
	if err != nil {
		return nil, err
	}

	if err := h.store.DeleteHostingAccount(ctx, acc.ID); err != nil {
		return nil, fmt.Errorf("failed to delete hosting account: %w", err)
	}

	// Cleanup website record if one exists
	websites, _ := h.store.ListWebsitesByOrg(ctx, acc.OrganizationID)
	for _, ws := range websites {
		if strings.EqualFold(ws.PrimaryDomain, acc.Domain) || ws.SystemUser == acc.Username {
			_ = h.store.DeleteWebsite(ctx, ws.ID)
		}
	}

	return map[string]interface{}{
		"status":     "success",
		"message":    fmt.Sprintf("Account %s (%s) terminated successfully", acc.Username, acc.Domain),
		"account_id": acc.ID,
	}, nil
}

func (h *ProvisioningHandler) executeChangePackage(r *http.Request, p *ProvisioningWebhookPayload) (interface{}, error) {
	ctx := r.Context()
	acc, err := h.findAccountByTarget(ctx, p.Domain, p.Username)
	if err != nil {
		return nil, err
	}

	var newPlan *store.HostingPlan
	if p.PlanID != "" {
		if planUUID, err := uuid.Parse(p.PlanID); err == nil {
			newPlan, _ = h.store.GetPlanByID(ctx, planUUID)
		}
	}
	if newPlan == nil && p.Plan != "" {
		plans, _ := h.store.ListPlans(ctx)
		for _, pl := range plans {
			if strings.EqualFold(pl.Name, p.Plan) || strings.EqualFold(pl.Slug, p.Plan) {
				newPlan = pl
				break
			}
		}
	}
	if newPlan == nil {
		return nil, fmt.Errorf("target hosting package '%s' not found", p.Plan)
	}

	acc.PlanID = newPlan.ID
	acc.PlanName = newPlan.Name
	acc.DiskLimitMB = newPlan.DiskSpaceMB
	acc.BandwidthLimitMB = newPlan.BandwidthMB
	acc.WebsitesLimit = newPlan.MaxWebsites
	acc.DatabasesLimit = newPlan.MaxDatabases
	acc.MailboxesLimit = newPlan.MaxMailboxes
	acc.UpdatedAt = time.Now().UTC()

	if err := h.store.UpdateHostingAccount(ctx, acc); err != nil {
		return nil, fmt.Errorf("failed to upgrade account package: %w", err)
	}

	return map[string]interface{}{
		"status":           "success",
		"message":          fmt.Sprintf("Account package changed to %s", newPlan.Name),
		"account_id":       acc.ID,
		"new_plan":         newPlan.Name,
		"disk_limit_mb":    acc.DiskLimitMB,
		"bandwidth_limit":  acc.BandwidthLimitMB,
	}, nil
}

func (h *ProvisioningHandler) findAccountByTarget(ctx context.Context, domain, username string) (*store.HostingAccount, error) {
	cleanDomain := strings.ToLower(strings.TrimSpace(domain))
	cleanUser := strings.ToLower(strings.TrimSpace(username))

	if cleanDomain != "" {
		if acc, err := h.store.GetHostingAccountByDomain(ctx, cleanDomain); err == nil && acc != nil {
			return acc, nil
		}
	}
	if cleanUser != "" {
		if acc, err := h.store.GetHostingAccountByUsername(ctx, cleanUser); err == nil && acc != nil {
			return acc, nil
		}
	}

	return nil, fmt.Errorf("hosting account not found for domain '%s' or username '%s'", cleanDomain, cleanUser)
}

// ----------------------------------------------------------------------------
// Authentication & Signature Verification
// ----------------------------------------------------------------------------

func (h *ProvisioningHandler) verifyWebhookAuth(r *http.Request, body []byte, provider string) (*store.ProvisioningIntegration, error) {
	// Look up integration ID if specified in headers or query
	var specificID *uuid.UUID
	idStr := r.Header.Get("X-Integration-ID")
	if idStr == "" {
		idStr = r.URL.Query().Get("integration_id")
	}
	if idStr != "" {
		if parsed, err := uuid.Parse(idStr); err == nil {
			specificID = &parsed
		}
	}

	// Fetch active integrations
	var integrations []*store.ProvisioningIntegration
	items, _ := h.store.ListProvisioningIntegrations(r.Context(), uuid.Nil)
	for _, it := range items {
		if it.IsActive {
			if specificID != nil && it.ID == *specificID {
				integrations = []*store.ProvisioningIntegration{it}
				break
			}
			if provider == "" || strings.EqualFold(it.Provider, provider) || it.Provider == "custom" {
				integrations = append(integrations, it)
			}
		}
	}

	// If no integrations configured yet, check system fallback token
	if len(integrations) == 0 {
		incomingSecret := extractIncomingToken(r)
		if incomingSecret != "" && h.cfg.JWTSecret != "" && subtleConstantCompare(incomingSecret, h.cfg.JWTSecret) {
			return nil, nil // Authenticated via master key
		}
		return nil, fmt.Errorf("no active provisioning integrations configured for provider '%s'", provider)
	}

	// Check token headers or HMAC signatures
	incomingSignature := extractIncomingSignature(r)
	incomingToken := extractIncomingToken(r)

	for _, item := range integrations {
		secretKey := item.RawSecretKey

		// Check Token match directly
		if incomingToken != "" {
			if secretKey != "" && subtleConstantCompare(incomingToken, secretKey) {
				return item, nil
			}
			hash := sha256.Sum256([]byte(incomingToken))
			hexHash := hex.EncodeToString(hash[:])
			if subtleConstantCompare(hexHash, item.SecretKeyHash) {
				return item, nil
			}
		}

		// Check HMAC-SHA256 signature
		if incomingSignature != "" && secretKey != "" {
			mac := hmac.New(sha256.New, []byte(secretKey))
			mac.Write(body)
			expectedSig := hex.EncodeToString(mac.Sum(nil))

			cleanIncoming := strings.TrimPrefix(incomingSignature, "sha256=")
			if hmac.Equal([]byte(cleanIncoming), []byte(expectedSig)) {
				return item, nil
			}
		}
	}

	return nil, fmt.Errorf("invalid HMAC signature or secret token")
}

func extractIncomingSignature(r *http.Request) string {
	headers := []string{
		"X-Hostvra-Signature",
		"X-WHMCS-Signature",
		"X-Blesta-Signature",
		"X-Signature",
		"X-Hub-Signature-256",
	}
	for _, h := range headers {
		if val := r.Header.Get(h); val != "" {
			return strings.TrimSpace(val)
		}
	}
	return r.URL.Query().Get("signature")
}

func extractIncomingToken(r *http.Request) string {
	headers := []string{
		"X-Hostvra-Token",
		"X-WHMCS-Token",
		"X-Blesta-Token",
		"X-Secret-Token",
		"X-Api-Key",
		"X-Auth-Token",
	}
	for _, h := range headers {
		if val := r.Header.Get(h); val != "" {
			return strings.TrimSpace(val)
		}
	}
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	return r.URL.Query().Get("token")
}

func subtleConstantCompare(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}

func (h *ProvisioningHandler) parseWebhookPayload(r *http.Request, body []byte) (*ProvisioningWebhookPayload, error) {
	p := &ProvisioningWebhookPayload{
		ExtraParams: make(map[string]string),
	}

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") || len(body) > 0 && body[0] == '{' {
		if err := json.Unmarshal(body, p); err != nil {
			return nil, err
		}
	} else {
		// Form-urlencoded
		if err := r.ParseForm(); err == nil {
			for k, v := range r.PostForm {
				if len(v) > 0 {
					p.ExtraParams[k] = v[0]
				}
			}
			p.Action = r.PostFormValue("action")
			if p.Action == "" {
				p.Action = r.PostFormValue("cmd")
			}
			if p.Action == "" {
				p.Action = r.PostFormValue("event")
			}
			p.Domain = r.PostFormValue("domain")
			p.Username = r.PostFormValue("username")
			p.Password = r.PostFormValue("password")
			p.Email = r.PostFormValue("email")
			p.Plan = r.PostFormValue("plan")
			if p.Plan == "" {
				p.Plan = r.PostFormValue("package")
			}
			p.PlanID = r.PostFormValue("plan_id")
			p.ServerID = r.PostFormValue("server_id")
			p.ResellerID = r.PostFormValue("reseller_id")
			p.Reason = r.PostFormValue("reason")
		}
	}

	if p.Action == "" {
		p.Action = r.URL.Query().Get("action")
	}
	if p.Action == "" {
		p.Action = "create_account"
	}
	return p, nil
}

func (h *ProvisioningHandler) logAndRespondError(w http.ResponseWriter, r *http.Request, integration *store.ProvisioningIntegration, provider, action, username, domain string, statusCode int, message, detail string) {
	var integrationID *uuid.UUID
	var orgID uuid.UUID
	if integration != nil {
		idCopy := integration.ID
		integrationID = &idCopy
		orgID = integration.OrganizationID
	}

	_ = h.store.LogProvisioningWebhook(r.Context(), &store.ProvisioningWebhookLog{
		ID:             uuid.New(),
		OrganizationID: orgID,
		IntegrationID:  integrationID,
		Provider:       provider,
		Action:         action,
		Username:       username,
		Domain:         domain,
		Status:         "failed",
		HTTPStatus:     statusCode,
		ErrorMessage:   fmt.Sprintf("%s: %s", message, detail),
		IPAddress:      r.RemoteAddr,
		CreatedAt:      time.Now().UTC(),
	})

	response.Error(w, statusCode, "PROVISIONING_ERROR", message, detail, "")
}

func matchedID(u uuid.UUID) uuid.UUID {
	return u
}

// ----------------------------------------------------------------------------
// Webhook Replay Endpoint
// ----------------------------------------------------------------------------

func (h *ProvisioningHandler) ReplayWebhookLog(w http.ResponseWriter, r *http.Request) {
	logIDStr := chi.URLParam(r, "id")
	logID, err := uuid.Parse(logIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid log ID", nil, "")
		return
	}

	logItem, err := h.store.GetProvisioningWebhookLogByID(r.Context(), logID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Provisioning log not found", nil, "")
		return
	}

	// Re-construct payload
	payloadBytes, _ := json.Marshal(logItem.Payload)
	var payload ProvisioningWebhookPayload
	_ = json.Unmarshal(payloadBytes, &payload)
	if payload.Action == "" {
		payload.Action = logItem.Action
	}
	if payload.Domain == "" {
		payload.Domain = logItem.Domain
	}
	if payload.Username == "" {
		payload.Username = logItem.Username
	}

	var integration *store.ProvisioningIntegration
	if logItem.IntegrationID != nil {
		integration, _ = h.store.GetProvisioningIntegrationByID(r.Context(), *logItem.IntegrationID)
	}

	var result interface{}
	var execErr error

	switch strings.ToLower(payload.Action) {
	case "create_account", "create", "provision":
		result, execErr = h.executeCreateAccount(r, integration, &payload)
	case "suspend":
		result, execErr = h.executeSuspendAccount(r, &payload)
	case "unsuspend":
		result, execErr = h.executeUnsuspendAccount(r, &payload)
	case "terminate", "delete":
		result, execErr = h.executeTerminateAccount(r, &payload)
	case "change_package", "change_plan":
		result, execErr = h.executeChangePackage(r, &payload)
	default:
		execErr = fmt.Errorf("unsupported action for replay: %s", payload.Action)
	}

	status := "replayed"
	httpStatus := http.StatusOK
	var errMsg string
	if execErr != nil {
		status = "failed"
		httpStatus = http.StatusBadRequest
		errMsg = execErr.Error()
	}

	respBytes, _ := json.Marshal(result)
	_ = h.store.LogProvisioningWebhook(r.Context(), &store.ProvisioningWebhookLog{
		ID:             uuid.New(),
		OrganizationID: logItem.OrganizationID,
		IntegrationID:  logItem.IntegrationID,
		Provider:       logItem.Provider,
		Action:         payload.Action,
		Username:       payload.Username,
		Domain:         payload.Domain,
		Payload:        payload,
		Status:         status,
		HTTPStatus:     httpStatus,
		ResponseBody:   string(respBytes),
		ErrorMessage:   errMsg,
		IPAddress:      r.RemoteAddr,
		CreatedAt:      time.Now().UTC(),
	})

	if execErr != nil {
		response.Error(w, http.StatusBadRequest, "REPLAY_FAILED", "Failed to replay webhook action", execErr.Error(), "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Webhook action replayed successfully",
		"result":  result,
	}, nil)
}

// ----------------------------------------------------------------------------
// Integrations Management CRUD
// ----------------------------------------------------------------------------

type CreateIntegrationRequest struct {
	Provider        string  `json:"provider"` // whmcs, blesta, custom
	Name            string  `json:"name"`
	DefaultServerID *string `json:"default_server_id,omitempty"`
	DefaultPlanID   *string `json:"default_plan_id,omitempty"`
	IsActive        bool    `json:"is_active"`
}

func (h *ProvisioningHandler) ListIntegrations(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	list, err := h.store.ListProvisioningIntegrations(r.Context(), claims.OrganizationID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list integrations", err.Error(), "")
		return
	}
	response.JSON(w, http.StatusOK, list, nil)
}

func (h *ProvisioningHandler) CreateIntegration(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	var req CreateIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if req.Name == "" {
		req.Name = fmt.Sprintf("%s Integration", strings.ToUpper(req.Provider))
	}
	if req.Provider == "" {
		req.Provider = "whmcs"
	}

	// Generate high-entropy 32-byte secret key (64 hex characters)
	randomBytes := make([]byte, 32)
	_, _ = rand.Read(randomBytes)
	rawSecret := hex.EncodeToString(randomBytes)

	hash := sha256.Sum256([]byte(rawSecret))
	secretHash := hex.EncodeToString(hash[:])

	var sID *uuid.UUID
	if req.DefaultServerID != nil && *req.DefaultServerID != "" {
		if parsed, err := uuid.Parse(*req.DefaultServerID); err == nil {
			sID = &parsed
		}
	}
	var pID *uuid.UUID
	if req.DefaultPlanID != nil && *req.DefaultPlanID != "" {
		if parsed, err := uuid.Parse(*req.DefaultPlanID); err == nil {
			pID = &parsed
		}
	}

	item := &store.ProvisioningIntegration{
		ID:              uuid.New(),
		OrganizationID:  claims.OrganizationID,
		Provider:        strings.ToLower(strings.TrimSpace(req.Provider)),
		Name:            strings.TrimSpace(req.Name),
		SecretKeyHash:   secretHash,
		RawSecretKey:    rawSecret,
		DefaultServerID: sID,
		DefaultPlanID:   pID,
		IsActive:        true,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if err := h.store.CreateProvisioningIntegration(r.Context(), item); err != nil {
		response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create integration", err.Error(), "")
		return
	}

	response.JSON(w, http.StatusCreated, item, nil)
}

func (h *ProvisioningHandler) DeleteIntegration(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid integration ID", nil, "")
		return
	}

	if err := h.store.DeleteProvisioningIntegration(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete integration", err.Error(), "")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "deleted"}, nil)
}

func (h *ProvisioningHandler) ListWebhookLogs(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	logs, err := h.store.ListProvisioningWebhookLogs(r.Context(), claims.OrganizationID, 100)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list webhook logs", err.Error(), "")
		return
	}
	response.JSON(w, http.StatusOK, logs, nil)
}
