package ssl

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPrepareDNS01Challenge_Manual(t *testing.T) {
	tempDir := t.TempDir()
	mgr := NewSSLManager(WithCertDir(tempDir))

	req := IssueRequest{
		PrimaryDomain: "example.com",
		Wildcard:      true,
		Provider:      "manual",
	}

	info, err := mgr.PrepareDNS01Challenge(context.Background(), req)
	if err != nil {
		t.Fatalf("PrepareDNS01Challenge failed: %v", err)
	}

	if info.Domain != "example.com" {
		t.Errorf("expected domain example.com, got %s", info.Domain)
	}
	if info.TXTHost != "_acme-challenge.example.com" {
		t.Errorf("expected txt host _acme-challenge.example.com, got %s", info.TXTHost)
	}
	if info.TXTValue == "" {
		t.Errorf("expected non-empty txt value")
	}

	// Verify it was stored in manual provider
	stored, ok := mgr.ManualProvider.GetChallenge("example.com")
	if !ok || stored.TXTValue != info.TXTValue {
		t.Errorf("failed to retrieve stored manual challenge")
	}
}

func TestVerifyDNSPropagation(t *testing.T) {
	mgr := NewSSLManager(WithDNSResolver(func(host string) ([]string, error) {
		if host == "_acme-challenge.example.com" {
			return []string{"match-secret-value-123"}, nil
		}
		return nil, nil
	}))

	ok, err := mgr.VerifyDNSPropagation(context.Background(), "_acme-challenge.example.com", "match-secret-value-123")
	if err != nil || !ok {
		t.Errorf("expected verification success, got ok=%v, err=%v", ok, err)
	}

	ok, err = mgr.VerifyDNSPropagation(context.Background(), "_acme-challenge.example.com", "wrong-value")
	if err != nil || ok {
		t.Errorf("expected verification failure for wrong value, got ok=%v, err=%v", ok, err)
	}
}

func TestIssueWildcardDNS01(t *testing.T) {
	tempDir := t.TempDir()
	reloaded := false

	mgr := NewSSLManager(
		WithCertDir(tempDir),
		WithReloadCmd(func(ctx context.Context) error {
			reloaded = true
			return nil
		}),
	)

	req := IssueRequest{
		PrimaryDomain: "mycorp.com",
		Wildcard:      true,
		Provider:      "local",
	}

	cert, err := mgr.IssueWildcardDNS01(context.Background(), req)
	if err != nil {
		t.Fatalf("IssueWildcardDNS01 failed: %v", err)
	}

	if !reloaded {
		t.Errorf("expected webserver reload on issuance")
	}
	if !cert.IsWildcard {
		t.Errorf("expected is_wildcard to be true")
	}
	if cert.Domain != "mycorp.com" {
		t.Errorf("expected domain mycorp.com, got %s", cert.Domain)
	}

	// Verify SANs contain both mycorp.com and *.mycorp.com
	hasWildcard := false
	for _, san := range cert.SANs {
		if san == "*.mycorp.com" {
			hasWildcard = true
			break
		}
	}
	if !hasWildcard {
		t.Errorf("SANs missing *.mycorp.com: %v", cert.SANs)
	}

	// Verify files created on disk
	fullchain := filepath.Join(tempDir, "mycorp.com", "fullchain.pem")
	privkey := filepath.Join(tempDir, "mycorp.com", "privkey.pem")

	if _, err := os.Stat(fullchain); os.IsNotExist(err) {
		t.Errorf("fullchain.pem not created")
	}
	if _, err := os.Stat(privkey); os.IsNotExist(err) {
		t.Errorf("privkey.pem not created")
	}
}

func TestImportCustomCert_ValidAndInvalid(t *testing.T) {
	tempDir := t.TempDir()
	mgr := NewSSLManager(WithCertDir(tempDir))

	// Generate key pair 1
	priv1, _ := rsa.GenerateKey(rand.Reader, 2048)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1337),
		Subject:      pkix.Name{CommonName: "custom.domain.com"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(60 * 24 * time.Hour),
		DNSNames:     []string{"custom.domain.com", "*.custom.domain.com"},
	}
	der1, _ := x509.CreateCertificate(rand.Reader, &template, &template, &priv1.PublicKey, priv1)
	certPEM1 := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der1})
	keyPEM1 := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv1)})

	// Generate key pair 2 (mismatched)
	priv2, _ := rsa.GenerateKey(rand.Reader, 2048)
	keyPEM2 := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv2)})

	// 1. Test mismatched keypair rejection
	_, err := mgr.ImportCustomCert(context.Background(), CustomCertRequest{
		Domain:  "custom.domain.com",
		CertPEM: string(certPEM1),
		KeyPEM:  string(keyPEM2),
	})
	if err == nil {
		t.Fatalf("expected error on mismatched keypair, got nil")
	}

	// 2. Test matching keypair acceptance
	info, err := mgr.ImportCustomCert(context.Background(), CustomCertRequest{
		Domain:  "custom.domain.com",
		CertPEM: string(certPEM1),
		KeyPEM:  string(keyPEM1),
	})
	if err != nil {
		t.Fatalf("ImportCustomCert failed for valid keypair: %v", err)
	}

	if info.Domain != "custom.domain.com" {
		t.Errorf("expected domain custom.domain.com, got %s", info.Domain)
	}
	if !info.IsWildcard {
		t.Errorf("expected is_wildcard to be true due to *.custom.domain.com SAN")
	}
}

func TestListInstalledCertificatesAndRevoke(t *testing.T) {
	tempDir := t.TempDir()
	mgr := NewSSLManager(WithCertDir(tempDir))

	// Issue 2 certificates
	_, err := mgr.IssueHTTP01(context.Background(), IssueRequest{PrimaryDomain: "site1.com"})
	if err != nil {
		t.Fatalf("issue site1 failed: %v", err)
	}
	_, err = mgr.IssueWildcardDNS01(context.Background(), IssueRequest{PrimaryDomain: "site2.com", Wildcard: true})
	if err != nil {
		t.Fatalf("issue site2 failed: %v", err)
	}

	list, err := mgr.ListInstalledCertificates(context.Background())
	if err != nil {
		t.Fatalf("ListInstalledCertificates failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 installed certificates, got %d", len(list))
	}

	// Revoke site1
	if err := mgr.RevokeOrDeleteCertificate(context.Background(), "site1.com"); err != nil {
		t.Fatalf("RevokeOrDeleteCertificate failed: %v", err)
	}

	listAfter, err := mgr.ListInstalledCertificates(context.Background())
	if err != nil {
		t.Fatalf("ListInstalledCertificates after revoke failed: %v", err)
	}
	if len(listAfter) != 1 {
		t.Errorf("expected 1 installed certificate after revoke, got %d", len(listAfter))
	}
	if listAfter[0].Domain != "site2.com" {
		t.Errorf("expected remaining domain site2.com, got %s", listAfter[0].Domain)
	}
}

func TestCloudflareProvider_MockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.Contains(auth, "test-token") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if strings.Contains(r.URL.Path, "/zones") && r.Method == http.MethodGet {
			resp := cfZoneResponse{
				Success: true,
				Result: []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{
					{ID: "zone-12345", Name: "example.com"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(r.URL.Path, "/dns_records") && r.Method == http.MethodPost {
			resp := cfRecordResponse{
				Success: true,
				Result: struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{
					ID:   "rec-99999",
					Name: "_acme-challenge.example.com",
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(r.URL.Path, "/dns_records/rec-99999") && r.Method == http.MethodDelete {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cf := NewCloudflareProvider("test-token", "")
	cf.BaseURL = server.URL
	cf.HTTPClient = server.Client()

	err := cf.CreateTXTRecord(context.Background(), "example.com", "_acme-challenge.example.com", "chal-value-abc")
	if err != nil {
		t.Fatalf("Cloudflare CreateTXTRecord failed: %v", err)
	}

	if cf.ZoneID != "zone-12345" {
		t.Errorf("expected zone id zone-12345, got %s", cf.ZoneID)
	}

	err = cf.DeleteTXTRecord(context.Background(), "example.com", "_acme-challenge.example.com", "chal-value-abc")
	if err != nil {
		t.Fatalf("Cloudflare DeleteTXTRecord failed: %v", err)
	}
}

func TestDigitalOceanProvider_MockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer do-token-123" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if strings.Contains(r.URL.Path, "/domains/example.com/records") && r.Method == http.MethodPost {
			resp := doRecordResponse{
				DomainRecord: struct {
					ID   int    `json:"id"`
					Type string `json:"type"`
					Name string `json:"name"`
					Data string `json:"data"`
				}{
					ID:   8888,
					Type: "TXT",
					Name: "_acme-challenge",
					Data: "do-chal-val",
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(r.URL.Path, "/domains/example.com/records/8888") && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	do := NewDigitalOceanProvider("do-token-123")
	do.BaseURL = server.URL
	do.HTTPClient = server.Client()

	err := do.CreateTXTRecord(context.Background(), "example.com", "_acme-challenge.example.com", "do-chal-val")
	if err != nil {
		t.Fatalf("DigitalOcean CreateTXTRecord failed: %v", err)
	}

	err = do.DeleteTXTRecord(context.Background(), "example.com", "_acme-challenge.example.com", "do-chal-val")
	if err != nil {
		t.Fatalf("DigitalOcean DeleteTXTRecord failed: %v", err)
	}
}
