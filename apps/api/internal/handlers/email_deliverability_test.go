package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/email/migration"
	"hostvra/agent/pkg/email/rspamd"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func setupEmailDeliverabilityTest(t *testing.T) (*EmailDeliverabilityHandler, *store.MemoryStore, uuid.UUID, *auth.Claims) {
	t.Helper()
	cfg := &config.Config{JWTSecret: "test_secret_32_characters_long_jwt"}
	memStore := store.NewMemoryStoreWithPath("")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)

	orgID := uuid.New()
	userID := uuid.New()

	org := &store.Organization{
		ID:        orgID,
		Name:      "Deliverability Test Org",
		Slug:      "deliv-org",
		CreatedAt: time.Now(),
	}
	_ = memStore.CreateOrganization(context.Background(), org)

	claims := &auth.Claims{
		UserID:         userID,
		Email:          "admin@deliv-org.com",
		Role:           "admin",
		OrganizationID: orgID,
		IsSuperAdmin:   true,
	}

	h := NewEmailDeliverabilityHandler(cfg, memStore, auditLogger)
	return h, memStore, orgID, claims
}

func TestEmailDeliverability_RelayFlow(t *testing.T) {
	h, _, _, claims := setupEmailDeliverabilityTest(t)

	// 1. Create Relay
	reqBody := CreateRelayRequest{
		Name:         "SendGrid Relay",
		Provider:     "sendgrid",
		Host:         "smtp.sendgrid.net",
		Port:         587,
		Username:     "apikey",
		Password:     "SG.secret_key_123",
		Encryption:   "starttls",
		SenderDomain: "example.com",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	r := httptest.NewRequest("POST", "/api/v1/email/relays", bytes.NewReader(bodyBytes))
	r = r.WithContext(auth.WithClaims(r.Context(), claims))
	w := httptest.NewRecorder()

	h.CreateRelay(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                       `json:"success"`
		Data    store.EmailSMTPRelayRecord `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	created := resp.Data

	if created.PasswordEncrypted != "••••••••" {
		t.Errorf("expected masked password, got %s", created.PasswordEncrypted)
	}

	// 2. List Relays
	r = httptest.NewRequest("GET", "/api/v1/email/relays", nil)
	r = r.WithContext(auth.WithClaims(r.Context(), claims))
	w = httptest.NewRecorder()

	h.ListRelays(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var listResp struct {
		Success bool                         `json:"success"`
		Data    []store.EmailSMTPRelayRecord `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if len(listResp.Data) != 1 {
		t.Fatalf("expected 1 relay, got %d", len(listResp.Data))
	}

	// 3. Update Relay
	updateBody := CreateRelayRequest{
		Name:     "SendGrid Enterprise Relay",
		Host:     "smtp.sendgrid.net",
		Port:         587,
		Username: "apikey",
	}
	uBytes, _ := json.Marshal(updateBody)
	r = httptest.NewRequest("PUT", "/api/v1/email/relays/"+created.ID.String(), bytes.NewReader(uBytes))
	r = r.WithContext(auth.WithClaims(r.Context(), claims))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID.String())
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()

	h.UpdateRelay(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Delete Relay
	r = httptest.NewRequest("DELETE", "/api/v1/email/relays/"+created.ID.String(), nil)
	r = r.WithContext(auth.WithClaims(r.Context(), claims))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()

	h.DeleteRelay(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

func TestEmailDeliverability_SpamAndQuarantineFlow(t *testing.T) {
	h, memStore, orgID, claims := setupEmailDeliverabilityTest(t)

	// Mock Rspamd Manager with temp config dir
	tmpDir := t.TempDir()
	h.SetRspamdManager(rspamd.NewManager(tmpDir))

	// 1. Get Spam Settings Defaults
	r := httptest.NewRequest("GET", "/api/v1/email/spam/settings", nil)
	r = r.WithContext(auth.WithClaims(r.Context(), claims))
	w := httptest.NewRecorder()

	h.GetSpamSettings(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Update Spam Settings
	setReq := store.EmailSpamFilterSettingsRecord{
		RejectScore:             13.0,
		AddHeaderScore:          5.5,
		GreylistScore:           3.5,
		EnableGreylisting:       true,
		EnableBayes:             true,
		QuarantineEnabled:       true,
		QuarantineRetentionDays: 21,
	}
	sBytes, _ := json.Marshal(setReq)
	r = httptest.NewRequest("PUT", "/api/v1/email/spam/settings", bytes.NewReader(sBytes))
	r = r.WithContext(auth.WithClaims(r.Context(), claims))
	w = httptest.NewRecorder()

	h.UpdateSpamSettings(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	// 3. Quarantine Item
	qID := uuid.New()
	mailPath := tmpDir + "/msg.eml"
	_ = os.WriteFile(mailPath, []byte("From: spammer@external.com\r\nTo: user@deliv-org.com\r\nSubject: Phishing Test\r\n\r\nTest content"), 0644)

	qItem := &store.EmailSpamQuarantineItemRecord{
		ID:             qID,
		OrganizationID: orgID,
		Recipient:      "user@deliv-org.com",
		Sender:         "spammer@external.com",
		Subject:        "Phishing Test",
		Score:          16.2,
		Symbols:        `["FORGED_RECIPIENTS"]`,
		Status:         "quarantined",
		MessageID:      "<test-phish@external.com>",
		MailPath:       mailPath,
	}
	_ = memStore.CreateSpamQuarantineItem(context.Background(), qItem)

	r = httptest.NewRequest("GET", "/api/v1/email/spam/quarantine", nil)
	r = r.WithContext(auth.WithClaims(r.Context(), claims))
	w = httptest.NewRecorder()

	h.ListQuarantine(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// Release quarantine item
	r = httptest.NewRequest("POST", "/api/v1/email/spam/quarantine/"+qID.String()+"/release", nil)
	r = r.WithContext(auth.WithClaims(r.Context(), claims))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", qID.String())
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()

	h.ReleaseQuarantineItem(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEmailDeliverability_MigrationFlow(t *testing.T) {
	h, memStore, _, claims := setupEmailDeliverabilityTest(t)

	// Create test mailbox
	mbID := uuid.New()
	mb := &store.EmailMailbox{
		ID:        mbID,
		DomainID:  uuid.New(),
		Email:     "dest@deliv-org.com",
		LocalPart: "dest",
		Name:      "Destination User",
		IsActive:  true,
	}
	_ = memStore.CreateEmailMailbox(context.Background(), mb)

	// 1. Start Migration
	migReq := StartMigrationRequest{
		DestinationMailboxID: mbID,
		Source: migration.SourceConnectionOptions{
			Host:     "imap.mockhost.com",
			Port:     993,
			Username: "source@mockhost.com",
			Password: "mockpassword",
			UseSSL:   true,
		},
	}
	mBytes, _ := json.Marshal(migReq)

	r := httptest.NewRequest("POST", "/api/v1/email/migrations", bytes.NewReader(mBytes))
	r = r.WithContext(auth.WithClaims(r.Context(), claims))
	w := httptest.NewRecorder()

	h.StartMigration(w, r)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                           `json:"success"`
		Data    store.EmailIMAPMigrationRecord `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	migID := resp.Data.ID

	// 2. Get Migration
	r = httptest.NewRequest("GET", "/api/v1/email/migrations/"+migID.String(), nil)
	r = r.WithContext(auth.WithClaims(r.Context(), claims))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", migID.String())
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()

	h.GetMigration(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// 3. Cancel Migration
	r = httptest.NewRequest("POST", "/api/v1/email/migrations/"+migID.String()+"/cancel", nil)
	r = r.WithContext(auth.WithClaims(r.Context(), claims))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()

	h.CancelMigration(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// 4. Delete Migration
	r = httptest.NewRequest("DELETE", "/api/v1/email/migrations/"+migID.String(), nil)
	r = r.WithContext(auth.WithClaims(r.Context(), claims))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()

	h.DeleteMigration(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}
