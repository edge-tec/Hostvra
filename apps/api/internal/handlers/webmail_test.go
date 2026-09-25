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

	"hostvra/agent/pkg/email/dovecot"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/dns"
	"hostvra/api/internal/store"
)

func setupWebmailTestEnv(t *testing.T) (*WebmailHandler, *EmailHandler, store.Store, uuid.UUID, uuid.UUID, uuid.UUID, *store.EmailMailbox) {
	st := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auditLogger := audit.NewLogger(st, logger)
	dnsSvc := dns.NewService()
	cfg := &config.Config{JWTSecret: "test-webmail-secret-key-32b-length"}

	webmailHandler := NewWebmailHandler(cfg, st, auditLogger)
	emailHandler := NewEmailHandler(cfg, st, dnsSvc, auditLogger)

	orgID := uuid.New()
	userID := uuid.New()
	serverID := uuid.New()

	_ = st.CreateOrganization(context.Background(), &store.Organization{
		ID:   orgID,
		Name: "Webmail Test Org",
		Slug: "webmail-test-org",
	})

	_ = st.CreateServer(context.Background(), &store.Server{
		ID:             serverID,
		OrganizationID: orgID,
		Name:           "mail-node-1",
		IPAddress:      "198.51.100.10",
		Status:         "online",
	})

	domainID := uuid.New()
	_ = st.CreateEmailDomain(context.Background(), &store.EmailDomain{
		ID:                domainID,
		OrganizationID:    orgID,
		ServerID:          serverID,
		Domain:            "enterprise.net",
		MailHostname:      "mail.enterprise.net",
		StorageLimitBytes: 53687091200,
		Status:            "active",
		DKIMSelector:      "default",
	})

	// Pre-create mailbox with Dovecot SHA512-CRYPT password
	mailboxID := uuid.New()
	passHash := dovecot.HashPassword("SecretPass789!@#")
	mb := &store.EmailMailbox{
		ID:           mailboxID,
		DomainID:     domainID,
		ServerID:     serverID,
		LocalPart:    "ceo",
		Email:        "ceo@enterprise.net",
		PasswordHash: passHash,
		Name:         "Chief Executive",
		QuotaBytes:   10737418240,
		IsActive:     true,
	}
	_ = st.CreateEmailMailbox(context.Background(), mb)

	return webmailHandler, emailHandler, st, orgID, userID, serverID, mb
}

func TestWebmailHandler_DraftLifecycle(t *testing.T) {
	wmHandler, _, _, orgID, userID, _, mb := setupWebmailTestEnv(t)
	claims := &auth.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           "owner",
	}
	ctx := context.WithValue(context.Background(), auth.UserContextKey, claims)

	// 1. Save Draft
	draftReq := SendWebmailMessageRequest{
		AccountEmail: mb.Email,
		ToEmail:      "investor@venture.com",
		Subject:      "Quarterly Earnings Report Q3",
		BodyText:     "Please find the summary of our fiscal operations attached.",
	}
	draftBody, _ := json.Marshal(draftReq)
	req := httptest.NewRequest("POST", "/api/v1/webmail/messages/draft", bytes.NewReader(draftBody)).WithContext(ctx)
	w := httptest.NewRecorder()

	wmHandler.SaveDraft(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("SaveDraft failed with code %d: %s", w.Code, w.Body.String())
	}

	var draftResp struct {
		Data store.WebmailMessage `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &draftResp)
	msgID := draftResp.Data.ID

	if draftResp.Data.Folder != "drafts" {
		t.Fatalf("Expected message to be in drafts folder, got: folder=%s", draftResp.Data.Folder)
	}

	// 2. List Messages for Mailbox
	listReq := httptest.NewRequest("GET", "/api/v1/webmail/messages?mailbox_id="+mb.ID.String()+"&folder=drafts", nil).WithContext(ctx)
	w = httptest.NewRecorder()
	wmHandler.ListMessages(w, listReq)

	if w.Code != http.StatusOK {
		t.Fatalf("ListMessages failed: %s", w.Body.String())
	}

	var listResp struct {
		Data struct {
			Messages []store.WebmailMessage `json:"messages"`
			Total    int                    `json:"total"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if len(listResp.Data.Messages) != 1 {
		t.Fatalf("Expected 1 draft message, got %d (raw: %s)", len(listResp.Data.Messages), w.Body.String())
	}

	// 3. Update Message Flags (Star and Read)
	starred := true
	unread := false
	flagReqBody, _ := json.Marshal(UpdateMessageFlagsRequest{
		IsStarred: &starred,
		IsUnread:  &unread,
	})
	rCtx := chi.NewRouteContext()
	rCtx.URLParams.Add("id", msgID.String())
	flagReq := httptest.NewRequest("PUT", "/api/v1/webmail/messages/"+msgID.String()+"/flag", bytes.NewReader(flagReqBody)).WithContext(context.WithValue(ctx, chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()

	wmHandler.UpdateMessageFlags(w, flagReq)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateMessageFlags failed: %s", w.Body.String())
	}

	// 4. Move Message to Archive
	moveReqBody, _ := json.Marshal(MoveMessageRequest{TargetFolder: "archive"})
	moveReq := httptest.NewRequest("PUT", "/api/v1/webmail/messages/"+msgID.String()+"/folder", bytes.NewReader(moveReqBody)).WithContext(context.WithValue(ctx, chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()

	wmHandler.MoveMessage(w, moveReq)
	if w.Code != http.StatusOK {
		t.Fatalf("MoveMessage failed: %s", w.Body.String())
	}

	// 5. Get Message Detail
	getReq := httptest.NewRequest("GET", "/api/v1/webmail/messages/"+msgID.String(), nil).WithContext(context.WithValue(ctx, chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	wmHandler.GetMessage(w, getReq)

	if w.Code != http.StatusOK {
		t.Fatalf("GetMessage failed: %s", w.Body.String())
	}
	var getResp struct {
		Data store.WebmailMessage `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &getResp)
	if getResp.Data.Folder != "archive" || !getResp.Data.IsStarred || getResp.Data.IsUnread {
		t.Fatalf("Message state mismatch: folder=%s, starred=%v, unread=%v", getResp.Data.Folder, getResp.Data.IsStarred, getResp.Data.IsUnread)
	}

	// 6. Delete Message
	delReq := httptest.NewRequest("DELETE", "/api/v1/webmail/messages/"+msgID.String(), nil).WithContext(context.WithValue(ctx, chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	wmHandler.DeleteMessage(w, delReq)
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteMessage failed: %s", w.Body.String())
	}
}

func TestWebmailHandler_XSSSanitization(t *testing.T) {
	wmHandler, _, st, _, _, _, mb := setupWebmailTestEnv(t)

	// Inject message with malicious XSS script and onload handlers into store
	maliciousHTML := `<p>Hello customer,</p><script>alert("XSS stolen cookie: "+document.cookie)</script><img src="x" onerror="stealCredentials()"><a href="javascript:alert(1)">Click for Free Gift</a>`
	msgID := uuid.New()
	msg := &store.WebmailMessage{
		ID:           msgID,
		MailboxID:    mb.ID,
		AccountEmail: mb.Email,
		Folder:       "inbox",
		FromName:     "Spammer",
		FromEmail:    "badguy@malicious.com",
		ToName:       mb.Name,
		ToEmail:      mb.Email,
		Subject:      "Dangerous Email",
		BodyText:     "Hello customer",
		BodyHTML:     maliciousHTML,
		IsUnread:     true,
	}
	_ = st.CreateWebmailMessage(context.Background(), msg)

	rCtx := chi.NewRouteContext()
	rCtx.URLParams.Add("id", msgID.String())
	req := httptest.NewRequest("GET", "/api/v1/webmail/messages/"+msgID.String(), nil).WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCtx))
	w := httptest.NewRecorder()

	wmHandler.GetMessage(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GetMessage failed: %s", w.Body.String())
	}

	var resp struct {
		Data store.WebmailMessage `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	sanitized := resp.Data.BodyHTML
	if strings.Contains(sanitized, "<script") {
		t.Fatalf("Security failure: script tag not stripped from sanitized HTML: %s", sanitized)
	}
	if strings.Contains(sanitized, "onerror=") {
		t.Fatalf("Security failure: onerror event not stripped from sanitized HTML: %s", sanitized)
	}
	if strings.Contains(sanitized, "javascript:") {
		t.Fatalf("Security failure: javascript URI not stripped: %s", sanitized)
	}
	if !strings.Contains(sanitized, "Hello customer") {
		t.Fatalf("Expected benign HTML content to be preserved: %s", sanitized)
	}
}

func TestWebmailHandler_DirectAuth(t *testing.T) {
	wmHandler, _, _, _, _, _, mb := setupWebmailTestEnv(t)

	// Valid credentials
	validBody, _ := json.Marshal(WebmailAuthRequest{
		Email:    mb.Email,
		Password: "SecretPass789!@#",
	})
	req := httptest.NewRequest("POST", "/api/v1/webmail/auth", bytes.NewReader(validBody))
	w := httptest.NewRecorder()

	wmHandler.DirectAuth(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected auth success, got %d: %s", w.Code, w.Body.String())
	}

	var authResp struct {
		Data struct {
			Token   string              `json:"token"`
			Mailbox *store.EmailMailbox `json:"mailbox"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &authResp)
	if authResp.Data.Token == "" || authResp.Data.Mailbox.Email != mb.Email {
		t.Fatalf("Auth response invalid: %+v", authResp.Data)
	}

	// Invalid password
	invalidBody, _ := json.Marshal(WebmailAuthRequest{
		Email:    mb.Email,
		Password: "WrongPassword!@#",
	})
	req = httptest.NewRequest("POST", "/api/v1/webmail/auth", bytes.NewReader(invalidBody))
	w = httptest.NewRecorder()

	wmHandler.DirectAuth(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized for wrong password, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEmailHandler_AdvancedEndpoints(t *testing.T) {
	_, emailHandler, st, orgID, userID, serverID, _ := setupWebmailTestEnv(t)
	claims := &auth.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           "owner",
	}
	ctx := context.WithValue(context.Background(), auth.UserContextKey, claims)

	// 1. List Queue
	qReq := httptest.NewRequest("GET", "/api/v1/email/queue", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	emailHandler.ListQueue(w, qReq)
	if w.Code != http.StatusOK {
		t.Fatalf("ListQueue failed: %s", w.Body.String())
	}

	// 2. Flush Queue
	flushReq := httptest.NewRequest("POST", "/api/v1/email/queue/flush", nil).WithContext(ctx)
	w = httptest.NewRecorder()
	emailHandler.FlushQueue(w, flushReq)
	if w.Code != http.StatusOK {
		t.Fatalf("FlushQueue failed: %s", w.Body.String())
	}

	// 3. List Services
	svcReq := httptest.NewRequest("GET", "/api/v1/email/services", nil).WithContext(ctx)
	w = httptest.NewRecorder()
	emailHandler.ListServices(w, svcReq)
	if w.Code != http.StatusOK {
		t.Fatalf("ListServices failed: %s", w.Body.String())
	}

	// 4. Suppressions
	supReqBody, _ := json.Marshal(AddSuppressionRequest{
		Email:      "bounced-user@invalidsite.net",
		Reason:     "hard_bounce",
		BounceCode: "550 5.1.1 User unknown",
	})
	supReq := httptest.NewRequest("POST", "/api/v1/email/suppressions", bytes.NewReader(supReqBody)).WithContext(ctx)
	w = httptest.NewRecorder()
	emailHandler.AddSuppression(w, supReq)
	if w.Code != http.StatusCreated {
		t.Fatalf("AddSuppression failed: %s", w.Body.String())
	}

	// Check IsSuppressed
	isSup, err := st.IsEmailSuppressed(ctx, serverID, "bounced-user@invalidsite.net")
	if err != nil || !isSup {
		t.Fatalf("Expected email to be suppressed, got %v (err: %v)", isSup, err)
	}

	// List Suppressions
	listSupReq := httptest.NewRequest("GET", "/api/v1/email/suppressions", nil).WithContext(ctx)
	w = httptest.NewRecorder()
	emailHandler.ListSuppressions(w, listSupReq)
	if w.Code != http.StatusOK {
		t.Fatalf("ListSuppressions failed: %s", w.Body.String())
	}

	// 5. SMTP Settings
	smtpReq := httptest.NewRequest("GET", "/api/v1/email/smtp-settings?domain=enterprise.net", nil).WithContext(ctx)
	w = httptest.NewRecorder()
	emailHandler.GetSMTPSettings(w, smtpReq)
	if w.Code != http.StatusOK {
		t.Fatalf("GetSMTPSettings failed: %s", w.Body.String())
	}

	var smtpResp struct {
		Data store.SMTPSettings `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &smtpResp)
	if smtpResp.Data.SMTPPort != 587 || smtpResp.Data.IMAPPort != 993 {
		t.Fatalf("Unexpected SMTP/IMAP ports: %+v", smtpResp.Data)
	}
}
