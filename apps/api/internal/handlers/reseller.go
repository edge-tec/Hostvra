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
	"hostvra/api/internal/rbac"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type ResellerHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewResellerHandler(cfg *config.Config, s store.Store, a *audit.Logger) *ResellerHandler {
	return &ResellerHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
}

// ----------------------------------------------------------------------------
// Requests & Models
// ----------------------------------------------------------------------------

type CreateResellerProfileRequest struct {
	UserID               string   `json:"user_id"`
	MaxAccounts          int      `json:"max_accounts"`
	AllocatedDiskMB      int64    `json:"allocated_disk_mb"`
	AllocatedBandwidthMB int64    `json:"allocated_bandwidth_mb"`
	AllowOverselling     bool     `json:"allow_overselling"`
	BrandName            string   `json:"brand_name"`
	BrandLogoURL         string   `json:"brand_logo_url"`
	CustomNameservers    []string `json:"custom_nameservers"`
	CustomEmailSender    string   `json:"custom_email_sender"`
}

type UpdateResellerProfileRequest struct {
	MaxAccounts          *int      `json:"max_accounts,omitempty"`
	AllocatedDiskMB      *int64    `json:"allocated_disk_mb,omitempty"`
	AllocatedBandwidthMB *int64    `json:"allocated_bandwidth_mb,omitempty"`
	AllowOverselling     *bool     `json:"allow_overselling,omitempty"`
	BrandName            *string   `json:"brand_name,omitempty"`
	BrandLogoURL         *string   `json:"brand_logo_url,omitempty"`
	CustomNameservers    *[]string `json:"custom_nameservers,omitempty"`
	CustomEmailSender    *string   `json:"custom_email_sender,omitempty"`
	IsActive             *bool     `json:"is_active,omitempty"`
}

type BrandingResponse struct {
	BrandName         string   `json:"brand_name"`
	BrandLogoURL      string   `json:"brand_logo_url"`
	CustomNameservers []string `json:"custom_nameservers"`
	CustomEmailSender string   `json:"custom_email_sender"`
}

// ----------------------------------------------------------------------------
// Reseller Profile Management (Admin)
// ----------------------------------------------------------------------------

func (h *ResellerHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var req CreateResellerProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	userUUID, err := uuid.Parse(req.UserID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Valid user ID is required", nil, "")
		return
	}

	targetUser, err := h.store.GetUserByID(r.Context(), userUUID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "Target user not found", nil, "")
		return
	}

	// Check if reseller profile already exists for this user
	if existing, _ := h.store.GetResellerProfileByUserID(r.Context(), userUUID); existing != nil {
		response.Error(w, http.StatusConflict, "EXISTS", "User already has a reseller profile", nil, "")
		return
	}

	// Update user role to reseller
	_ = h.store.UpdateUserRole(r.Context(), userUUID, rbac.RoleReseller)

	brandName := strings.TrimSpace(req.BrandName)
	if brandName == "" {
		brandName = targetUser.FullName + " Hosting"
	}
	nameservers := req.CustomNameservers
	if len(nameservers) == 0 {
		nameservers = []string{"ns1.hostvra.com", "ns2.hostvra.com"}
	}
	emailSender := req.CustomEmailSender
	if emailSender == "" {
		emailSender = "noreply@hostvra.com"
	}

	profile := &store.ResellerProfile{
		ID:                   uuid.New(),
		OrganizationID:       claims.OrganizationID,
		UserID:               userUUID,
		MaxAccounts:          req.MaxAccounts,
		AllocatedDiskMB:      req.AllocatedDiskMB,
		AllocatedBandwidthMB: req.AllocatedBandwidthMB,
		AllowOverselling:     req.AllowOverselling,
		BrandName:            brandName,
		BrandLogoURL:         req.BrandLogoURL,
		CustomNameservers:    nameservers,
		CustomEmailSender:    emailSender,
		IsActive:             true,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}

	if err := h.store.CreateResellerProfile(r.Context(), profile); err != nil {
		response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create reseller profile", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "reseller.profile.create", "reseller_profile", profile.ID.String(), "success", fmt.Sprintf("Provisioned reseller profile for user %s", targetUser.Email), nil)

	response.JSON(w, http.StatusCreated, profile, nil)
}

func (h *ResellerHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	profiles, err := h.store.ListResellerProfiles(r.Context(), claims.OrganizationID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list reseller profiles", err.Error(), "")
		return
	}
	response.JSON(w, http.StatusOK, profiles, &response.Meta{Total: len(profiles)})
}

func (h *ResellerHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	idStr := chi.URLParam(r, "id")

	var profile *store.ResellerProfile
	var err error

	if idStr != "" && idStr != "me" {
		id, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid profile ID", nil, "")
			return
		}
		profile, err = h.store.GetResellerProfileByID(r.Context(), id)
	} else {
		// Caller's own profile
		profile, err = h.store.GetResellerProfileByUserID(r.Context(), claims.UserID)
	}

	if err != nil || profile == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Reseller profile not found", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, profile, nil)
}

func (h *ResellerHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	idStr := chi.URLParam(r, "id")

	var profile *store.ResellerProfile
	var err error

	if idStr != "" && idStr != "me" {
		id, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid profile ID", nil, "")
			return
		}
		profile, err = h.store.GetResellerProfileByID(r.Context(), id)
	} else {
		profile, err = h.store.GetResellerProfileByUserID(r.Context(), claims.UserID)
	}

	if err != nil || profile == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Reseller profile not found", nil, "")
		return
	}

	var req UpdateResellerProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	// Only Admin/Owner can alter resource limits
	isAdmin := claims.Role == rbac.RoleOwner || claims.Role == rbac.RoleAdmin
	if isAdmin {
		if req.MaxAccounts != nil {
			profile.MaxAccounts = *req.MaxAccounts
		}
		if req.AllocatedDiskMB != nil {
			profile.AllocatedDiskMB = *req.AllocatedDiskMB
		}
		if req.AllocatedBandwidthMB != nil {
			profile.AllocatedBandwidthMB = *req.AllocatedBandwidthMB
		}
		if req.AllowOverselling != nil {
			profile.AllowOverselling = *req.AllowOverselling
		}
		if req.IsActive != nil {
			profile.IsActive = *req.IsActive
		}
	}

	// Both Reseller and Admin can update white-label branding
	if req.BrandName != nil {
		profile.BrandName = strings.TrimSpace(*req.BrandName)
	}
	if req.BrandLogoURL != nil {
		profile.BrandLogoURL = strings.TrimSpace(*req.BrandLogoURL)
	}
	if req.CustomNameservers != nil {
		profile.CustomNameservers = *req.CustomNameservers
	}
	if req.CustomEmailSender != nil {
		profile.CustomEmailSender = strings.TrimSpace(*req.CustomEmailSender)
	}

	if err := h.store.UpdateResellerProfile(r.Context(), profile); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update reseller profile", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "reseller.profile.update", "reseller_profile", profile.ID.String(), "success", fmt.Sprintf("Updated reseller profile for user %s", profile.UserID), nil)

	response.JSON(w, http.StatusOK, profile, nil)
}

// ----------------------------------------------------------------------------
// Reseller Quotas & Usage Pool Monitoring
// ----------------------------------------------------------------------------

func (h *ResellerHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	targetUserID := claims.UserID

	// Allow admin to query for a specific reseller user
	if qUser := r.URL.Query().Get("user_id"); qUser != "" {
		if claims.Role == rbac.RoleOwner || claims.Role == rbac.RoleAdmin {
			if parsed, err := uuid.Parse(qUser); err == nil {
				targetUserID = parsed
			}
		}
	}

	usage, err := h.store.GetResellerUsage(r.Context(), targetUserID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Reseller profile not found for user", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, usage, nil)
}

// ----------------------------------------------------------------------------
// White-Label Branding Endpoints
// ----------------------------------------------------------------------------

func (h *ResellerHandler) GetBranding(w http.ResponseWriter, r *http.Request) {
	// Default branding
	branding := BrandingResponse{
		BrandName:         "Hostvra",
		BrandLogoURL:      "/logo.svg",
		CustomNameservers: []string{"ns1.hostvra.com", "ns2.hostvra.com"},
		CustomEmailSender: "noreply@hostvra.com",
	}

	// 1. Check if authenticated as a reseller
	if claims, ok := auth.GetClaims(r.Context()); ok && claims != nil {
		if prof, err := h.store.GetResellerProfileByUserID(r.Context(), claims.UserID); err == nil && prof != nil {
			if prof.BrandName != "" {
				branding.BrandName = prof.BrandName
			}
			if prof.BrandLogoURL != "" {
				branding.BrandLogoURL = prof.BrandLogoURL
			}
			if len(prof.CustomNameservers) > 0 {
				branding.CustomNameservers = prof.CustomNameservers
			}
			if prof.CustomEmailSender != "" {
				branding.CustomEmailSender = prof.CustomEmailSender
			}
			response.JSON(w, http.StatusOK, branding, nil)
			return
		}
	}

	// 2. Query param or host-based inspection
	if rIDStr := r.URL.Query().Get("reseller_id"); rIDStr != "" {
		if rID, err := uuid.Parse(rIDStr); err == nil {
			if prof, err := h.store.GetResellerProfileByUserID(r.Context(), rID); err == nil && prof != nil {
				if prof.BrandName != "" {
					branding.BrandName = prof.BrandName
				}
				if prof.BrandLogoURL != "" {
					branding.BrandLogoURL = prof.BrandLogoURL
				}
				if len(prof.CustomNameservers) > 0 {
					branding.CustomNameservers = prof.CustomNameservers
				}
				if prof.CustomEmailSender != "" {
					branding.CustomEmailSender = prof.CustomEmailSender
				}
			}
		}
	}

	response.JSON(w, http.StatusOK, branding, nil)
}

func (h *ResellerHandler) UpdateBranding(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	profile, err := h.store.GetResellerProfileByUserID(r.Context(), claims.UserID)
	if err != nil || profile == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Reseller profile not found", nil, "")
		return
	}

	var req BrandingResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if req.BrandName != "" {
		profile.BrandName = strings.TrimSpace(req.BrandName)
	}
	if req.BrandLogoURL != "" {
		profile.BrandLogoURL = strings.TrimSpace(req.BrandLogoURL)
	}
	if len(req.CustomNameservers) > 0 {
		profile.CustomNameservers = req.CustomNameservers
	}
	if req.CustomEmailSender != "" {
		profile.CustomEmailSender = strings.TrimSpace(req.CustomEmailSender)
	}

	if err := h.store.UpdateResellerProfile(r.Context(), profile); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update branding", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "reseller.branding.update", "reseller_profile", profile.ID.String(), "success", fmt.Sprintf("Updated white-label branding: %s", profile.BrandName), nil)

	response.JSON(w, http.StatusOK, BrandingResponse{
		BrandName:         profile.BrandName,
		BrandLogoURL:      profile.BrandLogoURL,
		CustomNameservers: profile.CustomNameservers,
		CustomEmailSender: profile.CustomEmailSender,
	}, nil)
}
