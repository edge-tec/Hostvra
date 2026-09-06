package dkim

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

type DKIMKeyPair struct {
	Domain        string
	Selector      string
	KeySize       int
	PrivateKeyPEM string
	PublicKeyDNS  string
}

// GenerateDKIMKey creates a 2048-bit RSA key pair and formats it for DNS TXT record
func GenerateDKIMKey(domain, selector string, keySize int) (*DKIMKeyPair, error) {
	if selector == "" {
		selector = "default"
	}
	if keySize == 0 {
		keySize = 2048
	}

	privKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Encode private key to PEM
	privASN1 := x509.MarshalPKCS1PrivateKey(privKey)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privASN1,
	}
	privPEM := string(pem.EncodeToMemory(privBlock))

	// Encode public key for DNS
	pubASN1, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubBase64 := base64.StdEncoding.EncodeToString(pubASN1)
	pubDNS := fmt.Sprintf("v=DKIM1; k=rsa; p=%s", pubBase64)

	return &DKIMKeyPair{
		Domain:        domain,
		Selector:      selector,
		KeySize:       keySize,
		PrivateKeyPEM: privPEM,
		PublicKeyDNS:  pubDNS,
	}, nil
}

// SaveDKIMKey writes the private and public keys to standard hostvra directory
func SaveDKIMKey(baseDir string, key *DKIMKeyPair) error {
	if baseDir == "" {
		baseDir = "/var/lib/hostvra/dkim"
	}
	domainDir := filepath.Join(baseDir, key.Domain)
	if err := os.MkdirAll(domainDir, 0700); err != nil {
		return fmt.Errorf("failed to create dkim directory: %w", err)
	}

	privPath := filepath.Join(domainDir, fmt.Sprintf("%s.private", key.Selector))
	if err := os.WriteFile(privPath, []byte(key.PrivateKeyPEM), 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	dnsPath := filepath.Join(domainDir, fmt.Sprintf("%s.txt", key.Selector))
	txtContent := fmt.Sprintf("%s._domainkey.%s IN TXT \"%s\"\n", key.Selector, key.Domain, key.PublicKeyDNS)
	if err := os.WriteFile(dnsPath, []byte(txtContent), 0644); err != nil {
		return fmt.Errorf("failed to write dns txt file: %w", err)
	}

	return nil
}
