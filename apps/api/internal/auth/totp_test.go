package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

func TestTOTP_Encryption(t *testing.T) {
	appSecret := "super-secure-production-jwt-key-32-chars-long!"
	secret := "JBSWY3DPEHPK3PXP"

	ciphertext, err := EncryptSecret(secret, appSecret)
	if err != nil {
		t.Fatalf("failed to encrypt secret: %v", err)
	}
	if ciphertext == secret {
		t.Fatalf("ciphertext must not equal plaintext")
	}

	decrypted, err := DecryptSecret(ciphertext, appSecret)
	if err != nil {
		t.Fatalf("failed to decrypt secret: %v", err)
	}
	if decrypted != secret {
		t.Fatalf("expected decrypted '%s', got '%s'", secret, decrypted)
	}
}

func TestTOTP_GenerateAndValidate(t *testing.T) {
	secret, url, qrCode, err := GenerateTOTPSecret("admin@hostvra.com")
	if err != nil {
		t.Fatalf("failed to generate TOTP secret: %v", err)
	}
	if secret == "" || url == "" || qrCode == "" {
		t.Fatalf("expected non-empty secret, url, and qrCode")
	}

	// Generate valid code for current time
	passcode, err := totp.GenerateCodeCustom(secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	if !ValidateTOTPCode(passcode, secret) {
		t.Fatalf("expected passcode '%s' to be valid for secret '%s'", passcode, secret)
	}

	// Bad code must fail
	if ValidateTOTPCode("000000", secret) && passcode != "000000" {
		t.Fatalf("expected invalid passcode to fail")
	}
}

func TestTOTP_RecoveryCodes(t *testing.T) {
	rawCodes, hashes, err := GenerateRecoveryCodes(8)
	if err != nil {
		t.Fatalf("failed to generate recovery codes: %v", err)
	}
	if len(rawCodes) != 8 || len(hashes) != 8 {
		t.Fatalf("expected 8 codes and hashes")
	}

	for i, raw := range rawCodes {
		computedHash := HashRecoveryCode(raw)
		if computedHash != hashes[i] {
			t.Fatalf("hash mismatch for code %s: expected %s, got %s", raw, hashes[i], computedHash)
		}

		// Also check with lowercase and without dashes
		clean := NormalizeRecoveryCode(raw)
		if HashRecoveryCode(clean) != hashes[i] {
			t.Fatalf("normalized hash mismatch for code %s", clean)
		}
	}
}

func TestTOTP_ChallengeToken(t *testing.T) {
	jwtSecret := "my-jwt-secret-key-12345"
	userID := uuid.New()
	orgID := uuid.New()

	tokenStr, err := GenerateTwoFactorChallengeToken(userID, orgID, "user@example.com", "customer", false, jwtSecret)
	if err != nil {
		t.Fatalf("failed to generate challenge token: %v", err)
	}

	claims, err := ValidateTwoFactorChallengeToken(tokenStr, jwtSecret)
	if err != nil {
		t.Fatalf("failed to validate challenge token: %v", err)
	}
	if claims.UserID != userID || claims.Email != "user@example.com" || claims.Purpose != "2fa_challenge" {
		t.Fatalf("claims mismatch: %+v", claims)
	}

	// Invalid secret must fail
	_, err = ValidateTwoFactorChallengeToken(tokenStr, "wrong-secret")
	if err == nil {
		t.Fatalf("expected validation with wrong secret to fail")
	}
}
