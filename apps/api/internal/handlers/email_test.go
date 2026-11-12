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
	_ = st.CreateOrganization(context.Background(), &store.Organization{
		ID:   orgID,
		Name: "Test Org",
		Slug: "test-org",
	})
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
	_ = st.CreateOrganization(context.Background(), &store.Organization{
		ID:   orgID,
		Name: "Test Org",
		Slug: "test-org",
	})
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
	claims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: orgID,
		Role:           "owner",
	}
	ctx := context.WithValue(context.Background(), auth.UserContextKey, claims)

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

func TestEmailHandler_StrictMultiTenantIsolation(t *testing.T) {
	st := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auditLogger := audit.NewLogger(st, logger)
	dnsSvc := dns.NewService()
	cfg := &config.Config{JWTSecret: "test-secret"}
	handler := NewEmailHandler(cfg, st, dnsSvc, auditLogger)

	// Org A
	orgA := uuid.New()
	userA := uuid.New()
	serverA := uuid.New()
	_ = st.CreateOrganization(context.Background(), &store.Organization{ID: orgA, Name: "Org A", Slug: "org-a"})
	_ = st.CreateServer(context.Background(), &store.Server{ID: serverA, OrganizationID: orgA, Name: "srv-a", IPAddress: "198.51.100.1"})

	// Org B
	orgB := uuid.New()
	userB := uuid.New()
	serverB := uuid.New()
	_ = st.CreateOrganization(context.Background(), &store.Organization{ID: orgB, Name: "Org B", Slug: "org-b"})
	_ = st.CreateServer(context.Background(), &store.Server{ID: serverB, OrganizationID: orgB, Name: "srv-b", IPAddress: "198.51.100.2"})

	claimsA := &auth.Claims{UserID: userA, OrganizationID: orgA, Role: "owner"}
	ctxA := context.WithValue(context.Background(), auth.UserContextKey, claimsA)

	claimsB := &auth.Claims{UserID: userB, OrganizationID: orgB, Role: "owner"}
	ctxB := context.WithValue(context.Background(), auth.UserContextKey, claimsB)

	// 1. Create Domain for Org A
	bodyA, _ := json.Marshal(CreateEmailDomainRequest{ServerID: serverA.String(), Domain: "orga-mail.com"})
	reqA := httptest.NewRequest("POST", "/api/v1/email/domains", bytes.NewReader(bodyA)).WithContext(ctxA)
	wA := httptest.NewRecorder()
	handler.CreateDomain(wA, reqA)
	if wA.Code != http.StatusCreated {
		t.Fatalf("Failed to create domain for Org A: %s", wA.Body.String())
	}
	var domAResp struct {
		Data store.EmailDomain `json:"data"`
	}
	_ = json.Unmarshal(wA.Body.Bytes(), &domAResp)
	domAID := domAResp.Data.ID

	// Create Mailbox for Org A
	mbBodyA, _ := json.Marshal(CreateMailboxRequest{DomainID: domAID.String(), LocalPart: "alice", Password: "Password123!", Name: "Alice"})
	mbReqA := httptest.NewRequest("POST", "/api/v1/email/mailboxes", bytes.NewReader(mbBodyA)).WithContext(ctxA)
	mbWA := httptest.NewRecorder()
	handler.CreateMailbox(mbWA, mbReqA)
	if mbWA.Code != http.StatusCreated {
		t.Fatalf("Failed to create mailbox for Org A: %s", mbWA.Body.String())
	}

	// 2. Create Domain for Org B
	bodyB, _ := json.Marshal(CreateEmailDomainRequest{ServerID: serverB.String(), Domain: "orgb-mail.com"})
	reqB := httptest.NewRequest("POST", "/api/v1/email/domains", bytes.NewReader(bodyB)).WithContext(ctxB)
	wB := httptest.NewRecorder()
	handler.CreateDomain(wB, reqB)
	if wB.Code != http.StatusCreated {
		t.Fatalf("Failed to create domain for Org B: %s", wB.Body.String())
	}
	var domBResp struct {
		Data store.EmailDomain `json:"data"`
	}
	_ = json.Unmarshal(wB.Body.Bytes(), &domBResp)
	domBID := domBResp.Data.ID

	// Create Mailbox for Org B
	mbBodyB, _ := json.Marshal(CreateMailboxRequest{DomainID: domBID.String(), LocalPart: "bob", Password: "Password123!", Name: "Bob"})
	mbReqB := httptest.NewRequest("POST", "/api/v1/email/mailboxes", bytes.NewReader(mbBodyB)).WithContext(ctxB)
	mbWB := httptest.NewRecorder()
	handler.CreateMailbox(mbWB, mbReqB)
	if mbWB.Code != http.StatusCreated {
		t.Fatalf("Failed to create mailbox for Org B: %s", mbWB.Body.String())
	}

	// 3. Org A calls ListDomains -> Must ONLY see orga-mail.com, NEVER orgb-mail.com
	listDomReqA := httptest.NewRequest("GET", "/api/v1/email/domains", nil).WithContext(ctxA)
	listDomWA := httptest.NewRecorder()
	handler.ListDomains(listDomWA, listDomReqA)
	if listDomWA.Code != http.StatusOK {
		t.Fatalf("ListDomains for Org A failed: %s", listDomWA.Body.String())
	}
	var listDomARes struct {
		Data []*store.EmailDomain `json:"data"`
	}
	_ = json.Unmarshal(listDomWA.Body.Bytes(), &listDomARes)
	if len(listDomARes.Data) != 1 || listDomARes.Data[0].Domain != "orga-mail.com" {
		t.Fatalf("Org A saw incorrect domains: %v", listDomARes.Data)
	}

	// 4. Org B calls ListDomains -> Must ONLY see orgb-mail.com, NEVER orga-mail.com
	listDomReqB := httptest.NewRequest("GET", "/api/v1/email/domains", nil).WithContext(ctxB)
	listDomWB := httptest.NewRecorder()
	handler.ListDomains(listDomWB, listDomReqB)
	if listDomWB.Code != http.StatusOK {
		t.Fatalf("ListDomains for Org B failed: %s", listDomWB.Body.String())
	}
	var listDomBRes struct {
		Data []*store.EmailDomain `json:"data"`
	}
	_ = json.Unmarshal(listDomWB.Body.Bytes(), &listDomBRes)
	if len(listDomBRes.Data) != 1 || listDomBRes.Data[0].Domain != "orgb-mail.com" {
		t.Fatalf("Org B saw incorrect domains: %v", listDomBRes.Data)
	}

	// 5. Org A calls ListMailboxes -> Must ONLY see alice@orga-mail.com, NEVER bob@orgb-mail.com
	listMbReqA := httptest.NewRequest("GET", "/api/v1/email/mailboxes", nil).WithContext(ctxA)
	listMbWA := httptest.NewRecorder()
	handler.ListMailboxes(listMbWA, listMbReqA)
	if listMbWA.Code != http.StatusOK {
		t.Fatalf("ListMailboxes for Org A failed: %s", listMbWA.Body.String())
	}
	var listMbARes struct {
		Data []*store.EmailMailbox `json:"data"`
	}
	_ = json.Unmarshal(listMbWA.Body.Bytes(), &listMbARes)
	if len(listMbARes.Data) != 1 || listMbARes.Data[0].Email != "alice@orga-mail.com" {
		t.Fatalf("Org A saw incorrect mailboxes: %v", listMbARes.Data)
	}

	// 6. Org A attempts to call ListMailboxes with domain_id of Org B -> Must be Forbidden
	crossMbReq := httptest.NewRequest("GET", "/api/v1/email/mailboxes?domain_id="+domBID.String(), nil).WithContext(ctxA)
	crossMbW := httptest.NewRecorder()
	handler.ListMailboxes(crossMbW, crossMbReq)
	if crossMbW.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when Org A requests Org B's mailboxes, got %d", crossMbW.Code)
	}

	// 7. Org A attempts to GetDomain of Org B -> Must be Forbidden
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", domBID.String())
	crossDomReq := httptest.NewRequest("GET", "/api/v1/email/domains/"+domBID.String(), nil)
	crossDomReq = crossDomReq.WithContext(context.WithValue(ctxA, chi.RouteCtxKey, rctx))
	crossDomW := httptest.NewRecorder()
	handler.GetDomain(crossDomW, crossDomReq)
	if crossDomW.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when Org A gets Org B's domain, got %d", crossDomW.Code)
	}
}



