package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid or expired authentication token")
)

type Claims struct {
	UserID         uuid.UUID `json:"user_id"`
	Email          string    `json:"email"`
	OrganizationID uuid.UUID `json:"org_id"`
	Role           string    `json:"role"`
	IsSuperAdmin   bool      `json:"is_superadmin"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func GenerateTokenPair(userID, orgID uuid.UUID, email, role string, isSuperAdmin bool, secret string, accessDuration, refreshDuration time.Duration) (*TokenPair, string, error) {
	now := time.Now().UTC()
	accessExpiresAt := now.Add(accessDuration)

	claims := &Claims{
		UserID:         userID,
		Email:          email,
		OrganizationID: orgID,
		Role:           role,
		IsSuperAdmin:   isSuperAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "hostvra-core-api",
			Subject:   userID.String(),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessTokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, "", err
	}

	// Generate opaque high-entropy refresh token
	refreshBytes := make([]byte, 32)
	if _, err := rand.Read(refreshBytes); err != nil {
		return nil, "", err
	}
	refreshToken := hex.EncodeToString(refreshBytes)

	// Compute SHA256 of refresh token for database storage
	h := sha256.Sum256([]byte(refreshToken))
	refreshTokenHash := hex.EncodeToString(h[:])

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpiresAt,
	}, refreshTokenHash, nil
}

func ValidateAccessToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func HashOpaqueToken(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(h[:])
}
