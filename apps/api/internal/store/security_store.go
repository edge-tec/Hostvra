package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MEMORY STORE: TWO-FACTOR AUTHENTICATION & SECURITY
// ============================================================================

func (m *MemoryStore) UpdateUserTwoFactor(ctx context.Context, userID uuid.UUID, enabled bool, secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[userID]
	if !exists {
		return ErrNotFound
	}
	user.TwoFactorEnabled = enabled
	user.UpdatedAt = time.Now().UTC()
	if secret != "" || !enabled {
		m.twoFactorSecrets[userID] = secret
	}
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetUserTwoFactorSecret(ctx context.Context, userID uuid.UUID) (string, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return "", false, ErrNotFound
	}
	secret := m.twoFactorSecrets[userID]
	return secret, user.TwoFactorEnabled, nil
}

func (m *MemoryStore) SaveTwoFactorRecoveryCodes(ctx context.Context, userID uuid.UUID, codeHashes []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.twoFactorRecoveryCodes[userID] = codeHashes
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) VerifyAndConsumeTwoFactorRecoveryCode(ctx context.Context, userID uuid.UUID, codeHash string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	codes, exists := m.twoFactorRecoveryCodes[userID]
	if !exists || len(codes) == 0 {
		return false, nil
	}

	for i, c := range codes {
		if c == codeHash {
			// Consume: remove from slice
			m.twoFactorRecoveryCodes[userID] = append(codes[:i], codes[i+1:]...)
			m.saveToDiskLocked()
			return true, nil
		}
	}
	return false, nil
}

func (m *MemoryStore) CountValidTwoFactorRecoveryCodes(ctx context.Context, userID uuid.UUID) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	codes, exists := m.twoFactorRecoveryCodes[userID]
	if !exists {
		return 0, nil
	}
	return len(codes), nil
}

// ----------------------------------------------------------------------------
// MEMORY STORE: SCANS & QUARANTINE
// ----------------------------------------------------------------------------

func (m *MemoryStore) CreateSecurityScan(ctx context.Context, scan *SecurityScan) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if scan.ID == uuid.Nil {
		scan.ID = uuid.New()
	}
	if scan.CreatedAt.IsZero() {
		scan.CreatedAt = time.Now().UTC()
	}
	m.securityScans[scan.ID] = scan
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) UpdateSecurityScan(ctx context.Context, scan *SecurityScan) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.securityScans[scan.ID] = scan
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetSecurityScan(ctx context.Context, id uuid.UUID) (*SecurityScan, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	scan, exists := m.securityScans[id]
	if !exists {
		return nil, ErrNotFound
	}
	return scan, nil
}

func (m *MemoryStore) ListSecurityScans(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*SecurityScan, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*SecurityScan
	for _, s := range m.securityScans {
		if orgID == uuid.Nil || s.OrganizationID == orgID {
			list = append(list, s)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	total := len(list)
	if offset >= total {
		return []*SecurityScan{}, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}
	return list[offset:end], total, nil
}

func (m *MemoryStore) CreateQuarantineItem(ctx context.Context, item *QuarantineItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.QuarantinedAt.IsZero() {
		item.QuarantinedAt = time.Now().UTC()
	}
	m.quarantineItems[item.ID] = item
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetQuarantineItem(ctx context.Context, id uuid.UUID) (*QuarantineItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.quarantineItems[id]
	if !exists {
		return nil, ErrNotFound
	}
	return item, nil
}

func (m *MemoryStore) ListQuarantineItems(ctx context.Context, orgID uuid.UUID, status string) ([]*QuarantineItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*QuarantineItem
	for _, q := range m.quarantineItems {
		if (orgID == uuid.Nil || q.OrganizationID == orgID) &&
			(status == "" || q.Status == status) {
			list = append(list, q)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].QuarantinedAt.After(list[j].QuarantinedAt)
	})
	return list, nil
}

func (m *MemoryStore) UpdateQuarantineStatus(ctx context.Context, id uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, exists := m.quarantineItems[id]
	if !exists {
		return ErrNotFound
	}
	item.Status = status
	now := time.Now().UTC()
	if status == "restored" {
		item.RestoredAt = &now
	} else if status == "deleted" {
		item.DeletedAt = &now
	}
	m.saveToDiskLocked()
	return nil
}

// ============================================================================
// POSTGRES STORE: TWO-FACTOR AUTHENTICATION & SECURITY
// ============================================================================

func (p *PostgresStore) UpdateUserTwoFactor(ctx context.Context, userID uuid.UUID, enabled bool, secret string) error {
	query := `UPDATE users SET two_factor_enabled = $1, two_factor_secret = $2, updated_at = NOW() WHERE id = $3`
	_, err := p.db.ExecContext(ctx, query, enabled, secret, userID)
	return err
}

func (p *PostgresStore) GetUserTwoFactorSecret(ctx context.Context, userID uuid.UUID) (string, bool, error) {
	query := `SELECT COALESCE(two_factor_secret, ''), two_factor_enabled FROM users WHERE id = $1`
	var secret string
	var enabled bool
	err := p.db.QueryRowContext(ctx, query, userID).Scan(&secret, &enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, ErrNotFound
	}
	return secret, enabled, err
}

func (p *PostgresStore) SaveTwoFactorRecoveryCodes(ctx context.Context, userID uuid.UUID, codeHashes []string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Invalidate previous recovery codes
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_two_factor_recovery_codes WHERE user_id = $1`, userID); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO user_two_factor_recovery_codes (id, user_id, code_hash, created_at) VALUES ($1, $2, $3, NOW())`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, hash := range codeHashes {
		if _, err := stmt.ExecContext(ctx, uuid.New(), userID, hash); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (p *PostgresStore) VerifyAndConsumeTwoFactorRecoveryCode(ctx context.Context, userID uuid.UUID, codeHash string) (bool, error) {
	query := `
		UPDATE user_two_factor_recovery_codes
		SET used_at = NOW()
		WHERE id = (
			SELECT id FROM user_two_factor_recovery_codes
			WHERE user_id = $1 AND code_hash = $2 AND used_at IS NULL
			LIMIT 1
		)
		RETURNING id
	`
	var consumedID uuid.UUID
	err := p.db.QueryRowContext(ctx, query, userID, codeHash).Scan(&consumedID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (p *PostgresStore) CountValidTwoFactorRecoveryCodes(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM user_two_factor_recovery_codes WHERE user_id = $1 AND used_at IS NULL`
	var count int
	err := p.db.QueryRowContext(ctx, query, userID).Scan(&count)
	return count, err
}

func (p *PostgresStore) CreateSecurityScan(ctx context.Context, scan *SecurityScan) error {
	if scan.ID == uuid.Nil {
		scan.ID = uuid.New()
	}
	findingsJSON, err := json.Marshal(scan.Findings)
	if err != nil {
		findingsJSON = []byte("[]")
	}

	query := `
		INSERT INTO security_scans (
			id, organization_id, server_id, website_id, scan_type, target_path,
			status, total_files, infected_files, findings, error_message,
			started_at, completed_at, created_at, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, NOW(), $14
		)
	`
	_, err = p.db.ExecContext(ctx, query,
		scan.ID, scan.OrganizationID, scan.ServerID, scan.WebsiteID, scan.ScanType, scan.TargetPath,
		scan.Status, scan.TotalFiles, scan.InfectedFiles, findingsJSON, scan.ErrorMessage,
		scan.StartedAt, scan.CompletedAt, scan.CreatedBy,
	)
	return err
}

func (p *PostgresStore) UpdateSecurityScan(ctx context.Context, scan *SecurityScan) error {
	findingsJSON, err := json.Marshal(scan.Findings)
	if err != nil {
		findingsJSON = []byte("[]")
	}

	query := `
		UPDATE security_scans SET
			status = $1, total_files = $2, infected_files = $3, findings = $4,
			error_message = $5, started_at = $6, completed_at = $7
		WHERE id = $8
	`
	_, err = p.db.ExecContext(ctx, query,
		scan.Status, scan.TotalFiles, scan.InfectedFiles, findingsJSON,
		scan.ErrorMessage, scan.StartedAt, scan.CompletedAt, scan.ID,
	)
	return err
}

func (p *PostgresStore) GetSecurityScan(ctx context.Context, id uuid.UUID) (*SecurityScan, error) {
	query := `
		SELECT id, organization_id, server_id, website_id, scan_type, target_path,
		       status, total_files, infected_files, findings, error_message,
		       started_at, completed_at, created_at, created_by
		FROM security_scans WHERE id = $1
	`
	scan := &SecurityScan{}
	var findingsBytes []byte
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&scan.ID, &scan.OrganizationID, &scan.ServerID, &scan.WebsiteID, &scan.ScanType, &scan.TargetPath,
		&scan.Status, &scan.TotalFiles, &scan.InfectedFiles, &findingsBytes, &scan.ErrorMessage,
		&scan.StartedAt, &scan.CompletedAt, &scan.CreatedAt, &scan.CreatedBy,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(findingsBytes, &scan.Findings)
	return scan, nil
}

func (p *PostgresStore) ListSecurityScans(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*SecurityScan, int, error) {
	var countQuery string
	var query string
	var args []interface{}

	if orgID != uuid.Nil {
		countQuery = `SELECT COUNT(*) FROM security_scans WHERE organization_id = $1`
		query = `
			SELECT id, organization_id, server_id, website_id, scan_type, target_path,
			       status, total_files, infected_files, findings, error_message,
			       started_at, completed_at, created_at, created_by
			FROM security_scans
			WHERE organization_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{orgID, limit, offset}
	} else {
		countQuery = `SELECT COUNT(*) FROM security_scans`
		query = `
			SELECT id, organization_id, server_id, website_id, scan_type, target_path,
			       status, total_files, infected_files, findings, error_message,
			       started_at, completed_at, created_at, created_by
			FROM security_scans
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset}
	}

	var total int
	if orgID != uuid.Nil {
		_ = p.db.QueryRowContext(ctx, countQuery, orgID).Scan(&total)
	} else {
		_ = p.db.QueryRowContext(ctx, countQuery).Scan(&total)
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*SecurityScan
	for rows.Next() {
		scan := &SecurityScan{}
		var findingsBytes []byte
		if err := rows.Scan(
			&scan.ID, &scan.OrganizationID, &scan.ServerID, &scan.WebsiteID, &scan.ScanType, &scan.TargetPath,
			&scan.Status, &scan.TotalFiles, &scan.InfectedFiles, &findingsBytes, &scan.ErrorMessage,
			&scan.StartedAt, &scan.CompletedAt, &scan.CreatedAt, &scan.CreatedBy,
		); err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal(findingsBytes, &scan.Findings)
		list = append(list, scan)
	}
	return list, total, nil
}

func (p *PostgresStore) CreateQuarantineItem(ctx context.Context, item *QuarantineItem) error {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	query := `
		INSERT INTO security_quarantine (
			id, scan_id, organization_id, server_id, website_id, original_path,
			quarantine_filename, file_size, file_mode, file_owner, file_group,
			sha256_hash, threat_name, status, quarantined_at, quarantined_by
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, NOW(), $15
		)
	`
	_, err := p.db.ExecContext(ctx, query,
		item.ID, item.ScanID, item.OrganizationID, item.ServerID, item.WebsiteID, item.OriginalPath,
		item.QuarantineFilename, item.FileSize, item.FileMode, item.FileOwner, item.FileGroup,
		item.SHA256Hash, item.ThreatName, item.Status, item.QuarantinedBy,
	)
	return err
}

func (p *PostgresStore) GetQuarantineItem(ctx context.Context, id uuid.UUID) (*QuarantineItem, error) {
	query := `
		SELECT id, scan_id, organization_id, server_id, website_id, original_path,
		       quarantine_filename, file_size, file_mode, file_owner, file_group,
		       sha256_hash, threat_name, status, restored_at, deleted_at,
		       quarantined_at, quarantined_by
		FROM security_quarantine WHERE id = $1
	`
	item := &QuarantineItem{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&item.ID, &item.ScanID, &item.OrganizationID, &item.ServerID, &item.WebsiteID, &item.OriginalPath,
		&item.QuarantineFilename, &item.FileSize, &item.FileMode, &item.FileOwner, &item.FileGroup,
		&item.SHA256Hash, &item.ThreatName, &item.Status, &item.RestoredAt, &item.DeletedAt,
		&item.QuarantinedAt, &item.QuarantinedBy,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return item, err
}

func (p *PostgresStore) ListQuarantineItems(ctx context.Context, orgID uuid.UUID, status string) ([]*QuarantineItem, error) {
	var query string
	var args []interface{}

	if orgID != uuid.Nil && status != "" {
		query = `SELECT id, scan_id, organization_id, server_id, website_id, original_path,
		                quarantine_filename, file_size, file_mode, file_owner, file_group,
		                sha256_hash, threat_name, status, restored_at, deleted_at,
		                quarantined_at, quarantined_by
		         FROM security_quarantine
		         WHERE organization_id = $1 AND status = $2
		         ORDER BY quarantined_at DESC`
		args = []interface{}{orgID, status}
	} else if orgID != uuid.Nil {
		query = `SELECT id, scan_id, organization_id, server_id, website_id, original_path,
		                quarantine_filename, file_size, file_mode, file_owner, file_group,
		                sha256_hash, threat_name, status, restored_at, deleted_at,
		                quarantined_at, quarantined_by
		         FROM security_quarantine
		         WHERE organization_id = $1
		         ORDER BY quarantined_at DESC`
		args = []interface{}{orgID}
	} else {
		query = `SELECT id, scan_id, organization_id, server_id, website_id, original_path,
		                quarantine_filename, file_size, file_mode, file_owner, file_group,
		                sha256_hash, threat_name, status, restored_at, deleted_at,
		                quarantined_at, quarantined_by
		         FROM security_quarantine
		         ORDER BY quarantined_at DESC`
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*QuarantineItem
	for rows.Next() {
		item := &QuarantineItem{}
		if err := rows.Scan(
			&item.ID, &item.ScanID, &item.OrganizationID, &item.ServerID, &item.WebsiteID, &item.OriginalPath,
			&item.QuarantineFilename, &item.FileSize, &item.FileMode, &item.FileOwner, &item.FileGroup,
			&item.SHA256Hash, &item.ThreatName, &item.Status, &item.RestoredAt, &item.DeletedAt,
			&item.QuarantinedAt, &item.QuarantinedBy,
		); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, nil
}

func (p *PostgresStore) UpdateQuarantineStatus(ctx context.Context, id uuid.UUID, status string) error {
	var query string
	if status == "restored" {
		query = `UPDATE security_quarantine SET status = $1, restored_at = NOW() WHERE id = $2`
	} else if status == "deleted" {
		query = `UPDATE security_quarantine SET status = $1, deleted_at = NOW() WHERE id = $2`
	} else {
		query = `UPDATE security_quarantine SET status = $1 WHERE id = $2`
	}
	_, err := p.db.ExecContext(ctx, query, status, id)
	return err
}
