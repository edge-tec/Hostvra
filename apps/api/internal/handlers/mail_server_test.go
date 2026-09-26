package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"hostvra/api/internal/auth"
	"hostvra/api/internal/store"
)

func TestMailServerSubsystem_APIEndpoints(t *testing.T) {
	handler, _, _, orgID, userID, serverID := setupEmailTestEnv(t)

	claims := &auth.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           "owner",
	}
	ctx := context.WithValue(context.Background(), auth.UserContextKey, claims)

	// 1. Run Preflight Checks via API
	preflightBody, _ := json.Marshal(MailPreflightRequest{
		Hostname: "mail.hostvra-enterprise.com",
	})
	req := httptest.NewRequest("POST", "/api/v1/mail/servers/preflight", bytes.NewReader(preflightBody)).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.RunMailServerPreflight(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("RunMailServerPreflight failed: %d: %s", w.Code, w.Body.String())
	}

	// 2. Create Mail Server via API
	createBody, _ := json.Marshal(CreateMailServerRequest{
		NodeServerID:             serverID.String(),
		Name:                     "Primary Enterprise Mail Node",
		Hostname:                 "mail.hostvra-enterprise.com",
		PrimaryDomain:            "hostvra-enterprise.com",
		AdditionalDomains:        []string{"smtp.hostvra-enterprise.com"},
		IPv4Address:              "192.0.2.100",
		Timezone:                 "UTC",
		StorageLocation:          "/var/mail/vhosts",
		MailboxStorageLimitBytes: 107374182400, // 100GB
		MaxMailboxSizeBytes:      21474836480,  // 20GB
		MaxAttachmentSizeBytes:   52428800,     // 50MB
		SMTPPort:                 25,
		SMTPSubmissionPort:       587,
		SMTPSPort:                465,
		IMAPPort:                 143,
		IMAPSPort:                993,
		POP3Port:                 110,
		POP3SPort:                995,
		TLSEnabled:               true,
		SpamFilterEnabled:        true,
		AntivirusEnabled:         false,
		DKIMEnabled:              true,
		SPFEnabled:               true,
		DMARCEnabled:             true,
		WebmailEnabled:           true,
		RateLimitPerMailboxHr:    500,
		RateLimitPerDomainHr:     5000,
		RateLimitPerIPHr:         10000,
		AuthFailureThreshold:     5,
		InstallPackages:          false,
	})
	req = httptest.NewRequest("POST", "/api/v1/mail/servers", bytes.NewReader(createBody)).WithContext(ctx)
	w = httptest.NewRecorder()
	handler.CreateMailServer(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateMailServer failed: %d: %s", w.Code, w.Body.String())
	}

	var createdResp struct {
		Data store.MailServer `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &createdResp); err != nil {
		t.Fatalf("failed to decode created mail server: %v", err)
	}
	serverUUID := createdResp.Data.ID
	if createdResp.Data.Hostname != "mail.hostvra-enterprise.com" {
		t.Errorf("expected hostname mail.hostvra-enterprise.com, got %s", createdResp.Data.Hostname)
	}

	// 3. List Mail Servers
	req = httptest.NewRequest("GET", "/api/v1/mail/servers", nil).WithContext(ctx)
	w = httptest.NewRecorder()
	handler.ListMailServers(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListMailServers failed: %d: %s", w.Code, w.Body.String())
	}

	var listResp struct {
		Data []*store.MailServer `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil || len(listResp.Data) == 0 {
		t.Fatalf("expected at least 1 mail server in list, got %v", listResp.Data)
	}

	// 4. Get Mail Server by ID
	rCtx := chi.NewRouteContext()
	rCtx.URLParams.Add("id", serverUUID.String())
	req = httptest.NewRequest("GET", "/api/v1/mail/servers/"+serverUUID.String(), nil).WithContext(context.WithValue(ctx, chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.GetMailServer(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GetMailServer failed: %d: %s", w.Code, w.Body.String())
	}

	// 5. Update Mail Server
	updateBody, _ := json.Marshal(UpdateMailServerRequest{
		Name:                  "Primary Enterprise Mail Node (Updated)",
		Hostname:              "mail.hostvra-enterprise.com",
		PrimaryDomain:         "hostvra-enterprise.com",
		RateLimitPerMailboxHr: 800,
		Status:                "active",
		SpamFilterEnabled:     true,
		AntivirusEnabled:      true,
		DKIMEnabled:           true,
	})
	rCtx = chi.NewRouteContext()
	rCtx.URLParams.Add("id", serverUUID.String())
	req = httptest.NewRequest("PUT", "/api/v1/mail/servers/"+serverUUID.String(), bytes.NewReader(updateBody)).WithContext(context.WithValue(ctx, chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.UpdateMailServer(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateMailServer failed: %d: %s", w.Code, w.Body.String())
	}

	// 6. Diagnostics & Spam stats
	req = httptest.NewRequest("GET", "/api/v1/mail/diagnostics?domain=hostvra-enterprise.com", nil).WithContext(ctx)
	w = httptest.NewRecorder()
	handler.GetMailDiagnostics(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GetMailDiagnostics failed: %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("GET", "/api/v1/mail/spam", nil).WithContext(ctx)
	w = httptest.NewRecorder()
	handler.GetSpamProtectionStats(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GetSpamProtectionStats failed: %d: %s", w.Code, w.Body.String())
	}

	// 7. Delete Mail Server
	rCtx = chi.NewRouteContext()
	rCtx.URLParams.Add("id", serverUUID.String())
	req = httptest.NewRequest("DELETE", "/api/v1/mail/servers/"+serverUUID.String(), nil).WithContext(context.WithValue(ctx, chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.DeleteMailServer(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteMailServer failed: %d: %s", w.Code, w.Body.String())
	}
}
