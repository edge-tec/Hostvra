package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"hostvra/api/internal/auth"
	"hostvra/api/internal/response"
)

type Verify2FARequest struct {
	TempToken string `json:"temp_token"`
	Code      string `json:"code"`
}

type Setup2FARequest struct {
	CurrentPassword string `json:"current_password"`
}

type Enable2FARequest struct {
	Code string `json:"code"`
}

type Disable2FARequest struct {
	CurrentPassword string `json:"current_password"`
	Code            string `json:"code"`
}

type RegenerateRecoveryCodesRequest struct {
	CurrentPassword string `json:"current_password"`
}

// Verify2FAChallenge verifies an ephemeral 2FA challenge code or recovery code during login.
func (h *AuthHandler) Verify2FAChallenge(w http.ResponseWriter, r *http.Request) {
	var req Verify2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	req.TempToken = strings.TrimSpace(req.TempToken)
	req.Code = strings.TrimSpace(req.Code)
	if req.TempToken == "" || req.Code == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Both temp_token and code are required", nil, "")
		return
	}

	// 1. Validate challenge token signature & expiration
	claims, err := auth.ValidateTwoFactorChallengeToken(req.TempToken, h.cfg.JWTSecret)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "INVALID_CHALLENGE", "Two-factor challenge token has expired or is invalid. Please sign in again.", nil, "")
		return
	}

	// 2. Fetch encrypted secret and verify 2FA is active
	encSecret, enabled, err := h.store.GetUserTwoFactorSecret(r.Context(), claims.UserID)
	if err != nil || !enabled {
		response.Error(w, http.StatusBadRequest, "2FA_NOT_ENABLED", "Two-factor authentication is not active for this account", nil, "")
		return
	}

	verified := false
	isRecoveryCode := false

	// 3. Try standard 6-digit TOTP code
	cleanCode := strings.TrimSpace(req.Code)
	if len(cleanCode) == 6 {
		decryptedSecret, decErr := auth.DecryptSecret(encSecret, h.cfg.JWTSecret)
		if decErr == nil && auth.ValidateTOTPCode(cleanCode, decryptedSecret) {
			verified = true
		}
	}

	// 4. If TOTP failed or user provided a recovery code, test recovery code hashes
	if !verified {
		normRecovery := auth.NormalizeRecoveryCode(cleanCode)
		if len(normRecovery) >= 8 {
			codeHash := auth.HashRecoveryCode(normRecovery)
			consumed, recErr := h.store.VerifyAndConsumeTwoFactorRecoveryCode(r.Context(), claims.UserID, codeHash)
			if recErr == nil && consumed {
				verified = true
				isRecoveryCode = true
			}
		}
	}

	if !verified {
		h.audit.Log(r.Context(), r, "auth.login_2fa", "user", claims.UserID.String(), "failure", "Invalid code", map[string]interface{}{
			"email": claims.Email,
		})
		response.Error(w, http.StatusUnauthorized, "INVALID_CODE", "Invalid two-factor authentication code or recovery code", nil, "")
		return
	}

	// 5. Code is valid! Issue full JWT session tokens
	user, err := h.store.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "User account not found", nil, "")
		return
	}
	org, _ := h.store.GetOrganizationByID(r.Context(), claims.OrganizationID)

	tokens, _, err := auth.GenerateTokenPair(
		user.ID, claims.OrganizationID, user.Email, claims.Role, user.IsSuperAdmin,
		h.cfg.JWTSecret, h.cfg.JWTElementsHours, h.cfg.RefreshTokenDays,
	)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to mint session tokens", nil, "")
		return
	}

	_ = h.store.UpdateUserLastLogin(r.Context(), user.ID, r.RemoteAddr)

	h.audit.Log(r.Context(), r, "auth.login_2fa", "user", user.ID.String(), "success", "", map[string]interface{}{
		"email":            user.Email,
		"role":             claims.Role,
		"is_recovery_code": isRecoveryCode,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"user":             user,
		"org":              org,
		"tokens":           tokens,
		"role":             claims.Role,
		"is_superadmin":    user.IsSuperAdmin,
		"used_recovery":    isRecoveryCode,
	}, nil)
}

// Get2FAStatus returns the 2FA enrollment status and count of remaining recovery codes.
func (h *AuthHandler) Get2FAStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	_, enabled, err := h.store.GetUserTwoFactorSecret(r.Context(), claims.UserID)
	if err != nil {
		enabled = false
	}

	remainingCodes := 0
	if enabled {
		remainingCodes, _ = h.store.CountValidTwoFactorRecoveryCodes(r.Context(), claims.UserID)
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"two_factor_enabled":       enabled,
		"recovery_codes_remaining": remainingCodes,
	}, nil)
}

// Setup2FA initiates TOTP enrollment, generates a secret, and returns the QR code data URI.
// Requires re-authenticating with current password for security.
func (h *AuthHandler) Setup2FA(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	var req Setup2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CurrentPassword == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Current account password is required to configure 2FA", nil, "")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "User profile not found", nil, "")
		return
	}

	// Verify current password
	valid, _ := auth.VerifyPassword(req.CurrentPassword, user.PasswordHash)
	if !valid {
		response.Error(w, http.StatusUnauthorized, "INVALID_PASSWORD", "Incorrect account password", nil, "")
		return
	}

	// Generate new TOTP secret & QR code
	secret, otpauthURL, qrCodeDataURL, err := auth.GenerateTOTPSecret(user.Email)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TOTP_GENERATE_ERROR", "Failed to generate two-factor secret: "+err.Error(), nil, "")
		return
	}

	// Encrypt secret at rest using application encryption key
	encSecret, err := auth.EncryptSecret(secret, h.cfg.JWTSecret)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "ENCRYPTION_ERROR", "Failed to secure two-factor secret", nil, "")
		return
	}

	// Save pending secret (enabled remains false until first token is verified)
	if err := h.store.UpdateUserTwoFactor(r.Context(), user.ID, false, encSecret); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to save pending two-factor secret", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "auth.2fa_setup_initiated", "user", user.ID.String(), "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"secret":      secret,
		"otpauth_url": otpauthURL,
		"qr_code":     qrCodeDataURL,
	}, nil)
}

// Enable2FA verifies the user's first 6-digit TOTP code, activates 2FA, and returns single-use recovery codes.
func (h *AuthHandler) Enable2FA(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	var req Enable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Code) == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "6-digit verification code is required", nil, "")
		return
	}

	encSecret, _, err := h.store.GetUserTwoFactorSecret(r.Context(), claims.UserID)
	if err != nil || encSecret == "" {
		response.Error(w, http.StatusBadRequest, "2FA_NOT_INITIALIZED", "Two-factor enrollment has not been initialized. Please start setup again.", nil, "")
		return
	}

	decryptedSecret, err := auth.DecryptSecret(encSecret, h.cfg.JWTSecret)
	if err != nil || !auth.ValidateTOTPCode(req.Code, decryptedSecret) {
		response.Error(w, http.StatusBadRequest, "INVALID_CODE", "Invalid verification code. Check your authenticator app time and enter the 6-digit code.", nil, "")
		return
	}

	// Generate 8 cryptographically secure recovery codes
	rawCodes, codeHashes, err := auth.GenerateRecoveryCodes(8)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "RECOVERY_CODE_ERROR", "Failed to generate recovery codes", nil, "")
		return
	}

	// Save recovery code hashes
	if err := h.store.SaveTwoFactorRecoveryCodes(r.Context(), claims.UserID, codeHashes); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to store recovery codes", nil, "")
		return
	}

	// Activate 2FA on the user account
	if err := h.store.UpdateUserTwoFactor(r.Context(), claims.UserID, true, encSecret); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to activate two-factor authentication", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "auth.2fa_enabled", "user", claims.UserID.String(), "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"message":        "Two-factor authentication has been successfully enabled.",
		"recovery_codes": rawCodes,
	}, nil)
}

// Disable2FA disables two-factor authentication.
// Requires re-authenticating with current password and a valid TOTP code or recovery code.
func (h *AuthHandler) Disable2FA(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	var req Disable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Password and authentication code are required", nil, "")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "User profile not found", nil, "")
		return
	}

	// Verify current password
	valid, _ := auth.VerifyPassword(req.CurrentPassword, user.PasswordHash)
	if !valid {
		response.Error(w, http.StatusUnauthorized, "INVALID_PASSWORD", "Incorrect account password", nil, "")
		return
	}

	// Verify TOTP code or recovery code
	encSecret, enabled, _ := h.store.GetUserTwoFactorSecret(r.Context(), claims.UserID)
	if enabled {
		codeVerified := false
		cleanCode := strings.TrimSpace(req.Code)
		if len(cleanCode) == 6 {
			decryptedSecret, _ := auth.DecryptSecret(encSecret, h.cfg.JWTSecret)
			if auth.ValidateTOTPCode(cleanCode, decryptedSecret) {
				codeVerified = true
			}
		}
		if !codeVerified && len(cleanCode) >= 8 {
			codeHash := auth.HashRecoveryCode(cleanCode)
			consumed, _ := h.store.VerifyAndConsumeTwoFactorRecoveryCode(r.Context(), claims.UserID, codeHash)
			if consumed {
				codeVerified = true
			}
		}

		if !codeVerified {
			response.Error(w, http.StatusUnauthorized, "INVALID_CODE", "Invalid two-factor code or recovery code", nil, "")
			return
		}
	}

	// Disable 2FA, erase secret, and clear recovery codes
	if err := h.store.UpdateUserTwoFactor(r.Context(), claims.UserID, false, ""); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to disable two-factor authentication", nil, "")
		return
	}
	_ = h.store.SaveTwoFactorRecoveryCodes(r.Context(), claims.UserID, []string{})

	h.audit.Log(r.Context(), r, "auth.2fa_disabled", "user", claims.UserID.String(), "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Two-factor authentication has been disabled.",
	}, nil)
}

// RegenerateRecoveryCodes issues a new set of 8 recovery codes and invalidates all previous ones.
func (h *AuthHandler) RegenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	var req RegenerateRecoveryCodesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CurrentPassword == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Account password is required to regenerate recovery codes", nil, "")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "User profile not found", nil, "")
		return
	}

	valid, _ := auth.VerifyPassword(req.CurrentPassword, user.PasswordHash)
	if !valid {
		response.Error(w, http.StatusUnauthorized, "INVALID_PASSWORD", "Incorrect account password", nil, "")
		return
	}

	_, enabled, _ := h.store.GetUserTwoFactorSecret(r.Context(), claims.UserID)
	if !enabled {
		response.Error(w, http.StatusBadRequest, "2FA_NOT_ENABLED", "Two-factor authentication is not active", nil, "")
		return
	}

	rawCodes, codeHashes, err := auth.GenerateRecoveryCodes(8)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "RECOVERY_CODE_ERROR", "Failed to generate recovery codes", nil, "")
		return
	}

	if err := h.store.SaveTwoFactorRecoveryCodes(r.Context(), claims.UserID, codeHashes); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to store recovery codes", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "auth.2fa_recovery_regenerated", "user", claims.UserID.String(), "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"recovery_codes": rawCodes,
	}, nil)
}
