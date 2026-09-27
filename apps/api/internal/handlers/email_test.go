package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	_ = st.UpdateSystemSettings(ctx, &store.SystemSettings{
		ServerIP:           "185.193.17.42",
		MailServerIPMode:   "manual",
		MailServerPublicIP: "185.193.17.42",
	})

	testDomain := fmt.Sprintf("dns-test-%s.com", uuid.New().String()[:8])

	// 1. Create Domain without ServerID and with no servers pre-enrolled (UI behavior)
	reqBody, _ := json.Marshal(CreateEmailDomainRequest{
		Domain: testDomain,
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

	if resp.Data.Domain != testDomain {
		t.Errorf("expected domain %s, got %s", testDomain, resp.Data.Domain)
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
	var foundARecord, foundSPFRecord bool
	for _, rec := range dnsRecords {
		recordTypes[rec.RecordType] = true
		if rec.RecordType == "A" && rec.Host == "mail" {
			foundARecord = true
			if rec.Expected != "185.193.17.42" {
				t.Errorf("expected mail A record to be 185.193.17.42, got %s", rec.Expected)
			}
			if rec.Message != "Primary mail server address" {
				t.Errorf("expected mail A record message to be 'Primary mail server address', got %s", rec.Message)
			}
		}
		if rec.RecordType == "TXT" && strings.HasPrefix(rec.Expected, "v=spf1") {
			foundSPFRecord = true
			expectedSPF := "v=spf1 mx ip4:185.193.17.42 ~all"
			if rec.Expected != expectedSPF {
				t.Errorf("expected SPF %q, got %q", expectedSPF, rec.Expected)
			}
		}
		if rec.RecordType == "TXT" && strings.HasPrefix(rec.Expected, "v=DKIM1") {
			if !strings.Contains(rec.Expected, "p=") {
				t.Errorf("DKIM record missing public key: %s", rec.Expected)
			}
		}
		if rec.RecordType == "TXT" && strings.HasPrefix(rec.Expected, "v=DMARC1") {
			if !strings.Contains(rec.Expected, fmt.Sprintf("rua=mailto:dmarc@%s", testDomain)) {
				t.Errorf("DMARC record missing rua report: %s", rec.Expected)
			}
		}
	}

	if !foundARecord {
		t.Errorf("expected mail A record to be present in DNS records")
	}
	if !foundSPFRecord {
		t.Errorf("expected SPF record to be present in DNS records")
	}
	if !recordTypes["MX"] || !recordTypes["TXT"] || !recordTypes["A"] || !recordTypes["CNAME"] {
		t.Errorf("expected MX, TXT, A, and CNAME records, got types: %+v", recordTypes)
	}
}

func TestEmailHandler_GetDomainDNS_RejectsPrivateAndLoopbackIPs(t *testing.T) {
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

	// Create domain first with valid public IP setting
	_ = st.UpdateSystemSettings(ctx, &store.SystemSettings{
		ServerIP:           "185.193.17.42",
		MailServerIPMode:   "manual",
		MailServerPublicIP: "185.193.17.42",
	})

	testDomain := fmt.Sprintf("reject-test-%s.com", uuid.New().String()[:8])
	reqBody, _ := json.Marshal(CreateEmailDomainRequest{Domain: testDomain})
	req := httptest.NewRequest("POST", "/api/v1/email/domains", bytes.NewReader(reqBody)).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.CreateDomain(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create domain: %d: %s", w.Code, w.Body.String())
	}

	var createResp struct {
		Data struct {
			ID uuid.UUID `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	domainID := createResp.Data.ID

	// Test invalid IPs: loopback, RFC1918 private, link-local, 0.0.0.0
	invalidIPs := []string{
		"127.0.0.1",
		"127.0.1.1",
		"0.0.0.0",
		"10.0.0.1",
		"172.16.0.1",
		"192.168.1.100",
		"169.254.1.1",
	}

	for _, invalidIP := range invalidIPs {
		_ = st.UpdateSystemSettings(ctx, &store.SystemSettings{
			ServerIP:           invalidIP,
			MailServerIPMode:   "manual",
			MailServerPublicIP: invalidIP,
		})

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", domainID.String())
		dnsReq := httptest.NewRequest("GET", "/api/v1/email/domains/"+domainID.String()+"/dns", nil)
		dnsReq = dnsReq.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
		dnsRec := httptest.NewRecorder()

		handler.GetDomainDNS(dnsRec, dnsReq)
		if dnsRec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422 Unprocessable Entity for invalid IP %q, got %d: %s", invalidIP, dnsRec.Code, dnsRec.Body.String())
		}
		if !strings.Contains(dnsRec.Body.String(), "NO_PUBLIC_IP") {
			t.Errorf("expected NO_PUBLIC_IP error code in response for IP %q, got: %s", invalidIP, dnsRec.Body.String())
		}
	}

	// Now update to a valid public IP and verify it succeeds
	validIP := "185.193.17.50"
	_ = st.UpdateSystemSettings(ctx, &store.SystemSettings{
		ServerIP:           validIP,
		MailServerIPMode:   "manual",
		MailServerPublicIP: validIP,
	})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", domainID.String())
	dnsReq := httptest.NewRequest("GET", "/api/v1/email/domains/"+domainID.String()+"/dns", nil)
	dnsReq = dnsReq.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
	dnsRec := httptest.NewRecorder()

	handler.GetDomainDNS(dnsRec, dnsReq)
	if dnsRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid public IP %s, got %d: %s", validIP, dnsRec.Code, dnsRec.Body.String())
	}

	var okResp struct {
		Data []store.DNSVerificationResult `json:"data"`
	}
	_ = json.Unmarshal(dnsRec.Body.Bytes(), &okResp)
	var foundMailA bool
	for _, r := range okResp.Data {
		if r.RecordType == "A" && r.Host == "mail" {
			foundMailA = true
			if r.Expected != validIP {
				t.Errorf("expected A record to point to %s, got %s", validIP, r.Expected)
			}
		}
	}
	if !foundMailA {
		t.Errorf("mail A record not found in DNS response")
	}
}

func TestEmailHandler_VerifyDomainDNS(t *testing.T) {
	handler, memStore, _, orgID, _, serverID := setupEmailTestEnv(t)
	ctx := context.Background()

	domainID := uuid.New()
	testDomain := "verifytest.org"

	_ = memStore.CreateEmailDomain(ctx, &store.EmailDomain{
		ID:             domainID,
		OrganizationID: orgID,
		ServerID:       serverID,
		Domain:         testDomain,
		MailHostname:   "mail." + testDomain,
		DKIMSelector:   "default",
		IsDNSVerified:  false,
	})

	_ = memStore.UpdateSystemSettings(ctx, &store.SystemSettings{
		ServerIP:           "185.193.17.42",
		MailServerIPMode:   "manual",
		MailServerPublicIP: "185.193.17.42",
	})

	// 1. Live Query (domain not propagated on public DNS)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", domainID.String())
	req := httptest.NewRequest("GET", "/api/v1/email/domains/"+domainID.String()+"/verify-dns", nil)
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.VerifyDomainDNS(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res struct {
		Data struct {
			Domain        string                        `json:"domain"`
			AllVerified   bool                          `json:"all_verified"`
			IsDNSVerified bool                          `json:"is_dns_verified"`
			Records       []store.DNSVerificationResult `json:"records"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if res.Data.Domain != testDomain {
		t.Errorf("expected domain %s, got %s", testDomain, res.Data.Domain)
	}
	if len(res.Data.Records) == 0 {
		t.Errorf("expected records to be returned")
	}

	// 2. Force / Simulate Verify (Simulates all records passing and verify tick mark set)
	reqForce := httptest.NewRequest("GET", "/api/v1/email/domains/"+domainID.String()+"/verify-dns?force=true", nil)
	reqForce = reqForce.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
	wForce := httptest.NewRecorder()

	handler.VerifyDomainDNS(wForce, reqForce)
	if wForce.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", wForce.Code, wForce.Body.String())
	}

	var resForce struct {
		Data struct {
			Domain        string                        `json:"domain"`
			AllVerified   bool                          `json:"all_verified"`
			IsDNSVerified bool                          `json:"is_dns_verified"`
			MXValid       bool                          `json:"mx_valid"`
			SPFValid      bool                          `json:"spf_valid"`
			DKIMValid     bool                          `json:"dkim_valid"`
			Records       []store.DNSVerificationResult `json:"records"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wForce.Body.Bytes(), &resForce); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if !resForce.Data.AllVerified {
		t.Errorf("expected AllVerified to be true with force=true")
	}
	if !resForce.Data.IsDNSVerified {
		t.Errorf("expected IsDNSVerified to be true with force=true")
	}

	// Verify that the domain in the database/store now has IsDNSVerified = true
	savedDomain, err := memStore.GetEmailDomainByID(ctx, domainID)
	if err != nil || !savedDomain.IsDNSVerified {
		t.Errorf("expected domain in store to be marked IsDNSVerified=true, got err=%v, isDNSVerified=%v", err, savedDomain.IsDNSVerified)
	}
}


