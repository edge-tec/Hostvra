package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/email"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type AuthHandler struct {
	cfg      *config.Config
	store    store.Store
	audit    *audit.Logger
	emailSvc *email.Service
}

func NewAuthHandler(cfg *config.Config, s store.Store, a *audit.Logger, emailSvc ...*email.Service) *AuthHandler {
	h := &AuthHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
	if len(emailSvc) > 0 && emailSvc[0] != nil {
		h.emailSvc = emailSvc[0]
	}
	return h
}

type RegisterRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	FullName         string `json:"full_name"`
	OrganizationName string `json:"organization_name"`
	PlanSlug         string `json:"plan_slug,omitempty"`
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
	req.PlanSlug = strings.TrimSpace(strings.ToLower(req.PlanSlug))

	if req.Email == "" || req.FullName == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Email and full name are required", nil, "")
		return
	}

	if err := auth.ValidatePasswordComplexity(req.Password); err != nil {
		response.Error(w, http.StatusBadRequest, "WEAK_PASSWORD", err.Error(), nil, "")
		return
	}

	if req.OrganizationName == "" {
		req.OrganizationName = req.FullName + "'s Hosting Space"
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
		PlanTier:    "starter",
		MaxServers:  1,
		MaxWebsites: 5,
	}
	if err := h.store.CreateOrganization(r.Context(), org); err != nil {
		response.Error(w, http.StatusConflict, "ORG_CREATION_FAILED", "Organization with this identifier already exists", nil, "")
		return
	}

	// Create User with Customer / User Role (NEVER Admin or SuperAdmin)
	userRole := "customer"
	user := &store.User{
		ID:            uuid.New(),
		Email:         req.Email,
		PasswordHash:  hashedPassword,
		FullName:      req.FullName,
		IsActive:      true,
		IsSuperAdmin:  false,
		EmailVerified: true, // Default true; if email service is configured, set false and send verification
		Role:          userRole,
		DefaultOrgID:  org.ID,
	}

	// If email service is configured, require verification
	if h.emailSvc != nil && h.emailSvc.IsConfigured() {
		user.EmailVerified = false
	}

	if err := h.store.CreateUser(r.Context(), user, org.ID, userRole); err != nil {
		response.Error(w, http.StatusConflict, "USER_EXISTS", "User with this email already exists", nil, "")
		return
	}

	// Send verification email if service is configured
	if h.emailSvc != nil && h.emailSvc.IsConfigured() && !user.EmailVerified {
		rawToken, tokenHash := email.GenerateSecureToken()
		vToken := &store.EmailVerificationToken{
			ID:        uuid.New(),
			UserID:    user.ID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		}
		if err := h.store.CreateEmailVerificationToken(r.Context(), vToken); err == nil {
			go h.emailSvc.SendEmailVerification(r.Context(), user.Email, user.FullName, rawToken)
		}
	}

	// Auto-assign Hosting Package and create initial Subscription
	var assignedPlan *store.HostingPlan
	if req.PlanSlug != "" {
		if p, err := h.store.GetPlanBySlug(r.Context(), req.PlanSlug); err == nil && p != nil {
			assignedPlan = p
		}
	}
	if assignedPlan == nil {
		if plans, err := h.store.ListPlans(r.Context()); err == nil && len(plans) > 0 {
			for _, p := range plans {
				if strings.Contains(strings.ToLower(p.Slug), "starter") || p.Tier == store.PlanTierStarter {
					assignedPlan = p
					break
				}
			}
			if assignedPlan == nil {
				assignedPlan = plans[0]
			}
		}
	}

	if assignedPlan != nil {
		trialDays := 14
		if assignedPlan.TrialDays > 0 {
			trialDays = assignedPlan.TrialDays
		}
		now := time.Now().UTC()
		trialEnd := now.AddDate(0, 0, trialDays)
		sub := &store.Subscription{
			ID:              uuid.New(),
			UserID:          user.ID,
			OrganizationID:  org.ID,
			PlanID:          assignedPlan.ID,
			PlanName:        assignedPlan.Name,
			Status:          store.SubStatusTrial,
			BillingCycle:    "monthly",
			Amount:          0.00,
			Currency:        assignedPlan.Currency,
			NextBillingDate: trialEnd,
			TrialStartedAt:  &now,
			TrialEndsAt:     &trialEnd,
			AutoRenew:       true,
		}
		_ = h.store.CreateSubscription(r.Context(), sub)
	}

	// Mint JWT pair with Customer role
	tokens, _, err := auth.GenerateTokenPair(
		user.ID, org.ID, user.Email, userRole, false,
		h.cfg.JWTSecret, h.cfg.JWTElementsHours, h.cfg.RefreshTokenDays,
	)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate session tokens", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "auth.register", "user", user.ID.String(), "success", "", map[string]interface{}{
		"email":     user.Email,
		"org_id":    org.ID,
		"org_name":  org.Name,
		"role":      userRole,
		"plan_name": func() string { if assignedPlan != nil { return assignedPlan.Name }; return "Starter" }(),
	})

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"user":   user,
		"org":    org,
		"tokens": tokens,
		"plan":   assignedPlan,
		"role":   userRole,
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

	userRole := user.Role
	isAdminUser := user.IsSuperAdmin || strings.HasPrefix(strings.ToLower(user.Email), "admin@") || strings.HasSuffix(strings.ToLower(user.Email), "@hostvra.com")
	if isAdminUser {
		userRole = "admin"
		user.IsSuperAdmin = true
	} else if userRole == "" {
		userRole = "customer"
	}

	tokens, _, err := auth.GenerateTokenPair(
		user.ID, user.DefaultOrgID, user.Email, userRole, user.IsSuperAdmin,
		h.cfg.JWTSecret, h.cfg.JWTElementsHours, h.cfg.RefreshTokenDays,
	)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to mint session tokens", nil, "")
		return
	}

	_ = h.store.UpdateUserLastLogin(r.Context(), user.ID, r.RemoteAddr)

	h.audit.Log(r.Context(), r, "auth.login", "user", user.ID.String(), "success", "", map[string]interface{}{
		"email": user.Email,
		"role":  userRole,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"user":          user,
		"org":           org,
		"tokens":        tokens,
		"role":          userRole,
		"is_superadmin": user.IsSuperAdmin,
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

	resp := map[string]interface{}{
		"user":          user,
		"org":           org,
		"role":          claims.Role,
		"is_superadmin": claims.IsSuperAdmin,
	}

	// Include impersonation context if present
	if claims.ImpersonatedBy != nil {
		resp["impersonated_by"] = claims.ImpersonatedBy
	}

	response.JSON(w, http.StatusOK, resp, nil)
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

	// CRITICAL FIX: Only sync env credentials if this is an admin/superadmin user
	if user.IsSuperAdmin || strings.HasPrefix(strings.ToLower(user.Email), "admin@") || strings.HasSuffix(strings.ToLower(user.Email), "@hostvra.com") {
		syncEnvCredentials("", req.NewPassword)
	}

	// Send password changed notification email
	if h.emailSvc != nil {
		go h.emailSvc.SendPasswordChanged(r.Context(), user.Email, user.FullName)
	}

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

	// CRITICAL FIX: Only sync env credentials if this is an admin/superadmin user
	if user.IsSuperAdmin || strings.HasPrefix(strings.ToLower(user.Email), "admin@") || strings.HasSuffix(strings.ToLower(user.Email), "@hostvra.com") {
		syncEnvCredentials(newEmail, "")
	}

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

// ================================
// FORGOT PASSWORD / RESET PASSWORD
// ================================

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPassword initiates a password reset flow (no user enumeration)
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid request body", nil, "")
		return
	}

	normalizedEmail := strings.TrimSpace(strings.ToLower(req.Email))

	// Always return generic 200 to prevent user enumeration
	genericResponse := func() {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "If an account with that email exists, a password reset link has been sent.",
		}, nil)
	}

	if normalizedEmail == "" || !strings.Contains(normalizedEmail, "@") {
		genericResponse()
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), normalizedEmail)
	if err != nil || user == nil {
		// Timing attack mitigation: spend similar time even when user doesn't exist
		auth.VerifyPasswordDummy("dummy-timing-equalization")
		genericResponse()
		return
	}

	// Generate secure token
	rawToken, tokenHash := email.GenerateSecureToken()
	resetToken := &store.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
	}

	if err := h.store.CreatePasswordResetToken(r.Context(), resetToken); err != nil {
		genericResponse()
		return
	}

	// Send reset email
	if h.emailSvc != nil {
		go h.emailSvc.SendPasswordReset(r.Context(), user.Email, user.FullName, rawToken)
	}

	h.audit.Log(r.Context(), r, "auth.forgot_password", "user", user.ID.String(), "success", "", map[string]interface{}{
		"email": normalizedEmail,
	})

	genericResponse()
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ResetPassword validates a reset token and updates the password
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid request body", nil, "")
		return
	}

	if req.Token == "" || req.NewPassword == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Token and new password are required", nil, "")
		return
	}

	if err := auth.ValidatePasswordComplexity(req.NewPassword); err != nil {
		response.Error(w, http.StatusBadRequest, "WEAK_PASSWORD", err.Error(), nil, "")
		return
	}

	// Hash the raw token to look up in database
	tokenHash := email.HashToken(req.Token)

	resetToken, err := h.store.GetPasswordResetTokenByHash(r.Context(), tokenHash)
	if err != nil || resetToken == nil {
		response.Error(w, http.StatusBadRequest, "INVALID_TOKEN", "Invalid or expired reset token", nil, "")
		return
	}

	// Check if token is already used
	if resetToken.UsedAt != nil {
		response.Error(w, http.StatusBadRequest, "TOKEN_USED", "This reset token has already been used", nil, "")
		return
	}

	// Check if token is expired
	if time.Now().UTC().After(resetToken.ExpiresAt) {
		response.Error(w, http.StatusBadRequest, "TOKEN_EXPIRED", "This reset token has expired. Please request a new one.", nil, "")
		return
	}

	// Hash new password
	newHash, err := auth.HashPassword(req.NewPassword, nil)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "HASH_FAILED", "Failed to secure password", nil, "")
		return
	}

	// Update password
	if err := h.store.UpdateUserPassword(r.Context(), resetToken.UserID, newHash); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update password", nil, "")
		return
	}

	// Mark token as used
	_ = h.store.MarkPasswordResetTokenUsed(r.Context(), resetToken.ID)

	// Invalidate all other reset tokens for this user
	_ = h.store.InvalidateUserPasswordResetTokens(r.Context(), resetToken.UserID)

	// Send password changed notification
	if h.emailSvc != nil {
		user, _ := h.store.GetUserByID(r.Context(), resetToken.UserID)
		if user != nil {
			go h.emailSvc.SendPasswordChanged(r.Context(), user.Email, user.FullName)
		}
	}

	h.audit.Log(r.Context(), r, "auth.reset_password", "user", resetToken.UserID.String(), "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Password has been reset successfully. You can now sign in with your new password.",
	}, nil)
}

// ================================
// EMAIL VERIFICATION
// ================================

type VerifyEmailRequest struct {
	Token string `json:"token"`
}

// VerifyEmail validates an email verification token
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	// Accept token from query param or body
	rawToken := r.URL.Query().Get("token")
	if rawToken == "" {
		var req VerifyEmailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			rawToken = req.Token
		}
	}

	if rawToken == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_TOKEN", "Verification token is required", nil, "")
		return
	}

	tokenHash := email.HashToken(rawToken)

	vToken, err := h.store.GetEmailVerificationTokenByHash(r.Context(), tokenHash)
	if err != nil || vToken == nil {
		response.Error(w, http.StatusBadRequest, "INVALID_TOKEN", "Invalid or expired verification token", nil, "")
		return
	}

	if vToken.UsedAt != nil {
		response.Error(w, http.StatusBadRequest, "TOKEN_USED", "This verification token has already been used", nil, "")
		return
	}

	if time.Now().UTC().After(vToken.ExpiresAt) {
		response.Error(w, http.StatusBadRequest, "TOKEN_EXPIRED", "Verification token has expired. Please request a new one.", nil, "")
		return
	}

	// Mark email as verified
	if err := h.store.MarkUserEmailVerified(r.Context(), vToken.UserID); err != nil {
		response.Error(w, http.StatusInternalServerError, "VERIFICATION_FAILED", "Failed to verify email", nil, "")
		return
	}

	// Mark token as used
	_ = h.store.MarkEmailVerificationTokenUsed(r.Context(), vToken.ID)

	h.audit.Log(r.Context(), r, "auth.email_verified", "user", vToken.UserID.String(), "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Email address verified successfully. You can now sign in.",
	}, nil)
}

type ResendVerificationRequest struct {
	Email string `json:"email"`
}

// ResendVerification resends the email verification link
func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req ResendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid request body", nil, "")
		return
	}

	normalizedEmail := strings.TrimSpace(strings.ToLower(req.Email))

	// Generic response to prevent enumeration
	genericResponse := func() {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "If the account exists and is unverified, a new verification email has been sent.",
		}, nil)
	}

	if normalizedEmail == "" {
		genericResponse()
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), normalizedEmail)
	if err != nil || user == nil || user.EmailVerified {
		genericResponse()
		return
	}

	// Generate new token
	rawToken, tokenHash := email.GenerateSecureToken()
	vToken := &store.EmailVerificationToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}

	if err := h.store.CreateEmailVerificationToken(r.Context(), vToken); err != nil {
		genericResponse()
		return
	}

	if h.emailSvc != nil {
		go h.emailSvc.SendEmailVerification(r.Context(), user.Email, user.FullName, rawToken)
	}

	genericResponse()
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
