package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/quota"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type AdminUsersHandler struct {
	store        store.Store
	quotaService *quota.Service
	audit        *audit.Logger
}

func NewAdminUsersHandler(s store.Store, q *quota.Service, a *audit.Logger) *AdminUsersHandler {
	return &AdminUsersHandler{
		store:        s,
		quotaService: q,
		audit:        a,
	}
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
		item := &UserListItemDTO{
			ID:           u.ID,
			Email:        u.Email,
			FullName:     u.FullName,
			Role:         u.Role,
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

	effective, err := h.quotaService.ResolveEffectivePlan(r.Context(), claims.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "PLAN_RESOLUTION_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, effective, nil)
}
