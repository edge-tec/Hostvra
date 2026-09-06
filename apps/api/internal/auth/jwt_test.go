package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTTokens(t *testing.T) {
	secret := "test-secret-key-1234567890-hostvra-platform"
	userID := uuid.New()
	orgID := uuid.New()
	email := "admin@hostvra.com"
	role := "owner"

	tokens, hash, err := GenerateTokenPair(userID, orgID, email, role, false, secret, 1*time.Hour, 24*time.Hour)
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}

	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("Expected tokens to be populated")
	}

	if hash == "" {
		t.Fatal("Expected refresh token hash to be populated")
	}

	// Validate Access Token
	claims, err := ValidateAccessToken(tokens.AccessToken, secret)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}

	if claims.UserID != userID || claims.Email != email || claims.Role != role {
		t.Fatalf("Claims mismatch: got %v, expected user %v, email %s, role %s", claims, userID, email, role)
	}

	// Test invalid secret rejection
	_, err = ValidateAccessToken(tokens.AccessToken, "tampered-wrong-secret")
	if err == nil {
		t.Fatal("Expected token validation to fail with wrong secret")
	}
}
