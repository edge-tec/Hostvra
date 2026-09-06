package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
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
