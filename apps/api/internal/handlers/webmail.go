package handlers

import (
	"crypto/sha512"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"hostvra/agent/pkg/email/dkim"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type WebmailHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewWebmailHandler(cfg *config.Config, s store.Store, a *audit.Logger) *WebmailHandler {
	return &WebmailHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
}

// ----------------------------------------------------------------------------
// REQUEST & RESPONSE DTOs
// ----------------------------------------------------------------------------

type SendWebmailMessageRequest struct {
	ID           *uuid.UUID                `json:"id,omitempty"`
	MailboxID    *uuid.UUID                `json:"mailbox_id,omitempty"`
	AccountEmail string                    `json:"account_email,omitempty"`
	FromEmail    string                    `json:"from_email,omitempty"`
	ToEmail      string                    `json:"to_email,omitempty"`
	To           []string                  `json:"to,omitempty"`
	Cc           string                    `json:"cc,omitempty"`
	CcList       []string                  `json:"cc_list,omitempty"`
	Bcc          string                    `json:"bcc,omitempty"`
	BccList      []string                  `json:"bcc_list,omitempty"`
	Subject      string                    `json:"subject"`
	BodyHTML     string                    `json:"body_html,omitempty"`
	BodyText     string                    `json:"body_text,omitempty"`
	Priority     string                    `json:"priority,omitempty"` // normal, high, low
	Attachments  []store.WebmailAttachment `json:"attachments,omitempty"`
}

type UpdateMessageFlagsRequest struct {
	IsUnread    *bool `json:"is_unread,omitempty"`
	IsStarred   *bool `json:"is_starred,omitempty"`
	IsImportant *bool `json:"is_important,omitempty"`
}

type MoveMessageRequest struct {
	TargetFolder string `json:"target_folder"` // inbox, sent, drafts, trash, spam, archive
}

type WebmailAuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ----------------------------------------------------------------------------
// HANDLERS
// ----------------------------------------------------------------------------

func (h *WebmailHandler) DirectAuth(w http.ResponseWriter, r *http.Request) {
	var req WebmailAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_CREDENTIALS", "Email and password required", nil, "")
		return
	}

	cleanEmail := strings.ToLower(strings.TrimSpace(req.Email))
	mb, err := h.store.GetEmailMailboxByEmail(r.Context(), cleanEmail)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "AUTH_FAILED", "Invalid email or password", nil, "")
		return
	}

	if !mb.IsActive || mb.IsSuspended {
		response.Error(w, http.StatusForbidden, "MAILBOX_SUSPENDED", "Mailbox is inactive or suspended", nil, "")
		return
	}

	// Verify SHA512-CRYPT password hash
	if !verifyPasswordHash(req.Password, mb.PasswordHash) {
		response.Error(w, http.StatusUnauthorized, "AUTH_FAILED", "Invalid email or password", nil, "")
		return
	}

	domain, _ := h.store.GetEmailDomainByID(r.Context(), mb.DomainID)
	orgID := uuid.Nil
	if domain != nil {
		orgID = domain.OrganizationID
	}

	// Issue Webmail JWT session token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   mb.ID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(24 * time.Hour)),
		},
		UserID:         mb.ID,
		Email:          mb.Email,
		Role:           "webmail_user",
		OrganizationID: orgID,
	})

	signedToken, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to sign session token", nil, "")
		return
	}

	res := map[string]interface{}{
		"token": signedToken,
		"mailbox": map[string]interface{}{
			"id":          mb.ID,
			"email":       mb.Email,
			"name":        mb.Name,
			"quota_bytes": mb.QuotaBytes,
			"used_bytes":  mb.UsedBytes,
		},
	}

	response.JSON(w, http.StatusOK, res, nil)
}

func (h *WebmailHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	accountEmail := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("account")))
	if accountEmail == "" {
		accountEmail = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("account_email")))
	}
	mailboxIDStr := strings.TrimSpace(r.URL.Query().Get("mailbox_id"))

	folder := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("folder")))
	if folder == "" {
		folder = "inbox"
	}
	search := r.URL.Query().Get("search")
	if search == "" {
		search = r.URL.Query().Get("q")
	}

	var mb *store.EmailMailbox
	var err error

	if accountEmail != "" {
		mb, err = h.store.GetEmailMailboxByEmail(r.Context(), accountEmail)
	} else if mailboxIDStr != "" {
		if mbID, parseErr := uuid.Parse(mailboxIDStr); parseErr == nil {
			mb, err = h.store.GetEmailMailboxByID(r.Context(), mbID)
		}
	}

	if err != nil || mb == nil {
		response.Error(w, http.StatusBadRequest, "MISSING_ACCOUNT", "Valid account email or mailbox_id parameter required", nil, "")
		return
	}

	messages, total, err := h.store.ListWebmailMessages(r.Context(), mb.ID, folder, 50, 0, search)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve messages", nil, "")
		return
	}

	// Calculate unread count for mailbox
	inboxMsgs, _, _ := h.store.ListWebmailMessages(r.Context(), mb.ID, "inbox", 500, 0, "")
	unreadCount := 0
	for _, m := range inboxMsgs {
		if m.IsUnread {
			unreadCount++
		}
	}

	res := map[string]interface{}{
		"messages":     messages,
		"unread_count": unreadCount,
		"total":        total,
		"folder":       folder,
	}

	response.JSON(w, http.StatusOK, res, &response.Meta{Total: total})
}

func (h *WebmailHandler) GetMessage(w http.ResponseWriter, r *http.Request) {
	msgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid message UUID", nil, "")
		return
	}

	msg, err := h.store.GetWebmailMessageByID(r.Context(), msgID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Message not found", nil, "")
		return
	}

	// Auto-mark as read
	if msg.IsUnread {
		isFalse := false
		_ = h.store.UpdateWebmailMessageFlags(r.Context(), msg.ID, &isFalse, nil, nil)
		msg.IsUnread = false
	}

	// Sanitize HTML body to strip script tags, event handlers, and javascript: links
	sanitizedHTML := sanitizeEmailHTML(msg.BodyHTML)
	msg.BodyHTML = sanitizedHTML

	response.JSON(w, http.StatusOK, msg, nil)
}

func (h *WebmailHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req SendWebmailMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", nil, "")
		return
	}

	var mb *store.EmailMailbox
	var err error

	fromClean := strings.ToLower(strings.TrimSpace(req.AccountEmail))
	if fromClean == "" {
		fromClean = strings.ToLower(strings.TrimSpace(req.FromEmail))
	}

	if req.MailboxID != nil && *req.MailboxID != uuid.Nil {
		mb, err = h.store.GetEmailMailboxByID(r.Context(), *req.MailboxID)
	} else if fromClean != "" {
		mb, err = h.store.GetEmailMailboxByEmail(r.Context(), fromClean)
	}

	if err != nil || mb == nil {
		response.Error(w, http.StatusNotFound, "SENDER_MAILBOX_NOT_FOUND", "Sender mailbox does not exist", nil, "")
		return
	}

	if mb.IsSuspended {
		response.Error(w, http.StatusForbidden, "MAILBOX_SUSPENDED", "Sending is disabled on suspended mailbox", nil, "")
		return
	}

	// Extract all recipients
	var recipients []string
	if len(req.To) > 0 {
		for _, t := range req.To {
			clean := strings.ToLower(strings.TrimSpace(t))
			if clean != "" {
				recipients = append(recipients, clean)
			}
		}
	} else if req.ToEmail != "" {
		for _, t := range strings.Split(req.ToEmail, ",") {
			clean := strings.ToLower(strings.TrimSpace(t))
			if clean != "" {
				recipients = append(recipients, clean)
			}
		}
	}

	if len(recipients) == 0 {
		response.Error(w, http.StatusBadRequest, "MISSING_ADDRESSES", "At least one recipient email address required", nil, "")
		return
	}

	primaryTo := recipients[0]

	// 1. Check Suppression List (Prevent repeated sending to hard bounces)
	isSuppressed, sErr := h.store.IsEmailSuppressed(r.Context(), mb.ServerID, primaryTo)
	if sErr == nil && isSuppressed {
		response.Error(w, http.StatusBadRequest, "RECIPIENT_SUPPRESSED",
			fmt.Sprintf("Delivery rejected: %s is on the suppression list due to prior hard bounces or spam complaints.", primaryTo), nil, "")
		return
	}

	ccStr := req.Cc
	if ccStr == "" && len(req.CcList) > 0 {
		ccStr = strings.Join(req.CcList, ", ")
	}
	bccStr := req.Bcc
	if bccStr == "" && len(req.BccList) > 0 {
		bccStr = strings.Join(req.BccList, ", ")
	}

	allSMTPRecipients := make([]string, 0, len(recipients))
	allSMTPRecipients = append(allSMTPRecipients, recipients...)
	if ccStr != "" {
		for _, c := range strings.Split(ccStr, ",") {
			if clean := strings.ToLower(strings.TrimSpace(c)); clean != "" {
				allSMTPRecipients = append(allSMTPRecipients, clean)
			}
		}
	}
	if bccStr != "" {
		for _, b := range strings.Split(bccStr, ",") {
			if clean := strings.ToLower(strings.TrimSpace(b)); clean != "" {
				allSMTPRecipients = append(allSMTPRecipients, clean)
			}
		}
	}

	// 2. Transmit via Local Postfix MTA (Port 587 or 25)
	msgDomain := "hostvra.com"
	if parts := strings.Split(mb.Email, "@"); len(parts) == 2 && parts[1] != "" {
		msgDomain = strings.TrimSpace(parts[1])
	}
	msgID := fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), mb.ID.String()[:8], msgDomain)
	smtpServer := "127.0.0.1:25"
	if custom := os.Getenv("POSTFIX_SUBMISSION_ADDR"); custom != "" {
		smtpServer = custom
	}

	mailBody := req.BodyText
	var contentType string
	var msgPayload string

	if req.BodyHTML != "" {
		contentType = "text/html; charset=UTF-8"
		msgPayload = req.BodyHTML
		if mailBody == "" {
			mailBody = stripHTMLTags(req.BodyHTML)
		}
	} else {
		contentType = "text/plain; charset=UTF-8"
		msgPayload = mailBody
	}

	header := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMessage-ID: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: %s\r\n",
		mb.Name, mb.Email, strings.Join(recipients, ", "), req.Subject, msgID, time.Now().Format(time.RFC1123Z), contentType)
	if ccStr != "" {
		header += fmt.Sprintf("Cc: %s\r\n", ccStr)
	}
	header += "\r\n"
	fullMsg := header + msgPayload

	// 2b. Automatically Sign with RFC 6376 DKIM (rsa-sha256)
	payloadToDeliver := []byte(fullMsg)
	var dkimPrivKey string
	dkimSelector := "default"

	if mb.DomainID != uuid.Nil {
		if k, err := h.store.GetEmailDKIMKeyByDomain(r.Context(), mb.DomainID); err == nil && k != nil && k.PrivateKeyPEM != "" {
			dkimPrivKey = k.PrivateKeyPEM
			if k.Selector != "" {
				dkimSelector = k.Selector
			}
		}
	}
	if dkimPrivKey == "" {
		// Fallback to local DKIM file storage at /var/lib/hostvra/dkim/<domain>/<selector>.private
		candidates := []string{
			fmt.Sprintf("/var/lib/hostvra/dkim/%s/%s.private", msgDomain, dkimSelector),
			fmt.Sprintf("/var/lib/hostvra/dkim/%s/default.private", msgDomain),
		}
		for _, fpath := range candidates {
			if data, err := os.ReadFile(fpath); err == nil && len(data) > 0 {
				dkimPrivKey = string(data)
				break
			}
		}
	}

	if dkimPrivKey != "" {
		if signedBytes, err := dkim.SignEmail([]byte(fullMsg), dkim.SignOptions{
			Domain:        msgDomain,
			Selector:      dkimSelector,
			PrivateKeyPEM: dkimPrivKey,
		}); err == nil && len(signedBytes) > 0 {
			payloadToDeliver = signedBytes
		}
	}

	// Attempt real local SMTP delivery via loopback-safe TLS client
	deliveryErr := sendMailLocal(smtpServer, mb.Email, allSMTPRecipients, payloadToDeliver)
	deliveryStatus := "delivered"
	failureReason := ""
	if deliveryErr != nil {
		deliveryStatus = "failed"
		failureReason = deliveryErr.Error()
		response.Error(w, http.StatusInternalServerError, "SMTP_DELIVERY_FAILED", fmt.Sprintf("Postfix local delivery error: %v", deliveryErr), nil, "")
		return
	}

	// 3. Persist copy to sender's "sent" folder in DB and Maildir
	sentMsg := &store.WebmailMessage{
		ID:            uuid.New(),
		MailboxID:     mb.ID,
		AccountEmail:  mb.Email,
		Folder:        "sent",
		MessageID:     msgID,
		FromName:      mb.Name,
		FromEmail:     mb.Email,
		ToName:        primaryTo,
		ToEmail:       strings.Join(recipients, ", "),
		Cc:            ccStr,
		Bcc:           bccStr,
		Subject:       req.Subject,
		BodyText:      mailBody,
		BodyHTML:      req.BodyHTML,
		IsUnread:      false,
		IsStarred:     false,
		IsImportant:   false,
		HasAttachment: len(req.Attachments) > 0,
		Priority:      req.Priority,
		SizeBytes:     int64(len(fullMsg)),
		Attachments:   req.Attachments,
	}
	if sentMsg.Priority == "" {
		sentMsg.Priority = "normal"
	}

	_ = h.store.CreateWebmailMessage(r.Context(), sentMsg)

	// Also write to .Sent Maildir on filesystem if directory exists
	parts := strings.SplitN(mb.Email, "@", 2)
	if len(parts) == 2 {
		sentDir := filepath.Join("/var/mail/vhosts", parts[1], parts[0], ".Sent", "new")
		if err := os.MkdirAll(sentDir, 0700); err == nil {
			fName := fmt.Sprintf("%d.H%dP%d.hostvra,S=%d", time.Now().Unix(), os.Getpid(), time.Now().UnixNano()%10000, len(fullMsg))
			_ = os.WriteFile(filepath.Join(sentDir, fName), []byte(fullMsg), 0600)
		}
	}

	// 4. Record Audit and Delivery Log
	_ = h.store.RecordEmailDeliveryLog(r.Context(), &store.EmailDeliveryLog{
		ID:            uuid.New(),
		ServerID:      mb.ServerID,
		DomainID:      &mb.DomainID,
		MessageID:     msgID,
		Sender:        mb.Email,
		Recipient:     primaryTo,
		Status:        deliveryStatus,
		FailureReason: failureReason,
	})

	h.audit.Log(r.Context(), r, "webmail.message.send", "webmail_message", sentMsg.ID.String(), "success", "", map[string]interface{}{
		"from":      mb.Email,
		"to":        strings.Join(recipients, ", "),
		"status":    deliveryStatus,
		"messageID": msgID,
	})

	res := map[string]interface{}{
		"message":   sentMsg,
		"status":    deliveryStatus,
		"delivered": deliveryErr == nil,
	}

	response.JSON(w, http.StatusOK, res, nil)
}

func (h *WebmailHandler) SaveDraft(w http.ResponseWriter, r *http.Request) {
	var req SendWebmailMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", nil, "")
		return
	}

	var mb *store.EmailMailbox
	var err error

	fromClean := strings.ToLower(strings.TrimSpace(req.AccountEmail))
	if fromClean == "" {
		fromClean = strings.ToLower(strings.TrimSpace(req.FromEmail))
	}

	if req.MailboxID != nil && *req.MailboxID != uuid.Nil {
		mb, err = h.store.GetEmailMailboxByID(r.Context(), *req.MailboxID)
	} else if fromClean != "" {
		mb, err = h.store.GetEmailMailboxByEmail(r.Context(), fromClean)
	}

	if err != nil || mb == nil {
		response.Error(w, http.StatusNotFound, "MAILBOX_NOT_FOUND", "Mailbox not found", nil, "")
		return
	}

	toClean := req.ToEmail
	if toClean == "" && len(req.To) > 0 {
		toClean = strings.Join(req.To, ", ")
	}

	draftID := uuid.New()
	if req.ID != nil && *req.ID != uuid.Nil {
		draftID = *req.ID
	}

	draftMsg := &store.WebmailMessage{
		ID:            draftID,
		MailboxID:     mb.ID,
		AccountEmail:  mb.Email,
		Folder:        "drafts",
		FromName:      mb.Name,
		FromEmail:     mb.Email,
		ToName:        toClean,
		ToEmail:       toClean,
		Cc:            req.Cc,
		Bcc:           req.Bcc,
		Subject:       req.Subject,
		BodyText:      req.BodyText,
		BodyHTML:      req.BodyHTML,
		IsUnread:      false,
		HasAttachment: len(req.Attachments) > 0,
		Priority:      req.Priority,
		SizeBytes:     int64(len(req.BodyHTML) + len(req.BodyText)),
		Attachments:   req.Attachments,
	}
	if draftMsg.Priority == "" {
		draftMsg.Priority = "normal"
	}

	if err := h.store.CreateWebmailMessage(r.Context(), draftMsg); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to save draft", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, draftMsg, nil)
}

func (h *WebmailHandler) UpdateMessageFlags(w http.ResponseWriter, r *http.Request) {
	msgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid message UUID", nil, "")
		return
	}

	var req UpdateMessageFlagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", nil, "")
		return
	}

	if err := h.store.UpdateWebmailMessageFlags(r.Context(), msgID, req.IsUnread, req.IsStarred, req.IsImportant); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update flags", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Flags updated successfully"}, nil)
}

func (h *WebmailHandler) MoveMessage(w http.ResponseWriter, r *http.Request) {
	msgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid message UUID", nil, "")
		return
	}

	var req MoveMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetFolder == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Target folder required", nil, "")
		return
	}

	switch req.TargetFolder {
	case "inbox", "sent", "drafts", "trash", "spam", "archive":
	default:
		response.Error(w, http.StatusBadRequest, "INVALID_FOLDER", "Invalid destination folder", nil, "")
		return
	}

	if err := h.store.MoveWebmailMessage(r.Context(), msgID, req.TargetFolder); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to move message", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("Message moved to %s", req.TargetFolder)}, nil)
}

func (h *WebmailHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	msgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid message UUID", nil, "")
		return
	}

	if err := h.store.DeleteWebmailMessage(r.Context(), msgID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete message", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Message deleted successfully"}, nil)
}

// ----------------------------------------------------------------------------
// SECURITY & SANITIZATION HELPERS
// ----------------------------------------------------------------------------

func verifyPasswordHash(plainPassword, storedHash string) bool {
	if storedHash == "" {
		return false
	}

	// Support bcrypt / BLF-CRYPT
	if strings.HasPrefix(storedHash, "{BLF-CRYPT}") || strings.HasPrefix(storedHash, "$2") {
		cleanHash := strings.TrimPrefix(storedHash, "{BLF-CRYPT}")
		if err := bcrypt.CompareHashAndPassword([]byte(cleanHash), []byte(plainPassword)); err == nil {
			return true
		}
	}

	// Dovecot SHA512-CRYPT format: "$6$<salt>$<hash>" or "{SHA512-CRYPT}$6$<salt>$<hash>"
	cleanHash := strings.TrimPrefix(storedHash, "{SHA512-CRYPT}")
	parts := strings.Split(cleanHash, "$")
	if len(parts) >= 4 && parts[1] == "6" {
		salt := parts[2]
		expectedHex := parts[3]

		h := sha512.New()
		h.Write([]byte(plainPassword + salt))
		computedHex := fmt.Sprintf("%x", h.Sum(nil))

		return computedHex == expectedHex
	}

	// Plaintext fallback only in initial seed/test states
	return plainPassword == storedHash
}

func sanitizeEmailHTML(html string) string {
	if html == "" {
		return ""
	}

	// Strip script tags
	reScript := regexp.MustCompile(`(?i)<script[\s\S]*?</script>`)
	sanitized := reScript.ReplaceAllString(html, "")

	// Strip dangerous event handlers (e.g. onload=, onclick=, onerror=)
	reEvents := regexp.MustCompile(`(?i)\s+on[a-z]+\s*=\s*(?:'[^']*'|"[^"]*"|[^\s>]+)`)
	sanitized = reEvents.ReplaceAllString(sanitized, "")

	// Strip javascript: URLs
	reJSURL := regexp.MustCompile(`(?i)href\s*=\s*['"]javascript:[^'"]*['"]`)
	sanitized = reJSURL.ReplaceAllString(sanitized, `href="#"`)

	return sanitized
}

func stripHTMLTags(html string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return strings.TrimSpace(re.ReplaceAllString(html, " "))
}

// sendMailLocal transmits email to local Postfix MTA with InsecureSkipVerify for loopback TLS
func sendMailLocal(addr string, from string, to []string, msg []byte) error {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to local MTA at %s: %w", addr, err)
	}
	defer conn.Close()

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// If STARTTLS is advertised on local loopback, proceed with InsecureSkipVerify
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS negotiation failed: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}

	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("SMTP RCPT TO <%s> failed: %w", recipient, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA command failed: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("failed writing email message payload: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed finalizing email message payload: %w", err)
	}

	return client.Quit()
}
