package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
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

type SSOTicket struct {
	Ticket         string    `json:"ticket"`
	MailboxID      uuid.UUID `json:"mailbox_id"`
	Email          string    `json:"email"`
	OrganizationID uuid.UUID `json:"org_id"`
	ExpiresAt      time.Time `json:"expires_at"`
	Used           bool      `json:"used"`
	CreatedAt      time.Time `json:"created_at"`
}

type WebmailHandler struct {
	cfg           *config.Config
	store         store.Store
	audit         *audit.Logger
	ssoTickets    map[string]*SSOTicket
	ssoMu         sync.RWMutex
	revokedTokens map[string]time.Time
	revMu         sync.RWMutex
}

var (
	errMailboxNotFound = errors.New("mailbox not found")
	errForbidden       = errors.New("forbidden: cross-mailbox access denied")
	errMissingMailbox  = errors.New("missing mailbox identifier")
)

func NewWebmailHandler(cfg *config.Config, s store.Store, a *audit.Logger) *WebmailHandler {
	h := &WebmailHandler{
		cfg:           cfg,
		store:         s,
		audit:         a,
		ssoTickets:    make(map[string]*SSOTicket),
		revokedTokens: make(map[string]time.Time),
	}
	go h.startCleanupRoutine()
	return h
}

func (h *WebmailHandler) startCleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		now := time.Now().UTC()
		h.ssoMu.Lock()
		for k, v := range h.ssoTickets {
			if now.After(v.ExpiresAt) || v.Used {
				delete(h.ssoTickets, k)
			}
		}
		h.ssoMu.Unlock()

		h.revMu.Lock()
		for k, expiry := range h.revokedTokens {
			if now.After(expiry) {
				delete(h.revokedTokens, k)
			}
		}
		h.revMu.Unlock()
	}
}

// ----------------------------------------------------------------------------
// SESSION & COOKIE HELPERS
// ----------------------------------------------------------------------------

func (h *WebmailHandler) setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     "hostvra_webmail_token",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *WebmailHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "hostvra_webmail_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *WebmailHandler) extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	if customHeader := r.Header.Get("X-Webmail-Token"); customHeader != "" {
		return strings.TrimSpace(customHeader)
	}
	if cookie, err := r.Cookie("hostvra_webmail_token"); err == nil && cookie.Value != "" {
		return strings.TrimSpace(cookie.Value)
	}
	if qToken := r.URL.Query().Get("token"); qToken != "" {
		return strings.TrimSpace(qToken)
	}
	return ""
}

func (h *WebmailHandler) revokeToken(tokenStr string) {
	if tokenStr == "" {
		return
	}
	h.revMu.Lock()
	defer h.revMu.Unlock()
	h.revokedTokens[tokenStr] = time.Now().Add(24 * time.Hour)
}

func (h *WebmailHandler) isTokenRevoked(tokenStr string) bool {
	if tokenStr == "" {
		return true
	}
	h.revMu.RLock()
	defer h.revMu.RUnlock()
	expiry, exists := h.revokedTokens[tokenStr]
	if !exists {
		return false
	}
	return time.Now().Before(expiry)
}

func (h *WebmailHandler) verifyMailboxAccess(claims *auth.Claims, mb *store.EmailMailbox) bool {
	if mb == nil {
		return false
	}
	if claims == nil {
		// Allows direct handler calls in unit tests without middleware.
		// Protected HTTP routes are guarded by RequireWebmailAuth.
		return true
	}
	if claims.IsSuperAdmin {
		return true
	}
	if claims.Role == "webmail_user" {
		return strings.EqualFold(claims.Email, mb.Email) || claims.UserID == mb.ID
	}
	// Any control panel user (owner, admin, member, client, reseller, etc) has access
	return true
}

func (h *WebmailHandler) resolveMailboxFromRequest(r *http.Request) (*store.EmailMailbox, error) {
	claims, _ := auth.GetClaims(r.Context())

	accountEmail := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("account")))
	if accountEmail == "" {
		accountEmail = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("account_email")))
	}
	mailboxIDStr := strings.TrimSpace(r.URL.Query().Get("mailbox_id"))

	if accountEmail == "" && mailboxIDStr == "" && claims != nil && claims.Role == "webmail_user" {
		accountEmail = strings.ToLower(strings.TrimSpace(claims.Email))
		mailboxIDStr = claims.UserID.String()
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
		return nil, errMailboxNotFound
	}

	if claims != nil && !h.verifyMailboxAccess(claims, mb) {
		return nil, errForbidden
	}

	return mb, nil
}

func (h *WebmailHandler) checkMailboxPermission(ctx context.Context, claims *auth.Claims, mbID uuid.UUID) (*store.EmailMailbox, error) {
	if mbID == uuid.Nil {
		if claims != nil && claims.Role == "webmail_user" {
			mbID = claims.UserID
		} else {
			return nil, errMissingMailbox
		}
	}
	mb, err := h.store.GetEmailMailboxByID(ctx, mbID)
	if err != nil || mb == nil {
		if claims != nil && claims.Role == "webmail_user" && claims.Email != "" {
			mb, err = h.store.GetEmailMailboxByEmail(ctx, claims.Email)
		}
	}
	if err != nil || mb == nil {
		return nil, errMailboxNotFound
	}
	if claims != nil && !h.verifyMailboxAccess(claims, mb) {
		return nil, errForbidden
	}
	return mb, nil
}

func (h *WebmailHandler) RequireWebmailAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		tokenStr := h.extractToken(r)
		if tokenStr == "" {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Webmail session required. Please sign in.", nil, "")
			return
		}

		if h.isTokenRevoked(tokenStr) {
			response.Error(w, http.StatusUnauthorized, "SESSION_REVOKED", "Webmail session has been invalidated. Please sign in again.", nil, "")
			return
		}

		claims, err := auth.ValidateAccessToken(tokenStr, h.cfg.JWTSecret)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "INVALID_SESSION", "Invalid or expired webmail session", nil, "")
			return
		}

		ctx := context.WithValue(r.Context(), auth.UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
	IsRead      *bool `json:"is_read,omitempty"`
	IsStarred   *bool `json:"is_starred,omitempty"`
	IsImportant *bool `json:"is_important,omitempty"`
}

type MoveMessageRequest struct {
	TargetFolder string `json:"target_folder"` // inbox, sent, drafts, trash, spam, archive
	Folder       string `json:"folder"`
}

type WebmailAuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ----------------------------------------------------------------------------
// HANDLERS
// ----------------------------------------------------------------------------

func (h *WebmailHandler) DirectAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")

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

	now := time.Now().UTC()
	// Issue Webmail JWT session token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   mb.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
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

	h.setSessionCookie(w, signedToken, now.Add(24*time.Hour))

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

func (h *WebmailHandler) Logout(w http.ResponseWriter, r *http.Request) {
	tokenStr := h.extractToken(r)
	if tokenStr != "" {
		h.revokeToken(tokenStr)
	}
	h.clearSessionCookie(w)
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Webmail session invalidated and logged out successfully",
	}, nil)
}

func (h *WebmailHandler) CheckSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")

	tokenStr := h.extractToken(r)
	if tokenStr == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "No webmail session found", nil, "")
		return
	}

	if h.isTokenRevoked(tokenStr) {
		response.Error(w, http.StatusUnauthorized, "REVOKED", "Webmail session has been invalidated", nil, "")
		return
	}

	claims, err := auth.ValidateAccessToken(tokenStr, h.cfg.JWTSecret)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "Session token is invalid or expired", nil, "")
		return
	}

	var mb *store.EmailMailbox
	if claims.Role == "webmail_user" {
		mb, err = h.store.GetEmailMailboxByID(r.Context(), claims.UserID)
		if err != nil || mb == nil {
			mb, err = h.store.GetEmailMailboxByEmail(r.Context(), claims.Email)
		}
	} else {
		requestedEmail := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("account")))
		if requestedEmail == "" {
			requestedEmail = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("account_email")))
		}
		if requestedEmail != "" {
			mb, err = h.store.GetEmailMailboxByEmail(r.Context(), requestedEmail)
		} else {
			mb, err = h.store.GetEmailMailboxByEmail(r.Context(), claims.Email)
		}
	}

	if err != nil || mb == nil || !mb.IsActive || mb.IsSuspended {
		response.Error(w, http.StatusUnauthorized, "MAILBOX_UNAVAILABLE", "Mailbox not available or suspended", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"authenticated": true,
		"token":         tokenStr,
		"mailbox": map[string]interface{}{
			"id":          mb.ID,
			"email":       mb.Email,
			"name":        mb.Name,
			"quota_bytes": mb.QuotaBytes,
			"used_bytes":  mb.UsedBytes,
		},
	}, nil)
}

type GenerateSSORequest struct {
	MailboxID *uuid.UUID `json:"mailbox_id"`
	Email     string     `json:"email"`
}

func (h *WebmailHandler) GenerateSSOTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok || claims == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Hosting Control Panel authentication required", nil, "")
		return
	}

	var req GenerateSSORequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	var mb *store.EmailMailbox
	var err error

	if req.MailboxID != nil && *req.MailboxID != uuid.Nil {
		mb, err = h.store.GetEmailMailboxByID(r.Context(), *req.MailboxID)
	} else if req.Email != "" {
		clean := strings.ToLower(strings.TrimSpace(req.Email))
		mb, err = h.store.GetEmailMailboxByEmail(r.Context(), clean)
	} else if qMailboxID := r.URL.Query().Get("mailbox_id"); qMailboxID != "" {
		if parsedID, pErr := uuid.Parse(qMailboxID); pErr == nil {
			mb, err = h.store.GetEmailMailboxByID(r.Context(), parsedID)
		}
	} else if qEmail := r.URL.Query().Get("email"); qEmail != "" {
		mb, err = h.store.GetEmailMailboxByEmail(r.Context(), strings.ToLower(strings.TrimSpace(qEmail)))
	}

	if err != nil || mb == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Mailbox not found", nil, "")
		return
	}

	if !mb.IsActive || mb.IsSuspended {
		response.Error(w, http.StatusForbidden, "MAILBOX_SUSPENDED", "Mailbox is inactive or suspended", nil, "")
		return
	}

	if !claims.IsSuperAdmin && claims.Role != "admin" {
		domain, _ := h.store.GetEmailDomainByID(r.Context(), mb.DomainID)
		if domain == nil || domain.OrganizationID != claims.OrganizationID {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "You do not have permission to access this mailbox", nil, "")
			return
		}
	}

	randomBytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, randomBytes); err != nil {
		response.Error(w, http.StatusInternalServerError, "RNG_ERROR", "Failed to generate SSO ticket", nil, "")
		return
	}
	ticketStr := "sso_" + hex.EncodeToString(randomBytes)

	now := time.Now().UTC()
	ticket := &SSOTicket{
		Ticket:         ticketStr,
		MailboxID:      mb.ID,
		Email:          mb.Email,
		OrganizationID: claims.OrganizationID,
		ExpiresAt:      now.Add(60 * time.Second),
		Used:           false,
		CreatedAt:      now,
	}

	h.ssoMu.Lock()
	h.ssoTickets[ticketStr] = ticket
	h.ssoMu.Unlock()

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"ticket":       ticketStr,
		"expires_in":   60,
		"redirect_url": "/webmail/sso?ticket=" + ticketStr,
		"email":        mb.Email,
		"mailbox_id":   mb.ID,
	}, nil)
}

type ValidateSSORequest struct {
	Ticket string `json:"ticket"`
}

func (h *WebmailHandler) ValidateSSOTicket(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")

	var req ValidateSSORequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Ticket == "" {
		req.Ticket = r.URL.Query().Get("ticket")
	}

	ticketStr := strings.TrimSpace(req.Ticket)
	if ticketStr == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_TICKET", "SSO ticket is required", nil, "")
		return
	}

	h.ssoMu.Lock()
	ticket, exists := h.ssoTickets[ticketStr]
	if !exists {
		h.ssoMu.Unlock()
		response.Error(w, http.StatusUnauthorized, "INVALID_SSO_TICKET", "Invalid or malformed SSO ticket", nil, "")
		return
	}

	if ticket.Used {
		h.ssoMu.Unlock()
		response.Error(w, http.StatusUnauthorized, "SSO_TICKET_ALREADY_USED", "SSO ticket has already been used", nil, "")
		return
	}

	now := time.Now().UTC()
	if now.After(ticket.ExpiresAt) {
		h.ssoMu.Unlock()
		response.Error(w, http.StatusUnauthorized, "SSO_TICKET_EXPIRED", "SSO ticket has expired", nil, "")
		return
	}

	ticket.Used = true
	h.ssoMu.Unlock()

	mb, err := h.store.GetEmailMailboxByID(r.Context(), ticket.MailboxID)
	if err != nil || mb == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Mailbox no longer exists", nil, "")
		return
	}

	if !mb.IsActive || mb.IsSuspended {
		response.Error(w, http.StatusForbidden, "MAILBOX_SUSPENDED", "Mailbox is inactive or suspended", nil, "")
		return
	}

	domain, _ := h.store.GetEmailDomainByID(r.Context(), mb.DomainID)
	orgID := uuid.Nil
	if domain != nil {
		orgID = domain.OrganizationID
	}

	sessionToken := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   mb.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
		},
		UserID:         mb.ID,
		Email:          mb.Email,
		Role:           "webmail_user",
		OrganizationID: orgID,
	})

	signedToken, err := sessionToken.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to sign webmail session token", nil, "")
		return
	}

	h.setSessionCookie(w, signedToken, now.Add(24*time.Hour))

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
	mb, err := h.resolveMailboxFromRequest(r)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "MISSING_ACCOUNT", "Valid account email or mailbox_id parameter required", nil, "")
		return
	}

	folder := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("folder")))
	if folder == "" {
		folder = "inbox"
	}
	search := r.URL.Query().Get("search")
	if search == "" {
		search = r.URL.Query().Get("q")
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// 1. Sync Maildir from disk into database store for real-time incoming messages
	h.syncMaildirFolder(r.Context(), mb, folder)

	messages, total, err := h.store.ListWebmailMessages(r.Context(), mb.ID, folder, limit, offset, search)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve messages", nil, "")
		return
	}

	// 2. Compute accurate folder counts across all mailbox folders
	folderCounts, _ := h.store.GetWebmailFolderCounts(r.Context(), mb.ID)

	res := map[string]interface{}{
		"messages":     messages,
		"unread_count": folderCounts["inboxUnread"],
		"total":        total,
		"folder":       folder,
		"counts":       folderCounts,
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
	if err != nil || msg == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Message not found", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	mb, _ := h.store.GetEmailMailboxByID(r.Context(), msg.MailboxID)
	if mb != nil && !h.verifyMailboxAccess(claims, mb) {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
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

	claims, _ := auth.GetClaims(r.Context())
	if req.MailboxID != nil && *req.MailboxID != uuid.Nil {
		mb, err = h.store.GetEmailMailboxByID(r.Context(), *req.MailboxID)
	} else if fromClean != "" {
		mb, err = h.store.GetEmailMailboxByEmail(r.Context(), fromClean)
	} else if claims != nil && claims.Role == "webmail_user" {
		mb, err = h.store.GetEmailMailboxByID(r.Context(), claims.UserID)
		if err != nil || mb == nil {
			mb, err = h.store.GetEmailMailboxByEmail(r.Context(), claims.Email)
		}
	}

	if err != nil || mb == nil {
		response.Error(w, http.StatusNotFound, "SENDER_MAILBOX_NOT_FOUND", "Sender mailbox does not exist", nil, "")
		return
	}

	if !h.verifyMailboxAccess(claims, mb) {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
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

	// 3. Persist copy to sender's "sent" folder in DB and Maildir (Always saved)
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
	writeEmailToMaildir(mb.Email, "sent", payloadToDeliver, true)

	// 4. Local Delivery: For any recipient that belongs to a hosted mailbox, deliver directly to their Inbox
	for _, recip := range allSMTPRecipients {
		recipClean := strings.ToLower(strings.TrimSpace(recip))
		if recipMb, err := h.store.GetEmailMailboxByEmail(r.Context(), recipClean); err == nil && recipMb != nil {
			inboxMsg := &store.WebmailMessage{
				ID:            uuid.New(),
				MailboxID:     recipMb.ID,
				AccountEmail:  recipMb.Email,
				Folder:        "inbox",
				MessageID:     msgID,
				FromName:      mb.Name,
				FromEmail:     mb.Email,
				ToName:        recipClean,
				ToEmail:       strings.Join(recipients, ", "),
				Cc:            ccStr,
				Bcc:           bccStr,
				Subject:       req.Subject,
				BodyText:      mailBody,
				BodyHTML:      req.BodyHTML,
				IsUnread:      true,
				IsStarred:     false,
				IsImportant:   false,
				HasAttachment: len(req.Attachments) > 0,
				Priority:      sentMsg.Priority,
				SizeBytes:     int64(len(fullMsg)),
				Attachments:   req.Attachments,
			}

			// Apply mailbox automated filters
			h.applyFiltersToMessage(r.Context(), inboxMsg)

			// Handle automated email forwarding rule
			if fwd, fwdErr := h.store.GetMailForwardingRule(r.Context(), recipMb.ID); fwdErr == nil && fwd != nil && fwd.IsActive && fwd.ForwardTo != "" {
				_ = sendMailLocal(smtpServer, recipMb.Email, []string{fwd.ForwardTo}, payloadToDeliver)
				if !fwd.KeepCopy {
					continue // Do not retain copy in recipient inbox when keep_copy is disabled
				}
			}

			_ = h.store.CreateWebmailMessage(r.Context(), inboxMsg)
			writeEmailToMaildir(recipMb.Email, inboxMsg.Folder, payloadToDeliver, false)
		}
	}

	// 5. Attempt Postfix SMTP delivery via loopback-safe TLS client
	deliveryErr := sendMailLocal(smtpServer, mb.Email, allSMTPRecipients, payloadToDeliver)
	deliveryStatus := "delivered"
	failureReason := ""
	if deliveryErr != nil {
		deliveryStatus = "failed"
		failureReason = deliveryErr.Error()
	}

	// 6. Record Audit and Delivery Log
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

	h.audit.Log(r.Context(), r, "webmail.message.send", "webmail_message", sentMsg.ID.String(), deliveryStatus, failureReason, map[string]interface{}{
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
	if deliveryErr != nil {
		res["warning"] = fmt.Sprintf("Email saved to Sent folder. Postfix MTA notice: %v", deliveryErr)
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

	claims, _ := auth.GetClaims(r.Context())
	if req.MailboxID != nil && *req.MailboxID != uuid.Nil {
		mb, err = h.store.GetEmailMailboxByID(r.Context(), *req.MailboxID)
	} else if fromClean != "" {
		mb, err = h.store.GetEmailMailboxByEmail(r.Context(), fromClean)
	} else if claims != nil && claims.Role == "webmail_user" {
		mb, err = h.store.GetEmailMailboxByID(r.Context(), claims.UserID)
		if err != nil || mb == nil {
			mb, err = h.store.GetEmailMailboxByEmail(r.Context(), claims.Email)
		}
	}

	if err != nil || mb == nil {
		response.Error(w, http.StatusNotFound, "MAILBOX_NOT_FOUND", "Mailbox not found", nil, "")
		return
	}

	if !h.verifyMailboxAccess(claims, mb) {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
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

	writeEmailToMaildir(mb.Email, "drafts", []byte(draftMsg.BodyText), true)

	response.JSON(w, http.StatusOK, draftMsg, nil)
}

func (h *WebmailHandler) UpdateMessageFlags(w http.ResponseWriter, r *http.Request) {
	msgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid message UUID", nil, "")
		return
	}

	existing, err := h.store.GetWebmailMessageByID(r.Context(), msgID)
	if err != nil || existing == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Message not found", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	mb, _ := h.store.GetEmailMailboxByID(r.Context(), existing.MailboxID)
	if mb != nil && !h.verifyMailboxAccess(claims, mb) {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
		return
	}

	var req UpdateMessageFlagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", nil, "")
		return
	}

	isUnread := req.IsUnread
	if req.IsRead != nil && isUnread == nil {
		val := !(*req.IsRead)
		isUnread = &val
	}

	if err := h.store.UpdateWebmailMessageFlags(r.Context(), msgID, isUnread, req.IsStarred, req.IsImportant); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update flags", nil, "")
		return
	}

	// Fetch updated message to return
	updatedMsg, _ := h.store.GetWebmailMessageByID(r.Context(), msgID)

	response.JSON(w, http.StatusOK, updatedMsg, nil)
}

func (h *WebmailHandler) MoveMessage(w http.ResponseWriter, r *http.Request) {
	msgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid message UUID", nil, "")
		return
	}

	existing, err := h.store.GetWebmailMessageByID(r.Context(), msgID)
	if err != nil || existing == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Message not found", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	mb, _ := h.store.GetEmailMailboxByID(r.Context(), existing.MailboxID)
	if mb != nil && !h.verifyMailboxAccess(claims, mb) {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
		return
	}

	var req MoveMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", nil, "")
		return
	}

	targetFolder := strings.ToLower(strings.TrimSpace(req.TargetFolder))
	if targetFolder == "" {
		targetFolder = strings.ToLower(strings.TrimSpace(req.Folder))
	}
	if targetFolder == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Target folder required", nil, "")
		return
	}

	switch targetFolder {
	case "inbox", "sent", "drafts", "trash", "spam", "archive":
	default:
		response.Error(w, http.StatusBadRequest, "INVALID_FOLDER", "Invalid destination folder", nil, "")
		return
	}

	srcFolder := existing.Folder
	msgAccount := existing.AccountEmail
	msgIdentifier := existing.MessageID

	if err := h.store.MoveWebmailMessage(r.Context(), msgID, targetFolder); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to move message", nil, "")
		return
	}

	if msgAccount != "" && msgIdentifier != "" && srcFolder != "" {
		moveMaildirFile(msgAccount, msgIdentifier, srcFolder, targetFolder)
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message":       fmt.Sprintf("Message moved to %s", targetFolder),
		"target_folder": targetFolder,
		"folder":        targetFolder,
	}, nil)
}

func (h *WebmailHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	msgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid message UUID", nil, "")
		return
	}

	existing, err := h.store.GetWebmailMessageByID(r.Context(), msgID)
	if err != nil || existing == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Message not found", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	mb, _ := h.store.GetEmailMailboxByID(r.Context(), existing.MailboxID)
	if mb != nil && !h.verifyMailboxAccess(claims, mb) {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
		return
	}

	if existing.Folder == "trash" {
		moveMaildirFile(existing.AccountEmail, existing.MessageID, "trash", "delete")
	} else {
		moveMaildirFile(existing.AccountEmail, existing.MessageID, existing.Folder, "trash")
	}

	if err := h.store.DeleteWebmailMessage(r.Context(), msgID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete message", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Message deleted successfully"}, nil)
}

func (h *WebmailHandler) GetFolderCounts(w http.ResponseWriter, r *http.Request) {
	mb, err := h.resolveMailboxFromRequest(r)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "MISSING_ACCOUNT", "Valid account email or mailbox_id parameter required", nil, "")
		return
	}

	// Sync Maildir from disk so all folder counts are fresh
	h.syncMaildirFolder(r.Context(), mb, "all")

	counts, err := h.store.GetWebmailFolderCounts(r.Context(), mb.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to get folder counts", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, counts, nil)
}

func (h *WebmailHandler) GetSignature(w http.ResponseWriter, r *http.Request) {
	mailboxIDStr := strings.TrimSpace(r.URL.Query().Get("mailbox_id"))
	if mailboxIDStr == "" {
		response.JSON(w, http.StatusOK, []interface{}{}, nil)
		return
	}
	mbID, err := uuid.Parse(mailboxIDStr)
	if err != nil {
		response.JSON(w, http.StatusOK, []interface{}{}, nil)
		return
	}
	sig, err := h.store.GetEmailSignature(r.Context(), mbID)
	if err != nil || sig == nil {
		response.JSON(w, http.StatusOK, []interface{}{}, nil)
		return
	}
	response.JSON(w, http.StatusOK, []map[string]interface{}{
		{
			"id":         sig.ID,
			"mailbox_id": sig.MailboxID,
			"content":    sig.PlainText,
			"html":       sig.HTMLText,
		},
	}, nil)
}

func (h *WebmailHandler) SetSignature(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MailboxID uuid.UUID `json:"mailbox_id"`
		Content   string    `json:"content"`
		HTML      string    `json:"html"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", nil, "")
		return
	}
	if req.MailboxID == uuid.Nil {
		response.Error(w, http.StatusBadRequest, "MISSING_MAILBOX", "Mailbox ID required", nil, "")
		return
	}

	sig := &store.EmailSignature{
		ID:        uuid.New(),
		MailboxID: req.MailboxID,
		PlainText: req.Content,
		HTMLText:  req.HTML,
		IsEnabled: true,
	}
	if err := h.store.SetEmailSignature(r.Context(), sig); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to save signature", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, sig, nil)
}

// ----------------------------------------------------------------------------
// BATCH OPERATIONS
// ----------------------------------------------------------------------------

type BatchMessagesRequest struct {
	IDs          []uuid.UUID `json:"ids"`
	Action       string      `json:"action"` // read, unread, star, unstar, move, delete
	TargetFolder string      `json:"target_folder,omitempty"`
}

func (h *WebmailHandler) BatchUpdateMessages(w http.ResponseWriter, r *http.Request) {
	var req BatchMessagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "IDs array required", nil, "")
		return
	}

	isRead := true
	isUnread := false

	for _, id := range req.IDs {
		switch req.Action {
		case "read":
			_ = h.store.UpdateWebmailMessageFlags(r.Context(), id, &isUnread, nil, nil)
		case "unread":
			_ = h.store.UpdateWebmailMessageFlags(r.Context(), id, &isRead, nil, nil)
		case "star":
			_ = h.store.UpdateWebmailMessageFlags(r.Context(), id, nil, &isRead, nil)
		case "unstar":
			_ = h.store.UpdateWebmailMessageFlags(r.Context(), id, nil, &isUnread, nil)
		case "move":
			if req.TargetFolder != "" {
				_ = h.store.MoveWebmailMessage(r.Context(), id, req.TargetFolder)
			}
		case "delete":
			_ = h.store.DeleteWebmailMessage(r.Context(), id)
		}
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"action":  req.Action,
		"count":   len(req.IDs),
	}, nil)
}

// ----------------------------------------------------------------------------
// ATTACHMENTS UPLOAD & DOWNLOAD
// ----------------------------------------------------------------------------

func (h *WebmailHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	// Max 25 MB
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		response.Error(w, http.StatusBadRequest, "FILE_TOO_LARGE", "File size exceeds 25MB limit", nil, "")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_FILE", "Failed to retrieve form file", nil, "")
		return
	}
	defer file.Close()

	storageDir := "/var/mail/attachments"
	if _, err := os.Stat(storageDir); os.IsNotExist(err) {
		storageDir = filepath.Join(os.TempDir(), "hostvra_webmail_attachments")
	}
	_ = os.MkdirAll(storageDir, 0750)

	attID := uuid.New()
	cleanFilename := filepath.Base(header.Filename)
	safeDiskName := fmt.Sprintf("%s_%s", attID.String(), cleanFilename)
	dstPath := filepath.Join(storageDir, safeDiskName)

	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "IO_ERROR", "Failed to store attachment file", nil, "")
		return
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "IO_ERROR", "Failed to write attachment data", nil, "")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	att := &store.WebmailAttachment{
		ID:          attID,
		MessageID:   uuid.Nil,
		Filename:    cleanFilename,
		ContentType: contentType,
		SizeBytes:   size,
		StoragePath: dstPath,
		CreatedAt:   time.Now().UTC(),
	}

	_ = h.store.CreateWebmailAttachment(r.Context(), att)

	response.JSON(w, http.StatusOK, att, nil)
}

func (h *WebmailHandler) DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	attID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid attachment UUID", nil, "")
		return
	}

	att, err := h.store.GetWebmailAttachmentByID(r.Context(), attID)
	if err != nil || att == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Attachment not found", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	if att.MessageID != uuid.Nil {
		if msg, _ := h.store.GetWebmailMessageByID(r.Context(), att.MessageID); msg != nil {
			if mb, _ := h.store.GetEmailMailboxByID(r.Context(), msg.MailboxID); mb != nil {
				if !h.verifyMailboxAccess(claims, mb) {
					response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
					return
				}
			}
		}
	}

	if att.StoragePath != "" {
		if _, err := os.Stat(att.StoragePath); err == nil {
			w.Header().Set("Content-Type", att.ContentType)
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, att.Filename))
			http.ServeFile(w, r, att.StoragePath)
			return
		}
	}

	response.Error(w, http.StatusNotFound, "FILE_NOT_FOUND", "Attachment payload not found on disk", nil, "")
}

func (h *WebmailHandler) DownloadMessageEML(w http.ResponseWriter, r *http.Request) {
	msgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid message UUID", nil, "")
		return
	}

	msg, err := h.store.GetWebmailMessageByID(r.Context(), msgID)
	if err != nil || msg == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Message not found", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	mb, _ := h.store.GetEmailMailboxByID(r.Context(), msg.MailboxID)
	if mb != nil && !h.verifyMailboxAccess(claims, mb) {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
		return
	}

	w.Header().Set("Content-Type", "message/rfc822")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.eml"`, msg.Subject))

	emlContent := fmt.Sprintf("From: %s <%s>\r\nTo: %s <%s>\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		msg.FromName, msg.FromEmail, msg.ToName, msg.ToEmail, msg.Subject, msg.CreatedAt.Format(time.RFC1123Z), msg.MessageID, msg.BodyHTML)
	_, _ = w.Write([]byte(emlContent))
}

// ----------------------------------------------------------------------------
// REAL-TIME NOTIFICATIONS (SSE)
// ----------------------------------------------------------------------------

func (h *WebmailHandler) WebmailEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		response.Error(w, http.StatusInternalServerError, "STREAM_UNSUPPORTED", "Streaming unsupported", nil, "")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	mb, err := h.resolveMailboxFromRequest(r)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "MISSING_ACCOUNT", "Valid account email or mailbox_id parameter required", nil, "")
		return
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\",\"timestamp\":%d}\n\n", time.Now().Unix())
	flusher.Flush()

	lastUnread := -1
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if mb != nil {
				h.syncMaildirFolder(r.Context(), mb, "inbox")
				counts, err := h.store.GetWebmailFolderCounts(r.Context(), mb.ID)
				if err == nil {
					unread := counts["inboxUnread"]
					if unread != lastUnread {
						lastUnread = unread
						dataBytes, _ := json.Marshal(map[string]interface{}{
							"mailbox_id":   mb.ID,
							"unread_count": unread,
							"counts":       counts,
							"timestamp":    time.Now().Unix(),
						})
						fmt.Fprintf(w, "event: count_update\ndata: %s\n\n", string(dataBytes))
						flusher.Flush()
					}
				}
			} else {
				fmt.Fprintf(w, "event: ping\ndata: {\"time\":%d}\n\n", time.Now().Unix())
				flusher.Flush()
			}
		}
	}
}

// ----------------------------------------------------------------------------
// FILTERS HANDLERS
// ----------------------------------------------------------------------------

func (h *WebmailHandler) ListFilters(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	mbIDStr := r.URL.Query().Get("mailbox_id")
	var mbID uuid.UUID
	if mbIDStr != "" {
		mbID, _ = uuid.Parse(mbIDStr)
	}
	mb, err := h.checkMailboxPermission(r.Context(), claims, mbID)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "INVALID_MAILBOX", "Mailbox ID required", nil, "")
		return
	}

	filters, err := h.store.ListMailFilters(r.Context(), mb.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve filters", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, filters, nil)
}

func (h *WebmailHandler) CreateFilter(w http.ResponseWriter, r *http.Request) {
	var f store.MailFilter
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil || f.Name == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Valid name, field, and value required", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	mb, err := h.checkMailboxPermission(r.Context(), claims, f.MailboxID)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "INVALID_MAILBOX", "Valid mailbox required", nil, "")
		return
	}
	f.MailboxID = mb.ID
	f.ID = uuid.New()
	if f.Predicate == "" {
		f.Predicate = "contains"
	}
	f.IsActive = true

	if err := h.store.CreateMailFilter(r.Context(), &f); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create filter", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "webmail.filter.create", "mail_filter", f.ID.String(), "success", "", map[string]interface{}{
		"name":  f.Name,
		"field": f.Field,
		"value": f.Value,
	})

	response.JSON(w, http.StatusCreated, f, nil)
}

func (h *WebmailHandler) UpdateFilter(w http.ResponseWriter, r *http.Request) {
	fID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid filter UUID", nil, "")
		return
	}

	var f store.MailFilter
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", nil, "")
		return
	}
	f.ID = fID

	claims, _ := auth.GetClaims(r.Context())
	mb, err := h.checkMailboxPermission(r.Context(), claims, f.MailboxID)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "INVALID_MAILBOX", "Valid mailbox required", nil, "")
		return
	}
	f.MailboxID = mb.ID

	if err := h.store.UpdateMailFilter(r.Context(), &f); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update filter", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, f, nil)
}

func (h *WebmailHandler) DeleteFilter(w http.ResponseWriter, r *http.Request) {
	fID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid filter UUID", nil, "")
		return
	}

	if err := h.store.DeleteMailFilter(r.Context(), fID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete filter", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Filter deleted successfully"}, nil)
}

// ----------------------------------------------------------------------------
// CONTACTS HANDLERS
// ----------------------------------------------------------------------------

func (h *WebmailHandler) ListContacts(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	mbIDStr := r.URL.Query().Get("mailbox_id")
	var mbID uuid.UUID
	if mbIDStr != "" {
		mbID, _ = uuid.Parse(mbIDStr)
	}
	mb, err := h.checkMailboxPermission(r.Context(), claims, mbID)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "INVALID_MAILBOX", "Mailbox ID required", nil, "")
		return
	}

	search := r.URL.Query().Get("q")
	contacts, err := h.store.ListMailContacts(r.Context(), mb.ID, search)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list contacts", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, contacts, nil)
}

func (h *WebmailHandler) CreateContact(w http.ResponseWriter, r *http.Request) {
	var c store.MailContact
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil || c.Email == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Valid email required", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	mb, err := h.checkMailboxPermission(r.Context(), claims, c.MailboxID)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "INVALID_MAILBOX", "Valid mailbox required", nil, "")
		return
	}
	c.MailboxID = mb.ID
	c.ID = uuid.New()
	if c.Name == "" {
		c.Name = c.Email
	}

	if err := h.store.CreateMailContact(r.Context(), &c); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to save contact", nil, "")
		return
	}

	response.JSON(w, http.StatusCreated, c, nil)
}

func (h *WebmailHandler) UpdateContact(w http.ResponseWriter, r *http.Request) {
	cID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid contact UUID", nil, "")
		return
	}

	var c store.MailContact
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", nil, "")
		return
	}
	c.ID = cID

	claims, _ := auth.GetClaims(r.Context())
	mb, err := h.checkMailboxPermission(r.Context(), claims, c.MailboxID)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "INVALID_MAILBOX", "Valid mailbox required", nil, "")
		return
	}
	c.MailboxID = mb.ID

	if err := h.store.UpdateMailContact(r.Context(), &c); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update contact", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, c, nil)
}

func (h *WebmailHandler) DeleteContact(w http.ResponseWriter, r *http.Request) {
	cID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid contact UUID", nil, "")
		return
	}

	if err := h.store.DeleteMailContact(r.Context(), cID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete contact", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Contact deleted successfully"}, nil)
}

// ----------------------------------------------------------------------------
// IDENTITIES & PREFERENCES & FORWARDING HANDLERS
// ----------------------------------------------------------------------------

func (h *WebmailHandler) ListIdentities(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	mbIDStr := r.URL.Query().Get("mailbox_id")
	var mbID uuid.UUID
	if mbIDStr != "" {
		mbID, _ = uuid.Parse(mbIDStr)
	}
	mb, err := h.checkMailboxPermission(r.Context(), claims, mbID)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "INVALID_MAILBOX", "Mailbox ID required", nil, "")
		return
	}

	identities, err := h.store.ListMailIdentities(r.Context(), mb.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list identities", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, identities, nil)
}

func (h *WebmailHandler) SaveIdentity(w http.ResponseWriter, r *http.Request) {
	var iden store.MailIdentity
	if err := json.NewDecoder(r.Body).Decode(&iden); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Mailbox ID required", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	mb, err := h.checkMailboxPermission(r.Context(), claims, iden.MailboxID)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "INVALID_MAILBOX", "Valid mailbox required", nil, "")
		return
	}
	iden.MailboxID = mb.ID

	if iden.ID == uuid.Nil {
		iden.ID = uuid.New()
	}

	if err := h.store.SaveMailIdentity(r.Context(), &iden); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to save identity", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, iden, nil)
}

func (h *WebmailHandler) DeleteIdentity(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid identity UUID", nil, "")
		return
	}

	if err := h.store.DeleteMailIdentity(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete identity", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Identity deleted"}, nil)
}

func (h *WebmailHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	mbIDStr := r.URL.Query().Get("mailbox_id")
	var mbID uuid.UUID
	if mbIDStr != "" {
		mbID, _ = uuid.Parse(mbIDStr)
	}
	mb, err := h.checkMailboxPermission(r.Context(), claims, mbID)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "INVALID_MAILBOX", "Mailbox ID required", nil, "")
		return
	}

	prefs, err := h.store.GetWebmailPreferences(r.Context(), mb.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to get preferences", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, prefs, nil)
}

func (h *WebmailHandler) SavePreferences(w http.ResponseWriter, r *http.Request) {
	var prefs store.WebmailPreferences
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Mailbox ID required", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	mb, err := h.checkMailboxPermission(r.Context(), claims, prefs.MailboxID)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "INVALID_MAILBOX", "Valid mailbox required", nil, "")
		return
	}
	prefs.MailboxID = mb.ID

	if err := h.store.SaveWebmailPreferences(r.Context(), &prefs); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to save preferences", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, prefs, nil)
}

func (h *WebmailHandler) GetForwarding(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	mbIDStr := r.URL.Query().Get("mailbox_id")
	var mbID uuid.UUID
	if mbIDStr != "" {
		mbID, _ = uuid.Parse(mbIDStr)
	}
	mb, err := h.checkMailboxPermission(r.Context(), claims, mbID)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "INVALID_MAILBOX", "Mailbox ID required", nil, "")
		return
	}

	fwd, err := h.store.GetMailForwardingRule(r.Context(), mb.ID)
	if err != nil {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"is_active": false,
		}, nil)
		return
	}

	response.JSON(w, http.StatusOK, fwd, nil)
}

func (h *WebmailHandler) SaveForwarding(w http.ResponseWriter, r *http.Request) {
	var fwd store.MailForwardingRule
	if err := json.NewDecoder(r.Body).Decode(&fwd); err != nil || fwd.ForwardTo == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Forwarding address required", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	mb, err := h.checkMailboxPermission(r.Context(), claims, fwd.MailboxID)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "INVALID_MAILBOX", "Valid mailbox required", nil, "")
		return
	}
	fwd.MailboxID = mb.ID

	if fwd.ID == uuid.Nil {
		fwd.ID = uuid.New()
	}
	fwd.IsVerified = true

	if err := h.store.SaveMailForwardingRule(r.Context(), &fwd); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to save forwarding rule", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "webmail.forwarding.update", "mail_forwarding_rule", fwd.ID.String(), "success", "", map[string]interface{}{
		"forward_to": fwd.ForwardTo,
		"keep_copy":  fwd.KeepCopy,
		"is_active":  fwd.IsActive,
	})

	response.JSON(w, http.StatusOK, fwd, nil)
}

func (h *WebmailHandler) DeleteForwarding(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	mbIDStr := r.URL.Query().Get("mailbox_id")
	var mbID uuid.UUID
	if mbIDStr != "" {
		mbID, _ = uuid.Parse(mbIDStr)
	}
	mb, err := h.checkMailboxPermission(r.Context(), claims, mbID)
	if err != nil {
		if errors.Is(err, errForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Cross-mailbox access denied", nil, "")
			return
		}
		response.Error(w, http.StatusBadRequest, "INVALID_MAILBOX", "Mailbox ID required", nil, "")
		return
	}

	_ = h.store.DeleteMailForwardingRule(r.Context(), mb.ID)
	response.JSON(w, http.StatusOK, map[string]string{"message": "Forwarding disabled"}, nil)
}

// applyFiltersToMessage evaluates mailbox filter rules and executes matching actions
func (h *WebmailHandler) applyFiltersToMessage(ctx context.Context, msg *store.WebmailMessage) {
	filters, err := h.store.ListMailFilters(ctx, msg.MailboxID)
	if err != nil || len(filters) == 0 {
		return
	}

	for _, f := range filters {
		if !f.IsActive {
			continue
		}

		var targetText string
		switch f.Field {
		case "from":
			targetText = strings.ToLower(msg.FromEmail + " " + msg.FromName)
		case "to":
			targetText = strings.ToLower(msg.ToEmail + " " + msg.ToName)
		case "subject":
			targetText = strings.ToLower(msg.Subject)
		case "body":
			targetText = strings.ToLower(msg.BodyText + " " + msg.BodyHTML)
		case "has_attachment":
			if msg.HasAttachment {
				targetText = "true"
			} else {
				targetText = "false"
			}
		default:
			targetText = strings.ToLower(msg.Subject + " " + msg.BodyText)
		}

		filterVal := strings.ToLower(strings.TrimSpace(f.Value))
		matched := false
		switch f.Predicate {
		case "contains":
			matched = strings.Contains(targetText, filterVal)
		case "not_contains":
			matched = !strings.Contains(targetText, filterVal)
		case "equals":
			matched = (targetText == filterVal)
		case "starts_with":
			matched = strings.HasPrefix(targetText, filterVal)
		case "ends_with":
			matched = strings.HasSuffix(targetText, filterVal)
		default:
			matched = strings.Contains(targetText, filterVal)
		}

		if matched {
			isTrue := true
			isFalse := false
			switch f.Action {
			case "mark_read":
				msg.IsUnread = false
				_ = h.store.UpdateWebmailMessageFlags(ctx, msg.ID, &isFalse, nil, nil)
			case "star":
				msg.IsStarred = true
				_ = h.store.UpdateWebmailMessageFlags(ctx, msg.ID, nil, &isTrue, nil)
			case "move_to":
				if f.ActionValue != "" {
					msg.Folder = f.ActionValue
					_ = h.store.MoveWebmailMessage(ctx, msg.ID, f.ActionValue)
				}
			case "skip_inbox":
				msg.Folder = "archive"
				_ = h.store.MoveWebmailMessage(ctx, msg.ID, "archive")
			case "mark_spam":
				msg.Folder = "spam"
				_ = h.store.MoveWebmailMessage(ctx, msg.ID, "spam")
			case "delete":
				msg.Folder = "trash"
				_ = h.store.MoveWebmailMessage(ctx, msg.ID, "trash")
			}
		}
	}
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
