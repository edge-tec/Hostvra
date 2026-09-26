package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type AuthHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewAuthHandler(cfg *config.Config, s store.Store, a *audit.Logger) *AuthHandler {
	return &AuthHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
}

type RegisterRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	FullName         string `json:"full_name"`
	OrganizationName string `json:"organization_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.FullName = strings.TrimSpace(req.FullName)
	req.OrganizationName = strings.TrimSpace(req.OrganizationName)

	if req.Email == "" || req.FullName == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Email and full name are required", nil, "")
		return
	}

	if err := auth.ValidatePasswordComplexity(req.Password); err != nil {
		response.Error(w, http.StatusBadRequest, "WEAK_PASSWORD", err.Error(), nil, "")
		return
	}

	if req.OrganizationName == "" {
		req.OrganizationName = req.FullName + "'s Org"
	}

	// Hash password with Argon2id
	hashedPassword, err := auth.HashPassword(req.Password, nil)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to secure password", nil, "")
		return
	}

	// Create Organization
	orgSlug := strings.ToLower(strings.ReplaceAll(req.OrganizationName, " ", "-")) + "-" + uuid.New().String()[:8]
	org := &store.Organization{
		ID:          uuid.New(),
		Name:        req.OrganizationName,
		Slug:        orgSlug,
		PlanTier:    "free",
		MaxServers:  1,
		MaxWebsites: 5,
	}
	if err := h.store.CreateOrganization(r.Context(), org); err != nil {
		response.Error(w, http.StatusConflict, "ORG_CREATION_FAILED", "Organization with this identifier already exists", nil, "")
		return
	}

	// Create User as Owner
	user := &store.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FullName:     req.FullName,
		IsActive:     true,
		IsSuperAdmin: false,
	}

	if err := h.store.CreateUser(r.Context(), user, org.ID, "owner"); err != nil {
		response.Error(w, http.StatusConflict, "USER_EXISTS", "User with this email already exists", nil, "")
		return
	}

	// Mint JWT pair
	tokens, _, err := auth.GenerateTokenPair(
		user.ID, org.ID, user.Email, "owner", false,
		h.cfg.JWTSecret, h.cfg.JWTElementsHours, h.cfg.RefreshTokenDays,
	)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate session tokens", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "auth.register", "user", user.ID.String(), "success", "", map[string]interface{}{
		"email":   user.Email,
		"org_id":  org.ID,
		"org_name": org.Name,
	})

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"user":   user,
		"org":    org,
		"tokens": tokens,
	}, nil)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	user, err := h.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		auth.VerifyPasswordDummy(req.Password)
		response.Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password", nil, "")
		return
	}

	valid, err := auth.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !valid {
		h.audit.Log(r.Context(), r, "auth.login", "user", user.ID.String(), "failure", "Invalid password", map[string]interface{}{
			"email": req.Email,
		})
		response.Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password", nil, "")
		return
	}

	if !user.IsActive {
		response.Error(w, http.StatusForbidden, "ACCOUNT_SUSPENDED", "User account has been deactivated", nil, "")
		return
	}

	// Retrieve Org
	org, _ := h.store.GetOrganizationByID(r.Context(), user.DefaultOrgID)

	tokens, _, err := auth.GenerateTokenPair(
		user.ID, user.DefaultOrgID, user.Email, user.Role, user.IsSuperAdmin,
		h.cfg.JWTSecret, h.cfg.JWTElementsHours, h.cfg.RefreshTokenDays,
	)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to mint session tokens", nil, "")
		return
	}

	_ = h.store.UpdateUserLastLogin(r.Context(), user.ID, r.RemoteAddr)

	h.audit.Log(r.Context(), r, "auth.login", "user", user.ID.String(), "success", "", map[string]interface{}{
		"email": user.Email,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"user":   user,
		"org":    org,
		"tokens": tokens,
	}, nil)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication claims", nil, "")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "User profile not found", nil, "")
		return
	}

	org, _ := h.store.GetOrganizationByID(r.Context(), claims.OrganizationID)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"user": user,
		"org":  org,
		"role": claims.Role,
	}, nil)
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword updates the logged-in user's password
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NewPassword == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "New password is required", nil, "")
		return
	}

	if len(req.NewPassword) < 6 {
		response.Error(w, http.StatusBadRequest, "WEAK_PASSWORD", "Password must be at least 6 characters", nil, "")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "User profile not found", nil, "")
		return
	}

	// Verify current password if provided
	if req.CurrentPassword != "" {
		valid, _ := auth.VerifyPassword(req.CurrentPassword, user.PasswordHash)
		if !valid {
			response.Error(w, http.StatusUnauthorized, "INVALID_CURRENT_PASSWORD", "Current password does not match", nil, "")
			return
		}
	}

	newHash, err := auth.HashPassword(req.NewPassword, nil)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "HASH_FAILED", "Failed to secure password", nil, "")
		return
	}

	if err := h.store.UpdateUserPassword(r.Context(), user.ID, newHash); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update password", nil, "")
		return
	}

	// Persist to /etc/hostvra/api.env if running on server
	syncEnvCredentials("", req.NewPassword)

	h.audit.Log(r.Context(), r, "auth.password_change", "user", user.ID.String(), "success", "", map[string]interface{}{
		"email": user.Email,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Password updated successfully",
	}, nil)
}

type ChangeEmailRequest struct {
	Email string `json:"email"`
}

// ChangeEmail updates the logged-in user's email address
func (h *AuthHandler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	var req ChangeEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid request body", nil, "")
		return
	}

	newEmail := strings.TrimSpace(strings.ToLower(req.Email))
	if newEmail == "" || !strings.Contains(newEmail, "@") {
		response.Error(w, http.StatusBadRequest, "INVALID_EMAIL", "A valid email address is required", nil, "")
		return
	}

	// Check if already used by another account
	existing, err := h.store.GetUserByEmail(r.Context(), newEmail)
	if err == nil && existing != nil && existing.ID != claims.UserID {
		response.Error(w, http.StatusConflict, "EMAIL_EXISTS", "This email is already in use by another account", nil, "")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "User profile not found", nil, "")
		return
	}

	if err := h.store.UpdateUserEmail(r.Context(), user.ID, newEmail); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update email address", nil, "")
		return
	}

	// Persist to /etc/hostvra/api.env if running on server
	syncEnvCredentials(newEmail, "")

	h.audit.Log(r.Context(), r, "auth.email_change", "user", user.ID.String(), "success", "", map[string]interface{}{
		"old_email": user.Email,
		"new_email": newEmail,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"email":   newEmail,
		"message": "Email address updated successfully",
	}, nil)
}

func syncEnvCredentials(newEmail, newPassword string) {
	envPath := "/etc/hostvra/api.env"
	data, err := os.ReadFile(envPath)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if newEmail != "" && strings.HasPrefix(line, "INITIAL_ADMIN_EMAIL=") {
			lines[i] = "INITIAL_ADMIN_EMAIL=" + newEmail
		}
		if newPassword != "" && strings.HasPrefix(line, "INITIAL_ADMIN_PASSWORD=") {
			lines[i] = "INITIAL_ADMIN_PASSWORD=" + newPassword
		}
	}
	_ = os.WriteFile(envPath, []byte(strings.Join(lines, "\n")), 0600)
}
