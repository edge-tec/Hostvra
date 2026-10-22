package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestAuth2FA_CompleteLifecycle(t *testing.T) {
	s := store.NewMemoryStore()
	cfg := &config.Config{
		JWTSecret:         "test-super-secret-jwt-key-for-2fa-testing-32b",
		JWTElementsHours:  1,
		RefreshTokenDays: 7,
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)
	authHandler := NewAuthHandler(cfg, s, auditLogger)

	// 1. Create Organization & User
	orgID := uuid.New()
	_ = s.CreateOrganization(context.Background(), &store.Organization{
		ID:   orgID,
		Name: "Test 2FA Org",
		Slug: "test-2fa-org",
	})

	userID := uuid.New()
	rawPassword := "SecurePassword123!"
	pwdHash, _ := auth.HashPassword(rawPassword, nil)
	user := &store.User{
		ID:               userID,
		Email:            "totpuser@example.com",
		PasswordHash:     pwdHash,
		FullName:         "TOTP Tester",
		IsActive:         true,
		TwoFactorEnabled: false,
		DefaultOrgID:     orgID,
		Role:             "customer",
	}
	_ = s.CreateUser(context.Background(), user, orgID, "customer")

	// 2. Normal Login (2FA not enabled yet)
	loginBody, _ := json.Marshal(map[string]string{
		"email":    user.Email,
		"password": rawPassword,
	})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	rec := httptest.NewRecorder()
	authHandler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on initial login, got %d: %s", rec.Code, rec.Body.String())
	}
	var loginRes struct {
		Data struct {
			TwoFactorRequired bool            `json:"two_factor_required"`
			Tokens            *auth.TokenPair `json:"tokens"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &loginRes)
	if loginRes.Data.TwoFactorRequired || loginRes.Data.Tokens == nil {
		t.Fatalf("expected direct login tokens without 2FA required")
	}

	accessToken := loginRes.Data.Tokens.AccessToken

	// Helper to create authenticated request with claims context
	authCtx := func(r *http.Request) *http.Request {
		claims := &auth.Claims{
			UserID:         userID,
			OrganizationID: orgID,
			Email:          user.Email,
			Role:           "customer",
		}
		return r.WithContext(auth.WithClaims(r.Context(), claims))
	}

	// 3. Initiate 2FA Setup with wrong password -> 401
	setupBadBody, _ := json.Marshal(map[string]string{"current_password": "WrongPassword!"})
	req = httptest.NewRequest("POST", "/api/v1/auth/2fa/setup", bytes.NewReader(setupBadBody))
	req = authCtx(req)
	rec = httptest.NewRecorder()
	authHandler.Setup2FA(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on 2fa setup with bad password, got %d", rec.Code)
	}

	// 4. Initiate 2FA Setup with correct password -> 200 (returns secret & QR code data URI)
	setupGoodBody, _ := json.Marshal(map[string]string{"current_password": rawPassword})
	req = httptest.NewRequest("POST", "/api/v1/auth/2fa/setup", bytes.NewReader(setupGoodBody))
	req = authCtx(req)
	rec = httptest.NewRecorder()
	authHandler.Setup2FA(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on 2fa setup, got %d: %s", rec.Code, rec.Body.String())
	}

	var setupRes struct {
		Data struct {
			Secret     string `json:"secret"`
			OtpauthURL string `json:"otpauth_url"`
			QRCode     string `json:"qr_code"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &setupRes)
	if setupRes.Data.Secret == "" || setupRes.Data.QRCode == "" {
		t.Fatalf("expected secret and qr_code in setup response")
	}

	totpSecret := setupRes.Data.Secret

	// 5. Try enabling 2FA with invalid code -> 400
	badEnableBody, _ := json.Marshal(map[string]string{"code": "000000"})
	req = httptest.NewRequest("POST", "/api/v1/auth/2fa/enable", bytes.NewReader(badEnableBody))
	req = authCtx(req)
	rec = httptest.NewRecorder()
	authHandler.Enable2FA(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on enable with invalid code, got %d", rec.Code)
	}

	// 6. Enable 2FA with valid 6-digit TOTP code -> 200 (returns 8 recovery codes)
	validCode, _ := totp.GenerateCodeCustom(totpSecret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	goodEnableBody, _ := json.Marshal(map[string]string{"code": validCode})
	req = httptest.NewRequest("POST", "/api/v1/auth/2fa/enable", bytes.NewReader(goodEnableBody))
	req = authCtx(req)
	rec = httptest.NewRecorder()
	authHandler.Enable2FA(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on enable with valid code, got %d: %s", rec.Code, rec.Body.String())
	}

	var enableRes struct {
		Data struct {
			Success       bool     `json:"success"`
			RecoveryCodes []string `json:"recovery_codes"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &enableRes)
	if !enableRes.Data.Success || len(enableRes.Data.RecoveryCodes) != 8 {
		t.Fatalf("expected success and 8 recovery codes, got %d codes", len(enableRes.Data.RecoveryCodes))
	}
	recoveryCodes := enableRes.Data.RecoveryCodes

	// 7. Check 2FA Status -> enabled = true, remaining = 8
	req = httptest.NewRequest("GET", "/api/v1/auth/2fa/status", nil)
	req = authCtx(req)
	rec = httptest.NewRecorder()
	authHandler.Get2FAStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on 2fa status, got %d", rec.Code)
	}
	var statusRes struct {
		Data struct {
			TwoFactorEnabled       bool `json:"two_factor_enabled"`
			RecoveryCodesRemaining int  `json:"recovery_codes_remaining"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &statusRes)
	if !statusRes.Data.TwoFactorEnabled || statusRes.Data.RecoveryCodesRemaining != 8 {
		t.Fatalf("expected 2FA enabled with 8 codes, got: %+v", statusRes.Data)
	}

	// 8. Now attempt Login -> MUST require 2FA challenge!
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	rec = httptest.NewRecorder()
	authHandler.Login(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with 2FA required on login, got %d", rec.Code)
	}
	var challengeRes struct {
		Data struct {
			TwoFactorRequired bool   `json:"two_factor_required"`
			TempToken         string `json:"temp_token"`
			UserID            string `json:"user_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &challengeRes)
	if !challengeRes.Data.TwoFactorRequired || challengeRes.Data.TempToken == "" {
		t.Fatalf("expected two_factor_required=true and temp_token")
	}

	challengeToken := challengeRes.Data.TempToken

	// 9. Verify 2FA challenge with valid TOTP code -> 200 with full tokens!
	validCode2, _ := totp.GenerateCodeCustom(totpSecret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	verifyBody, _ := json.Marshal(map[string]string{
		"temp_token": challengeToken,
		"code":       validCode2,
	})
	req = httptest.NewRequest("POST", "/api/v1/auth/2fa/verify", bytes.NewReader(verifyBody))
	rec = httptest.NewRecorder()
	authHandler.Verify2FAChallenge(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on 2fa verify, got %d: %s", rec.Code, rec.Body.String())
	}
	var verifyRes struct {
		Data struct {
			Tokens *auth.TokenPair `json:"tokens"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &verifyRes)
	if verifyRes.Data.Tokens == nil || verifyRes.Data.Tokens.AccessToken == "" {
		t.Fatalf("expected tokens after 2fa verification")
	}

	// 10. Test Login using Recovery Code
	// Get fresh challenge token
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	rec = httptest.NewRecorder()
	authHandler.Login(rec, req)
	_ = json.Unmarshal(rec.Body.Bytes(), &challengeRes)
	challengeToken2 := challengeRes.Data.TempToken

	// Use first recovery code
	usedRecoveryCode := recoveryCodes[0]
	verifyRecoveryBody, _ := json.Marshal(map[string]string{
		"temp_token": challengeToken2,
		"code":       usedRecoveryCode,
	})
	req = httptest.NewRequest("POST", "/api/v1/auth/2fa/verify", bytes.NewReader(verifyRecoveryBody))
	rec = httptest.NewRecorder()
	authHandler.Verify2FAChallenge(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on 2fa verify with recovery code, got %d: %s", rec.Code, rec.Body.String())
	}

	// 11. Test that the used recovery code CANNOT be used again (single-use protection)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	rec = httptest.NewRecorder()
	authHandler.Login(rec, req)
	_ = json.Unmarshal(rec.Body.Bytes(), &challengeRes)
	challengeToken3 := challengeRes.Data.TempToken

	req = httptest.NewRequest("POST", "/api/v1/auth/2fa/verify", bytes.NewReader(verifyRecoveryBody))
	rec = httptest.NewRecorder()
	authHandler.Verify2FAChallenge(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when reusing consumed recovery code, got %d", rec.Code)
	}

	// Verify remaining recovery count decreased to 7
	req = httptest.NewRequest("GET", "/api/v1/auth/2fa/status", nil)
	req = authCtx(req)
	rec = httptest.NewRecorder()
	authHandler.Get2FAStatus(rec, req)
	_ = json.Unmarshal(rec.Body.Bytes(), &statusRes)
	if statusRes.Data.RecoveryCodesRemaining != 7 {
		t.Fatalf("expected 7 recovery codes remaining after using 1, got %d", statusRes.Data.RecoveryCodesRemaining)
	}

	// 12. Regenerate Recovery Codes
	regenBody, _ := json.Marshal(map[string]string{"current_password": rawPassword})
	req = httptest.NewRequest("POST", "/api/v1/auth/2fa/recovery-codes", bytes.NewReader(regenBody))
	req = authCtx(req)
	rec = httptest.NewRecorder()
	authHandler.RegenerateRecoveryCodes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on regenerate recovery codes, got %d: %s", rec.Code, rec.Body.String())
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &enableRes)
	if len(enableRes.Data.RecoveryCodes) != 8 {
		t.Fatalf("expected 8 newly generated recovery codes")
	}

	// 13. Disable 2FA
	disableCode, _ := totp.GenerateCodeCustom(totpSecret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	disableBody, _ := json.Marshal(map[string]string{
		"current_password": rawPassword,
		"code":             disableCode,
	})
	req = httptest.NewRequest("POST", "/api/v1/auth/2fa/disable", bytes.NewReader(disableBody))
	req = authCtx(req)
	rec = httptest.NewRecorder()
	authHandler.Disable2FA(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on disable 2fa, got %d: %s", rec.Code, rec.Body.String())
	}

	// 14. Verify Login is now direct again without 2FA
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	rec = httptest.NewRecorder()
	authHandler.Login(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on login after 2fa disabled, got %d", rec.Code)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &loginRes)
	if loginRes.Data.TwoFactorRequired || loginRes.Data.Tokens == nil {
		t.Fatalf("expected direct login tokens after 2fa disabled")
	}
	_ = accessToken
	_ = challengeToken3
}
