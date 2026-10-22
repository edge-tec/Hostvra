package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MEMORY STORE: SMTP RELAY, SPAM FILTERING & IMAP MIGRATIONS
// ============================================================================

// 1. External SMTP Relays
func (m *MemoryStore) ListEmailSMTPRelays(ctx context.Context, orgID uuid.UUID, domainID *uuid.UUID) ([]*EmailSMTPRelayRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*EmailSMTPRelayRecord
	for _, r := range m.emailSMTPRelays {
		if r.OrganizationID == orgID {
			if domainID != nil {
				if r.DomainID != nil && *r.DomainID == *domainID {
					copyR := *r
					list = append(list, &copyR)
				}
			} else {
				copyR := *r
				list = append(list, &copyR)
			}
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	return list, nil
}

func (m *MemoryStore) GetEmailSMTPRelayByID(ctx context.Context, id uuid.UUID) (*EmailSMTPRelayRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	r, ok := m.emailSMTPRelays[id]
	if !ok {
		return nil, ErrNotFound
	}
	copyR := *r
	return &copyR, nil
}

func (m *MemoryStore) CreateEmailSMTPRelay(ctx context.Context, relay *EmailSMTPRelayRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if relay.ID == uuid.Nil {
		relay.ID = uuid.New()
	}
	now := time.Now().UTC()
	relay.CreatedAt = now
	relay.UpdatedAt = now

	copyR := *relay
	m.emailSMTPRelays[relay.ID] = &copyR
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) UpdateEmailSMTPRelay(ctx context.Context, relay *EmailSMTPRelayRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.emailSMTPRelays[relay.ID]
	if !ok {
		return ErrNotFound
	}

	relay.CreatedAt = existing.CreatedAt
	relay.UpdatedAt = time.Now().UTC()

	copyR := *relay
	m.emailSMTPRelays[relay.ID] = &copyR
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteEmailSMTPRelay(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.emailSMTPRelays[id]; !ok {
		return ErrNotFound
	}

	delete(m.emailSMTPRelays, id)
	m.saveToDiskLocked()
	return nil
}

// 2. Anti-Spam Filter Settings
func (m *MemoryStore) spamSettingsKey(orgID uuid.UUID, domainID *uuid.UUID) string {
	if domainID != nil {
		return fmt.Sprintf("%s:%s", orgID, *domainID)
	}
	return orgID.String()
}

func (m *MemoryStore) GetEmailSpamSettings(ctx context.Context, orgID uuid.UUID, domainID *uuid.UUID) (*EmailSpamFilterSettingsRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.spamSettingsKey(orgID, domainID)
	s, ok := m.emailSpamSettings[key]
	if !ok {
		// Fallback to org default if domain-specific not configured
		if domainID != nil {
			if orgDefault, okOrg := m.emailSpamSettings[orgID.String()]; okOrg {
				copyS := *orgDefault
				return &copyS, nil
			}
		}
		// Return sensible defaults if no setting stored yet
		return &EmailSpamFilterSettingsRecord{
			ID:                      uuid.New(),
			OrganizationID:          orgID,
			DomainID:                domainID,
			Engine:                  "rspamd",
			RejectScore:             15.0,
			AddHeaderScore:          6.0,
			GreylistScore:           4.0,
			EnableGreylisting:       true,
			EnableBayes:             true,
			EnableClamAV:            false,
			QuarantineEnabled:       true,
			QuarantineRetentionDays: 14,
			CreatedAt:               time.Now().UTC(),
			UpdatedAt:               time.Now().UTC(),
		}, nil
	}

	copyS := *s
	return &copyS, nil
}

func (m *MemoryStore) SaveEmailSpamSettings(ctx context.Context, settings *EmailSpamFilterSettingsRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if settings.ID == uuid.Nil {
		settings.ID = uuid.New()
	}
	now := time.Now().UTC()
	key := m.spamSettingsKey(settings.OrganizationID, settings.DomainID)
	if existing, ok := m.emailSpamSettings[key]; ok {
		settings.CreatedAt = existing.CreatedAt
	} else if settings.CreatedAt.IsZero() {
		settings.CreatedAt = now
	}
	settings.UpdatedAt = now

	copyS := *settings
	m.emailSpamSettings[key] = &copyS
	m.saveToDiskLocked()
	return nil
}

// 3. Spam Quarantine Items
func (m *MemoryStore) ListSpamQuarantineItems(ctx context.Context, orgID uuid.UUID, domainID *uuid.UUID, limit int) ([]*EmailSpamQuarantineItemRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*EmailSpamQuarantineItemRecord
	for _, it := range m.emailSpamQuarantineItems {
		if it.OrganizationID == orgID {
			if domainID == nil || it.DomainID == *domainID {
				copyIt := *it
				list = append(list, &copyIt)
			}
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

func (m *MemoryStore) GetSpamQuarantineItemByID(ctx context.Context, id uuid.UUID) (*EmailSpamQuarantineItemRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	it, ok := m.emailSpamQuarantineItems[id]
	if !ok {
		return nil, ErrNotFound
	}
	copyIt := *it
	return &copyIt, nil
}

func (m *MemoryStore) CreateSpamQuarantineItem(ctx context.Context, item *EmailSpamQuarantineItemRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}

	copyIt := *item
	m.emailSpamQuarantineItems[item.ID] = &copyIt
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) UpdateSpamQuarantineItemStatus(ctx context.Context, id uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	it, ok := m.emailSpamQuarantineItems[id]
	if !ok {
		return ErrNotFound
	}
	it.Status = status
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteSpamQuarantineItem(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.emailSpamQuarantineItems[id]; !ok {
		return ErrNotFound
	}
	delete(m.emailSpamQuarantineItems, id)
	m.saveToDiskLocked()
	return nil
}

// 4. Live IMAP Mailbox Migrations
func (m *MemoryStore) CreateIMAPMigration(ctx context.Context, migration *EmailIMAPMigrationRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if migration.ID == uuid.Nil {
		migration.ID = uuid.New()
	}
	now := time.Now().UTC()
	migration.CreatedAt = now
	migration.UpdatedAt = now

	copyM := *migration
	m.emailIMAPMigrations[migration.ID] = &copyM
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetIMAPMigrationByID(ctx context.Context, id uuid.UUID) (*EmailIMAPMigrationRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mig, ok := m.emailIMAPMigrations[id]
	if !ok {
		return nil, ErrNotFound
	}
	copyM := *mig
	return &copyM, nil
}

func (m *MemoryStore) ListIMAPMigrations(ctx context.Context, orgID uuid.UUID, limit int) ([]*EmailIMAPMigrationRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*EmailIMAPMigrationRecord
	for _, mig := range m.emailIMAPMigrations {
		if mig.OrganizationID == orgID {
			copyM := *mig
			list = append(list, &copyM)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

func (m *MemoryStore) UpdateIMAPMigration(ctx context.Context, migration *EmailIMAPMigrationRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.emailIMAPMigrations[migration.ID]
	if !ok {
		return ErrNotFound
	}

	migration.CreatedAt = existing.CreatedAt
	migration.UpdatedAt = time.Now().UTC()

	copyM := *migration
	m.emailIMAPMigrations[migration.ID] = &copyM
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteIMAPMigration(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.emailIMAPMigrations[id]; !ok {
		return ErrNotFound
	}
	delete(m.emailIMAPMigrations, id)
	m.saveToDiskLocked()
	return nil
}

// ============================================================================
// POSTGRES STORE: SMTP RELAY, SPAM FILTERING & IMAP MIGRATIONS
// ============================================================================

func (s *PostgresStore) ListEmailSMTPRelays(ctx context.Context, orgID uuid.UUID, domainID *uuid.UUID) ([]*EmailSMTPRelayRecord, error) {
	query := `
		SELECT id, organization_id, domain_id, name, provider, host, port, username, password_encrypted,
		       auth_type, encryption, sender_domain, is_verified, verified_at, status, last_test_at,
		       last_test_status, last_test_error, rate_limit_per_second, rate_limit_per_day, created_at, updated_at
		FROM email_smtp_relays
		WHERE organization_id = $1 AND ($2::uuid IS NULL OR domain_id = $2)
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, orgID, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*EmailSMTPRelayRecord
	for rows.Next() {
		var r EmailSMTPRelayRecord
		err := rows.Scan(
			&r.ID, &r.OrganizationID, &r.DomainID, &r.Name, &r.Provider, &r.Host, &r.Port, &r.Username,
			&r.PasswordEncrypted, &r.AuthType, &r.Encryption, &r.SenderDomain, &r.IsVerified, &r.VerifiedAt,
			&r.Status, &r.LastTestAt, &r.LastTestStatus, &r.LastTestError, &r.RateLimitPerSecond,
			&r.RateLimitPerDay, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, &r)
	}
	return list, nil
}

func (s *PostgresStore) GetEmailSMTPRelayByID(ctx context.Context, id uuid.UUID) (*EmailSMTPRelayRecord, error) {
	query := `
		SELECT id, organization_id, domain_id, name, provider, host, port, username, password_encrypted,
		       auth_type, encryption, sender_domain, is_verified, verified_at, status, last_test_at,
		       last_test_status, last_test_error, rate_limit_per_second, rate_limit_per_day, created_at, updated_at
		FROM email_smtp_relays
		WHERE id = $1
		LIMIT 1
	`
	var r EmailSMTPRelayRecord
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&r.ID, &r.OrganizationID, &r.DomainID, &r.Name, &r.Provider, &r.Host, &r.Port, &r.Username,
		&r.PasswordEncrypted, &r.AuthType, &r.Encryption, &r.SenderDomain, &r.IsVerified, &r.VerifiedAt,
		&r.Status, &r.LastTestAt, &r.LastTestStatus, &r.LastTestError, &r.RateLimitPerSecond,
		&r.RateLimitPerDay, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *PostgresStore) CreateEmailSMTPRelay(ctx context.Context, relay *EmailSMTPRelayRecord) error {
	if relay.ID == uuid.Nil {
		relay.ID = uuid.New()
	}
	now := time.Now().UTC()
	relay.CreatedAt = now
	relay.UpdatedAt = now

	query := `
		INSERT INTO email_smtp_relays (
			id, organization_id, domain_id, name, provider, host, port, username, password_encrypted,
			auth_type, encryption, sender_domain, is_verified, verified_at, status, last_test_at,
			last_test_status, last_test_error, rate_limit_per_second, rate_limit_per_day, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
		)
	`
	_, err := s.db.ExecContext(ctx, query,
		relay.ID, relay.OrganizationID, relay.DomainID, relay.Name, relay.Provider, relay.Host, relay.Port,
		relay.Username, relay.PasswordEncrypted, relay.AuthType, relay.Encryption, relay.SenderDomain,
		relay.IsVerified, relay.VerifiedAt, relay.Status, relay.LastTestAt, relay.LastTestStatus,
		relay.LastTestError, relay.RateLimitPerSecond, relay.RateLimitPerDay, relay.CreatedAt, relay.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) UpdateEmailSMTPRelay(ctx context.Context, relay *EmailSMTPRelayRecord) error {
	relay.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE email_smtp_relays SET
			domain_id = $2, name = $3, provider = $4, host = $5, port = $6, username = $7,
			password_encrypted = $8, auth_type = $9, encryption = $10, sender_domain = $11,
			is_verified = $12, verified_at = $13, status = $14, last_test_at = $15,
			last_test_status = $16, last_test_error = $17, rate_limit_per_second = $18,
			rate_limit_per_day = $19, updated_at = $20
		WHERE id = $1
	`
	res, err := s.db.ExecContext(ctx, query,
		relay.ID, relay.DomainID, relay.Name, relay.Provider, relay.Host, relay.Port,
		relay.Username, relay.PasswordEncrypted, relay.AuthType, relay.Encryption, relay.SenderDomain,
		relay.IsVerified, relay.VerifiedAt, relay.Status, relay.LastTestAt, relay.LastTestStatus,
		relay.LastTestError, relay.RateLimitPerSecond, relay.RateLimitPerDay, relay.UpdatedAt,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteEmailSMTPRelay(ctx context.Context, id uuid.UUID) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM email_smtp_relays WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) GetEmailSpamSettings(ctx context.Context, orgID uuid.UUID, domainID *uuid.UUID) (*EmailSpamFilterSettingsRecord, error) {
	query := `
		SELECT id, organization_id, domain_id, engine, reject_score, add_header_score, greylist_score,
		       enable_greylisting, enable_bayes, enable_clamav, quarantine_enabled, quarantine_retention_days,
		       created_at, updated_at
		FROM email_spam_filter_settings
		WHERE organization_id = $1 AND ($2::uuid IS NULL OR domain_id = $2)
		LIMIT 1
	`
	var set EmailSpamFilterSettingsRecord
	err := s.db.QueryRowContext(ctx, query, orgID, domainID).Scan(
		&set.ID, &set.OrganizationID, &set.DomainID, &set.Engine, &set.RejectScore, &set.AddHeaderScore,
		&set.GreylistScore, &set.EnableGreylisting, &set.EnableBayes, &set.EnableClamAV,
		&set.QuarantineEnabled, &set.QuarantineRetentionDays, &set.CreatedAt, &set.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		// Defaults
		return &EmailSpamFilterSettingsRecord{
			ID:                      uuid.New(),
			OrganizationID:          orgID,
			DomainID:                domainID,
			Engine:                  "rspamd",
			RejectScore:             15.0,
			AddHeaderScore:          6.0,
			GreylistScore:           4.0,
			EnableGreylisting:       true,
			EnableBayes:             true,
			EnableClamAV:            false,
			QuarantineEnabled:       true,
			QuarantineRetentionDays: 14,
			CreatedAt:               time.Now().UTC(),
			UpdatedAt:               time.Now().UTC(),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &set, nil
}

func (s *PostgresStore) SaveEmailSpamSettings(ctx context.Context, settings *EmailSpamFilterSettingsRecord) error {
	if settings.ID == uuid.Nil {
		settings.ID = uuid.New()
	}
	now := time.Now().UTC()
	if settings.CreatedAt.IsZero() {
		settings.CreatedAt = now
	}
	settings.UpdatedAt = now

	query := `
		INSERT INTO email_spam_filter_settings (
			id, organization_id, domain_id, engine, reject_score, add_header_score, greylist_score,
			enable_greylisting, enable_bayes, enable_clamav, quarantine_enabled, quarantine_retention_days,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		ON CONFLICT (id) DO UPDATE SET
			reject_score = EXCLUDED.reject_score,
			add_header_score = EXCLUDED.add_header_score,
			greylist_score = EXCLUDED.greylist_score,
			enable_greylisting = EXCLUDED.enable_greylisting,
			enable_bayes = EXCLUDED.enable_bayes,
			enable_clamav = EXCLUDED.enable_clamav,
			quarantine_enabled = EXCLUDED.quarantine_enabled,
			quarantine_retention_days = EXCLUDED.quarantine_retention_days,
			updated_at = EXCLUDED.updated_at
	`
	_, err := s.db.ExecContext(ctx, query,
		settings.ID, settings.OrganizationID, settings.DomainID, settings.Engine, settings.RejectScore,
		settings.AddHeaderScore, settings.GreylistScore, settings.EnableGreylisting, settings.EnableBayes,
		settings.EnableClamAV, settings.QuarantineEnabled, settings.QuarantineRetentionDays, settings.CreatedAt, settings.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) ListSpamQuarantineItems(ctx context.Context, orgID uuid.UUID, domainID *uuid.UUID, limit int) ([]*EmailSpamQuarantineItemRecord, error) {
	query := `
		SELECT id, organization_id, domain_id, recipient, sender, subject, score, symbols, status,
		       message_id, mail_path, created_at
		FROM email_spam_quarantine_items
		WHERE organization_id = $1 AND ($2::uuid IS NULL OR domain_id = $2)
		ORDER BY created_at DESC
		LIMIT $3
	`
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, query, orgID, domainID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*EmailSpamQuarantineItemRecord
	for rows.Next() {
		var it EmailSpamQuarantineItemRecord
		err := rows.Scan(
			&it.ID, &it.OrganizationID, &it.DomainID, &it.Recipient, &it.Sender, &it.Subject,
			&it.Score, &it.Symbols, &it.Status, &it.MessageID, &it.MailPath, &it.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, &it)
	}
	return list, nil
}

func (s *PostgresStore) GetSpamQuarantineItemByID(ctx context.Context, id uuid.UUID) (*EmailSpamQuarantineItemRecord, error) {
	query := `
		SELECT id, organization_id, domain_id, recipient, sender, subject, score, symbols, status,
		       message_id, mail_path, created_at
		FROM email_spam_quarantine_items
		WHERE id = $1
		LIMIT 1
	`
	var it EmailSpamQuarantineItemRecord
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&it.ID, &it.OrganizationID, &it.DomainID, &it.Recipient, &it.Sender, &it.Subject,
		&it.Score, &it.Symbols, &it.Status, &it.MessageID, &it.MailPath, &it.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (s *PostgresStore) CreateSpamQuarantineItem(ctx context.Context, item *EmailSpamQuarantineItemRecord) error {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO email_spam_quarantine_items (
			id, organization_id, domain_id, recipient, sender, subject, score, symbols, status,
			message_id, mail_path, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := s.db.ExecContext(ctx, query,
		item.ID, item.OrganizationID, item.DomainID, item.Recipient, item.Sender, item.Subject,
		item.Score, item.Symbols, item.Status, item.MessageID, item.MailPath, item.CreatedAt,
	)
	return err
}

func (s *PostgresStore) UpdateSpamQuarantineItemStatus(ctx context.Context, id uuid.UUID, status string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE email_spam_quarantine_items SET status = $2 WHERE id = $1`, id, status)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteSpamQuarantineItem(ctx context.Context, id uuid.UUID) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM email_spam_quarantine_items WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) CreateIMAPMigration(ctx context.Context, migration *EmailIMAPMigrationRecord) error {
	if migration.ID == uuid.Nil {
		migration.ID = uuid.New()
	}
	now := time.Now().UTC()
	migration.CreatedAt = now
	migration.UpdatedAt = now

	query := `
		INSERT INTO email_imap_migrations (
			id, organization_id, destination_mailbox_id, destination_email, source_host, source_port,
			source_username, source_password_encrypted, source_use_ssl, status, total_folders, current_folder,
			total_messages, synced_messages, skipped_duplicates, failed_messages, bytes_transferred,
			error_message, started_at, completed_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
		)
	`
	_, err := s.db.ExecContext(ctx, query,
		migration.ID, migration.OrganizationID, migration.DestinationMailboxID, migration.DestinationEmail,
		migration.SourceHost, migration.SourcePort, migration.SourceUsername, migration.SourcePasswordEncrypted,
		migration.SourceUseSSL, migration.Status, migration.TotalFolders, migration.CurrentFolder,
		migration.TotalMessages, migration.SyncedMessages, migration.SkippedDuplicates, migration.FailedMessages,
		migration.BytesTransferred, migration.ErrorMessage, migration.StartedAt, migration.CompletedAt,
		migration.CreatedAt, migration.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) GetIMAPMigrationByID(ctx context.Context, id uuid.UUID) (*EmailIMAPMigrationRecord, error) {
	query := `
		SELECT id, organization_id, destination_mailbox_id, destination_email, source_host, source_port,
		       source_username, source_password_encrypted, source_use_ssl, status, total_folders, current_folder,
		       total_messages, synced_messages, skipped_duplicates, failed_messages, bytes_transferred,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM email_imap_migrations
		WHERE id = $1
		LIMIT 1
	`
	var m EmailIMAPMigrationRecord
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&m.ID, &m.OrganizationID, &m.DestinationMailboxID, &m.DestinationEmail, &m.SourceHost, &m.SourcePort,
		&m.SourceUsername, &m.SourcePasswordEncrypted, &m.SourceUseSSL, &m.Status, &m.TotalFolders, &m.CurrentFolder,
		&m.TotalMessages, &m.SyncedMessages, &m.SkippedDuplicates, &m.FailedMessages, &m.BytesTransferred,
		&m.ErrorMessage, &m.StartedAt, &m.CompletedAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *PostgresStore) ListIMAPMigrations(ctx context.Context, orgID uuid.UUID, limit int) ([]*EmailIMAPMigrationRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, organization_id, destination_mailbox_id, destination_email, source_host, source_port,
		       source_username, source_password_encrypted, source_use_ssl, status, total_folders, current_folder,
		       total_messages, synced_messages, skipped_duplicates, failed_messages, bytes_transferred,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM email_imap_migrations
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := s.db.QueryContext(ctx, query, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*EmailIMAPMigrationRecord
	for rows.Next() {
		var m EmailIMAPMigrationRecord
		err := rows.Scan(
			&m.ID, &m.OrganizationID, &m.DestinationMailboxID, &m.DestinationEmail, &m.SourceHost, &m.SourcePort,
			&m.SourceUsername, &m.SourcePasswordEncrypted, &m.SourceUseSSL, &m.Status, &m.TotalFolders, &m.CurrentFolder,
			&m.TotalMessages, &m.SyncedMessages, &m.SkippedDuplicates, &m.FailedMessages, &m.BytesTransferred,
			&m.ErrorMessage, &m.StartedAt, &m.CompletedAt, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, &m)
	}
	return list, nil
}

func (s *PostgresStore) UpdateIMAPMigration(ctx context.Context, migration *EmailIMAPMigrationRecord) error {
	migration.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE email_imap_migrations SET
			status = $2, total_folders = $3, current_folder = $4, total_messages = $5,
			synced_messages = $6, skipped_duplicates = $7, failed_messages = $8,
			bytes_transferred = $9, error_message = $10, started_at = $11, completed_at = $12,
			updated_at = $13
		WHERE id = $1
	`
	res, err := s.db.ExecContext(ctx, query,
		migration.ID, migration.Status, migration.TotalFolders, migration.CurrentFolder,
		migration.TotalMessages, migration.SyncedMessages, migration.SkippedDuplicates, migration.FailedMessages,
		migration.BytesTransferred, migration.ErrorMessage, migration.StartedAt, migration.CompletedAt,
		migration.UpdatedAt,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteIMAPMigration(ctx context.Context, id uuid.UUID) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM email_imap_migrations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
