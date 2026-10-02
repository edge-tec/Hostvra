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
	"hostvra/api/internal/email"
	"hostvra/api/internal/quota"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type AdminUsersHandler struct {
	cfg          *config.Config
	store        store.Store
	quotaService *quota.Service
	audit        *audit.Logger
	emailSvc     *email.Service
}

func NewAdminUsersHandler(cfg *config.Config, s store.Store, q *quota.Service, a *audit.Logger, emailSvc ...*email.Service) *AdminUsersHandler {
	h := &AdminUsersHandler{
		cfg:          cfg,
		store:        s,
		quotaService: q,
		audit:        a,
	}
	if len(emailSvc) > 0 && emailSvc[0] != nil {
		h.emailSvc = emailSvc[0]
	}
	return h
}

// UserListItemDTO represents user summary in Admin User Management
type UserListItemDTO struct {
	ID                 uuid.UUID                `json:"id"`
	Email              string                   `json:"email"`
	FullName           string                   `json:"full_name"`
	Role               string                   `json:"role"`
	IsActive           bool                     `json:"is_active"`
	IsSuperAdmin       bool                     `json:"is_superadmin"`
	LastLoginAt        *string                  `json:"last_login_at,omitempty"`
	CreatedAt          string                   `json:"created_at"`
	EffectivePlan      *store.EffectiveUserPlan `json:"effective_plan,omitempty"`
	HasCustomOverrides bool                     `json:"has_custom_overrides"`
}

// ListUsers handles GET /api/v1/admin/users
func (h *AdminUsersHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.ListUsers(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "USERS_LIST_FAILED", err.Error(), nil, "")
		return
	}

	result := make([]*UserListItemDTO, 0, len(users))
	for _, u := range users {
		displayRole := u.Role
		if u.IsSuperAdmin {
			displayRole = "superadmin"
		} else if displayRole == "owner" || displayRole == "" {
			displayRole = "customer"
			_ = h.store.UpdateUserRole(r.Context(), u.ID, "customer")
		}

		item := &UserListItemDTO{
			ID:           u.ID,
			Email:        u.Email,
			FullName:     u.FullName,
			Role:         displayRole,
			IsActive:     u.IsActive,
			IsSuperAdmin: u.IsSuperAdmin,
			CreatedAt:    u.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if u.LastLoginAt != nil {
			t := u.LastLoginAt.Format("2006-01-02 15:04:05")
			item.LastLoginAt = &t
		}

		// Resolve effective plan & check overrides
		if eff, err := h.quotaService.ResolveEffectivePlan(r.Context(), u.ID); err == nil {
			item.EffectivePlan = eff
			item.HasCustomOverrides = (eff.Overrides != nil)
		}

		result = append(result, item)
	}

	response.JSON(w, http.StatusOK, result, &response.Meta{
		Total: len(result),
	})
}

// GetUserDetails handles GET /api/v1/admin/users/{id}
func (h *AdminUsersHandler) GetUserDetails(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user UUID", nil, "")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "User profile not found", nil, "")
		return
	}

	if !user.IsSuperAdmin && (user.Role == "owner" || user.Role == "") {
		user.Role = "customer"
		_ = h.store.UpdateUserRole(r.Context(), user.ID, "customer")
	}

	if r.URL.Query().Get("refresh") == "true" || r.URL.Query().Get("sync") == "true" {
		h.quotaService.InvalidateUserStorageCache(userID)
	}

	effective, err := h.quotaService.ResolveEffectivePlan(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "QUOTA_RESOLUTION_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"user":           user,
		"effective_plan": effective,
	}, nil)
}

type UpdateUserPlanRequest struct {
	PlanID       string `json:"plan_id"`
	BillingCycle string `json:"billing_cycle"` // monthly, yearly
}

// UpdateUserPlan handles PUT /api/v1/admin/users/{id}/plan
func (h *AdminUsersHandler) UpdateUserPlan(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user UUID", nil, "")
		return
	}

	var req UpdateUserPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PlanID == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Plan ID is required", nil, "")
		return
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PLAN_ID", "Invalid plan UUID", nil, "")
		return
	}

	plan, err := h.store.GetPlanByID(r.Context(), planID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "PLAN_NOT_FOUND", "Hosting plan not found", nil, "")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found", nil, "")
		return
	}

	// Update existing subscription or create new one
	subs, err := h.store.ListSubscriptions(r.Context(), user.DefaultOrgID)
	var activeSub *store.Subscription
	if err == nil {
		for _, s := range subs {
			if s.UserID == user.ID {
				activeSub = s
				break
			}
		}
	}

	if activeSub != nil {
		activeSub.PlanID = plan.ID
		activeSub.PlanName = plan.Name
		activeSub.Amount = plan.PriceMonthly
		if req.BillingCycle == "yearly" {
			activeSub.BillingCycle = "yearly"
			activeSub.Amount = plan.PriceYearly
		}
		activeSub.Status = store.SubStatusActive
		_ = h.store.UpdateSubscription(r.Context(), activeSub)
	} else {
		newSub := &store.Subscription{
			ID:             uuid.New(),
			UserID:         user.ID,
			OrganizationID: user.DefaultOrgID,
			PlanID:         plan.ID,
			PlanName:       plan.Name,
			Status:         store.SubStatusActive,
			BillingCycle:   "monthly",
			Amount:         plan.PriceMonthly,
			Currency:       plan.Currency,
			AutoRenew:      true,
		}
		_ = h.store.CreateSubscription(r.Context(), newSub)
	}

	// Synchronize user's hosting accounts with the new plan configuration
	if accounts, err := h.store.ListHostingAccounts(r.Context(), user.DefaultOrgID, nil); err == nil {
		for _, acc := range accounts {
			if acc.UserID == user.ID {
				acc.PlanID = plan.ID
				acc.PlanName = plan.Name
				acc.DiskLimitMB = plan.DiskSpaceMB
				acc.BandwidthLimitMB = plan.BandwidthMB
				acc.WebsitesLimit = plan.MaxWebsites
				acc.DatabasesLimit = plan.MaxDatabases
				acc.MailboxesLimit = plan.MaxMailboxes
				_ = h.store.UpdateHostingAccount(r.Context(), acc)
			}
		}
	}

	h.audit.Log(r.Context(), r, "admin.user.change_plan", "user", userID.String(), "success", "", map[string]interface{}{
		"plan_id":   plan.ID,
		"plan_name": plan.Name,
	})

	effective, _ := h.quotaService.ResolveEffectivePlan(r.Context(), userID)
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message":        "User package plan successfully updated",
		"effective_plan": effective,
	}, nil)
}

// UpdateUserOverrides handles PUT /api/v1/admin/users/{id}/overrides
func (h *AdminUsersHandler) UpdateUserOverrides(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user UUID", nil, "")
		return
	}

	var override store.UserPlanOverride
	if err := json.NewDecoder(r.Body).Decode(&override); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	override.UserID = userID
	if err := h.store.UpsertUserPlanOverride(r.Context(), &override); err != nil {
		response.Error(w, http.StatusInternalServerError, "OVERRIDE_SAVE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "admin.user.set_overrides", "user", userID.String(), "success", "", map[string]interface{}{
		"notes": override.Notes,
	})

	effective, _ := h.quotaService.ResolveEffectivePlan(r.Context(), userID)
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message":        "User custom limit and permission overrides applied successfully",
		"effective_plan": effective,
	}, nil)
}

// DeleteUserOverrides handles DELETE /api/v1/admin/users/{id}/overrides
func (h *AdminUsersHandler) DeleteUserOverrides(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user UUID", nil, "")
		return
	}

	if err := h.store.DeleteUserPlanOverride(r.Context(), userID); err != nil {
		response.Error(w, http.StatusInternalServerError, "OVERRIDE_DELETE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "admin.user.clear_overrides", "user", userID.String(), "success", "", nil)

	effective, _ := h.quotaService.ResolveEffectivePlan(r.Context(), userID)
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message":        "User overrides removed; reverted to package defaults",
		"effective_plan": effective,
	}, nil)
}

type UpdateUserStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// UpdateUserStatus handles PUT /api/v1/admin/users/{id}/status
func (h *AdminUsersHandler) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user UUID", nil, "")
		return
	}

	var req UpdateUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	if err := h.store.UpdateUserStatus(r.Context(), userID, req.IsActive); err != nil {
		response.Error(w, http.StatusInternalServerError, "STATUS_UPDATE_FAILED", err.Error(), nil, "")
		return
	}

	action := "activated"
	if !req.IsActive {
		action = "suspended"
	}
	h.audit.Log(r.Context(), r, "admin.user.status", "user", userID.String(), "success", "", map[string]interface{}{
		"action": action,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message":   "User account status updated to " + action,
		"is_active": req.IsActive,
	}, nil)
}

type UpdateUserRoleRequest struct {
	Role string `json:"role"`
}

// UpdateUserRole handles PUT /api/v1/admin/users/{id}/role
func (h *AdminUsersHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user UUID", nil, "")
		return
	}

	var req UpdateUserRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Role == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Valid role is required", nil, "")
		return
	}

	cleanRole := strings.ToLower(strings.TrimSpace(req.Role))
	switch cleanRole {
	case "customer", "user", "admin":
	default:
		response.Error(w, http.StatusBadRequest, "INVALID_ROLE", "Role must be 'customer', 'user', or 'admin'", nil, "")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found", nil, "")
		return
	}

	if err := h.store.UpdateUserRole(r.Context(), userID, cleanRole); err != nil {
		response.Error(w, http.StatusInternalServerError, "ROLE_UPDATE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "admin.user.update_role", "user", userID.String(), "success", fmt.Sprintf("Changed role of %s to %s", user.Email, cleanRole), map[string]interface{}{
		"previous_role": user.Role,
		"new_role":      cleanRole,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "User role successfully updated",
		"role":    cleanRole,
	}, nil)
}

// DeleteUser handles DELETE /api/v1/admin/users/{id}
func (h *AdminUsersHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user UUID", nil, "")
		return
	}

	if err := h.store.DeleteUser(r.Context(), userID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_USER_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "admin.user.delete", "user", userID.String(), "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "User account deactivated successfully",
	}, nil)
}

// GetUserEffectivePlan handles GET /api/v1/user/plan (Customer Self-Service endpoint)
func (h *AdminUsersHandler) GetUserEffectivePlan(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication session", nil, "")
		return
	}

	if r.URL.Query().Get("refresh") == "true" || r.URL.Query().Get("sync") == "true" {
		h.quotaService.InvalidateUserStorageCache(claims.UserID)
	}

	effective, err := h.quotaService.ResolveEffectivePlan(r.Context(), claims.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "PLAN_RESOLUTION_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, effective, nil)
}

// ================================
// ADMIN: UPDATE USER EMAIL
// ================================

type AdminUpdateEmailRequest struct {
	Email string `json:"email"`
}

// AdminUpdateEmail handles PUT /api/v1/admin/users/{id}/email
func (h *AdminUsersHandler) AdminUpdateEmail(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user UUID", nil, "")
		return
	}

	var req AdminUpdateEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid request body", nil, "")
		return
	}

	newEmail := strings.TrimSpace(strings.ToLower(req.Email))
	if newEmail == "" || !strings.Contains(newEmail, "@") {
		response.Error(w, http.StatusBadRequest, "INVALID_EMAIL", "A valid email address is required", nil, "")
		return
	}

	// Check uniqueness
	existing, err := h.store.GetUserByEmail(r.Context(), newEmail)
	if err == nil && existing != nil && existing.ID != userID {
		response.Error(w, http.StatusConflict, "EMAIL_EXISTS", "This email is already in use by another account", nil, "")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found", nil, "")
		return
	}

	oldEmail := user.Email
	if err := h.store.UpdateUserEmail(r.Context(), userID, newEmail); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update email", nil, "")
		return
	}

	// If updating admin's own email, sync env
	if user.IsSuperAdmin {
		syncEnvCredentials(newEmail, "")
	}

	h.audit.Log(r.Context(), r, "admin.user.update_email", "user", userID.String(), "success", "", map[string]interface{}{
		"old_email": oldEmail,
		"new_email": newEmail,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"email":   newEmail,
		"message": "User email updated successfully",
	}, nil)
}

// ================================
// ADMIN: UPDATE USER PASSWORD
// ================================

type AdminUpdatePasswordRequest struct {
	Password string `json:"password"`
}

// AdminUpdatePassword handles PUT /api/v1/admin/users/{id}/password
func (h *AdminUsersHandler) AdminUpdatePassword(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user UUID", nil, "")
		return
	}

	var req AdminUpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Password is required", nil, "")
		return
	}

	if err := auth.ValidatePasswordComplexity(req.Password); err != nil {
		response.Error(w, http.StatusBadRequest, "WEAK_PASSWORD", err.Error(), nil, "")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found", nil, "")
		return
	}

	newHash, err := auth.HashPassword(req.Password, nil)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "HASH_FAILED", "Failed to secure password", nil, "")
		return
	}

	if err := h.store.UpdateUserPassword(r.Context(), userID, newHash); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update password", nil, "")
		return
	}

	// If updating admin's own password, sync env
	if user.IsSuperAdmin {
		syncEnvCredentials("", req.Password)
	}

	// Send password changed notification
	if h.emailSvc != nil {
		go h.emailSvc.SendPasswordChanged(r.Context(), user.Email, user.FullName)
	}

	h.audit.Log(r.Context(), r, "admin.user.update_password", "user", userID.String(), "success", "", map[string]interface{}{
		"email": user.Email,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "User password updated successfully",
	}, nil)
}

// ================================
// ADMIN: IMPERSONATE USER
// ================================

// ImpersonateUser handles POST /api/v1/admin/users/{id}/impersonate
func (h *AdminUsersHandler) ImpersonateUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	userIDStr := chi.URLParam(r, "id")
	targetUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user UUID", nil, "")
		return
	}

	// Cannot impersonate yourself
	if targetUserID == claims.UserID {
		response.Error(w, http.StatusBadRequest, "SELF_IMPERSONATION", "Cannot impersonate your own account", nil, "")
		return
	}

	targetUser, err := h.store.GetUserByID(r.Context(), targetUserID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "Target user not found", nil, "")
		return
	}

	// Cannot impersonate superadmin
	if targetUser.IsSuperAdmin {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cannot impersonate a superadmin account", nil, "")
		return
	}

	// Generate short-lived impersonation token (1 hour)
	impersonationDuration := 1 * time.Hour
	tokens, err := auth.GenerateImpersonationTokenPair(
		claims.UserID, targetUser.ID, targetUser.DefaultOrgID, targetUser.Email,
		h.cfg.JWTSecret, impersonationDuration,
	)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate impersonation token", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "admin.impersonation.started", "user", targetUserID.String(), "success", "", map[string]interface{}{
		"admin_id":    claims.UserID,
		"admin_email": claims.Email,
		"target_id":   targetUser.ID,
		"target_email":targetUser.Email,
		"duration":    impersonationDuration.String(),
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"tokens":     tokens,
		"user":       targetUser,
		"role":       "customer",
		"expires_at": tokens.ExpiresAt,
		"message":    fmt.Sprintf("Impersonation session started for %s (expires in 1 hour)", targetUser.Email),
	}, nil)
}
