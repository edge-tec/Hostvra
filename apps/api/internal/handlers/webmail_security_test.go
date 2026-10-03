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
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/email/dovecot"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

type webmailSecurityEnv struct {
	router      chi.Router
	wmHandler   *WebmailHandler
	store       store.Store
	cfg         *config.Config
	orgID       uuid.UUID
	userID      uuid.UUID
	serverID    uuid.UUID
	mailboxA    *store.EmailMailbox
	mailboxB    *store.EmailMailbox
	mailboxAPwd string
	mailboxBPwd string
	cpToken     string // Control Panel JWT for CP user
}

func setupWebmailSecurityEnv(t *testing.T) *webmailSecurityEnv {
	st := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auditLogger := audit.NewLogger(st, logger)
	cfg := &config.Config{JWTSecret: "test-webmail-strict-security-key-32b"}

	wmHandler := NewWebmailHandler(cfg, st, auditLogger)

	orgID := uuid.New()
	userID := uuid.New()
	serverID := uuid.New()

	_ = st.CreateOrganization(context.Background(), &store.Organization{
		ID:   orgID,
		Name: "Security Test Org",
		Slug: "sec-org",
	})

	_ = st.CreateServer(context.Background(), &store.Server{
		ID:             serverID,
		OrganizationID: orgID,
		Name:           "sec-mail-node",
		IPAddress:      "198.51.100.20",
		Status:         "online",
	})

	domainID := uuid.New()
	_ = st.CreateEmailDomain(context.Background(), &store.EmailDomain{
		ID:                domainID,
		OrganizationID:    orgID,
		ServerID:          serverID,
		Domain:            "hostvrasec.com",
		MailHostname:      "mail.hostvrasec.com",
		StorageLimitBytes: 107374182400,
		Status:            "active",
	})

	// Mailbox A (Alice)
	pwdA := "AliceP@ssw0rd999!"
	mbA := &store.EmailMailbox{
		ID:           uuid.New(),
		DomainID:     domainID,
		ServerID:     serverID,
		LocalPart:    "alice",
		Email:        "alice@hostvrasec.com",
		PasswordHash: dovecot.HashPassword(pwdA),
		Name:         "Alice Sec",
		QuotaBytes:   5368709120,
		IsActive:     true,
	}
	_ = st.CreateEmailMailbox(context.Background(), mbA)

	// Mailbox B (Bob)
	pwdB := "BobP@ssw0rd888!"
	mbB := &store.EmailMailbox{
		ID:           uuid.New(),
		DomainID:     domainID,
		ServerID:     serverID,
		LocalPart:    "bob",
		Email:        "bob@hostvrasec.com",
		PasswordHash: dovecot.HashPassword(pwdB),
		Name:         "Bob Sec",
		QuotaBytes:   5368709120,
		IsActive:     true,
	}
	_ = st.CreateEmailMailbox(context.Background(), mbB)

	// Create sample message for Bob
	msgBob := &store.WebmailMessage{
		ID:            uuid.New(),
		MailboxID:     mbB.ID,
		AccountEmail:  mbB.Email,
		MessageID:     fmt.Sprintf("<%s@hostvrasec.com>", uuid.New().String()),
		Folder:        "inbox",
		Subject:       "Confidential Financial Memo for Bob",
		FromEmail:     "boss@hostvrasec.com",
		FromName:      "The Boss",
		ToEmail:       mbB.Email,
		Snippet:       "Secret bonus details",
		BodyText:      "Here are the strictly confidential bonuses...",
		HasAttachment: false,
		SizeBytes:     1024,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	_ = st.CreateWebmailMessage(context.Background(), msgBob)

	// Generate Control Panel JWT for owner
	now := time.Now().UTC()
	cpClaims := &auth.Claims{
		UserID:         userID,
		Email:          "owner@hostvrasec.com",
		OrganizationID: orgID,
		Role:           "owner",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	cpJwt := jwt.NewWithClaims(jwt.SigningMethodHS256, cpClaims)
	cpTokenStr, _ := cpJwt.SignedString([]byte(cfg.JWTSecret))

	// Setup Router mirroring main.go
	r := chi.NewRouter()
	r.Route("/api/v1/webmail", func(wr chi.Router) {
		// Standalone Auth (Public)
		wr.Post("/auth", wmHandler.DirectAuth)

		// Webmail SSO
		wr.With(auth.Middleware(cfg.JWTSecret)).Post("/sso/generate", wmHandler.GenerateSSOTicket)
		wr.Post("/sso/validate", wmHandler.ValidateSSOTicket)
		wr.Get("/sso/validate", wmHandler.ValidateSSOTicket)

		// Session & Logout
		wr.Get("/session", wmHandler.CheckSession)
		wr.Post("/logout", wmHandler.Logout)

		// Protected Webmail routes
		wr.Group(func(pr chi.Router) {
			pr.Use(wmHandler.RequireWebmailAuth)
			pr.Get("/messages", wmHandler.ListMessages)
			pr.Get("/messages/{id}", wmHandler.GetMessage)
			pr.Post("/send", wmHandler.SendMessage)
			pr.Post("/draft", wmHandler.SaveDraft)
			pr.Get("/counts", wmHandler.GetFolderCounts)
			pr.Get("/filters", wmHandler.ListFilters)
			pr.Post("/filters", wmHandler.CreateFilter)
			pr.Get("/contacts", wmHandler.ListContacts)
			pr.Post("/contacts", wmHandler.CreateContact)
		})
	})

	return &webmailSecurityEnv{
		router:      r,
		wmHandler:   wmHandler,
		store:       st,
		cfg:         cfg,
		orgID:       orgID,
		userID:      userID,
		serverID:    serverID,
		mailboxA:    mbA,
		mailboxB:    mbB,
		mailboxAPwd: pwdA,
		mailboxBPwd: pwdB,
		cpToken:     cpTokenStr,
	}
}

// TEST 1: Open Webmail without authentication (API level verification)
// EXPECT: 401 Unauthorized, no mailbox data.
func TestSecurity_Test1_UnauthenticatedAccess(t *testing.T) {
	env := setupWebmailSecurityEnv(t)

	req := httptest.NewRequest("GET", "/api/v1/webmail/messages", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("TEST 1 FAILED: Expected 401 Unauthorized for unauthenticated request, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res["data"] != nil {
		t.Fatalf("TEST 1 FAILED: Mailbox data exposed in unauthenticated response: %v", res["data"])
	}
}

// TEST 2: Open /webmail/inbox endpoints without authentication
// EXPECT: 401 Unauthorized, no mailbox data.
func TestSecurity_Test2_DirectInboxEndpointUnauthenticated(t *testing.T) {
	env := setupWebmailSecurityEnv(t)

	endpoints := []string{
		"/api/v1/webmail/messages?folder=inbox",
		fmt.Sprintf("/api/v1/webmail/messages?account=%s", env.mailboxA.Email),
		fmt.Sprintf("/api/v1/webmail/counts?mailbox_id=%s", env.mailboxA.ID),
		"/api/v1/webmail/filters",
		"/api/v1/webmail/contacts",
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest("GET", ep, nil)
		w := httptest.NewRecorder()
		env.router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("TEST 2 FAILED: Expected 401 for unauthenticated request to %s, got %d. Body: %s", ep, w.Code, w.Body.String())
		}
	}
}

// TEST 3: Login with valid credentials
// EXPECT: 200 OK, authenticated session token returned, session cookie set.
func TestSecurity_Test3_LoginWithValidCredentials(t *testing.T) {
	env := setupWebmailSecurityEnv(t)

	body, _ := json.Marshal(map[string]string{
		"email":    env.mailboxA.Email,
		"password": env.mailboxAPwd,
	})
	req := httptest.NewRequest("POST", "/api/v1/webmail/auth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("TEST 3 FAILED: Expected 200 OK for valid login, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res struct {
		Data struct {
			Token   string                 `json:"token"`
			Mailbox map[string]interface{} `json:"mailbox"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("TEST 3 FAILED: Failed to unmarshal login response: %v", err)
	}

	if res.Data.Token == "" {
		t.Fatalf("TEST 3 FAILED: No authentication token in response")
	}

	// Verify session cookie was set
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "hostvra_webmail_token" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("TEST 3 FAILED: Expected hostvra_webmail_token cookie to be set")
	}
	if !sessionCookie.HttpOnly {
		t.Fatalf("TEST 3 FAILED: Session cookie must be HttpOnly")
	}
}

// TEST 4: Login with invalid credentials
// EXPECT: 401 Unauthorized, no session token returned.
func TestSecurity_Test4_LoginWithInvalidCredentials(t *testing.T) {
	env := setupWebmailSecurityEnv(t)

	body, _ := json.Marshal(map[string]string{
		"email":    env.mailboxA.Email,
		"password": "WrongPassword123!",
	})
	req := httptest.NewRequest("POST", "/api/v1/webmail/auth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("TEST 4 FAILED: Expected 401 Unauthorized for invalid login, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res["data"] != nil {
		t.Fatalf("TEST 4 FAILED: Authentication data leaked on failed login: %v", res["data"])
	}
}

// TEST 5: Login -> Logout -> Refresh/Check session
// EXPECT: Login -> 200, Logout -> 200 (cookie cleared), Check session -> 401 Unauthorized.
func TestSecurity_Test5_LoginLogoutRefresh(t *testing.T) {
	env := setupWebmailSecurityEnv(t)

	// Step 1: Login
	body, _ := json.Marshal(map[string]string{
		"email":    env.mailboxA.Email,
		"password": env.mailboxAPwd,
	})
	req := httptest.NewRequest("POST", "/api/v1/webmail/auth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("TEST 5 Setup failed: %d - %s", w.Code, w.Body.String())
	}

	var loginRes struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &loginRes)
	token := loginRes.Data.Token

	// Step 2: Logout
	logoutReq := httptest.NewRequest("POST", "/api/v1/webmail/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+token)
	logoutW := httptest.NewRecorder()
	env.router.ServeHTTP(logoutW, logoutReq)
	if logoutW.Code != http.StatusOK {
		t.Fatalf("TEST 5 Logout failed: %d - %s", logoutW.Code, logoutW.Body.String())
	}

	// Verify cookie expiration in logout response
	var clearedCookie *http.Cookie
	for _, c := range logoutW.Result().Cookies() {
		if c.Name == "hostvra_webmail_token" {
			clearedCookie = c
			break
		}
	}
	if clearedCookie == nil || clearedCookie.MaxAge > 0 {
		t.Fatalf("TEST 5 FAILED: Cookie was not cleared with MaxAge <= 0 on logout: %v", clearedCookie)
	}

	// Step 3: Check session with revoked token
	checkReq := httptest.NewRequest("GET", "/api/v1/webmail/session", nil)
	checkReq.Header.Set("Authorization", "Bearer "+token)
	checkW := httptest.NewRecorder()
	env.router.ServeHTTP(checkW, checkReq)

	if checkW.Code != http.StatusUnauthorized {
		t.Fatalf("TEST 5 FAILED: Expected 401 for session check after logout, got %d. Body: %s", checkW.Code, checkW.Body.String())
	}
}

// TEST 6: Login -> Logout -> Browser Back (replay revoked session token)
// EXPECT: 401 Unauthorized (session invalidated in revoked list).
func TestSecurity_Test6_LoginLogoutBrowserBackReplay(t *testing.T) {
	env := setupWebmailSecurityEnv(t)

	// Step 1: Login
	body, _ := json.Marshal(map[string]string{
		"email":    env.mailboxA.Email,
		"password": env.mailboxAPwd,
	})
	req := httptest.NewRequest("POST", "/api/v1/webmail/auth", bytes.NewReader(body))
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	var loginRes struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &loginRes)
	token := loginRes.Data.Token

	// Step 2: Logout (revokes token on server)
	logoutReq := httptest.NewRequest("POST", "/api/v1/webmail/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+token)
	logoutW := httptest.NewRecorder()
	env.router.ServeHTTP(logoutW, logoutReq)

	// Step 3: Simulate browser back / replayed request to protected endpoint
	backReq := httptest.NewRequest("GET", "/api/v1/webmail/messages", nil)
	backReq.Header.Set("Authorization", "Bearer "+token)
	backW := httptest.NewRecorder()
	env.router.ServeHTTP(backW, backReq)

	if backW.Code != http.StatusUnauthorized {
		t.Fatalf("TEST 6 FAILED: Browser back replay of revoked token must return 401, got %d. Body: %s", backW.Code, backW.Body.String())
	}
}

// TEST 7: Login -> Logout -> directly open /webmail/inbox
// EXPECT: 401 Unauthorized (protected route rejects stale/empty token).
func TestSecurity_Test7_LoginLogoutDirectProtectedURL(t *testing.T) {
	env := setupWebmailSecurityEnv(t)

	// Login and logout to ensure clean cycle
	body, _ := json.Marshal(map[string]string{
		"email":    env.mailboxA.Email,
		"password": env.mailboxAPwd,
	})
	req := httptest.NewRequest("POST", "/api/v1/webmail/auth", bytes.NewReader(body))
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	var loginRes struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &loginRes)

	logoutReq := httptest.NewRequest("POST", "/api/v1/webmail/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+loginRes.Data.Token)
	logoutW := httptest.NewRecorder()
	env.router.ServeHTTP(logoutW, logoutReq)

	// Request protected endpoint without token (as would happen after cookie cleared)
	directReq := httptest.NewRequest("GET", "/api/v1/webmail/messages?folder=inbox", nil)
	directW := httptest.NewRecorder()
	env.router.ServeHTTP(directW, directReq)

	if directW.Code != http.StatusUnauthorized {
		t.Fatalf("TEST 7 FAILED: Direct protected URL after logout must return 401, got %d", directW.Code)
	}
}

// TEST 8: Control Panel -> Webmail SSO
// EXPECT: Control Panel user can generate SSO ticket and redeem it for a valid Webmail session without mailbox password.
func TestSecurity_Test8_ControlPanelSSOFlow(t *testing.T) {
	env := setupWebmailSecurityEnv(t)

	// Step 1: Generate SSO Ticket using Control Panel JWT
	genBody, _ := json.Marshal(map[string]interface{}{
		"mailbox_id": env.mailboxA.ID,
	})
	genReq := httptest.NewRequest("POST", "/api/v1/webmail/sso/generate", bytes.NewReader(genBody))
	genReq.Header.Set("Authorization", "Bearer "+env.cpToken)
	genReq.Header.Set("Content-Type", "application/json")
	genW := httptest.NewRecorder()
	env.router.ServeHTTP(genW, genReq)

	if genW.Code != http.StatusOK {
		t.Fatalf("TEST 8 FAILED: Generate SSO ticket failed with %d: %s", genW.Code, genW.Body.String())
	}

	var genRes struct {
		Data struct {
			Ticket    string `json:"ticket"`
			ExpiresIn int    `json:"expires_in"`
			Redirect  string `json:"redirect_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(genW.Body.Bytes(), &genRes); err != nil {
		t.Fatalf("TEST 8 FAILED: Failed to parse SSO ticket response: %v", err)
	}

	if genRes.Data.Ticket == "" {
		t.Fatalf("TEST 8 FAILED: SSO ticket is empty")
	}
	if genRes.Data.ExpiresIn != 60 {
		t.Fatalf("TEST 8 FAILED: SSO ticket TTL must be 60 seconds, got %d", genRes.Data.ExpiresIn)
	}

	// Step 2: Validate SSO ticket (Redeem)
	valBody, _ := json.Marshal(map[string]string{
		"ticket": genRes.Data.Ticket,
	})
	valReq := httptest.NewRequest("POST", "/api/v1/webmail/sso/validate", bytes.NewReader(valBody))
	valReq.Header.Set("Content-Type", "application/json")
	valW := httptest.NewRecorder()
	env.router.ServeHTTP(valW, valReq)

	if valW.Code != http.StatusOK {
		t.Fatalf("TEST 8 FAILED: Validate SSO ticket failed with %d: %s", valW.Code, valW.Body.String())
	}

	var valRes struct {
		Data struct {
			Token   string                 `json:"token"`
			Mailbox map[string]interface{} `json:"mailbox"`
		} `json:"data"`
	}
	if err := json.Unmarshal(valW.Body.Bytes(), &valRes); err != nil {
		t.Fatalf("TEST 8 FAILED: Failed to parse validate response: %v", err)
	}

	if valRes.Data.Token == "" {
		t.Fatalf("TEST 8 FAILED: No webmail token returned upon SSO validation")
	}

	// Step 3: Access protected endpoint using the SSO-issued token
	inboxReq := httptest.NewRequest("GET", "/api/v1/webmail/messages", nil)
	inboxReq.Header.Set("Authorization", "Bearer "+valRes.Data.Token)
	inboxW := httptest.NewRecorder()
	env.router.ServeHTTP(inboxW, inboxReq)

	if inboxW.Code != http.StatusOK {
		t.Fatalf("TEST 8 FAILED: Accessing inbox with SSO-issued token failed with %d: %s", inboxW.Code, inboxW.Body.String())
	}
}

// TEST 9: Reuse an SSO token
// EXPECT: Rejected with 401 Unauthorized (SSO_TICKET_ALREADY_USED).
func TestSecurity_Test9_ReuseSSOTicket(t *testing.T) {
	env := setupWebmailSecurityEnv(t)

	// Step 1: Generate ticket
	genBody, _ := json.Marshal(map[string]interface{}{
		"mailbox_id": env.mailboxA.ID,
	})
	genReq := httptest.NewRequest("POST", "/api/v1/webmail/sso/generate", bytes.NewReader(genBody))
	genReq.Header.Set("Authorization", "Bearer "+env.cpToken)
	genW := httptest.NewRecorder()
	env.router.ServeHTTP(genW, genReq)

	var genRes struct {
		Data struct {
			Ticket string `json:"ticket"`
		} `json:"data"`
	}
	_ = json.Unmarshal(genW.Body.Bytes(), &genRes)
	ticket := genRes.Data.Ticket

	// Step 2: First use - must succeed
	valBody, _ := json.Marshal(map[string]string{"ticket": ticket})
	valReq1 := httptest.NewRequest("POST", "/api/v1/webmail/sso/validate", bytes.NewReader(valBody))
	valW1 := httptest.NewRecorder()
	env.router.ServeHTTP(valW1, valReq1)

	if valW1.Code != http.StatusOK {
		t.Fatalf("TEST 9 Setup failed: first ticket use should succeed, got %d", valW1.Code)
	}

	// Step 3: Second use - must be rejected
	valReq2 := httptest.NewRequest("POST", "/api/v1/webmail/sso/validate", bytes.NewReader(valBody))
	valW2 := httptest.NewRecorder()
	env.router.ServeHTTP(valW2, valReq2)

	if valW2.Code != http.StatusUnauthorized {
		t.Fatalf("TEST 9 FAILED: Reused SSO ticket was not rejected with 401, got %d. Body: %s", valW2.Code, valW2.Body.String())
	}
}

// TEST 10: Expired SSO token
// EXPECT: Rejected with 401 Unauthorized (SSO_TICKET_EXPIRED).
func TestSecurity_Test10_ExpiredSSOTicket(t *testing.T) {
	env := setupWebmailSecurityEnv(t)

	// Manually inject an already expired ticket into handler
	expiredTicket := "expired-test-ticket-xyz"
	env.wmHandler.ssoMu.Lock()
	env.wmHandler.ssoTickets[expiredTicket] = &SSOTicket{
		Ticket:    expiredTicket,
		MailboxID: env.mailboxA.ID,
		ExpiresAt: time.Now().Add(-10 * time.Second), // expired 10 seconds ago
		Used:      false,
	}
	env.wmHandler.ssoMu.Unlock()

	// Attempt validation
	valBody, _ := json.Marshal(map[string]string{"ticket": expiredTicket})
	valReq := httptest.NewRequest("POST", "/api/v1/webmail/sso/validate", bytes.NewReader(valBody))
	valW := httptest.NewRecorder()
	env.router.ServeHTTP(valW, valReq)

	if valW.Code != http.StatusUnauthorized {
		t.Fatalf("TEST 10 FAILED: Expired SSO ticket was not rejected with 401, got %d. Body: %s", valW.Code, valW.Body.String())
	}
}

// TEST 11: Attempt to access another mailbox's API/data (Cross-Mailbox Authorization)
// EXPECT: 403 Forbidden. No data leakage between Alice and Bob.
func TestSecurity_Test11_CrossMailboxAuthorizationEnforcement(t *testing.T) {
	env := setupWebmailSecurityEnv(t)

	// Step 1: Login as Alice
	loginBody, _ := json.Marshal(map[string]string{
		"email":    env.mailboxA.Email,
		"password": env.mailboxAPwd,
	})
	req := httptest.NewRequest("POST", "/api/v1/webmail/auth", bytes.NewReader(loginBody))
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	var loginRes struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &loginRes)
	aliceToken := loginRes.Data.Token

	// Step 2: Alice attempts to query Bob's messages by explicitly specifying account=bob@hostvrasec.com
	crossReq1 := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/webmail/messages?account=%s", env.mailboxB.Email), nil)
	crossReq1.Header.Set("Authorization", "Bearer "+aliceToken)
	crossW1 := httptest.NewRecorder()
	env.router.ServeHTTP(crossW1, crossReq1)

	if crossW1.Code != http.StatusForbidden {
		t.Fatalf("TEST 11 FAILED: Expected 403 Forbidden when Alice accesses Bob's account query param, got %d. Body: %s", crossW1.Code, crossW1.Body.String())
	}

	// Step 3: Alice attempts to query counts using Bob's mailbox_id
	crossReq2 := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/webmail/counts?mailbox_id=%s", env.mailboxB.ID), nil)
	crossReq2.Header.Set("Authorization", "Bearer "+aliceToken)
	crossW2 := httptest.NewRecorder()
	env.router.ServeHTTP(crossW2, crossReq2)

	if crossW2.Code != http.StatusForbidden {
		t.Fatalf("TEST 11 FAILED: Expected 403 Forbidden when Alice accesses Bob's mailbox_id, got %d. Body: %s", crossW2.Code, crossW2.Body.String())
	}

	// Step 4: Alice attempts to send email claiming to be Bob (From: bob@hostvrasec.com)
	sendPayload, _ := json.Marshal(map[string]interface{}{
		"account_email": env.mailboxB.Email,
		"to_email":      "victim@external.com",
		"subject":       "Forged memo from Bob",
		"body_text":     "This was forged by Alice",
	})
	crossReq3 := httptest.NewRequest("POST", "/api/v1/webmail/send", bytes.NewReader(sendPayload))
	crossReq3.Header.Set("Authorization", "Bearer "+aliceToken)
	crossReq3.Header.Set("Content-Type", "application/json")
	crossW3 := httptest.NewRecorder()
	env.router.ServeHTTP(crossW3, crossReq3)

	if crossW3.Code != http.StatusForbidden {
		t.Fatalf("TEST 11 FAILED: Expected 403 Forbidden when Alice attempts to send mail as Bob, got %d. Body: %s", crossW3.Code, crossW3.Body.String())
	}
}

// TEST 12: Control panel administrator / owner can send mail and manage any mailbox in their organization
func TestSecurity_Test12_ControlPanelUserCanAccessAndSendFromAnyMailbox(t *testing.T) {
	env := setupWebmailSecurityEnv(t)

	// Step 1: CP User sends email as Bob using CP token in Authorization
	sendPayload1, _ := json.Marshal(map[string]interface{}{
		"account_email": env.mailboxB.Email,
		"from_email":    env.mailboxB.Email,
		"to_email":      "client@external.com",
		"subject":       "Official message from Bob",
		"body_text":     "Sent by administrator on behalf of Bob",
	})
	req1 := httptest.NewRequest("POST", "/api/v1/webmail/send", bytes.NewReader(sendPayload1))
	req1.Header.Set("Authorization", "Bearer "+env.cpToken)
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	env.router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("TEST 12 FAILED: Expected 200 OK when Control Panel owner sends mail for Bob, got %d. Body: %s", w1.Code, w1.Body.String())
	}

	// Step 2: Login as Alice to get a webmail token
	loginBody, _ := json.Marshal(map[string]string{
		"email":    env.mailboxA.Email,
		"password": env.mailboxAPwd,
	})
	authReq := httptest.NewRequest("POST", "/api/v1/webmail/auth", bytes.NewReader(loginBody))
	authW := httptest.NewRecorder()
	env.router.ServeHTTP(authW, authReq)

	var loginRes struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(authW.Body.Bytes(), &loginRes)
	aliceToken := loginRes.Data.Token

	// Step 3: Request carrying a stale/different webmail token (Alice) in Authorization,
	// but accompanied by the Control Panel owner token in X-CP-Token.
	// Sending as Bob must SUCCEED because the user is verified as the Control Panel owner.
	sendPayload2, _ := json.Marshal(map[string]interface{}{
		"account_email": env.mailboxB.Email,
		"from_email":    env.mailboxB.Email,
		"to_email":      "client2@external.com",
		"subject":       "Another message from Bob",
		"body_text":     "Sent via Webmail Client in Control Panel",
	})
	req2 := httptest.NewRequest("POST", "/api/v1/webmail/send", bytes.NewReader(sendPayload2))
	req2.Header.Set("Authorization", "Bearer "+aliceToken)
	req2.Header.Set("X-CP-Token", env.cpToken)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	env.router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("TEST 12 FAILED: Expected 200 OK when request has X-CP-Token for admin sending as Bob, got %d. Body: %s", w2.Code, w2.Body.String())
	}
}

