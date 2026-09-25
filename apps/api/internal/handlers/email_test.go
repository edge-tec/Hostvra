package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/dns"
	"hostvra/api/internal/store"
)

func setupEmailTestEnv(t *testing.T) (*EmailHandler, store.Store, *dns.Service, uuid.UUID, uuid.UUID, uuid.UUID) {
	st := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auditLogger := audit.NewLogger(st, logger)
	dnsSvc := dns.NewService()
	cfg := &config.Config{JWTSecret: "test-secret"}

	handler := NewEmailHandler(cfg, st, dnsSvc, auditLogger)

	orgID := uuid.New()
	userID := uuid.New()
	serverID := uuid.New()

	_ = st.CreateOrganization(context.Background(), &store.Organization{
		ID:   orgID,
		Name: "Test Org",
		Slug: "test-org",
	})

	_ = st.CreateServer(context.Background(), &store.Server{
		ID:             serverID,
		OrganizationID: orgID,
		Name:           "mail-vps-1",
		IPAddress:      "192.0.2.25",
		Status:         "online",
	})

	return handler, st, dnsSvc, orgID, userID, serverID
}

func TestEmailHandler_FullLifecycle(t *testing.T) {
	handler, _, _, orgID, userID, serverID := setupEmailTestEnv(t)

	// Context with claims
	claims := &auth.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           "owner",
	}
	ctx := context.WithValue(context.Background(), auth.UserContextKey, claims)

	// 1. Create Domain
	reqBody, _ := json.Marshal(CreateEmailDomainRequest{
		ServerID: serverID.String(),
		Domain:   "hostvramail.com",
	})
	req := httptest.NewRequest("POST", "/api/v1/email/domains", bytes.NewReader(reqBody)).WithContext(ctx)
	w := httptest.NewRecorder()

	handler.CreateDomain(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateDomain failed with code %d: %s", w.Code, w.Body.String())
	}

	var createdDomainResp struct {
		Data store.EmailDomain `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &createdDomainResp)
	domainID := createdDomainResp.Data.ID

	// 2. List Domains
	req = httptest.NewRequest("GET", "/api/v1/email/domains", nil).WithContext(ctx)
	w = httptest.NewRecorder()
	handler.ListDomains(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListDomains failed with code %d: %s", w.Code, w.Body.String())
	}

	// 3. Create Mailbox
	mbBody, _ := json.Marshal(CreateMailboxRequest{
		DomainID:  domainID.String(),
		LocalPart: "support",
		Password:  "SecurePass123!@#",
		Name:      "Customer Support",
	})
	req = httptest.NewRequest("POST", "/api/v1/email/mailboxes", bytes.NewReader(mbBody)).WithContext(ctx)
	w = httptest.NewRecorder()
	handler.CreateMailbox(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateMailbox failed with code %d: %s", w.Code, w.Body.String())
	}

	// 4. List Mailboxes
	req = httptest.NewRequest("GET", "/api/v1/email/mailboxes?domain_id="+domainID.String(), nil).WithContext(ctx)
	w = httptest.NewRecorder()
	handler.ListMailboxes(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListMailboxes failed with code %d: %s", w.Code, w.Body.String())
	}

	// 5. Create Alias
	aliasBody, _ := json.Marshal(CreateAliasRequest{
		DomainID:            domainID.String(),
		SourceAddress:       "help@hostvramail.com",
		DestinationAddress: "support@hostvramail.com",
	})
	req = httptest.NewRequest("POST", "/api/v1/email/aliases", bytes.NewReader(aliasBody)).WithContext(ctx)
	w = httptest.NewRecorder()
	handler.CreateAlias(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateAlias failed with code %d: %s", w.Code, w.Body.String())
	}

	// 6. Check Health
	req = httptest.NewRequest("GET", "/api/v1/email/health?domain=hostvramail.com", nil).WithContext(ctx)
	w = httptest.NewRecorder()
	handler.CheckHealth(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("CheckHealth failed with code %d: %s", w.Code, w.Body.String())
	}

	// 7. Delete Domain with chi URLParam routing
	rCtx := chi.NewRouteContext()
	rCtx.URLParams.Add("id", domainID.String())
	delReq := httptest.NewRequest("DELETE", "/api/v1/email/domains/"+domainID.String(), nil).WithContext(context.WithValue(ctx, chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.DeleteDomain(w, delReq)
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteDomain failed with code %d: %s", w.Code, w.Body.String())
	}
}

func TestEmailHandler_DovecotSync(t *testing.T) {
	tempDir := t.TempDir()
	usersFile := tempDir + "/users"
	t.Setenv("DOVECOT_USERS_FILE", usersFile)

	handler, _, _, orgID, userID, serverID := setupEmailTestEnv(t)
	claims := &auth.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           "owner",
	}
	ctx := context.WithValue(context.Background(), auth.UserContextKey, claims)

	// Create Domain
	reqBody, _ := json.Marshal(CreateEmailDomainRequest{
		ServerID: serverID.String(),
		Domain:   "sync-test.com",
	})
	req := httptest.NewRequest("POST", "/api/v1/email/domains", bytes.NewReader(reqBody)).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.CreateDomain(w, req)

	var createdDomainResp struct {
		Data store.EmailDomain `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &createdDomainResp)
	domainID := createdDomainResp.Data.ID

	// Create Mailbox
	mbBody, _ := json.Marshal(CreateMailboxRequest{
		DomainID:   domainID.String(),
		LocalPart:  "billing",
		Password:   "MyPass123!@#",
		Name:       "Billing Team",
		QuotaBytes: 1073741824, // 1GB
	})
	req = httptest.NewRequest("POST", "/api/v1/email/mailboxes", bytes.NewReader(mbBody)).WithContext(ctx)
	w = httptest.NewRecorder()
	handler.CreateMailbox(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateMailbox failed: %s", w.Body.String())
	}

	// Verify users file was written and contains formatted user entry
	data, err := os.ReadFile(usersFile)
	if err != nil {
		t.Fatalf("expected dovecot users file to exist: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "billing@sync-test.com:") {
		t.Errorf("expected mailbox billing@sync-test.com in users file, got: %s", content)
	}
	if !strings.Contains(content, ":storage=1024M") {
		t.Errorf("expected quota rule :storage=1024M in users file, got: %s", content)
	}
}

func TestEmailHandler_CreateDomainWithoutServerID_AndDNSResolution(t *testing.T) {
	st := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auditLogger := audit.NewLogger(st, logger)
	dnsSvc := dns.NewService()
	cfg := &config.Config{JWTSecret: "test-secret"}
	handler := NewEmailHandler(cfg, st, dnsSvc, auditLogger)

	orgID := uuid.New()
	userID := uuid.New()
	claims := &auth.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           "owner",
	}
	ctx := context.WithValue(context.Background(), auth.UserContextKey, claims)

	// 1. Create Domain without ServerID and with no servers pre-enrolled (UI behavior)
	reqBody, _ := json.Marshal(CreateEmailDomainRequest{
		Domain: "enterprise-mail.com",
	})
	req := httptest.NewRequest("POST", "/api/v1/email/domains", bytes.NewReader(reqBody)).WithContext(ctx)
	w := httptest.NewRecorder()

	handler.CreateDomain(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created when adding domain without server_id, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			ID          uuid.UUID                     `json:"id"`
			Domain      string                        `json:"domain"`
			PublicDKIM  string                        `json:"public_dkim"`
			RequiredDNS []store.DNSVerificationResult `json:"required_dns"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Data.Domain != "enterprise-mail.com" {
		t.Errorf("expected domain enterprise-mail.com, got %s", resp.Data.Domain)
	}
	if resp.Data.PublicDKIM == "" || !strings.Contains(resp.Data.PublicDKIM, "v=DKIM1; k=rsa; p=") {
		t.Errorf("expected valid 2048-bit RSA DKIM public key, got: %s", resp.Data.PublicDKIM)
	}

	// 2. Query GetDomainDNS
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", resp.Data.ID.String())
	req = httptest.NewRequest("GET", "/api/v1/email/domains/"+resp.Data.ID.String()+"/dns", nil)
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()

	handler.GetDomainDNS(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GetDomainDNS failed: %d: %s", w.Code, w.Body.String())
	}

	var dnsRecords []store.DNSVerificationResult
	var dnsResp struct {
		Data []store.DNSVerificationResult `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &dnsResp)
	dnsRecords = dnsResp.Data

	recordTypes := make(map[string]bool)
	for _, rec := range dnsRecords {
		recordTypes[rec.RecordType] = true
		if rec.RecordType == "TXT" && strings.HasPrefix(rec.Expected, "v=spf1") {
			if !strings.Contains(rec.Expected, "ip4:") {
				t.Errorf("SPF record missing server IP: %s", rec.Expected)
			}
		}
		if rec.RecordType == "TXT" && strings.HasPrefix(rec.Expected, "v=DKIM1") {
			if !strings.Contains(rec.Expected, "p=") {
				t.Errorf("DKIM record missing public key: %s", rec.Expected)
			}
		}
		if rec.RecordType == "TXT" && strings.HasPrefix(rec.Expected, "v=DMARC1") {
			if !strings.Contains(rec.Expected, "rua=mailto:dmarc@enterprise-mail.com") {
				t.Errorf("DMARC record missing rua report: %s", rec.Expected)
			}
		}
	}

	if !recordTypes["MX"] || !recordTypes["TXT"] || !recordTypes["A"] || !recordTypes["CNAME"] {
		t.Errorf("expected MX, TXT, A, and CNAME records, got types: %+v", recordTypes)
	}
}

