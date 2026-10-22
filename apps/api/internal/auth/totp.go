package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	qrcode "github.com/skip2/go-qrcode"
)

var (
	ErrInvalidTOTPToken       = errors.New("invalid or expired two-factor code")
	ErrInvalidRecoveryCode    = errors.New("invalid recovery code")
	ErrInvalidChallengeToken  = errors.New("invalid or expired two-factor session challenge")
	ErrTOTPAlreadyConfigured  = errors.New("two-factor authentication is already active")
	ErrTOTPNotConfigured      = errors.New("two-factor authentication is not configured for this account")
)

// TwoFactorChallengeClaims is an ephemeral, short-lived claim for 2FA verification during login.
type TwoFactorChallengeClaims struct {
	UserID         uuid.UUID  `json:"user_id"`
	OrganizationID uuid.UUID  `json:"org_id"`
	Email          string     `json:"email"`
	Role           string     `json:"role"`
	IsSuperAdmin   bool       `json:"is_superadmin"`
	Purpose        string     `json:"purpose"` // always "2fa_challenge"
	jwt.RegisteredClaims
}

// DeriveEncryptionKey computes a 32-byte key for AES-256 from application secret.
func DeriveEncryptionKey(appSecret string) []byte {
	h := sha256.Sum256([]byte(appSecret))
	return h[:]
}

// EncryptSecret encrypts plaintext using AES-256-GCM with a random nonce.
func EncryptSecret(plaintext string, appSecret string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key := DeriveEncryptionKey(appSecret)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes cipher error: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm cipher error: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// DecryptSecret decrypts hex-encoded ciphertext using AES-256-GCM.
func DecryptSecret(ciphertextHex string, appSecret string) (string, error) {
	if ciphertextHex == "" {
		return "", nil
	}
	data, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", fmt.Errorf("invalid hex ciphertext: %w", err)
	}
	key := DeriveEncryptionKey(appSecret)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes cipher error: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm cipher error: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secret: %w", err)
	}
	return string(plaintext), nil
}

// GenerateTOTPSecret creates a new RFC 6238 TOTP key, otpauth URL, and base64 PNG QR code data URL.
func GenerateTOTPSecret(accountEmail string) (secret string, otpauthURL string, qrCodeDataURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Hostvra",
		AccountName: accountEmail,
		Period:      30,
		SecretSize:  20, // 160 bits standard
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	secret = key.Secret()
	otpauthURL = key.URL()

	// Generate QR code PNG
	pngBytes, err := qrcode.Encode(otpauthURL, qrcode.Medium, 256)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate QR code: %w", err)
	}

	qrCodeDataURL = "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	return secret, otpauthURL, qrCodeDataURL, nil
}

// ValidateTOTPCode validates a 6-digit TOTP code against decrypted base32 secret with +/- 1 step skew.
func ValidateTOTPCode(passcode string, secret string) bool {
	cleanPasscode := strings.TrimSpace(passcode)
	if len(cleanPasscode) != 6 {
		return false
	}
	valid, err := totp.ValidateCustom(cleanPasscode, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      1, // Allow 1 step (30s) before and after for clock skew
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	return err == nil && valid
}

const recoveryCharset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ" // Crockford-style base32 without ambiguous characters

// GenerateRecoveryCodes produces count cryptographically secure random recovery codes and their SHA-256 hashes.
func GenerateRecoveryCodes(count int) (rawCodes []string, codeHashes []string, err error) {
	if count <= 0 {
		count = 8
	}
	rawCodes = make([]string, count)
	codeHashes = make([]string, count)

	for i := 0; i < count; i++ {
		bytes := make([]byte, 12)
		if _, err := rand.Read(bytes); err != nil {
			return nil, nil, fmt.Errorf("crypto rand error: %w", err)
		}

		var sb strings.Builder
		for j, b := range bytes {
			if j > 0 && j%4 == 0 {
				sb.WriteByte('-')
			}
			sb.WriteByte(recoveryCharset[int(b)%len(recoveryCharset)])
		}
		raw := sb.String()
		rawCodes[i] = raw
		codeHashes[i] = HashRecoveryCode(raw)
	}

	return rawCodes, codeHashes, nil
}

// NormalizeRecoveryCode cleans and normalizes a user-entered recovery code.
func NormalizeRecoveryCode(code string) string {
	clean := strings.ToUpper(strings.TrimSpace(code))
	clean = strings.ReplaceAll(clean, "-", "")
	clean = strings.ReplaceAll(clean, " ", "")
	return clean
}

// HashRecoveryCode computes the SHA-256 hex string of a normalized recovery code.
func HashRecoveryCode(code string) string {
	norm := NormalizeRecoveryCode(code)
	h := sha256.Sum256([]byte(norm))
	return hex.EncodeToString(h[:])
}

// GenerateTwoFactorChallengeToken creates a 5-minute ephemeral challenge token for 2FA login.
func GenerateTwoFactorChallengeToken(userID, orgID uuid.UUID, email, role string, isSuperAdmin bool, secret string) (string, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(5 * time.Minute)

	claims := &TwoFactorChallengeClaims{
		UserID:         userID,
		OrganizationID: orgID,
		Email:          email,
		Role:           role,
		IsSuperAdmin:   isSuperAdmin,
		Purpose:        "2fa_challenge",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "hostvra-core-api",
			Subject:   userID.String(),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateTwoFactorChallengeToken verifies an ephemeral 2FA challenge token.
func ValidateTwoFactorChallengeToken(tokenString, secret string) (*TwoFactorChallengeClaims, error) {
	parsedToken, err := jwt.ParseWithClaims(tokenString, &TwoFactorChallengeClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidChallengeToken
		}
		return []byte(secret), nil
	})

	if err != nil || !parsedToken.Valid {
		return nil, ErrInvalidChallengeToken
	}

	claims, ok := parsedToken.Claims.(*TwoFactorChallengeClaims)
	if !ok || claims.Purpose != "2fa_challenge" {
		return nil, ErrInvalidChallengeToken
	}

	return claims, nil
}
