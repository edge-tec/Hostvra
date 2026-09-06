package auth

import (
	"testing"
)

func TestArgon2idPasswordHashing(t *testing.T) {
	password := "SecureHostvraP@ssw0rd!2026"

	// Test fast params for quick unit testing
	testParams := &Params{
		Memory:      16 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}

	hash, err := HashPassword(password, testParams)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if len(hash) == 0 {
		t.Fatal("Hash returned is empty")
	}

	// Verify with correct password
	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed with error: %v", err)
	}
	if !valid {
		t.Fatal("Expected password verification to succeed")
	}

	// Verify with incorrect password
	invalid, err := VerifyPassword("WrongP@ssword", hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed with error: %v", err)
	}
	if invalid {
		t.Fatal("Expected password verification to fail for incorrect password")
	}
}
