package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/email/migration"
	"hostvra/agent/pkg/email/relay"
	"hostvra/agent/pkg/email/rspamd"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

// EmailDeliverabilityHandler coordinates SMTP relays, Rspamd spam filtering, and IMAP migrations.
type EmailDeliverabilityHandler struct {
	cfg                *config.Config
	store              store.Store
	audit              *audit.Logger
	rspamdMgr          *rspamd.Manager
	activeMigrationsMu sync.RWMutex
	activeMigrations   map[uuid.UUID]context.CancelFunc
}

func NewEmailDeliverabilityHandler(cfg *config.Config, s store.Store, a *audit.Logger) *EmailDeliverabilityHandler {
	return &EmailDeliverabilityHandler{
		cfg:              cfg,
		store:            s,
		audit:            a,
		rspamdMgr:        rspamd.NewManager("/etc/rspamd"),
		activeMigrations: make(map[uuid.UUID]context.CancelFunc),
	}
}

func (h *EmailDeliverabilityHandler) SetRspamdManager(m *rspamd.Manager) {
	h.rspamdMgr = m
}

// Helper: verify organization claims

func (h *EmailDeliverabilityHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	response.JSON(w, status, data, nil)
}

func (h *EmailDeliverabilityHandler) respondError(w http.ResponseWriter, status int, code, msg string) {
	response.Error(w, status, code, msg, nil, "")
}

func (h *EmailDeliverabilityHandler) getClaims(r *http.Request) (*auth.Claims, error) {
	claims, _ := auth.GetClaims(r.Context())
	if claims == nil {
		return nil, errors.New("unauthorized: missing or invalid credentials")
	}
	return claims, nil
}

// ============================================================================
// 1. EXTERNAL SMTP RELAY ENDPOINTS
// ============================================================================

// ListRelays handles GET /api/v1/email/relays
func (h *EmailDeliverabilityHandler) ListRelays(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	var domainID *uuid.UUID
	if dStr := r.URL.Query().Get("domain_id"); dStr != "" {
		if dUUID, err := uuid.Parse(dStr); err == nil {
			domainID = &dUUID
		}
	}

	relays, err := h.store.ListEmailSMTPRelays(r.Context(), claims.OrganizationID, domainID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list SMTP relays")
		return
	}

	// Mask passwords in response
	for _, rel := range relays {
		if rel.PasswordEncrypted != "" {
			rel.PasswordEncrypted = "••••••••"
		}
	}

	h.respondJSON(w, http.StatusOK, relays)
}

// GetRelay handles GET /api/v1/email/relays/{id}
func (h *EmailDeliverabilityHandler) GetRelay(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid relay ID")
		return
	}

	relayRec, err := h.store.GetEmailSMTPRelayByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "NOT_FOUND", "SMTP relay not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch SMTP relay")
		return
	}

	if relayRec.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		h.respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
		return
	}

	if relayRec.PasswordEncrypted != "" {
		relayRec.PasswordEncrypted = "••••••••"
	}
	h.respondJSON(w, http.StatusOK, relayRec)
}

type CreateRelayRequest struct {
	DomainID           *uuid.UUID `json:"domain_id,omitempty"`
	Name               string     `json:"name"`
	Provider           string     `json:"provider"`
	Host               string     `json:"host"`
	Port               int        `json:"port"`
	Username           string     `json:"username"`
	Password           string     `json:"password"`
	AuthType           string     `json:"auth_type"`
	Encryption         string     `json:"encryption"`
	SenderDomain       string     `json:"sender_domain"`
	RateLimitPerSecond int        `json:"rate_limit_per_second"`
	RateLimitPerDay    int        `json:"rate_limit_per_day"`
	TestOnSave         bool       `json:"test_on_save"`
}

// CreateRelay handles POST /api/v1/email/relays
func (h *EmailDeliverabilityHandler) CreateRelay(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	var req CreateRelayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	cleanHost := strings.TrimSpace(req.Host)
	if cleanHost == "" {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "SMTP relay host is required")
		return
	}
	if req.Port <= 0 {
		req.Port = 587
	}
	if req.Provider == "" {
		req.Provider = "custom"
	}
	if req.Encryption == "" {
		req.Encryption = "starttls"
	}
	if req.AuthType == "" {
		req.AuthType = "plain"
	}
	if req.Name == "" {
		req.Name = fmt.Sprintf("%s Relay", req.Provider)
	}

	relayRec := &store.EmailSMTPRelayRecord{
		ID:                 uuid.New(),
		OrganizationID:     claims.OrganizationID,
		DomainID:           req.DomainID,
		Name:               req.Name,
		Provider:           req.Provider,
		Host:               cleanHost,
		Port:               req.Port,
		Username:           strings.TrimSpace(req.Username),
		PasswordEncrypted:  req.Password,
		AuthType:           req.AuthType,
		Encryption:         req.Encryption,
		SenderDomain:       strings.TrimSpace(strings.ToLower(req.SenderDomain)),
		Status:             "active",
		RateLimitPerSecond: req.RateLimitPerSecond,
		RateLimitPerDay:    req.RateLimitPerDay,
		LastTestStatus:     "pending",
	}

	// Test connection if requested
	if req.TestOnSave {
		testCtx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
		defer cancel()
		tRes := relay.TestRelayConnection(testCtx, relayRec.Host, relayRec.Port, relayRec.Username, req.Password, relayRec.Encryption, relayRec.AuthType)
		now := time.Now().UTC()
		relayRec.LastTestAt = &now
		if tRes.Success {
			relayRec.LastTestStatus = "success"
		} else {
			relayRec.LastTestStatus = "failed"
			relayRec.LastTestError = tRes.Error
		}
	}

	if err := h.store.CreateEmailSMTPRelay(r.Context(), relayRec); err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to save SMTP relay")
		return
	}

	h.audit.Log(r.Context(), r, "create", "email_smtp_relay", relayRec.ID.String(), "success", fmt.Sprintf("Configured external SMTP relay %s (%s:%d)", relayRec.Name, relayRec.Host, relayRec.Port), nil)

	// Return masked
	relayRec.PasswordEncrypted = "••••••••"
	h.respondJSON(w, http.StatusCreated, relayRec)
}

// UpdateRelay handles PUT /api/v1/email/relays/{id}
func (h *EmailDeliverabilityHandler) UpdateRelay(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid relay ID")
		return
	}

	existing, err := h.store.GetEmailSMTPRelayByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "NOT_FOUND", "SMTP relay not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch relay")
		return
	}

	if existing.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		h.respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
		return
	}

	var req CreateRelayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Host != "" {
		existing.Host = strings.TrimSpace(req.Host)
	}
	if req.Port > 0 {
		existing.Port = req.Port
	}
	if req.Username != "" {
		existing.Username = strings.TrimSpace(req.Username)
	}
	// Only update password if new non-masked password is provided
	if req.Password != "" && req.Password != "••••••••" {
		existing.PasswordEncrypted = req.Password
	}
	if req.AuthType != "" {
		existing.AuthType = req.AuthType
	}
	if req.Encryption != "" {
		existing.Encryption = req.Encryption
	}
	if req.SenderDomain != "" {
		existing.SenderDomain = strings.TrimSpace(strings.ToLower(req.SenderDomain))
	}
	existing.DomainID = req.DomainID
	existing.RateLimitPerSecond = req.RateLimitPerSecond
	existing.RateLimitPerDay = req.RateLimitPerDay

	if err := h.store.UpdateEmailSMTPRelay(r.Context(), existing); err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update relay")
		return
	}

	h.audit.Log(r.Context(), r, "update", "email_smtp_relay", existing.ID.String(), "success", fmt.Sprintf("Updated external SMTP relay %s", existing.Name), nil)

	existing.PasswordEncrypted = "••••••••"
	h.respondJSON(w, http.StatusOK, existing)
}

// DeleteRelay handles DELETE /api/v1/email/relays/{id}
func (h *EmailDeliverabilityHandler) DeleteRelay(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid relay ID")
		return
	}

	existing, err := h.store.GetEmailSMTPRelayByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "NOT_FOUND", "SMTP relay not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch relay")
		return
	}

	if existing.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		h.respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
		return
	}

	if err := h.store.DeleteEmailSMTPRelay(r.Context(), id); err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete relay")
		return
	}

	h.audit.Log(r.Context(), r, "delete", "email_smtp_relay", id.String(), "success", fmt.Sprintf("Removed external SMTP relay %s", existing.Name), nil)
	h.respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Relay removed"})
}

// TestRelayConnection handles POST /api/v1/email/relays/{id}/test
func (h *EmailDeliverabilityHandler) TestRelayConnection(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid relay ID")
		return
	}

	relayRec, err := h.store.GetEmailSMTPRelayByID(r.Context(), id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "NOT_FOUND", "SMTP relay not found")
		return
	}
	if relayRec.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		h.respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
		return
	}

	testCtx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	result := relay.TestRelayConnection(testCtx, relayRec.Host, relayRec.Port, relayRec.Username, relayRec.PasswordEncrypted, relayRec.Encryption, relayRec.AuthType)

	now := time.Now().UTC()
	relayRec.LastTestAt = &now
	if result.Success {
		relayRec.LastTestStatus = "success"
		relayRec.LastTestError = ""
	} else {
		relayRec.LastTestStatus = "failed"
		relayRec.LastTestError = result.Error
	}
	_ = h.store.UpdateEmailSMTPRelay(r.Context(), relayRec)

	h.respondJSON(w, http.StatusOK, result)
}

// VerifyDomainDeliverability handles POST /api/v1/email/relays/{id}/verify-dns
func (h *EmailDeliverabilityHandler) VerifyDomainDeliverability(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid relay ID")
		return
	}

	relayRec, err := h.store.GetEmailSMTPRelayByID(r.Context(), id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "NOT_FOUND", "SMTP relay not found")
		return
	}
	if relayRec.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		h.respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
		return
	}

	testCtx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	auditRes := relay.VerifyDomainDeliverability(testCtx, relayRec.SenderDomain, relayRec.Provider, "")

	// Update verification status in database
	relayRec.IsVerified = auditRes.IsFullyVerified
	if auditRes.IsFullyVerified {
		now := time.Now().UTC()
		relayRec.VerifiedAt = &now
	}
	_ = h.store.UpdateEmailSMTPRelay(r.Context(), relayRec)

	h.respondJSON(w, http.StatusOK, auditRes)
}

// ============================================================================
// 2. ANTI-SPAM FILTER & QUARANTINE ENDPOINTS
// ============================================================================

// GetSpamSettings handles GET /api/v1/email/spam/settings
func (h *EmailDeliverabilityHandler) GetSpamSettings(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	var domainID *uuid.UUID
	if dStr := r.URL.Query().Get("domain_id"); dStr != "" {
		if dUUID, err := uuid.Parse(dStr); err == nil {
			domainID = &dUUID
		}
	}

	settings, err := h.store.GetEmailSpamSettings(r.Context(), claims.OrganizationID, domainID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve spam settings")
		return
	}
	h.respondJSON(w, http.StatusOK, settings)
}

// UpdateSpamSettings handles PUT /api/v1/email/spam/settings
func (h *EmailDeliverabilityHandler) UpdateSpamSettings(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	var req store.EmailSpamFilterSettingsRecord
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	req.OrganizationID = claims.OrganizationID
	if req.RejectScore <= 0 {
		req.RejectScore = 15.0
	}
	if req.AddHeaderScore <= 0 {
		req.AddHeaderScore = 6.0
	}
	if req.GreylistScore <= 0 {
		req.GreylistScore = 4.0
	}

	if err := h.store.SaveEmailSpamSettings(r.Context(), &req); err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to save spam settings")
		return
	}

	// Apply thresholds to Rspamd daemon configuration
	_ = h.rspamdMgr.UpdateThresholds(r.Context(), req.RejectScore, req.AddHeaderScore, req.GreylistScore)

	h.audit.Log(r.Context(), r, "update", "email_spam_settings", req.ID.String(), "success", fmt.Sprintf("Updated anti-spam scores: reject=%.1f, header=%.1f, greylist=%.1f", req.RejectScore, req.AddHeaderScore, req.GreylistScore), nil)
	h.respondJSON(w, http.StatusOK, req)
}

// GetSpamTelemetry handles GET /api/v1/email/spam/telemetry
func (h *EmailDeliverabilityHandler) GetSpamTelemetry(w http.ResponseWriter, r *http.Request) {
	_, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	stats, err := h.rspamdMgr.GetTelemetry(r.Context())
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch Rspamd statistics")
		return
	}
	h.respondJSON(w, http.StatusOK, stats)
}

// ListQuarantine handles GET /api/v1/email/spam/quarantine
func (h *EmailDeliverabilityHandler) ListQuarantine(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	var domainID *uuid.UUID
	if dStr := r.URL.Query().Get("domain_id"); dStr != "" {
		if dUUID, err := uuid.Parse(dStr); err == nil {
			domainID = &dUUID
		}
	}

	items, err := h.store.ListSpamQuarantineItems(r.Context(), claims.OrganizationID, domainID, 100)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list quarantine items")
		return
	}
	h.respondJSON(w, http.StatusOK, items)
}

// ReleaseQuarantineItem handles POST /api/v1/email/spam/quarantine/{id}/release
func (h *EmailDeliverabilityHandler) ReleaseQuarantineItem(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid item ID")
		return
	}

	item, err := h.store.GetSpamQuarantineItemByID(r.Context(), id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "NOT_FOUND", "Quarantine item not found")
		return
	}
	if item.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		h.respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
		return
	}

	relResult := h.rspamdMgr.ReleaseQuarantinedMessage(r.Context(), item.MailPath, item.Recipient)
	if !relResult.Success {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", fmt.Sprintf("Failed to release quarantined message: %s", relResult.Error))
		return
	}

	_ = h.store.UpdateSpamQuarantineItemStatus(r.Context(), id, "released")
	h.audit.Log(r.Context(), r, "release", "email_spam_quarantine", id.String(), "success", fmt.Sprintf("Released quarantined email to %s", item.Recipient), nil)

	h.respondJSON(w, http.StatusOK, relResult)
}

// LearnHamQuarantineItem handles POST /api/v1/email/spam/quarantine/{id}/learn-ham
func (h *EmailDeliverabilityHandler) LearnHamQuarantineItem(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid item ID")
		return
	}

	item, err := h.store.GetSpamQuarantineItemByID(r.Context(), id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "NOT_FOUND", "Quarantine item not found")
		return
	}
	if item.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		h.respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
		return
	}

	_ = h.rspamdMgr.LearnHam(r.Context(), item.MailPath)
	_ = h.store.UpdateSpamQuarantineItemStatus(r.Context(), id, "learned_ham")
	h.audit.Log(r.Context(), r, "learn_ham", "email_spam_quarantine", id.String(), "success", fmt.Sprintf("Trained Bayes classifier as Ham for message %s", item.MessageID), nil)

	h.respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Bayesian filter trained as Ham"})
}

// DeleteQuarantineItem handles DELETE /api/v1/email/spam/quarantine/{id}
func (h *EmailDeliverabilityHandler) DeleteQuarantineItem(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid item ID")
		return
	}

	item, err := h.store.GetSpamQuarantineItemByID(r.Context(), id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "NOT_FOUND", "Quarantine item not found")
		return
	}
	if item.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		h.respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
		return
	}

	if err := h.store.DeleteSpamQuarantineItem(r.Context(), id); err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete item")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Quarantined item deleted"})
}

// ============================================================================
// 3. LIVE IMAP MAILBOX MIGRATION ENDPOINTS
// ============================================================================

// ListMigrations handles GET /api/v1/email/migrations
func (h *EmailDeliverabilityHandler) ListMigrations(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	migs, err := h.store.ListIMAPMigrations(r.Context(), claims.OrganizationID, 50)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list migrations")
		return
	}

	// Mask source passwords
	for _, m := range migs {
		if m.SourcePasswordEncrypted != "" {
			m.SourcePasswordEncrypted = "••••••••"
		}
	}

	h.respondJSON(w, http.StatusOK, migs)
}

// TestSourceIMAP handles POST /api/v1/email/migrations/test-source
func (h *EmailDeliverabilityHandler) TestSourceIMAP(w http.ResponseWriter, r *http.Request) {
	_, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	var opts migration.SourceConnectionOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	testCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	result := migration.TestSourceConnection(testCtx, opts)
	h.respondJSON(w, http.StatusOK, result)
}

type StartMigrationRequest struct {
	DestinationMailboxID uuid.UUID                       `json:"destination_mailbox_id"`
	Source               migration.SourceConnectionOptions `json:"source"`
}

// StartMigration handles POST /api/v1/email/migrations
func (h *EmailDeliverabilityHandler) StartMigration(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	var req StartMigrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	if req.DestinationMailboxID == uuid.Nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "destination_mailbox_id is required")
		return
	}
	if req.Source.Host == "" || req.Source.Username == "" || req.Source.Password == "" {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Source host, username, and password are required")
		return
	}
	if req.Source.Port <= 0 {
		if req.Source.UseSSL {
			req.Source.Port = 993
		} else {
			req.Source.Port = 143
		}
	}

	// Verify mailbox exists
	mb, err := h.store.GetEmailMailboxByID(r.Context(), req.DestinationMailboxID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "NOT_FOUND", "Destination mailbox not found")
		return
	}

	record := &store.EmailIMAPMigrationRecord{
		ID:                      uuid.New(),
		OrganizationID:          claims.OrganizationID,
		DestinationMailboxID:    req.DestinationMailboxID,
		DestinationEmail:        mb.Email,
		SourceHost:              strings.TrimSpace(req.Source.Host),
		SourcePort:              req.Source.Port,
		SourceUsername:          strings.TrimSpace(req.Source.Username),
		SourcePasswordEncrypted: req.Source.Password,
		SourceUseSSL:            req.Source.UseSSL,
		Status:                  "pending",
	}

	if err := h.store.CreateIMAPMigration(r.Context(), record); err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create migration record")
		return
	}

	// Spawn asynchronous transfer engine with cancellation context
	migCtx, cancel := context.WithCancel(context.Background())
	h.activeMigrationsMu.Lock()
	h.activeMigrations[record.ID] = cancel
	h.activeMigrationsMu.Unlock()

	task := migration.MigrationTask{
		ID:                     record.ID.String(),
		OrganizationID:         claims.OrganizationID.String(),
		DestinationMailboxID:   mb.ID.String(),
		DestinationEmail:       mb.Email,
		DestinationMaildirBase: filepath.Join("/var/mail/vhosts"),
		Source:                 req.Source,
	}

	go func(migID uuid.UUID, t migration.MigrationTask) {
		defer func() {
			h.activeMigrationsMu.Lock()
			delete(h.activeMigrations, migID)
			h.activeMigrationsMu.Unlock()
		}()

		_ = migration.RunMigration(migCtx, t, func(p migration.MigrationProgress) {
			m, err := h.store.GetIMAPMigrationByID(context.Background(), migID)
			if err == nil && m != nil {
				m.Status = p.Status
				m.TotalFolders = p.TotalFolders
				m.CurrentFolder = p.CurrentFolder
				m.TotalMessages = p.TotalMessages
				m.SyncedMessages = p.SyncedMessages
				m.SkippedDuplicates = p.SkippedDuplicates
				m.FailedMessages = p.FailedMessages
				m.BytesTransferred = p.BytesTransferred
				m.ErrorMessage = p.ErrorMessage
				now := time.Now().UTC()
				if p.Status == "running" && m.StartedAt == nil {
					m.StartedAt = &now
				}
				if p.Status == "completed" || p.Status == "failed" || p.Status == "cancelled" {
					m.CompletedAt = &now
				}
				_ = h.store.UpdateIMAPMigration(context.Background(), m)
			}
		})
	}(record.ID, task)

	h.audit.Log(r.Context(), r, "start_migration", "email_imap_migration", record.ID.String(), "success", fmt.Sprintf("Started IMAP migration into %s from %s", mb.Email, req.Source.Host), nil)

	record.SourcePasswordEncrypted = "••••••••"
	h.respondJSON(w, http.StatusAccepted, record)
}

// GetMigration handles GET /api/v1/email/migrations/{id}
func (h *EmailDeliverabilityHandler) GetMigration(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid migration ID")
		return
	}

	record, err := h.store.GetIMAPMigrationByID(r.Context(), id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "NOT_FOUND", "Migration not found")
		return
	}
	if record.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		h.respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
		return
	}

	record.SourcePasswordEncrypted = "••••••••"
	h.respondJSON(w, http.StatusOK, record)
}

// CancelMigration handles POST /api/v1/email/migrations/{id}/cancel
func (h *EmailDeliverabilityHandler) CancelMigration(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid migration ID")
		return
	}

	record, err := h.store.GetIMAPMigrationByID(r.Context(), id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "NOT_FOUND", "Migration not found")
		return
	}
	if record.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		h.respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
		return
	}

	h.activeMigrationsMu.RLock()
	cancel, ok := h.activeMigrations[id]
	h.activeMigrationsMu.RUnlock()

	if ok && cancel != nil {
		cancel()
	}

	record.Status = "cancelled"
	now := time.Now().UTC()
	record.CompletedAt = &now
	_ = h.store.UpdateIMAPMigration(r.Context(), record)

	h.audit.Log(r.Context(), r, "cancel_migration", "email_imap_migration", id.String(), "success", fmt.Sprintf("Cancelled IMAP migration into %s", record.DestinationEmail), nil)
	h.respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Migration cancelled"})
}

// DeleteMigration handles DELETE /api/v1/email/migrations/{id}
func (h *EmailDeliverabilityHandler) DeleteMigration(w http.ResponseWriter, r *http.Request) {
	claims, err := h.getClaims(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid migration ID")
		return
	}

	record, err := h.store.GetIMAPMigrationByID(r.Context(), id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "NOT_FOUND", "Migration not found")
		return
	}
	if record.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		h.respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
		return
	}

	if err := h.store.DeleteIMAPMigration(r.Context(), id); err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete migration")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Migration history deleted"})
}
