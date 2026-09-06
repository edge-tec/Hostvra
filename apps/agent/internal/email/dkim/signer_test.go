package dkim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateDKIMKey(t *testing.T) {
	domain := "example.com"
	selector := "hostvra"

	key, err := GenerateDKIMKey(domain, selector, 2048)
	if err != nil {
		t.Fatalf("GenerateDKIMKey failed: %v", err)
	}

	if !strings.HasPrefix(key.PrivateKeyPEM, "-----BEGIN RSA PRIVATE KEY-----") {
		t.Errorf("expected PEM encoded private key")
	}

	if !strings.HasPrefix(key.PublicKeyDNS, "v=DKIM1; k=rsa; p=") {
		t.Errorf("expected DKIM DNS TXT format, got: %s", key.PublicKeyDNS)
	}

	tmpDir, err := os.MkdirTemp("", "hostvra-dkim-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := SaveDKIMKey(tmpDir, key); err != nil {
		t.Fatalf("SaveDKIMKey failed: %v", err)
	}

	privPath := filepath.Join(tmpDir, domain, selector+".private")
	if _, err := os.Stat(privPath); err != nil {
		t.Errorf("private key file missing: %v", err)
	}

	txtPath := filepath.Join(tmpDir, domain, selector+".txt")
	txtData, err := os.ReadFile(txtPath)
	if err != nil || !strings.Contains(string(txtData), "_domainkey") {
		t.Errorf("dns txt file missing or invalid: %v", err)
	}
}
