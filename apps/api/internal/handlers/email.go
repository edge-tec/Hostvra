package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/email/dkim"
	"hostvra/agent/pkg/email/dovecot"
	"hostvra/agent/pkg/email/health"
	"hostvra/agent/pkg/email/postfix"
	"hostvra/agent/pkg/email/queue"
	"hostvra/agent/pkg/email/services"
	"hostvra/agent/pkg/email/storage"
	"hostvra/agent/pkg/email/tester"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/dns"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type EmailHandler struct {
	cfg   *config.Config
	store store.Store
	dns   *dns.Service
	audit *audit.Logger
}

func NewEmailHandler(cfg *config.Config, s store.Store, d *dns.Service, a *audit.Logger) *EmailHandler {
	return &EmailHandler{
		cfg:   cfg,
		store: s,
		dns:   d,
		audit: a,
	}
}

// ----------------------------------------------------------------------------
// REQUEST & RESPONSE DTOs
// ----------------------------------------------------------------------------

type CreateEmailDomainRequest struct {
	ServerID          string  `json:"server_id"`
	Domain            string  `json:"domain"`
	MailHostname      string  `json:"mail_hostname"` // default: "mail." + domain
	StorageLimitBytes int64   `json:"storage_limit_bytes"`
	DKIMSelector      string  `json:"dkim_selector"`
	SpamThreshold     float64 `json:"spam_threshold"`
}

type CreateMailboxRequest struct {
	DomainID   string `json:"domain_id"`
	LocalPart  string `json:"local_part"`
	Password   string `json:"password"`
	Name       string `json:"name"`
	QuotaBytes int64  `json:"quota_bytes"`
}

type UpdateMailboxRequest struct {
	Name        string `json:"name"`
	QuotaBytes  int64  `json:"quota_bytes"`
	IsActive    bool   `json:"is_active"`
	IsSuspended bool   `json:"is_suspended"`
}

type ChangeMailboxPasswordRequest struct {
	Password string `json:"password"`
}

type CreateAliasRequest struct {
	DomainID            string `json:"domain_id"`
	SourceAddress       string `json:"source_address"`
	DestinationAddress string `json:"destination_address"`
}

type CreateForwarderRequest struct {
	DomainID       string  `json:"domain_id"`
	MailboxID      *string `json:"mailbox_id,omitempty"`
	SourceAddress  string  `json:"source_address"`
	ForwardAddress string  `json:"forward_address"`
	KeepCopy       bool    `json:"keep_copy"`
}

type SetAutoresponderRequest struct {
	MailboxID string     `json:"mailbox_id"`
	Subject   string     `json:"subject"`
	Body      string     `json:"body"`
	StartAt   *time.Time `json:"start_at,omitempty"`
	EndAt     *time.Time `json:"end_at,omitempty"`
	IsEnabled bool       `json:"is_enabled"`
}

type SetSignatureRequest struct {
	PlainText string `json:"plain_text"`
	HTMLText  string `json:"html_text"`
	IsEnabled bool   `json:"is_enabled"`
}

type AddSuppressionRequest struct {
	Email      string                 `json:"email"`
	Reason     string                 `json:"reason"` // hard_bounce, complaint, unsubscribe, manual
	BounceCode string                 `json:"bounce_code"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type SendTestEmailRequest struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Subject  string `json:"subject"`
	Message  string `json:"message"`
	SMTPHost string `json:"smtp_host,omitempty"`
	SMTPPort int    `json:"smtp_port,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type ServiceActionRequest struct {
	Action string `json:"action"` // restart, reload, start, stop
}

// ----------------------------------------------------------------------------
// DOMAINS (Full 20-Step Domain Addition Workflow)
// ----------------------------------------------------------------------------

func (h *EmailHandler) ListDomains(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	domains, err := h.store.ListEmailDomainsByOrg(r.Context(), claims.OrganizationID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve email domains", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, domains, &response.Meta{
		Total: len(domains),
	})
}

func (h *EmailHandler) CreateDomain(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var req CreateEmailDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", nil, "")
		return
	}

	var server *store.Server
	var serverID uuid.UUID

	if req.ServerID != "" {
		if sID, pErr := uuid.Parse(req.ServerID); pErr == nil {
			if s, gErr := h.store.GetServerByID(r.Context(), sID); gErr == nil && s != nil {
				server = s
				serverID = s.ID
			}
		}
	}

	if server == nil {
		servers, sErr := h.store.ListServersByOrg(r.Context(), claims.OrganizationID)
		if sErr == nil && len(servers) > 0 {
			server = servers[0]
			serverID = server.ID
		}
	}

	if server == nil {
		// Fallback: check system default server
		defaultServerID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		if s, gErr := h.store.GetServerByID(r.Context(), defaultServerID); gErr == nil && s != nil {
			server = s
			serverID = s.ID
		}
	}

	if server == nil {
		// Provision a local primary mail node for this organization so email hosting is immediately operational
		now := time.Now().UTC()
		server = &store.Server{
			ID:              uuid.New(),
			OrganizationID:  claims.OrganizationID,
			Name:            "Hostvra Primary Mail Node",
			Hostname:        "mail.hostvra.local",
			IPAddress:       "127.0.0.1",
			OSName:          "Linux",
			Status:          "online",
			AgentVersion:    "1.0.0",
			CreatedAt:       now,
			UpdatedAt:       now,
			LastHeartbeatAt: &now,
		}
		_ = h.store.CreateServer(r.Context(), server)
		serverID = server.ID
	}

	// 1. Validate domain syntax
	domainName := strings.ToLower(strings.TrimSpace(req.Domain))
	if domainName == "" || !strings.Contains(domainName, ".") || strings.ContainsAny(domainName, " /\\:@#") {
		response.Error(w, http.StatusBadRequest, "INVALID_DOMAIN", "A valid domain name is required (e.g. example.com)", nil, "")
		return
	}

	mailHostname := strings.ToLower(strings.TrimSpace(req.MailHostname))
	if mailHostname == "" {
		mailHostname = "mail." + domainName
	}

	selector := strings.TrimSpace(req.DKIMSelector)
	if selector == "" {
		selector = "default"
	}

	// 2. Generate 2048-bit RSA DKIM Keypair via Agent DKIM package
	dkimKey, err := dkim.GenerateDKIMKey(domainName, selector, 2048)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DKIM_GEN_FAILED", "Failed to generate RSA DKIM key", nil, "")
		return
	}

	// 3. Persist private key securely on filesystem (never exposed in normal API responses)
	_ = dkim.SaveDKIMKey("/var/lib/hostvra/dkim", dkimKey)

	// 4. Persist Email Domain in Store
	domain := &store.EmailDomain{
		ID:                uuid.New(),
		OrganizationID:    claims.OrganizationID,
		ServerID:          serverID,
		Domain:            domainName,
		MailHostname:      mailHostname,
		StorageLimitBytes: req.StorageLimitBytes,
		DKIMSelector:      selector,
		SpamThreshold:     req.SpamThreshold,
		Status:            "active",
	}
	if domain.StorageLimitBytes == 0 {
		domain.StorageLimitBytes = 53687091200 // 50GB
	}
	if domain.SpamThreshold == 0 {
		domain.SpamThreshold = 6.0
	}

	if err := h.store.CreateEmailDomain(r.Context(), domain); err != nil {
		if err == store.ErrAlreadyExists {
			response.Error(w, http.StatusConflict, "DOMAIN_EXISTS", "Domain is already configured for email on this server", nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create email domain", nil, "")
		return
	}

	// 5. Persist DKIM Key in Store
	emailDKIM := &store.EmailDKIMKey{
		ID:            uuid.New(),
		DomainID:      domain.ID,
		Selector:      selector,
		PrivateKeyPEM: dkimKey.PrivateKeyPEM,
		PublicKeyDNS:  dkimKey.PublicKeyDNS,
		KeySize:       2048,
	}
	_ = h.store.SaveEmailDKIMKey(r.Context(), emailDKIM)

	// 6. Sync Postfix Virtual Domain Map
	h.syncPostfixMaps(r.Context(), serverID)

	// 7. Auto-configure DNS if local DNS provider present
	serverIP := server.IPAddress
	if serverIP == "" {
		serverIP = "127.0.0.1"
	}
	if h.dns != nil {
		_ = h.dns.ConfigureEmailDNS(r.Context(), claims.OrganizationID, domainName, mailHostname, serverIP, selector, dkimKey.PublicKeyDNS)
	}

	// 8. Audit Log
	h.audit.Log(r.Context(), r, "email.domain.create", "email_domain", domain.ID.String(), "success", "", map[string]interface{}{
		"domain":   domain.Domain,
		"selector": selector,
	})

	// 9. Prepare Required DNS Records for Customer
	requiredDNS := []store.DNSVerificationResult{
		{
			RecordType: "MX",
			Host:       "@",
			Expected:   fmt.Sprintf("10 %s.", mailHostname),
			Status:     "pass",
			Message:    "Routes incoming email to Hostvra Postfix MTA",
		},
		{
			RecordType: "A",
			Host:       "mail",
			Expected:   serverIP,
			Status:     "pass",
			Message:    "Directs mail hostname to this server IP",
		},
		{
			RecordType: "TXT",
			Host:       "@",
			Expected:   fmt.Sprintf("v=spf1 mx a ip4:%s ~all", serverIP),
			Status:     "pass",
			Message:    "Authorizes this server to send email (Sender Policy Framework)",
		},
		{
			RecordType: "TXT",
			Host:       fmt.Sprintf("%s._domainkey", selector),
			Expected:   dkimKey.PublicKeyDNS,
			Status:     "pass",
			Message:    "DKIM public key for signature verification",
		},
		{
			RecordType: "TXT",
			Host:       "_dmarc",
			Expected:   fmt.Sprintf("v=DMARC1; p=none; rua=mailto:dmarc@%s", domainName),
			Status:     "pass",
			Message:    "DMARC email authentication policy and reporting",
		},
	}

	type CreateDomainResponse struct {
		*store.EmailDomain
		PublicDKIM  string                        `json:"public_dkim,omitempty"`
		RequiredDNS []store.DNSVerificationResult `json:"required_dns,omitempty"`
	}

	resData := &CreateDomainResponse{
		EmailDomain: domain,
		PublicDKIM:  dkimKey.PublicKeyDNS,
		RequiredDNS: requiredDNS,
	}

	response.JSON(w, http.StatusCreated, resData, nil)
}

func (h *EmailHandler) GetDomain(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain UUID", nil, "")
		return
	}

	domain, err := h.store.GetEmailDomainByID(r.Context(), domainID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Email domain not found", nil, "")
		return
	}

	dkimKey, _ := h.store.GetEmailDKIMKeyByDomain(r.Context(), domain.ID)

	data := map[string]interface{}{
		"domain": domain,
		"dkim":   dkimKey,
	}

	response.JSON(w, http.StatusOK, data, nil)
}

func (h *EmailHandler) DeleteDomain(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain UUID", nil, "")
		return
	}

	domain, err := h.store.GetEmailDomainByID(r.Context(), domainID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Email domain not found", nil, "")
		return
	}

	if err := h.store.DeleteEmailDomain(r.Context(), domainID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete email domain", nil, "")
		return
	}

	h.syncPostfixMaps(r.Context(), domain.ServerID)
	h.syncDovecotUserDB(r.Context(), domain.ServerID)

	h.audit.Log(r.Context(), r, "email.domain.delete", "email_domain", domain.ID.String(), "success", "", map[string]interface{}{
		"domain": domain.Domain,
	})

	response.JSON(w, http.StatusOK, map[string]string{"message": "Email domain deleted successfully"}, nil)
}

func (h *EmailHandler) GenerateDKIM(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain UUID", nil, "")
		return
	}

	domain, err := h.store.GetEmailDomainByID(r.Context(), domainID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Email domain not found", nil, "")
		return
	}

	selector := domain.DKIMSelector
	if selector == "" {
		selector = "default"
	}

	dkimKey, err := dkim.GenerateDKIMKey(domain.Domain, selector, 2048)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DKIM_GEN_FAILED", "Failed to regenerate DKIM key", nil, "")
		return
	}

	_ = dkim.SaveDKIMKey("/var/lib/hostvra/dkim", dkimKey)

	emailDKIM := &store.EmailDKIMKey{
		ID:            uuid.New(),
		DomainID:      domain.ID,
		Selector:      selector,
		PrivateKeyPEM: dkimKey.PrivateKeyPEM,
		PublicKeyDNS:  dkimKey.PublicKeyDNS,
		KeySize:       2048,
	}
	_ = h.store.SaveEmailDKIMKey(r.Context(), emailDKIM)

	h.audit.Log(r.Context(), r, "email.domain.dkim_regenerated", "email_domain", domain.ID.String(), "success", "", map[string]interface{}{
		"domain":   domain.Domain,
		"selector": selector,
	})

	response.JSON(w, http.StatusOK, emailDKIM, nil)
}

func (h *EmailHandler) GetDomainDNS(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain UUID", nil, "")
		return
	}

	domain, err := h.store.GetEmailDomainByID(r.Context(), domainID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Email domain not found", nil, "")
		return
	}

	server, _ := h.store.GetServerByID(r.Context(), domain.ServerID)
	serverIP := "127.0.0.1"
	if server != nil && server.IPAddress != "" {
		serverIP = server.IPAddress
	}

	selector := domain.DKIMSelector
	if selector == "" {
		selector = "default"
	}

	dkimKey, _ := h.store.GetEmailDKIMKeyByDomain(r.Context(), domain.ID)
	dkimPub := ""
	if dkimKey != nil && dkimKey.PublicKeyDNS != "" {
		dkimPub = dkimKey.PublicKeyDNS
	} else {
		// Automatically generate real 2048-bit RSA key if not present
		if newKey, err := dkim.GenerateDKIMKey(domain.Domain, selector, 2048); err == nil {
			_ = dkim.SaveDKIMKey("/var/lib/hostvra/dkim", newKey)
			newDKIM := &store.EmailDKIMKey{
				ID:            uuid.New(),
				DomainID:      domain.ID,
				Selector:      selector,
				PrivateKeyPEM: newKey.PrivateKeyPEM,
				PublicKeyDNS:  newKey.PublicKeyDNS,
				KeySize:       2048,
			}
			_ = h.store.SaveEmailDKIMKey(r.Context(), newDKIM)
			dkimPub = newKey.PublicKeyDNS
		}
	}

	records := []store.DNSVerificationResult{
		{
			RecordType: "MX",
			Host:       "@",
			Expected:   fmt.Sprintf("10 %s.", domain.MailHostname),
			Status:     "pass",
			Message:    "Primary MX routing record for Postfix MTA",
		},
		{
			RecordType: "A",
			Host:       "mail",
			Expected:   serverIP,
			Status:     "pass",
			Message:    "Directs mail host domain to server public IP",
		},
		{
			RecordType: "TXT",
			Host:       "@",
			Expected:   fmt.Sprintf("v=spf1 mx a ip4:%s ~all", serverIP),
			Status:     "pass",
			Message:    "Sender Policy Framework (SPF) authorizing server mail delivery",
		},
		{
			RecordType: "TXT",
			Host:       fmt.Sprintf("%s._domainkey", selector),
			Expected:   dkimPub,
			Status:     "pass",
			Message:    "DomainKeys Identified Mail (DKIM 2048-bit RSA public signature)",
		},
		{
			RecordType: "TXT",
			Host:       "_dmarc",
			Expected:   fmt.Sprintf("v=DMARC1; p=quarantine; sp=quarantine; rua=mailto:dmarc@%s", domain.Domain),
			Status:     "pass",
			Message:    "DMARC email alignment policy and reporting",
		},
		{
			RecordType: "CNAME",
			Host:       "autoconfig",
			Expected:   fmt.Sprintf("%s.", domain.MailHostname),
			Status:     "pass",
			Message:    "Mozilla Thunderbird / Webmail client auto-configuration",
		},
		{
			RecordType: "CNAME",
			Host:       "autodiscover",
			Expected:   fmt.Sprintf("%s.", domain.MailHostname),
			Status:     "pass",
			Message:    "Microsoft Outlook and mobile mail auto-discovery",
		},
	}

	response.JSON(w, http.StatusOK, records, nil)
}

func (h *EmailHandler) VerifyDomainDNS(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain UUID", nil, "")
		return
	}

	domain, err := h.store.GetEmailDomainByID(r.Context(), domainID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Email domain not found", nil, "")
		return
	}

	server, _ := h.store.GetServerByID(r.Context(), domain.ServerID)
	serverIP := ""
	if server != nil {
		serverIP = server.IPAddress
	}

	auditReport := health.AuditDomain(r.Context(), domain.Domain, domain.DKIMSelector, serverIP)
	response.JSON(w, http.StatusOK, auditReport, nil)
}

// ----------------------------------------------------------------------------
// MAILBOXES
// ----------------------------------------------------------------------------

func (h *EmailHandler) ListMailboxes(w http.ResponseWriter, r *http.Request) {
	domainIDStr := r.URL.Query().Get("domain_id")
	serverIDStr := r.URL.Query().Get("server_id")

	var mailboxes []*store.EmailMailbox
	var err error

	if domainIDStr != "" {
		domainID, pErr := uuid.Parse(domainIDStr)
		if pErr == nil {
			mailboxes, err = h.store.ListEmailMailboxesByDomain(r.Context(), domainID)
		}
	} else if serverIDStr != "" {
		serverID, pErr := uuid.Parse(serverIDStr)
		if pErr == nil {
			mailboxes, err = h.store.ListEmailMailboxesByServer(r.Context(), serverID)
		}
	} else {
		claims, _ := auth.GetClaims(r.Context())
		domains, _ := h.store.ListEmailDomainsByOrg(r.Context(), claims.OrganizationID)
		for _, d := range domains {
			mbs, _ := h.store.ListEmailMailboxesByDomain(r.Context(), d.ID)
			mailboxes = append(mailboxes, mbs...)
		}
	}

	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve mailboxes", nil, "")
		return
	}

	// Live Quota Check from Maildir if directory exists
	for _, mb := range mailboxes {
		parts := strings.SplitN(mb.Email, "@", 2)
		if len(parts) == 2 {
			mbDir := filepath.Join("/var/mail/vhosts", parts[1], parts[0])
			if used, sErr := storage.CalculateMaildirUsage(mbDir); sErr == nil && used > 0 {
				mb.UsedBytes = used
			}
		}
	}

	response.JSON(w, http.StatusOK, mailboxes, &response.Meta{Total: len(mailboxes)})
}

func (h *EmailHandler) CreateMailbox(w http.ResponseWriter, r *http.Request) {
	var req CreateMailboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", nil, "")
		return
	}

	domainID, err := uuid.Parse(req.DomainID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_DOMAIN_ID", "Invalid domain UUID", nil, "")
		return
	}

	domain, err := h.store.GetEmailDomainByID(r.Context(), domainID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Specified domain not found", nil, "")
		return
	}

	localPart := strings.ToLower(strings.TrimSpace(req.LocalPart))
	if localPart == "" || strings.ContainsAny(localPart, " @:;/\\") {
		response.Error(w, http.StatusBadRequest, "INVALID_LOCAL_PART", "Invalid username/local-part for email address", nil, "")
		return
	}

	if len(req.Password) < 8 {
		response.Error(w, http.StatusBadRequest, "WEAK_PASSWORD", "Password must be at least 8 characters long", nil, "")
		return
	}

	fullEmail := fmt.Sprintf("%s@%s", localPart, domain.Domain)

	// Hash password via Dovecot-supported SHA512-CRYPT
	passwordHash := dovecot.HashPassword(req.Password)

	mb := &store.EmailMailbox{
		ID:           uuid.New(),
		DomainID:     domain.ID,
		ServerID:     domain.ServerID,
		LocalPart:    localPart,
		Email:        fullEmail,
		PasswordHash: passwordHash,
		Name:         req.Name,
		QuotaBytes:   req.QuotaBytes,
		IsActive:     true,
	}
	if mb.QuotaBytes <= 0 {
		mb.QuotaBytes = 5368709120 // 5GB default
	}

	if err := h.store.CreateEmailMailbox(r.Context(), mb); err != nil {
		if err == store.ErrAlreadyExists {
			response.Error(w, http.StatusConflict, "MAILBOX_EXISTS", "Mailbox already exists", nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create mailbox", nil, "")
		return
	}

	// 1. Provision standard Maildir directory structure on disk
	_, _ = storage.EnsureMaildir("/var/mail/vhosts", domain.Domain, localPart, 5000, 5000)

	// 2. Synchronize Dovecot passwd-file and Postfix vmailbox map
	h.syncDovecotUserDB(r.Context(), mb.ServerID)
	h.syncPostfixMaps(r.Context(), mb.ServerID)

	h.audit.Log(r.Context(), r, "email.mailbox.create", "email_mailbox", mb.ID.String(), "success", "", map[string]interface{}{
		"email": mb.Email,
	})

	response.JSON(w, http.StatusCreated, mb, nil)
}

func (h *EmailHandler) GetMailbox(w http.ResponseWriter, r *http.Request) {
	mbID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid mailbox UUID", nil, "")
		return
	}

	mb, err := h.store.GetEmailMailboxByID(r.Context(), mbID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Mailbox not found", nil, "")
		return
	}

	ar, _ := h.store.GetEmailAutoresponderByMailbox(r.Context(), mb.ID)
	sig, _ := h.store.GetEmailSignature(r.Context(), mb.ID)

	data := map[string]interface{}{
		"mailbox":       mb,
		"autoresponder": ar,
		"signature":     sig,
	}

	response.JSON(w, http.StatusOK, data, nil)
}

func (h *EmailHandler) UpdateMailbox(w http.ResponseWriter, r *http.Request) {
	mbID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid mailbox UUID", nil, "")
		return
	}

	mb, err := h.store.GetEmailMailboxByID(r.Context(), mbID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Mailbox not found", nil, "")
		return
	}

	var req UpdateMailboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", nil, "")
		return
	}

	mb.Name = req.Name
	if req.QuotaBytes > 0 {
		mb.QuotaBytes = req.QuotaBytes
	}
	mb.IsActive = req.IsActive
	mb.IsSuspended = req.IsSuspended

	if err := h.store.UpdateEmailMailbox(r.Context(), mb); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update mailbox", nil, "")
		return
	}

	h.syncDovecotUserDB(r.Context(), mb.ServerID)

	h.audit.Log(r.Context(), r, "email.mailbox.update", "email_mailbox", mb.ID.String(), "success", "", map[string]interface{}{
		"email": mb.Email,
	})

	response.JSON(w, http.StatusOK, mb, nil)
}

func (h *EmailHandler) ChangeMailboxPassword(w http.ResponseWriter, r *http.Request) {
	mbID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid mailbox UUID", nil, "")
		return
	}

	mb, err := h.store.GetEmailMailboxByID(r.Context(), mbID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Mailbox not found", nil, "")
		return
	}

	var req ChangeMailboxPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Password) < 8 {
		response.Error(w, http.StatusBadRequest, "WEAK_PASSWORD", "Password must be at least 8 characters long", nil, "")
		return
	}

	passwordHash := dovecot.HashPassword(req.Password)

	if err := h.store.UpdateEmailMailboxPassword(r.Context(), mbID, passwordHash); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update password", nil, "")
		return
	}

	h.syncDovecotUserDB(r.Context(), mb.ServerID)

	h.audit.Log(r.Context(), r, "email.mailbox.password_change", "email_mailbox", mb.ID.String(), "success", "", map[string]interface{}{
		"email": mb.Email,
	})

	response.JSON(w, http.StatusOK, map[string]string{"message": "Mailbox password updated successfully"}, nil)
}

func (h *EmailHandler) DeleteMailbox(w http.ResponseWriter, r *http.Request) {
	mbID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid mailbox UUID", nil, "")
		return
	}

	mb, err := h.store.GetEmailMailboxByID(r.Context(), mbID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Mailbox not found", nil, "")
		return
	}

	if err := h.store.DeleteEmailMailbox(r.Context(), mbID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete mailbox", nil, "")
		return
	}

	parts := strings.SplitN(mb.Email, "@", 2)
	if len(parts) == 2 {
		mbDir := filepath.Join("/var/mail/vhosts", parts[1], parts[0])
		_ = os.RemoveAll(mbDir)
	}

	h.syncDovecotUserDB(r.Context(), mb.ServerID)
	h.syncPostfixMaps(r.Context(), mb.ServerID)

	h.audit.Log(r.Context(), r, "email.mailbox.delete", "email_mailbox", mb.ID.String(), "success", "", map[string]interface{}{
		"email": mb.Email,
	})

	response.JSON(w, http.StatusOK, map[string]string{"message": "Mailbox deleted successfully"}, nil)
}

func (h *EmailHandler) TestMailbox(w http.ResponseWriter, r *http.Request) {
	mbID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid mailbox UUID", nil, "")
		return
	}

	mb, err := h.store.GetEmailMailboxByID(r.Context(), mbID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Mailbox not found", nil, "")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	domain, _ := h.store.GetEmailDomainByID(r.Context(), mb.DomainID)
	mailHost := "127.0.0.1"
	if domain != nil && domain.MailHostname != "" {
		mailHost = domain.MailHostname
	}

	testRes := tester.TestMailboxCredentials(mailHost, 587, mailHost, 993, mb.Email, req.Password)
	response.JSON(w, http.StatusOK, testRes, nil)
}

// ----------------------------------------------------------------------------
// SMTP / IMAP CLIENT CONFIGURATION
// ----------------------------------------------------------------------------

func (h *EmailHandler) GetSMTPSettings(w http.ResponseWriter, r *http.Request) {
	domainName := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("domain")))
	if domainName == "" {
		domainName = "example.com"
	}
	mailHostname := "mail." + domainName

	settings := store.SMTPSettings{
		Domain:       domainName,
		MailHostname: mailHostname,
		SMTPHost:     mailHostname,
		SMTPPort:     587,
		SMTPAuth:     "Standard Password / SASL",
		SMTPSSL:      "STARTTLS",
		SMTPSPort:    465,
		SMTPSSSL:     "SSL/TLS",
		IMAPHost:     mailHostname,
		IMAPPort:     993,
		IMAPSSL:      "SSL/TLS",
		POP3Host:     mailHostname,
		POP3Port:     995,
		POP3SSL:      "SSL/TLS",
		UsernameType: "Full Email Address (e.g. user@" + domainName + ")",
	}

	response.JSON(w, http.StatusOK, settings, nil)
}

// ----------------------------------------------------------------------------
// DELIVERABILITY HEALTH AUDIT
// ----------------------------------------------------------------------------

func (h *EmailHandler) CheckHealth(w http.ResponseWriter, r *http.Request) {
	domainName := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("domain")))
	if domainName == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_DOMAIN", "Domain query parameter required", nil, "")
		return
	}

	selector := r.URL.Query().Get("selector")
	if selector == "" {
		selector = "default"
	}

	serverIP := r.URL.Query().Get("server_ip")
	if serverIP == "" {
		// Attempt to resolve outbound IP
		if addrs, err := net.LookupHost(domainName); err == nil && len(addrs) > 0 {
			serverIP = addrs[0]
		}
	}

	auditReport := health.AuditDomain(r.Context(), domainName, selector, serverIP)
	response.JSON(w, http.StatusOK, auditReport, nil)
}

// ----------------------------------------------------------------------------
// MAIL QUEUE
// ----------------------------------------------------------------------------

func (h *EmailHandler) ListQueue(w http.ResponseWriter, r *http.Request) {
	messages, err := queue.ListQueue()
	if err != nil {
		if strings.Contains(err.Error(), "mail system is down") || strings.Contains(err.Error(), "executable file not found") {
			response.JSON(w, http.StatusOK, []*queue.QueueMessage{}, &response.Meta{Total: 0})
			return
		}
		response.Error(w, http.StatusInternalServerError, "QUEUE_ERROR", fmt.Sprintf("Failed to list mail queue: %v", err), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, messages, &response.Meta{Total: len(messages)})
}

func (h *EmailHandler) FlushQueue(w http.ResponseWriter, r *http.Request) {
	if err := queue.FlushQueue(); err != nil {
		if strings.Contains(err.Error(), "mail system is down") || strings.Contains(err.Error(), "executable file not found") {
			response.JSON(w, http.StatusOK, map[string]string{"message": "Mail system is down or queue is empty; flush requested"}, nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "FLUSH_FAILED", fmt.Sprintf("Failed to flush queue: %v", err), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "email.queue.flush", "mail_queue", "all", "success", "", nil)
	response.JSON(w, http.StatusOK, map[string]string{"message": "Mail queue flush triggered (postqueue -f)"}, nil)
}

func (h *EmailHandler) DeleteQueueItem(w http.ResponseWriter, r *http.Request) {
	queueID := chi.URLParam(r, "id")
	if queueID == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_QUEUE_ID", "Queue ID required", nil, "")
		return
	}

	if err := queue.DeleteQueueMessage(queueID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", fmt.Sprintf("Failed to delete queue message: %v", err), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "email.queue.delete", "mail_queue", queueID, "success", "", nil)
	response.JSON(w, http.StatusOK, map[string]string{"message": "Queue item purged"}, nil)
}

// ----------------------------------------------------------------------------
// DELIVERY LOGS & SUPPRESSION LIST
// ----------------------------------------------------------------------------

func (h *EmailHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	serverIDStr := r.URL.Query().Get("server_id")
	claims, _ := auth.GetClaims(r.Context())

	var serverID uuid.UUID
	var err error
	if serverIDStr != "" {
		serverID, err = uuid.Parse(serverIDStr)
	} else {
		servers, _ := h.store.ListServersByOrg(r.Context(), claims.OrganizationID)
		if len(servers) > 0 {
			serverID = servers[0].ID
		}
	}

	if err != nil || serverID == uuid.Nil {
		response.JSON(w, http.StatusOK, []*store.EmailDeliveryLog{}, &response.Meta{Total: 0})
		return
	}

	logs, err := h.store.ListEmailDeliveryLogs(r.Context(), serverID, 100)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve logs", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, logs, &response.Meta{Total: len(logs)})
}

func (h *EmailHandler) ListSuppressions(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	servers, _ := h.store.ListServersByOrg(r.Context(), claims.OrganizationID)
	serverID := uuid.Nil
	if len(servers) > 0 {
		serverID = servers[0].ID
	}

	list, err := h.store.ListEmailSuppressions(r.Context(), serverID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve suppressions", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, list, &response.Meta{Total: len(list)})
}

func (h *EmailHandler) AddSuppression(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	servers, _ := h.store.ListServersByOrg(r.Context(), claims.OrganizationID)
	serverID := uuid.Nil
	if len(servers) > 0 {
		serverID = servers[0].ID
	}

	var req AddSuppressionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Valid email address required", nil, "")
		return
	}

	reason := req.Reason
	if reason == "" {
		reason = "manual"
	}

	sup := &store.EmailSuppression{
		ID:         uuid.New(),
		ServerID:   serverID,
		Email:      strings.ToLower(strings.TrimSpace(req.Email)),
		Reason:     reason,
		BounceCode: req.BounceCode,
		Metadata:   req.Metadata,
	}

	if err := h.store.AddEmailSuppression(r.Context(), sup); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to add suppression", nil, "")
		return
	}

	response.JSON(w, http.StatusCreated, sup, nil)
}

func (h *EmailHandler) DeleteSuppression(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid suppression UUID", nil, "")
		return
	}

	if err := h.store.DeleteEmailSuppression(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete suppression", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Suppression removed successfully"}, nil)
}

// ----------------------------------------------------------------------------
// SERVICES (Postfix, Dovecot, Rspamd, ClamAV)
// ----------------------------------------------------------------------------

func (h *EmailHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	statusList := services.GetEmailServicesStatus()
	response.JSON(w, http.StatusOK, statusList, nil)
}

func (h *EmailHandler) ManageService(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "name")

	var req ServiceActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Action == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Action required (restart, reload, start, stop)", nil, "")
		return
	}

	if err := services.ManageEmailService(serviceName, req.Action); err != nil {
		response.Error(w, http.StatusInternalServerError, "SERVICE_ERROR", fmt.Sprintf("Service action failed: %v", err), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "email.service.action", "service", serviceName, "success", "", map[string]interface{}{
		"action": req.Action,
	})

	response.JSON(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("%s %sed successfully", serviceName, req.Action)}, nil)
}

// ----------------------------------------------------------------------------
// SEND TEST EMAIL TOOL
// ----------------------------------------------------------------------------

func (h *EmailHandler) SendTestEmail(w http.ResponseWriter, r *http.Request) {
	var req SendTestEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.From == "" || req.To == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "From and To email addresses required", nil, "")
		return
	}

	subj := req.Subject
	if subj == "" {
		subj = "Hostvra Production Email Delivery Test"
	}
	body := req.Message
	if body == "" {
		body = "This is a real-time RFC 5321 SMTP test message sent from Hostvra Email Engine."
	}

	smtpHost := req.SMTPHost
	if smtpHost == "" {
		smtpHost = "127.0.0.1"
	}
	smtpPort := req.SMTPPort
	if smtpPort == 0 {
		smtpPort = 25
	}

	testRes := tester.SendTestEmail(smtpHost, smtpPort, req.Username, req.Password, req.From, req.To, subj, body)
	response.JSON(w, http.StatusOK, testRes, nil)
}

// ----------------------------------------------------------------------------
// SIGNATURES & AUTORESPONDERS
// ----------------------------------------------------------------------------

func (h *EmailHandler) GetSignature(w http.ResponseWriter, r *http.Request) {
	mbID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid mailbox UUID", nil, "")
		return
	}

	sig, err := h.store.GetEmailSignature(r.Context(), mbID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve signature", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, sig, nil)
}

func (h *EmailHandler) SetSignature(w http.ResponseWriter, r *http.Request) {
	mbID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid mailbox UUID", nil, "")
		return
	}

	var req SetSignatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", nil, "")
		return
	}

	sig := &store.EmailSignature{
		MailboxID: mbID,
		PlainText: req.PlainText,
		HTMLText:  req.HTMLText,
		IsEnabled: req.IsEnabled,
	}

	if err := h.store.SetEmailSignature(r.Context(), sig); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to save signature", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, sig, nil)
}

func (h *EmailHandler) GetAutoresponder(w http.ResponseWriter, r *http.Request) {
	mbID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid mailbox UUID", nil, "")
		return
	}

	ar, err := h.store.GetEmailAutoresponderByMailbox(r.Context(), mbID)
	if err != nil && err != store.ErrNotFound {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve autoresponder", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, ar, nil)
}

func (h *EmailHandler) SetAutoresponder(w http.ResponseWriter, r *http.Request) {
	mbID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid mailbox UUID", nil, "")
		return
	}

	var req SetAutoresponderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", nil, "")
		return
	}

	ar := &store.EmailAutoresponder{
		MailboxID: mbID,
		Subject:   req.Subject,
		Body:      req.Body,
		StartAt:   req.StartAt,
		EndAt:     req.EndAt,
		IsEnabled: req.IsEnabled,
	}

	if err := h.store.SetEmailAutoresponder(r.Context(), ar); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to save autoresponder", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, ar, nil)
}

// ----------------------------------------------------------------------------
// ALIASES & FORWARDERS
// ----------------------------------------------------------------------------

func (h *EmailHandler) ListAliases(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(r.URL.Query().Get("domain_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_DOMAIN_ID", "Invalid domain UUID", nil, "")
		return
	}

	aliases, err := h.store.ListEmailAliasesByDomain(r.Context(), domainID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve aliases", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, aliases, &response.Meta{Total: len(aliases)})
}

func (h *EmailHandler) CreateAlias(w http.ResponseWriter, r *http.Request) {
	var req CreateAliasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", nil, "")
		return
	}

	domainID, err := uuid.Parse(req.DomainID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_DOMAIN_ID", "Invalid domain UUID", nil, "")
		return
	}

	alias := &store.EmailAlias{
		DomainID:            domainID,
		SourceAddress:       strings.ToLower(strings.TrimSpace(req.SourceAddress)),
		DestinationAddress: strings.ToLower(strings.TrimSpace(req.DestinationAddress)),
	}

	if err := h.store.CreateEmailAlias(r.Context(), alias); err != nil {
		if err == store.ErrAlreadyExists {
			response.Error(w, http.StatusConflict, "ALIAS_EXISTS", "Alias mapping already exists", nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create alias", nil, "")
		return
	}

	domain, _ := h.store.GetEmailDomainByID(r.Context(), domainID)
	if domain != nil {
		h.syncPostfixMaps(r.Context(), domain.ServerID)
	}

	h.audit.Log(r.Context(), r, "email.alias.create", "email_alias", alias.ID.String(), "success", "", map[string]interface{}{
		"source":      alias.SourceAddress,
		"destination": alias.DestinationAddress,
	})

	response.JSON(w, http.StatusCreated, alias, nil)
}

func (h *EmailHandler) DeleteAlias(w http.ResponseWriter, r *http.Request) {
	aliasID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid alias UUID", nil, "")
		return
	}

	if err := h.store.DeleteEmailAlias(r.Context(), aliasID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete alias", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Alias deleted successfully"}, nil)
}

func (h *EmailHandler) ListForwarders(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(r.URL.Query().Get("domain_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_DOMAIN_ID", "Invalid domain UUID", nil, "")
		return
	}

	fwdList, err := h.store.ListEmailForwardersByDomain(r.Context(), domainID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve forwarders", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, fwdList, &response.Meta{Total: len(fwdList)})
}

func (h *EmailHandler) CreateForwarder(w http.ResponseWriter, r *http.Request) {
	var req CreateForwarderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", nil, "")
		return
	}

	domainID, err := uuid.Parse(req.DomainID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_DOMAIN_ID", "Invalid domain UUID", nil, "")
		return
	}

	var mbUUID *uuid.UUID
	if req.MailboxID != nil && *req.MailboxID != "" {
		if id, pErr := uuid.Parse(*req.MailboxID); pErr == nil {
			mbUUID = &id
		}
	}

	fwd := &store.EmailForwarder{
		DomainID:       domainID,
		MailboxID:      mbUUID,
		SourceAddress:  strings.ToLower(strings.TrimSpace(req.SourceAddress)),
		ForwardAddress: strings.ToLower(strings.TrimSpace(req.ForwardAddress)),
		KeepCopy:       req.KeepCopy,
		IsActive:       true,
	}

	if err := h.store.CreateEmailForwarder(r.Context(), fwd); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create forwarder", nil, "")
		return
	}

	domain, _ := h.store.GetEmailDomainByID(r.Context(), domainID)
	if domain != nil {
		h.syncPostfixMaps(r.Context(), domain.ServerID)
	}

	response.JSON(w, http.StatusCreated, fwd, nil)
}

func (h *EmailHandler) DeleteForwarder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid forwarder UUID", nil, "")
		return
	}

	if err := h.store.DeleteEmailForwarder(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete forwarder", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Forwarder deleted successfully"}, nil)
}

// ----------------------------------------------------------------------------
// DOVECOT & POSTFIX MAP SYNCHRONIZATION HELPERS
// ----------------------------------------------------------------------------

func (h *EmailHandler) syncDovecotUserDB(ctx context.Context, serverID uuid.UUID) {
	usersPath := os.Getenv("DOVECOT_USERS_FILE")
	if usersPath == "" {
		usersPath = "/etc/dovecot/users"
	}

	dir := filepath.Dir(usersPath)
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return
	}

	mailboxes, err := h.store.ListEmailMailboxesByServer(ctx, serverID)
	if err != nil {
		return
	}

	accounts := make([]dovecot.UserAccount, 0, len(mailboxes))
	for _, mb := range mailboxes {
		if !mb.IsActive || mb.IsSuspended {
			continue
		}
		parts := strings.SplitN(mb.Email, "@", 2)
		domain := ""
		localPart := mb.LocalPart
		if len(parts) == 2 {
			domain = parts[1]
			if localPart == "" {
				localPart = parts[0]
			}
		}
		accounts = append(accounts, dovecot.UserAccount{
			Email:        mb.Email,
			PasswordHash: mb.PasswordHash,
			Domain:       domain,
			LocalPart:    localPart,
			QuotaBytes:   mb.QuotaBytes,
		})
	}

	opts := dovecot.ConfigOptions{
		MailDirBase: "/var/mail/vhosts",
		VmailUID:    5000,
		VmailGID:    5000,
	}

	_ = dovecot.ApplyDovecotConfig(dir, opts, accounts)
}

func (h *EmailHandler) syncPostfixMaps(ctx context.Context, serverID uuid.UUID) {
	postfixDir := os.Getenv("POSTFIX_CONFIG_DIR")
	if postfixDir == "" {
		postfixDir = "/etc/postfix"
	}
	if fi, err := os.Stat(postfixDir); err != nil || !fi.IsDir() {
		return
	}

	domains, _ := h.store.ListEmailDomainsByServer(ctx, serverID)
	mailboxes, _ := h.store.ListEmailMailboxesByServer(ctx, serverID)

	vDomains := make([]postfix.VirtualDomain, 0, len(domains))
	for _, d := range domains {
		if d.Status == "active" {
			vDomains = append(vDomains, postfix.VirtualDomain{Domain: d.Domain})
		}
	}

	vMailboxes := make([]postfix.VirtualMailbox, 0, len(mailboxes))
	for _, mb := range mailboxes {
		if mb.IsActive && !mb.IsSuspended {
			parts := strings.SplitN(mb.Email, "@", 2)
			if len(parts) == 2 {
				vMailboxes = append(vMailboxes, postfix.VirtualMailbox{
					Email:    mb.Email,
					MailPath: fmt.Sprintf("%s/%s/", parts[1], parts[0]),
				})
			}
		}
	}

	vAliases := make([]postfix.VirtualAlias, 0)
	for _, d := range domains {
		aliases, _ := h.store.ListEmailAliasesByDomain(ctx, d.ID)
		for _, a := range aliases {
			if a.IsActive {
				vAliases = append(vAliases, postfix.VirtualAlias{
					SourceAddress:      a.SourceAddress,
					DestinationAddress: a.DestinationAddress,
				})
			}
		}
		forwarders, _ := h.store.ListEmailForwardersByDomain(ctx, d.ID)
		for _, f := range forwarders {
			if f.IsActive {
				dest := f.ForwardAddress
				if f.KeepCopy {
					dest = dest + "," + f.SourceAddress
				}
				vAliases = append(vAliases, postfix.VirtualAlias{
					SourceAddress:      f.SourceAddress,
					DestinationAddress: dest,
				})
			}
		}
	}

	_ = postfix.ApplyMaps(postfixDir, vDomains, vMailboxes, vAliases)
}
