package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MEMORY STORE EMAIL IMPLEMENTATION
// ============================================================================

// Email Domains

func (m *MemoryStore) CreateEmailDomain(ctx context.Context, domain *EmailDomain) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, d := range m.emailDomains {
		if d.ServerID == domain.ServerID && d.Domain == domain.Domain && d.DeletedAt == nil {
			return ErrAlreadyExists
		}
	}

	if domain.ID == uuid.Nil {
		domain.ID = uuid.New()
	}
	now := time.Now().UTC()
	domain.CreatedAt = now
	domain.UpdatedAt = now
	if domain.Status == "" {
		domain.Status = "active"
	}
	if domain.StorageLimitBytes == 0 {
		domain.StorageLimitBytes = 53687091200 // 50GB
	}
	if domain.DKIMSelector == "" {
		domain.DKIMSelector = "default"
	}
	if domain.SpamThreshold == 0 {
		domain.SpamThreshold = 6.0
	}

	m.emailDomains[domain.ID] = domain
	return nil
}

func (m *MemoryStore) GetEmailDomainByID(ctx context.Context, id uuid.UUID) (*EmailDomain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	d, exists := m.emailDomains[id]
	if !exists || d.DeletedAt != nil {
		return nil, ErrNotFound
	}
	res := *d
	// Count mailboxes and aliases
	for _, mb := range m.emailMailboxes {
		if mb.DomainID == d.ID && mb.DeletedAt == nil {
			res.MailboxCount++
		}
	}
	for _, al := range m.emailAliases {
		if al.DomainID == d.ID {
			res.AliasCount++
		}
	}
	return &res, nil
}

func (m *MemoryStore) GetEmailDomainByName(ctx context.Context, serverID uuid.UUID, domainName string) (*EmailDomain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, d := range m.emailDomains {
		if d.ServerID == serverID && d.Domain == domainName && d.DeletedAt == nil {
			res := *d
			return &res, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) ListEmailDomainsByOrg(ctx context.Context, orgID uuid.UUID) ([]*EmailDomain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*EmailDomain, 0)
	for _, d := range m.emailDomains {
		if d.OrganizationID == orgID && d.DeletedAt == nil {
			res := *d
			for _, mb := range m.emailMailboxes {
				if mb.DomainID == d.ID && mb.DeletedAt == nil {
					res.MailboxCount++
				}
			}
			for _, al := range m.emailAliases {
				if al.DomainID == d.ID {
					res.AliasCount++
				}
			}
			list = append(list, &res)
		}
	}
	return list, nil
}

func (m *MemoryStore) ListEmailDomainsByServer(ctx context.Context, serverID uuid.UUID) ([]*EmailDomain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*EmailDomain, 0)
	for _, d := range m.emailDomains {
		if d.ServerID == serverID && d.DeletedAt == nil {
			res := *d
			list = append(list, &res)
		}
	}
	return list, nil
}

func (m *MemoryStore) UpdateEmailDomain(ctx context.Context, domain *EmailDomain) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	d, exists := m.emailDomains[domain.ID]
	if !exists || d.DeletedAt != nil {
		return ErrNotFound
	}
	domain.UpdatedAt = time.Now().UTC()
	m.emailDomains[domain.ID] = domain
	return nil
}

func (m *MemoryStore) DeleteEmailDomain(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	d, exists := m.emailDomains[id]
	if !exists || d.DeletedAt != nil {
		return ErrNotFound
	}
	now := time.Now().UTC()
	d.DeletedAt = &now
	d.UpdatedAt = now

	// Soft-delete associated mailboxes
	for _, mb := range m.emailMailboxes {
		if mb.DomainID == id && mb.DeletedAt == nil {
			mb.DeletedAt = &now
			mb.UpdatedAt = now
		}
	}
	return nil
}

// Email Mailboxes

func (m *MemoryStore) CreateEmailMailbox(ctx context.Context, mb *EmailMailbox) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.emailMailboxes {
		if existing.Email == mb.Email && existing.DeletedAt == nil {
			return ErrAlreadyExists
		}
	}

	if mb.ID == uuid.Nil {
		mb.ID = uuid.New()
	}
	now := time.Now().UTC()
	mb.CreatedAt = now
	mb.UpdatedAt = now
	mb.IsActive = true
	if mb.QuotaBytes == 0 {
		mb.QuotaBytes = 5368709120 // 5GB default
	}

	m.emailMailboxes[mb.ID] = mb
	return nil
}

func (m *MemoryStore) GetEmailMailboxByID(ctx context.Context, id uuid.UUID) (*EmailMailbox, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mb, exists := m.emailMailboxes[id]
	if !exists || mb.DeletedAt != nil {
		return nil, ErrNotFound
	}
	return mb, nil
}

func (m *MemoryStore) GetEmailMailboxByEmail(ctx context.Context, email string) (*EmailMailbox, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, mb := range m.emailMailboxes {
		if mb.Email == email && mb.DeletedAt == nil {
			return mb, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) ListEmailMailboxesByDomain(ctx context.Context, domainID uuid.UUID) ([]*EmailMailbox, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*EmailMailbox, 0)
	for _, mb := range m.emailMailboxes {
		if mb.DomainID == domainID && mb.DeletedAt == nil {
			list = append(list, mb)
		}
	}
	return list, nil
}

func (m *MemoryStore) ListEmailMailboxesByServer(ctx context.Context, serverID uuid.UUID) ([]*EmailMailbox, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*EmailMailbox, 0)
	for _, mb := range m.emailMailboxes {
		if mb.ServerID == serverID && mb.DeletedAt == nil {
			list = append(list, mb)
		}
	}
	return list, nil
}

func (m *MemoryStore) UpdateEmailMailbox(ctx context.Context, mb *EmailMailbox) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.emailMailboxes[mb.ID]
	if !exists || existing.DeletedAt != nil {
		return ErrNotFound
	}
	mb.UpdatedAt = time.Now().UTC()
	// Preserve password hash if empty in update
	if mb.PasswordHash == "" {
		mb.PasswordHash = existing.PasswordHash
	}
	m.emailMailboxes[mb.ID] = mb
	return nil
}

func (m *MemoryStore) UpdateEmailMailboxPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mb, exists := m.emailMailboxes[id]
	if !exists || mb.DeletedAt != nil {
		return ErrNotFound
	}
	mb.PasswordHash = passwordHash
	mb.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *MemoryStore) DeleteEmailMailbox(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mb, exists := m.emailMailboxes[id]
	if !exists || mb.DeletedAt != nil {
		return ErrNotFound
	}
	now := time.Now().UTC()
	mb.DeletedAt = &now
	mb.UpdatedAt = now
	return nil
}

// Email Aliases

func (m *MemoryStore) CreateEmailAlias(ctx context.Context, alias *EmailAlias) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, a := range m.emailAliases {
		if a.DomainID == alias.DomainID && a.SourceAddress == alias.SourceAddress && a.DestinationAddress == alias.DestinationAddress {
			return ErrAlreadyExists
		}
	}

	if alias.ID == uuid.Nil {
		alias.ID = uuid.New()
	}
	now := time.Now().UTC()
	alias.CreatedAt = now
	alias.UpdatedAt = now
	alias.IsActive = true

	m.emailAliases[alias.ID] = alias
	return nil
}

func (m *MemoryStore) ListEmailAliasesByDomain(ctx context.Context, domainID uuid.UUID) ([]*EmailAlias, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*EmailAlias, 0)
	for _, a := range m.emailAliases {
		if a.DomainID == domainID {
			list = append(list, a)
		}
	}
	return list, nil
}

func (m *MemoryStore) DeleteEmailAlias(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.emailAliases[id]; !exists {
		return ErrNotFound
	}
	delete(m.emailAliases, id)
	return nil
}

// Email Forwarders

func (m *MemoryStore) CreateEmailForwarder(ctx context.Context, fwd *EmailForwarder) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if fwd.ID == uuid.Nil {
		fwd.ID = uuid.New()
	}
	fwd.CreatedAt = time.Now().UTC()
	fwd.IsActive = true

	m.emailForwarders[fwd.ID] = fwd
	return nil
}

func (m *MemoryStore) ListEmailForwardersByDomain(ctx context.Context, domainID uuid.UUID) ([]*EmailForwarder, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*EmailForwarder, 0)
	for _, f := range m.emailForwarders {
		if f.DomainID == domainID {
			list = append(list, f)
		}
	}
	return list, nil
}

func (m *MemoryStore) DeleteEmailForwarder(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.emailForwarders[id]; !exists {
		return ErrNotFound
	}
	delete(m.emailForwarders, id)
	return nil
}

// Email Autoresponders

func (m *MemoryStore) SetEmailAutoresponder(ctx context.Context, ar *EmailAutoresponder) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if existing, exists := m.emailAutoresponders[ar.MailboxID]; exists {
		ar.ID = existing.ID
		ar.CreatedAt = existing.CreatedAt
		ar.UpdatedAt = now
	} else {
		if ar.ID == uuid.Nil {
			ar.ID = uuid.New()
		}
		ar.CreatedAt = now
		ar.UpdatedAt = now
	}
	m.emailAutoresponders[ar.MailboxID] = ar
	return nil
}

func (m *MemoryStore) GetEmailAutoresponderByMailbox(ctx context.Context, mailboxID uuid.UUID) (*EmailAutoresponder, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ar, exists := m.emailAutoresponders[mailboxID]
	if !exists {
		return nil, ErrNotFound
	}
	return ar, nil
}

func (m *MemoryStore) DeleteEmailAutoresponder(ctx context.Context, mailboxID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.emailAutoresponders[mailboxID]; !exists {
		return ErrNotFound
	}
	delete(m.emailAutoresponders, mailboxID)
	return nil
}

// DKIM Keys

func (m *MemoryStore) SaveEmailDKIMKey(ctx context.Context, dkim *EmailDKIMKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if dkim.ID == uuid.Nil {
		dkim.ID = uuid.New()
	}
	dkim.CreatedAt = now
	dkim.UpdatedAt = now
	if dkim.Selector == "" {
		dkim.Selector = "default"
	}
	if dkim.KeySize == 0 {
		dkim.KeySize = 2048
	}

	m.emailDKIMKeys[dkim.DomainID] = dkim
	return nil
}

func (m *MemoryStore) GetEmailDKIMKeyByDomain(ctx context.Context, domainID uuid.UUID) (*EmailDKIMKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dkim, exists := m.emailDKIMKeys[domainID]
	if !exists {
		return nil, ErrNotFound
	}
	return dkim, nil
}

// Delivery Logs

func (m *MemoryStore) RecordEmailDeliveryLog(ctx context.Context, log *EmailDeliveryLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}
	m.emailDeliveryLogs = append(m.emailDeliveryLogs, log)
	return nil
}

func (m *MemoryStore) ListEmailDeliveryLogs(ctx context.Context, serverID uuid.UUID, limit int) ([]*EmailDeliveryLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > 500 {
		limit = 100
	}
	logs := make([]*EmailDeliveryLog, 0)
	for i := len(m.emailDeliveryLogs) - 1; i >= 0; i-- {
		l := m.emailDeliveryLogs[i]
		if l.ServerID == serverID {
			logs = append(logs, l)
			if len(logs) >= limit {
				break
			}
		}
	}
	return logs, nil
}

// Email Signatures

func (m *MemoryStore) GetEmailSignature(ctx context.Context, mailboxID uuid.UUID) (*EmailSignature, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sig, ok := m.emailSignatures[mailboxID]
	if !ok {
		return &EmailSignature{
			ID:        uuid.New(),
			MailboxID: mailboxID,
			IsEnabled: false,
		}, nil
	}
	res := *sig
	return &res, nil
}

func (m *MemoryStore) SetEmailSignature(ctx context.Context, sig *EmailSignature) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sig.ID == uuid.Nil {
		sig.ID = uuid.New()
	}
	now := time.Now().UTC()
	if sig.CreatedAt.IsZero() {
		sig.CreatedAt = now
	}
	sig.UpdatedAt = now
	m.emailSignatures[sig.MailboxID] = sig
	return nil
}

// Email Suppressions

func (m *MemoryStore) AddEmailSuppression(ctx context.Context, sup *EmailSuppression) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.emailSuppressions {
		if s.ServerID == sup.ServerID && strings.EqualFold(s.Email, sup.Email) {
			s.Reason = sup.Reason
			s.BounceCode = sup.BounceCode
			s.Metadata = sup.Metadata
			return nil
		}
	}
	if sup.ID == uuid.Nil {
		sup.ID = uuid.New()
	}
	if sup.CreatedAt.IsZero() {
		sup.CreatedAt = time.Now().UTC()
	}
	m.emailSuppressions[sup.ID] = sup
	return nil
}

func (m *MemoryStore) ListEmailSuppressions(ctx context.Context, serverID uuid.UUID) ([]*EmailSuppression, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*EmailSuppression, 0)
	for _, s := range m.emailSuppressions {
		if s.ServerID == serverID {
			res := *s
			list = append(list, &res)
		}
	}
	return list, nil
}

func (m *MemoryStore) DeleteEmailSuppression(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.emailSuppressions, id)
	return nil
}

func (m *MemoryStore) IsEmailSuppressed(ctx context.Context, serverID uuid.UUID, email string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.emailSuppressions {
		if s.ServerID == serverID && strings.EqualFold(s.Email, email) {
			return true, nil
		}
	}
	return false, nil
}

// Webmail Messages

func (m *MemoryStore) ListWebmailMessages(ctx context.Context, mailboxID uuid.UUID, folder string, limit, offset int, search string) ([]*WebmailMessage, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	matched := make([]*WebmailMessage, 0)
	for _, msg := range m.webmailMessages {
		if msg.MailboxID == mailboxID {
			if folder == "starred" {
				if !msg.IsStarred {
					continue
				}
			} else if folder == "important" {
				if !msg.IsImportant {
					continue
				}
			} else if folder != "" && folder != "all" && msg.Folder != folder {
				continue
			}
			if search != "" {
				s := strings.ToLower(search)
				if !strings.Contains(strings.ToLower(msg.Subject), s) &&
					!strings.Contains(strings.ToLower(msg.FromEmail), s) &&
					!strings.Contains(strings.ToLower(msg.FromName), s) &&
					!strings.Contains(strings.ToLower(msg.Snippet), s) {
					continue
				}
			}
			res := *msg
			res.Attachments = make([]WebmailAttachment, 0)
			for _, att := range m.webmailAttachments {
				if att.MessageID == msg.ID {
					res.Attachments = append(res.Attachments, *att)
				}
			}
			matched = append(matched, &res)
		}
	}

	for i := 0; i < len(matched); i++ {
		for j := i + 1; j < len(matched); j++ {
			if matched[j].CreatedAt.After(matched[i].CreatedAt) {
				matched[i], matched[j] = matched[j], matched[i]
			}
		}
	}

	total := len(matched)
	if offset > total {
		return []*WebmailMessage{}, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}
	return matched[offset:end], total, nil
}

func (m *MemoryStore) GetWebmailMessageByID(ctx context.Context, id uuid.UUID) (*WebmailMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	msg, ok := m.webmailMessages[id]
	if !ok {
		return nil, ErrNotFound
	}
	res := *msg
	res.Attachments = make([]WebmailAttachment, 0)
	for _, att := range m.webmailAttachments {
		if att.MessageID == msg.ID {
			res.Attachments = append(res.Attachments, *att)
		}
	}
	return &res, nil
}

func (m *MemoryStore) CreateWebmailMessage(ctx context.Context, msg *WebmailMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	now := time.Now().UTC()
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now
	}
	msg.UpdatedAt = now
	if msg.Snippet == "" && msg.BodyText != "" {
		if len(msg.BodyText) > 120 {
			msg.Snippet = msg.BodyText[:120] + "..."
		} else {
			msg.Snippet = msg.BodyText
		}
	}
	m.webmailMessages[msg.ID] = msg
	for _, att := range msg.Attachments {
		att.MessageID = msg.ID
		if att.ID == uuid.Nil {
			att.ID = uuid.New()
		}
		att.CreatedAt = now
		m.webmailAttachments[att.ID] = &att
	}
	return nil
}

func (m *MemoryStore) UpdateWebmailMessageFlags(ctx context.Context, id uuid.UUID, isUnread, isStarred, isImportant *bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.webmailMessages[id]
	if !ok {
		return ErrNotFound
	}
	if isUnread != nil {
		msg.IsUnread = *isUnread
	}
	if isStarred != nil {
		msg.IsStarred = *isStarred
	}
	if isImportant != nil {
		msg.IsImportant = *isImportant
	}
	msg.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *MemoryStore) MoveWebmailMessage(ctx context.Context, id uuid.UUID, targetFolder string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.webmailMessages[id]
	if !ok {
		return ErrNotFound
	}
	msg.Folder = targetFolder
	msg.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *MemoryStore) DeleteWebmailMessage(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.webmailMessages[id]
	if !ok {
		return ErrNotFound
	}
	if msg.Folder == "trash" {
		delete(m.webmailMessages, id)
		for attID, att := range m.webmailAttachments {
			if att.MessageID == id {
				delete(m.webmailAttachments, attID)
			}
		}
	} else {
		msg.Folder = "trash"
		msg.UpdatedAt = time.Now().UTC()
	}
	return nil
}

func (m *MemoryStore) CreateWebmailAttachment(ctx context.Context, att *WebmailAttachment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if att.ID == uuid.Nil {
		att.ID = uuid.New()
	}
	if att.CreatedAt.IsZero() {
		att.CreatedAt = time.Now().UTC()
	}
	m.webmailAttachments[att.ID] = att
	return nil
}

func (m *MemoryStore) ListWebmailAttachments(ctx context.Context, messageID uuid.UUID) ([]*WebmailAttachment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*WebmailAttachment, 0)
	for _, att := range m.webmailAttachments {
		if att.MessageID == messageID {
			res := *att
			list = append(list, &res)
		}
	}
	return list, nil
}

// ============================================================================
// POSTGRES STORE EMAIL IMPLEMENTATION
// ============================================================================

func (p *PostgresStore) CreateEmailDomain(ctx context.Context, domain *EmailDomain) error {
	query := `
		INSERT INTO email_domains (
			id, organization_id, server_id, domain, mail_hostname, status,
			storage_limit_bytes, storage_used_bytes, spam_threshold, dkim_selector, is_catchall_enabled
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at
	`
	if domain.ID == uuid.Nil {
		domain.ID = uuid.New()
	}
	if domain.Status == "" {
		domain.Status = "active"
	}
	if domain.StorageLimitBytes == 0 {
		domain.StorageLimitBytes = 53687091200
	}
	if domain.DKIMSelector == "" {
		domain.DKIMSelector = "default"
	}
	if domain.SpamThreshold == 0 {
		domain.SpamThreshold = 6.0
	}

	return p.db.QueryRowContext(ctx, query,
		domain.ID, domain.OrganizationID, domain.ServerID, domain.Domain,
		domain.MailHostname, domain.Status, domain.StorageLimitBytes,
		domain.StorageUsedBytes, domain.SpamThreshold, domain.DKIMSelector,
		domain.IsCatchallEnabled,
	).Scan(&domain.CreatedAt, &domain.UpdatedAt)
}

func (p *PostgresStore) GetEmailDomainByID(ctx context.Context, id uuid.UUID) (*EmailDomain, error) {
	query := `
		SELECT id, organization_id, server_id, domain, mail_hostname, status,
		       storage_limit_bytes, storage_used_bytes, spam_threshold, dkim_selector,
		       is_catchall_enabled, catchall_mailbox_id, created_at, updated_at, deleted_at
		FROM email_domains
		WHERE id = $1 AND deleted_at IS NULL
	`
	d := &EmailDomain{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID, &d.OrganizationID, &d.ServerID, &d.Domain, &d.MailHostname,
		&d.Status, &d.StorageLimitBytes, &d.StorageUsedBytes, &d.SpamThreshold,
		&d.DKIMSelector, &d.IsCatchallEnabled, &d.CatchallMailboxID,
		&d.CreatedAt, &d.UpdatedAt, &d.DeletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return d, err
}

func (p *PostgresStore) GetEmailDomainByName(ctx context.Context, serverID uuid.UUID, domainName string) (*EmailDomain, error) {
	query := `
		SELECT id, organization_id, server_id, domain, mail_hostname, status,
		       storage_limit_bytes, storage_used_bytes, spam_threshold, dkim_selector,
		       is_catchall_enabled, catchall_mailbox_id, created_at, updated_at, deleted_at
		FROM email_domains
		WHERE server_id = $1 AND domain = $2 AND deleted_at IS NULL
	`
	d := &EmailDomain{}
	err := p.db.QueryRowContext(ctx, query, serverID, domainName).Scan(
		&d.ID, &d.OrganizationID, &d.ServerID, &d.Domain, &d.MailHostname,
		&d.Status, &d.StorageLimitBytes, &d.StorageUsedBytes, &d.SpamThreshold,
		&d.DKIMSelector, &d.IsCatchallEnabled, &d.CatchallMailboxID,
		&d.CreatedAt, &d.UpdatedAt, &d.DeletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return d, err
}

func (p *PostgresStore) ListEmailDomainsByOrg(ctx context.Context, orgID uuid.UUID) ([]*EmailDomain, error) {
	query := `
		SELECT d.id, d.organization_id, d.server_id, d.domain, d.mail_hostname, d.status,
		       d.storage_limit_bytes, d.storage_used_bytes, d.spam_threshold, d.dkim_selector,
		       d.is_catchall_enabled, d.catchall_mailbox_id, d.created_at, d.updated_at,
		       (SELECT COUNT(*) FROM email_mailboxes m WHERE m.domain_id = d.id AND m.deleted_at IS NULL) as mailbox_count,
		       (SELECT COUNT(*) FROM email_aliases a WHERE a.domain_id = d.id) as alias_count
		FROM email_domains d
		WHERE d.organization_id = $1 AND d.deleted_at IS NULL
		ORDER BY d.created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*EmailDomain, 0)
	for rows.Next() {
		d := &EmailDomain{}
		err := rows.Scan(
			&d.ID, &d.OrganizationID, &d.ServerID, &d.Domain, &d.MailHostname,
			&d.Status, &d.StorageLimitBytes, &d.StorageUsedBytes, &d.SpamThreshold,
			&d.DKIMSelector, &d.IsCatchallEnabled, &d.CatchallMailboxID,
			&d.CreatedAt, &d.UpdatedAt, &d.MailboxCount, &d.AliasCount,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, nil
}

func (p *PostgresStore) ListEmailDomainsByServer(ctx context.Context, serverID uuid.UUID) ([]*EmailDomain, error) {
	query := `
		SELECT id, organization_id, server_id, domain, mail_hostname, status,
		       storage_limit_bytes, storage_used_bytes, spam_threshold, dkim_selector,
		       is_catchall_enabled, catchall_mailbox_id, created_at, updated_at
		FROM email_domains
		WHERE server_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*EmailDomain, 0)
	for rows.Next() {
		d := &EmailDomain{}
		err := rows.Scan(
			&d.ID, &d.OrganizationID, &d.ServerID, &d.Domain, &d.MailHostname,
			&d.Status, &d.StorageLimitBytes, &d.StorageUsedBytes, &d.SpamThreshold,
			&d.DKIMSelector, &d.IsCatchallEnabled, &d.CatchallMailboxID,
			&d.CreatedAt, &d.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, nil
}

func (p *PostgresStore) UpdateEmailDomain(ctx context.Context, domain *EmailDomain) error {
	query := `
		UPDATE email_domains
		SET status = $2, storage_limit_bytes = $3, spam_threshold = $4,
		    is_catchall_enabled = $5, catchall_mailbox_id = $6, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	return p.db.QueryRowContext(ctx, query,
		domain.ID, domain.Status, domain.StorageLimitBytes, domain.SpamThreshold,
		domain.IsCatchallEnabled, domain.CatchallMailboxID,
	).Scan(&domain.UpdatedAt)
}

func (p *PostgresStore) DeleteEmailDomain(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE email_domains
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	res, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	// Soft delete mailboxes
	_, _ = p.db.ExecContext(ctx, `UPDATE email_mailboxes SET deleted_at = NOW() WHERE domain_id = $1 AND deleted_at IS NULL`, id)
	return nil
}

// Mailboxes Postgres

func (p *PostgresStore) CreateEmailMailbox(ctx context.Context, mb *EmailMailbox) error {
	query := `
		INSERT INTO email_mailboxes (
			id, domain_id, server_id, local_part, email, password_hash,
			name, quota_bytes, used_bytes, is_active, is_suspended
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at
	`
	if mb.ID == uuid.Nil {
		mb.ID = uuid.New()
	}
	if mb.QuotaBytes == 0 {
		mb.QuotaBytes = 5368709120
	}
	mb.IsActive = true

	return p.db.QueryRowContext(ctx, query,
		mb.ID, mb.DomainID, mb.ServerID, mb.LocalPart, mb.Email, mb.PasswordHash,
		mb.Name, mb.QuotaBytes, mb.UsedBytes, mb.IsActive, mb.IsSuspended,
	).Scan(&mb.CreatedAt, &mb.UpdatedAt)
}

func (p *PostgresStore) GetEmailMailboxByID(ctx context.Context, id uuid.UUID) (*EmailMailbox, error) {
	query := `
		SELECT id, domain_id, server_id, local_part, email, password_hash,
		       name, quota_bytes, used_bytes, is_active, is_suspended,
		       created_at, updated_at, deleted_at
		FROM email_mailboxes
		WHERE id = $1 AND deleted_at IS NULL
	`
	mb := &EmailMailbox{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&mb.ID, &mb.DomainID, &mb.ServerID, &mb.LocalPart, &mb.Email, &mb.PasswordHash,
		&mb.Name, &mb.QuotaBytes, &mb.UsedBytes, &mb.IsActive, &mb.IsSuspended,
		&mb.CreatedAt, &mb.UpdatedAt, &mb.DeletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return mb, err
}

func (p *PostgresStore) GetEmailMailboxByEmail(ctx context.Context, email string) (*EmailMailbox, error) {
	query := `
		SELECT id, domain_id, server_id, local_part, email, password_hash,
		       name, quota_bytes, used_bytes, is_active, is_suspended,
		       created_at, updated_at, deleted_at
		FROM email_mailboxes
		WHERE email = $1 AND deleted_at IS NULL
	`
	mb := &EmailMailbox{}
	err := p.db.QueryRowContext(ctx, query, email).Scan(
		&mb.ID, &mb.DomainID, &mb.ServerID, &mb.LocalPart, &mb.Email, &mb.PasswordHash,
		&mb.Name, &mb.QuotaBytes, &mb.UsedBytes, &mb.IsActive, &mb.IsSuspended,
		&mb.CreatedAt, &mb.UpdatedAt, &mb.DeletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return mb, err
}

func (p *PostgresStore) ListEmailMailboxesByDomain(ctx context.Context, domainID uuid.UUID) ([]*EmailMailbox, error) {
	query := `
		SELECT id, domain_id, server_id, local_part, email, name,
		       quota_bytes, used_bytes, is_active, is_suspended,
		       created_at, updated_at
		FROM email_mailboxes
		WHERE domain_id = $1 AND deleted_at IS NULL
		ORDER BY local_part ASC
	`
	rows, err := p.db.QueryContext(ctx, query, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*EmailMailbox, 0)
	for rows.Next() {
		mb := &EmailMailbox{}
		err := rows.Scan(
			&mb.ID, &mb.DomainID, &mb.ServerID, &mb.LocalPart, &mb.Email, &mb.Name,
			&mb.QuotaBytes, &mb.UsedBytes, &mb.IsActive, &mb.IsSuspended,
			&mb.CreatedAt, &mb.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, mb)
	}
	return list, nil
}

func (p *PostgresStore) ListEmailMailboxesByServer(ctx context.Context, serverID uuid.UUID) ([]*EmailMailbox, error) {
	query := `
		SELECT id, domain_id, server_id, local_part, email, name,
		       quota_bytes, used_bytes, is_active, is_suspended,
		       created_at, updated_at
		FROM email_mailboxes
		WHERE server_id = $1 AND deleted_at IS NULL
		ORDER BY email ASC
	`
	rows, err := p.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*EmailMailbox, 0)
	for rows.Next() {
		mb := &EmailMailbox{}
		err := rows.Scan(
			&mb.ID, &mb.DomainID, &mb.ServerID, &mb.LocalPart, &mb.Email, &mb.Name,
			&mb.QuotaBytes, &mb.UsedBytes, &mb.IsActive, &mb.IsSuspended,
			&mb.CreatedAt, &mb.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, mb)
	}
	return list, nil
}

func (p *PostgresStore) UpdateEmailMailbox(ctx context.Context, mb *EmailMailbox) error {
	query := `
		UPDATE email_mailboxes
		SET name = $2, quota_bytes = $3, is_active = $4, is_suspended = $5, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	return p.db.QueryRowContext(ctx, query,
		mb.ID, mb.Name, mb.QuotaBytes, mb.IsActive, mb.IsSuspended,
	).Scan(&mb.UpdatedAt)
}

func (p *PostgresStore) UpdateEmailMailboxPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `
		UPDATE email_mailboxes
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	res, err := p.db.ExecContext(ctx, query, id, passwordHash)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) DeleteEmailMailbox(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE email_mailboxes
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	res, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// Aliases Postgres

func (p *PostgresStore) CreateEmailAlias(ctx context.Context, alias *EmailAlias) error {
	query := `
		INSERT INTO email_aliases (id, domain_id, source_address, destination_address, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`
	if alias.ID == uuid.Nil {
		alias.ID = uuid.New()
	}
	alias.IsActive = true
	return p.db.QueryRowContext(ctx, query,
		alias.ID, alias.DomainID, alias.SourceAddress, alias.DestinationAddress, alias.IsActive,
	).Scan(&alias.CreatedAt, &alias.UpdatedAt)
}

func (p *PostgresStore) ListEmailAliasesByDomain(ctx context.Context, domainID uuid.UUID) ([]*EmailAlias, error) {
	query := `
		SELECT id, domain_id, source_address, destination_address, is_active, created_at, updated_at
		FROM email_aliases
		WHERE domain_id = $1
		ORDER BY source_address ASC
	`
	rows, err := p.db.QueryContext(ctx, query, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*EmailAlias, 0)
	for rows.Next() {
		a := &EmailAlias{}
		if err := rows.Scan(&a.ID, &a.DomainID, &a.SourceAddress, &a.DestinationAddress, &a.IsActive, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

func (p *PostgresStore) DeleteEmailAlias(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM email_aliases WHERE id = $1`
	res, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// Forwarders Postgres

func (p *PostgresStore) CreateEmailForwarder(ctx context.Context, fwd *EmailForwarder) error {
	query := `
		INSERT INTO email_forwarders (id, domain_id, mailbox_id, source_address, forward_address, keep_copy, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at
	`
	if fwd.ID == uuid.Nil {
		fwd.ID = uuid.New()
	}
	fwd.IsActive = true
	return p.db.QueryRowContext(ctx, query,
		fwd.ID, fwd.DomainID, fwd.MailboxID, fwd.SourceAddress, fwd.ForwardAddress, fwd.KeepCopy, fwd.IsActive,
	).Scan(&fwd.CreatedAt)
}

func (p *PostgresStore) ListEmailForwardersByDomain(ctx context.Context, domainID uuid.UUID) ([]*EmailForwarder, error) {
	query := `
		SELECT id, domain_id, mailbox_id, source_address, forward_address, keep_copy, is_active, created_at
		FROM email_forwarders
		WHERE domain_id = $1
		ORDER BY source_address ASC
	`
	rows, err := p.db.QueryContext(ctx, query, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*EmailForwarder, 0)
	for rows.Next() {
		f := &EmailForwarder{}
		if err := rows.Scan(&f.ID, &f.DomainID, &f.MailboxID, &f.SourceAddress, &f.ForwardAddress, &f.KeepCopy, &f.IsActive, &f.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, nil
}

func (p *PostgresStore) DeleteEmailForwarder(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM email_forwarders WHERE id = $1`
	res, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// Autoresponders Postgres

func (p *PostgresStore) SetEmailAutoresponder(ctx context.Context, ar *EmailAutoresponder) error {
	query := `
		INSERT INTO email_autoresponders (id, mailbox_id, subject, body, start_at, end_at, is_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (mailbox_id) DO UPDATE SET
			subject = EXCLUDED.subject,
			body = EXCLUDED.body,
			start_at = EXCLUDED.start_at,
			end_at = EXCLUDED.end_at,
			is_enabled = EXCLUDED.is_enabled,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	if ar.ID == uuid.Nil {
		ar.ID = uuid.New()
	}
	return p.db.QueryRowContext(ctx, query,
		ar.ID, ar.MailboxID, ar.Subject, ar.Body, ar.StartAt, ar.EndAt, ar.IsEnabled,
	).Scan(&ar.ID, &ar.CreatedAt, &ar.UpdatedAt)
}

func (p *PostgresStore) GetEmailAutoresponderByMailbox(ctx context.Context, mailboxID uuid.UUID) (*EmailAutoresponder, error) {
	query := `
		SELECT id, mailbox_id, subject, body, start_at, end_at, is_enabled, created_at, updated_at
		FROM email_autoresponders
		WHERE mailbox_id = $1
	`
	ar := &EmailAutoresponder{}
	err := p.db.QueryRowContext(ctx, query, mailboxID).Scan(
		&ar.ID, &ar.MailboxID, &ar.Subject, &ar.Body, &ar.StartAt, &ar.EndAt,
		&ar.IsEnabled, &ar.CreatedAt, &ar.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return ar, err
}

func (p *PostgresStore) DeleteEmailAutoresponder(ctx context.Context, mailboxID uuid.UUID) error {
	query := `DELETE FROM email_autoresponders WHERE mailbox_id = $1`
	res, err := p.db.ExecContext(ctx, query, mailboxID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DKIM Keys Postgres

func (p *PostgresStore) SaveEmailDKIMKey(ctx context.Context, dkim *EmailDKIMKey) error {
	query := `
		INSERT INTO email_dkim_keys (id, domain_id, selector, private_key_pem, public_key_dns, key_size)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (domain_id) DO UPDATE SET
			selector = EXCLUDED.selector,
			private_key_pem = EXCLUDED.private_key_pem,
			public_key_dns = EXCLUDED.public_key_dns,
			key_size = EXCLUDED.key_size,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	if dkim.ID == uuid.Nil {
		dkim.ID = uuid.New()
	}
	if dkim.Selector == "" {
		dkim.Selector = "default"
	}
	if dkim.KeySize == 0 {
		dkim.KeySize = 2048
	}
	return p.db.QueryRowContext(ctx, query,
		dkim.ID, dkim.DomainID, dkim.Selector, dkim.PrivateKeyPEM, dkim.PublicKeyDNS, dkim.KeySize,
	).Scan(&dkim.ID, &dkim.CreatedAt, &dkim.UpdatedAt)
}

func (p *PostgresStore) GetEmailDKIMKeyByDomain(ctx context.Context, domainID uuid.UUID) (*EmailDKIMKey, error) {
	query := `
		SELECT id, domain_id, selector, private_key_pem, public_key_dns, key_size, created_at, updated_at
		FROM email_dkim_keys
		WHERE domain_id = $1
	`
	dkim := &EmailDKIMKey{}
	err := p.db.QueryRowContext(ctx, query, domainID).Scan(
		&dkim.ID, &dkim.DomainID, &dkim.Selector, &dkim.PrivateKeyPEM,
		&dkim.PublicKeyDNS, &dkim.KeySize, &dkim.CreatedAt, &dkim.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return dkim, err
}

// Delivery Logs Postgres

func (p *PostgresStore) RecordEmailDeliveryLog(ctx context.Context, log *EmailDeliveryLog) error {
	query := `
		INSERT INTO email_delivery_logs (
			id, server_id, domain_id, message_id, queue_id, sender, recipient,
			status, spam_score, failure_reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at
	`
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	return p.db.QueryRowContext(ctx, query,
		log.ID, log.ServerID, log.DomainID, log.MessageID, log.QueueID,
		log.Sender, log.Recipient, log.Status, log.SpamScore, log.FailureReason,
	).Scan(&log.CreatedAt)
}

func (p *PostgresStore) ListEmailDeliveryLogs(ctx context.Context, serverID uuid.UUID, limit int) ([]*EmailDeliveryLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `
		SELECT id, server_id, domain_id, message_id, queue_id, sender, recipient,
		       status, spam_score, failure_reason, created_at
		FROM email_delivery_logs
		WHERE server_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := p.db.QueryContext(ctx, query, serverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*EmailDeliveryLog, 0)
	for rows.Next() {
		l := &EmailDeliveryLog{}
		err := rows.Scan(
			&l.ID, &l.ServerID, &l.DomainID, &l.MessageID, &l.QueueID,
			&l.Sender, &l.Recipient, &l.Status, &l.SpamScore, &l.FailureReason,
			&l.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, nil
}

// Postgres Email Signatures

func (p *PostgresStore) GetEmailSignature(ctx context.Context, mailboxID uuid.UUID) (*EmailSignature, error) {
	query := `
		SELECT id, mailbox_id, plain_text, html_text, is_enabled, created_at, updated_at
		FROM email_signatures
		WHERE mailbox_id = $1
	`
	s := &EmailSignature{}
	err := p.db.QueryRowContext(ctx, query, mailboxID).Scan(
		&s.ID, &s.MailboxID, &s.PlainText, &s.HTMLText, &s.IsEnabled, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return &EmailSignature{
			ID:        uuid.New(),
			MailboxID: mailboxID,
			IsEnabled: false,
		}, nil
	}
	return s, err
}

func (p *PostgresStore) SetEmailSignature(ctx context.Context, sig *EmailSignature) error {
	query := `
		INSERT INTO email_signatures (id, mailbox_id, plain_text, html_text, is_enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (mailbox_id) DO UPDATE
		SET plain_text = EXCLUDED.plain_text,
		    html_text = EXCLUDED.html_text,
		    is_enabled = EXCLUDED.is_enabled,
		    updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	if sig.ID == uuid.Nil {
		sig.ID = uuid.New()
	}
	return p.db.QueryRowContext(ctx, query,
		sig.ID, sig.MailboxID, sig.PlainText, sig.HTMLText, sig.IsEnabled,
	).Scan(&sig.ID, &sig.CreatedAt, &sig.UpdatedAt)
}

// Postgres Email Suppressions

func (p *PostgresStore) AddEmailSuppression(ctx context.Context, sup *EmailSuppression) error {
	query := `
		INSERT INTO email_suppressions (id, server_id, domain_id, email, reason, bounce_code, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (server_id, email) DO UPDATE
		SET reason = EXCLUDED.reason,
		    bounce_code = EXCLUDED.bounce_code,
		    metadata = EXCLUDED.metadata
		RETURNING created_at
	`
	if sup.ID == uuid.Nil {
		sup.ID = uuid.New()
	}
	metaJSON, _ := json.Marshal(sup.Metadata)
	return p.db.QueryRowContext(ctx, query,
		sup.ID, sup.ServerID, sup.DomainID, strings.ToLower(sup.Email),
		sup.Reason, sup.BounceCode, metaJSON,
	).Scan(&sup.CreatedAt)
}

func (p *PostgresStore) ListEmailSuppressions(ctx context.Context, serverID uuid.UUID) ([]*EmailSuppression, error) {
	query := `
		SELECT id, server_id, domain_id, email, reason, bounce_code, metadata, created_at
		FROM email_suppressions
		WHERE server_id = $1
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*EmailSuppression, 0)
	for rows.Next() {
		s := &EmailSuppression{}
		var metaRaw []byte
		err := rows.Scan(
			&s.ID, &s.ServerID, &s.DomainID, &s.Email, &s.Reason, &s.BounceCode, &metaRaw, &s.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if len(metaRaw) > 0 {
			_ = json.Unmarshal(metaRaw, &s.Metadata)
		}
		list = append(list, s)
	}
	return list, nil
}

func (p *PostgresStore) DeleteEmailSuppression(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM email_suppressions WHERE id = $1`
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgresStore) IsEmailSuppressed(ctx context.Context, serverID uuid.UUID, email string) (bool, error) {
	query := `SELECT 1 FROM email_suppressions WHERE server_id = $1 AND email = $2 LIMIT 1`
	var exists int
	err := p.db.QueryRowContext(ctx, query, serverID, strings.ToLower(email)).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// Postgres Webmail Messages

func (p *PostgresStore) ListWebmailMessages(ctx context.Context, mailboxID uuid.UUID, folder string, limit, offset int, search string) ([]*WebmailMessage, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	whereClause := "WHERE mailbox_id = $1"
	args := []interface{}{mailboxID}
	argIdx := 2

	if folder == "starred" {
		whereClause += " AND is_starred = TRUE"
	} else if folder == "important" {
		whereClause += " AND is_important = TRUE"
	} else if folder != "" && folder != "all" {
		whereClause += fmt.Sprintf(" AND folder = $%d", argIdx)
		args = append(args, folder)
		argIdx++
	}

	if search != "" {
		whereClause += fmt.Sprintf(" AND (subject ILIKE $%d OR from_email ILIKE $%d OR from_name ILIKE $%d OR snippet ILIKE $%d)", argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM webmail_messages %s", whereClause)
	var total int
	_ = p.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)

	query := fmt.Sprintf(`
		SELECT id, mailbox_id, account_email, folder, message_id, from_name, from_email,
		       to_name, to_email, cc, bcc, subject, snippet, body_text, body_html,
		       is_unread, is_starred, is_important, has_attachment, priority, size_bytes,
		       created_at, updated_at
		FROM webmail_messages
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]*WebmailMessage, 0)
	for rows.Next() {
		m := &WebmailMessage{}
		err := rows.Scan(
			&m.ID, &m.MailboxID, &m.AccountEmail, &m.Folder, &m.MessageID, &m.FromName, &m.FromEmail,
			&m.ToName, &m.ToEmail, &m.Cc, &m.Bcc, &m.Subject, &m.Snippet, &m.BodyText, &m.BodyHTML,
			&m.IsUnread, &m.IsStarred, &m.IsImportant, &m.HasAttachment, &m.Priority, &m.SizeBytes,
			&m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, m)
	}
	return list, total, nil
}

func (p *PostgresStore) GetWebmailMessageByID(ctx context.Context, id uuid.UUID) (*WebmailMessage, error) {
	query := `
		SELECT id, mailbox_id, account_email, folder, message_id, from_name, from_email,
		       to_name, to_email, cc, bcc, subject, snippet, body_text, body_html,
		       is_unread, is_starred, is_important, has_attachment, priority, size_bytes,
		       created_at, updated_at
		FROM webmail_messages
		WHERE id = $1
	`
	m := &WebmailMessage{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&m.ID, &m.MailboxID, &m.AccountEmail, &m.Folder, &m.MessageID, &m.FromName, &m.FromEmail,
		&m.ToName, &m.ToEmail, &m.Cc, &m.Bcc, &m.Subject, &m.Snippet, &m.BodyText, &m.BodyHTML,
		&m.IsUnread, &m.IsStarred, &m.IsImportant, &m.HasAttachment, &m.Priority, &m.SizeBytes,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Fetch attachments
	attQuery := `
		SELECT id, message_id, filename, content_type, size_bytes, storage_path, created_at
		FROM webmail_attachments
		WHERE message_id = $1
	`
	attRows, err := p.db.QueryContext(ctx, attQuery, m.ID)
	if err == nil {
		defer attRows.Close()
		for attRows.Next() {
			att := WebmailAttachment{}
			if err := attRows.Scan(&att.ID, &att.MessageID, &att.Filename, &att.ContentType, &att.SizeBytes, &att.StoragePath, &att.CreatedAt); err == nil {
				m.Attachments = append(m.Attachments, att)
			}
		}
	}

	return m, nil
}

func (p *PostgresStore) CreateWebmailMessage(ctx context.Context, msg *WebmailMessage) error {
	query := `
		INSERT INTO webmail_messages (
			id, mailbox_id, account_email, folder, message_id, from_name, from_email,
			to_name, to_email, cc, bcc, subject, snippet, body_text, body_html,
			is_unread, is_starred, is_important, has_attachment, priority, size_bytes,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21,
			NOW(), NOW()
		)
		RETURNING created_at, updated_at
	`
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	if msg.Snippet == "" && msg.BodyText != "" {
		if len(msg.BodyText) > 120 {
			msg.Snippet = msg.BodyText[:120] + "..."
		} else {
			msg.Snippet = msg.BodyText
		}
	}
	err := p.db.QueryRowContext(ctx, query,
		msg.ID, msg.MailboxID, msg.AccountEmail, msg.Folder, msg.MessageID, msg.FromName, msg.FromEmail,
		msg.ToName, msg.ToEmail, msg.Cc, msg.Bcc, msg.Subject, msg.Snippet, msg.BodyText, msg.BodyHTML,
		msg.IsUnread, msg.IsStarred, msg.IsImportant, msg.HasAttachment, msg.Priority, msg.SizeBytes,
	).Scan(&msg.CreatedAt, &msg.UpdatedAt)
	if err != nil {
		return err
	}

	for _, att := range msg.Attachments {
		att.MessageID = msg.ID
		_ = p.CreateWebmailAttachment(ctx, &att)
	}

	return nil
}

func (p *PostgresStore) UpdateWebmailMessageFlags(ctx context.Context, id uuid.UUID, isUnread, isStarred, isImportant *bool) error {
	query := `
		UPDATE webmail_messages
		SET is_unread = COALESCE($2, is_unread),
		    is_starred = COALESCE($3, is_starred),
		    is_important = COALESCE($4, is_important),
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query, id, isUnread, isStarred, isImportant)
	return err
}

func (p *PostgresStore) MoveWebmailMessage(ctx context.Context, id uuid.UUID, targetFolder string) error {
	query := `
		UPDATE webmail_messages
		SET folder = $2, updated_at = NOW()
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query, id, targetFolder)
	return err
}

func (p *PostgresStore) DeleteWebmailMessage(ctx context.Context, id uuid.UUID) error {
	var currentFolder string
	err := p.db.QueryRowContext(ctx, `SELECT folder FROM webmail_messages WHERE id = $1`, id).Scan(&currentFolder)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if currentFolder == "trash" {
		_, err = p.db.ExecContext(ctx, `DELETE FROM webmail_messages WHERE id = $1`, id)
		return err
	}
	_, err = p.db.ExecContext(ctx, `UPDATE webmail_messages SET folder = 'trash', updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (p *PostgresStore) CreateWebmailAttachment(ctx context.Context, att *WebmailAttachment) error {
	query := `
		INSERT INTO webmail_attachments (id, message_id, filename, content_type, size_bytes, storage_path, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING created_at
	`
	if att.ID == uuid.Nil {
		att.ID = uuid.New()
	}
	return p.db.QueryRowContext(ctx, query,
		att.ID, att.MessageID, att.Filename, att.ContentType, att.SizeBytes, att.StoragePath,
	).Scan(&att.CreatedAt)
}

func (p *PostgresStore) ListWebmailAttachments(ctx context.Context, messageID uuid.UUID) ([]*WebmailAttachment, error) {
	query := `
		SELECT id, message_id, filename, content_type, size_bytes, storage_path, created_at
		FROM webmail_attachments
		WHERE message_id = $1
	`
	rows, err := p.db.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*WebmailAttachment, 0)
	for rows.Next() {
		att := &WebmailAttachment{}
		err := rows.Scan(&att.ID, &att.MessageID, &att.Filename, &att.ContentType, &att.SizeBytes, &att.StoragePath, &att.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, att)
	}
	return list, nil
}

