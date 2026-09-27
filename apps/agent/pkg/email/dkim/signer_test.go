package dkim

import (
	"bytes"
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

func TestSignEmail(t *testing.T) {
	domain := "hostvra.com"
	selector := "default"

	key, err := GenerateDKIMKey(domain, selector, 2048)
	if err != nil {
		t.Fatalf("GenerateDKIMKey failed: %v", err)
	}

	rawEmail := []byte("From: info@hostvra.com\r\n" +
		"To: recipient@example.org\r\n" +
		"Subject: Production Email Delivery Test\r\n" +
		"Date: Sun, 27 Sep 2026 21:00:00 +0000\r\n" +
		"Message-ID: <12345.test@hostvra.com>\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		"This is a production test message body from Hostvra mail server.\r\n")

	signed, err := SignEmail(rawEmail, SignOptions{
		Domain:        domain,
		Selector:      selector,
		PrivateKeyPEM: key.PrivateKeyPEM,
	})
	if err != nil {
		t.Fatalf("SignEmail failed: %v", err)
	}

	signedStr := string(signed)
	if !strings.HasPrefix(signedStr, "DKIM-Signature: v=1; a=rsa-sha256; c=relaxed/relaxed;") {
		t.Errorf("DKIM-Signature header prefix missing or malformed: %s", signedStr[:min(100, len(signedStr))])
	}
	if !strings.Contains(signedStr, "d=hostvra.com;") {
		t.Errorf("DKIM domain parameter missing")
	}
	if !strings.Contains(signedStr, "s=default;") {
		t.Errorf("DKIM selector parameter missing")
	}
	if !strings.Contains(signedStr, "b=") {
		t.Errorf("DKIM signature value b= missing")
	}
	if !strings.Contains(signedStr, "bh=") {
		t.Errorf("DKIM body hash bh= missing")
	}

	// Verify original message is preserved after signature
	if !bytes.Contains(signed, rawEmail) {
		t.Errorf("signed message does not contain original email payload")
	}
}
