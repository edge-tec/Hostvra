package ssl

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type SSLManager struct{}

func NewSSLManager() *SSLManager {
	return &SSLManager{}
}

func (s *SSLManager) IssueCertbot(primaryDomain string, aliases []string, webroot string) (string, string, error) {
	domains := append([]string{primaryDomain}, aliases...)
	domainArgs := make([]string, 0)
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d != "" {
			domainArgs = append(domainArgs, "-d", d)
		}
	}

	certPath := fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", primaryDomain)
	keyPath := fmt.Sprintf("/etc/letsencrypt/live/%s/privkey.pem", primaryDomain)

	// Check if certbot is installed
	if _, err := exec.LookPath("certbot"); err == nil {
		args := append([]string{"certonly", "--webroot", "-w", webroot, "--non-interactive", "--agree-tos", "--register-unsafely-without-email"}, domainArgs...)
		cmd := exec.Command("certbot", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", "", fmt.Errorf("certbot issuance failed: %s (%w)", string(out), err)
		}
		return certPath, keyPath, nil
	}

	// Development / Self-Signed fallback for staging and test environments
	devCertDir := "/var/lib/hostvra/certs"
	if os.Geteuid() != 0 {
		devCertDir = "/tmp/hostvra-certs"
	}
	_ = os.MkdirAll(devCertDir, 0755)

	devCertPath := fmt.Sprintf("%s/%s.crt", devCertDir, primaryDomain)
	devKeyPath := fmt.Sprintf("%s/%s.key", devCertDir, primaryDomain)

	if _, err := os.Stat(devCertPath); os.IsNotExist(err) {
		// Generate self-signed cert for testing
		cmd := exec.Command("openssl", "req", "-x509", "-nodes", "-days", "90",
			"-newkey", "rsa:2048",
			"-keyout", devKeyPath,
			"-out", devCertPath,
			"-subj", fmt.Sprintf("/CN=%s/O=Hostvra Managed", primaryDomain),
		)
		_ = cmd.Run()
	}

	return devCertPath, devKeyPath, nil
}

func (s *SSLManager) ParseCertificateExpiry(certPath string) (*time.Time, string, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return nil, "", err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, "", fmt.Errorf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, "", err
	}

	issuer := cert.Issuer.CommonName
	if issuer == "" {
		issuer = cert.Issuer.Organization[0]
	}

	return &cert.NotAfter, issuer, nil
}
