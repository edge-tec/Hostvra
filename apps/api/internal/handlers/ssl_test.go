package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/ssl"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

func TestSSLHandler_API(t *testing.T) {
	tempDir := t.TempDir()
	certDir := filepath.Join(tempDir, "certs")
	t.Setenv("HOSTVRA_SSL_CERT_DIR", certDir)

	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)

	h := NewSSLHandler(cfg, s, auditLogger)

	// Create test website
	orgID := uuid.New()
	siteID := uuid.New()
	_ = s.CreateWebsite(context.Background(), &store.Website{
		ID:             siteID,
		OrganizationID: orgID,
		PrimaryDomain:  "testssl.com",
		DocumentRoot:   "/var/www/testssl.com",
		SystemUser:     "u_testssl",
		Status:         "running",
	})

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
				UserID:         uuid.New(),
				OrganizationID: orgID,
				Role:           "owner",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})

	r.Route("/api/v1/ssl", func(r chi.Router) {
		r.Get("/certificates", h.ListCertificates)
		r.Post("/issue", h.Issue)
		r.Post("/challenge", h.PrepareChallenge)
		r.Post("/verify-challenge", h.VerifyChallenge)
		r.Post("/custom", h.ImportCustom)
		r.Post("/renew/{id}", h.Renew)
		r.Post("/auto-renew", h.AutoRenew)
		r.Delete("/{id}", h.Delete)
	})

	// 1. Issue Wildcard Certificate
	issueReq := ssl.IssueRequest{
		PrimaryDomain: "testssl.com",
		Wildcard:      true,
		Provider:      "local",
		WebsiteID:     siteID.String(),
	}
	body, _ := json.Marshal(issueReq)
	req := httptest.NewRequest("POST", "/api/v1/ssl/issue", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("issue wildcard failed: status %d, body %s", rec.Code, rec.Body.String())
	}

	var issueResp response.Envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &issueResp)
	if !issueResp.Success {
		t.Fatalf("issue wildcard returned failure envelope: %s", rec.Body.String())
	}

	dataBytes, _ := json.Marshal(issueResp.Data)
	var issuedCert store.SSLCertificate
	_ = json.Unmarshal(dataBytes, &issuedCert)

	if !issuedCert.IsWildcard {
		t.Errorf("expected is_wildcard true, got %v", issuedCert.IsWildcard)
	}

	// Verify website SSL status updated
	site, _ := s.GetWebsiteByID(context.Background(), siteID)
	if !site.SSLEnabled {
		t.Errorf("expected website SSLEnabled to be true")
	}

	// 2. Prepare Manual Challenge
	chalReq := ssl.IssueRequest{
		PrimaryDomain: "manualwildcard.org",
		Wildcard:      true,
		Provider:      "manual",
	}
	body, _ = json.Marshal(chalReq)
	req = httptest.NewRequest("POST", "/api/v1/ssl/challenge", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("prepare challenge failed: status %d, body %s", rec.Code, rec.Body.String())
	}

	var chalEnv response.Envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &chalEnv)
	chalDataBytes, _ := json.Marshal(chalEnv.Data)
	var chalInfo ssl.ChallengeInfo
	_ = json.Unmarshal(chalDataBytes, &chalInfo)

	if chalInfo.TXTHost != "_acme-challenge.manualwildcard.org" {
		t.Errorf("unexpected txt host: %s", chalInfo.TXTHost)
	}

	// 3. Import Custom Certificate
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(9999),
		Subject:      pkix.Name{CommonName: "customsecure.net"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(60 * 24 * time.Hour),
		DNSNames:     []string{"customsecure.net", "*.customsecure.net"},
	}
	der, _ := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	customReq := ssl.CustomCertRequest{
		Domain:  "customsecure.net",
		CertPEM: string(certPEM),
		KeyPEM:  string(keyPEM),
	}
	body, _ = json.Marshal(customReq)
	req = httptest.NewRequest("POST", "/api/v1/ssl/custom", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("custom import failed: status %d, body %s", rec.Code, rec.Body.String())
	}

	// 4. List All Certificates
	req = httptest.NewRequest("GET", "/api/v1/ssl/certificates", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list certificates failed: status %d", rec.Code)
	}

	var listEnv response.Envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &listEnv)
	listBytes, _ := json.Marshal(listEnv.Data)
	var certList []*store.SSLCertificate
	_ = json.Unmarshal(listBytes, &certList)

	if len(certList) < 2 {
		t.Errorf("expected at least 2 certificates in list, got %d", len(certList))
	}

	// 5. Force Renew
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/ssl/renew/%s", issuedCert.ID.String()), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("renew failed: status %d, body %s", rec.Code, rec.Body.String())
	}

	// 6. Auto-Renew Scan
	req = httptest.NewRequest("POST", "/api/v1/ssl/auto-renew", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("auto-renew scan failed: status %d", rec.Code)
	}

	// 7. Delete Certificate
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/ssl/%s", issuedCert.ID.String()), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("delete certificate failed: status %d", rec.Code)
	}
}
