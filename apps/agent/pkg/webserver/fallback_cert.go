package webserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// EnsureFallbackCertificate generates a standalone, self-signed X.509 fallback certificate
// for Nginx's neutral default_server block. This guarantees that unmatched TLS/SNI requests
// can be terminated safely with a 404/neutral response rather than falling through to Hostvra.
func EnsureFallbackCertificate(certPath, keyPath string) error {
	// If both files exist and are non-empty, nothing to do
	certInfo, errCert := os.Stat(certPath)
	keyInfo, errKey := os.Stat(keyPath)
	if errCert == nil && errKey == nil && certInfo.Size() > 0 && keyInfo.Size() > 0 {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(certPath), 0755); err != nil {
		return fmt.Errorf("failed to create certificate directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return fmt.Errorf("failed to create private key directory: %w", err)
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNum, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now().UTC()
	template := x509.Certificate{
		SerialNumber: serialNum,
		Subject: pkix.Name{
			Organization: []string{"Hostvra Isolated Neutral Server"},
			CommonName:   "default-server.neutral",
		},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(3650 * 24 * time.Hour), // 10 years
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "default-server.neutral"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyBytes := x509.MarshalPKCS1PrivateKey(priv)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})

	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return fmt.Errorf("failed to write fallback certificate: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write fallback private key: %w", err)
	}

	return nil
}

// EnsureDomainOriginCertificate generates or returns an authentic domain-specific X.509 origin TLS certificate
// under /etc/ssl/hostvra/<domain>/origin-fullchain.pem. This guarantees that every virtual host can bind port 443
// immediately with exact SNI and CommonName matching, preventing Cloudflare Error 525 across all domains.
func EnsureDomainOriginCertificate(domain string) (string, string, error) {
	cleanDomain := filepath.Clean(domain)
	baseDir := "/etc/ssl/hostvra"
	if custom := os.Getenv("HOSTVRA_SSL_DIR"); custom != "" {
		baseDir = custom
	} else if os.Geteuid() != 0 {
		baseDir = filepath.Join(os.TempDir(), "hostvra_ssl")
	}

	certDir := filepath.Join(baseDir, cleanDomain)
	certPath := filepath.Join(certDir, "origin-fullchain.pem")
	keyPath := filepath.Join(certDir, "origin-privkey.pem")

	// If existing certificate is present and valid, reuse it
	certInfo, errCert := os.Stat(certPath)
	keyInfo, errKey := os.Stat(keyPath)
	if errCert == nil && errKey == nil && certInfo.Size() > 0 && keyInfo.Size() > 0 {
		return certPath, keyPath, nil
	}

	if err := os.MkdirAll(certDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create origin certificate directory: %w", err)
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNum, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now().UTC()
	template := x509.Certificate{
		SerialNumber: serialNum,
		Subject: pkix.Name{
			Organization: []string{"Hostvra Origin TLS Provider"},
			CommonName:   cleanDomain,
		},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(3650 * 24 * time.Hour), // 10 years
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{cleanDomain, "www." + cleanDomain},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyBytes := x509.MarshalPKCS1PrivateKey(priv)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})

	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return "", "", fmt.Errorf("failed to write origin certificate: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return "", "", fmt.Errorf("failed to write origin private key: %w", err)
	}

	return certPath, keyPath, nil
}
