package store

import (
	"context"
	"database/sql"
	"sort"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MEMORY STORE: ALERTS, INCIDENTS, CHANNELS & API KEYS
// ============================================================================

func (m *MemoryStore) CreateAlertRule(ctx context.Context, rule *AlertRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now().UTC()
	}

	copyRule := *rule
	m.alertRules[rule.ID] = &copyRule
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) ListAlertRules(ctx context.Context, orgID uuid.UUID) ([]*AlertRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*AlertRule
	for _, r := range m.alertRules {
		if r.OrganizationID == orgID {
			copyR := *r
			list = append(list, &copyR)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	return list, nil
}

func (m *MemoryStore) DeleteAlertRule(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.alertRules, id)
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) CreateAlertIncident(ctx context.Context, inc *AlertIncident) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if inc.ID == uuid.Nil {
		inc.ID = uuid.New()
	}
	if inc.StartedAt.IsZero() {
		inc.StartedAt = time.Now().UTC()
	}

	copyInc := *inc
	m.alertIncidents[inc.ID] = &copyInc
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) UpdateAlertIncident(ctx context.Context, inc *AlertIncident) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	copyInc := *inc
	m.alertIncidents[inc.ID] = &copyInc
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) ListAlertIncidents(ctx context.Context, limit int) ([]*AlertIncident, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*AlertIncident
	for _, inc := range m.alertIncidents {
		copyInc := *inc
		list = append(list, &copyInc)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].StartedAt.After(list[j].StartedAt)
	})
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

func (m *MemoryStore) CreateNotificationChannel(ctx context.Context, ch *NotificationChannel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ch.ID == uuid.Nil {
		ch.ID = uuid.New()
	}
	if ch.CreatedAt.IsZero() {
		ch.CreatedAt = time.Now().UTC()
	}

	copyCh := *ch
	m.notificationChannels[ch.ID] = &copyCh
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) ListNotificationChannels(ctx context.Context, orgID uuid.UUID) ([]*NotificationChannel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*NotificationChannel
	for _, ch := range m.notificationChannels {
		if ch.OrganizationID == orgID {
			copyCh := *ch
			list = append(list, &copyCh)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	return list, nil
}

func (m *MemoryStore) CreateAPIKey(ctx context.Context, key *StoredAPIKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if key.ID == uuid.Nil {
		key.ID = uuid.New()
	}
	if key.CreatedAt.IsZero() {
		key.CreatedAt = time.Now().UTC()
	}

	copyKey := *key
	m.storedAPIKeys[key.ID] = &copyKey
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) ListAPIKeys(ctx context.Context, orgID uuid.UUID) ([]*StoredAPIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*StoredAPIKey
	for _, k := range m.storedAPIKeys {
		if k.OrganizationID == orgID && k.RevokedAt == nil {
			copyKey := *k
			list = append(list, &copyKey)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	return list, nil
}

func (m *MemoryStore) RevokeAPIKey(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if k, ok := m.storedAPIKeys[id]; ok {
		now := time.Now().UTC()
		k.RevokedAt = &now
		m.saveToDiskLocked()
	}
	return nil
}

// ============================================================================
// POSTGRES STORE: ALERTS, INCIDENTS, CHANNELS & API KEYS
// ============================================================================

func (s *PostgresStore) CreateAlertRule(ctx context.Context, rule *AlertRule) error {
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO alert_rules (id, organization_id, name, type, threshold, duration_sec, severity, enabled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := s.db.ExecContext(ctx, query,
		rule.ID, rule.OrganizationID, rule.Name, rule.Type,
		rule.Threshold, rule.DurationSec, rule.Severity, rule.Enabled, rule.CreatedAt,
	)
	return err
}

func (s *PostgresStore) ListAlertRules(ctx context.Context, orgID uuid.UUID) ([]*AlertRule, error) {
	query := `
		SELECT id, organization_id, name, type, threshold, duration_sec, severity, enabled, created_at
		FROM alert_rules
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*AlertRule
	for rows.Next() {
		var r AlertRule
		if err := rows.Scan(
			&r.ID, &r.OrganizationID, &r.Name, &r.Type,
			&r.Threshold, &r.DurationSec, &r.Severity, &r.Enabled, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &r)
	}
	return list, rows.Err()
}

func (s *PostgresStore) DeleteAlertRule(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM alert_rules WHERE id = $1`, id)
	return err
}

func (s *PostgresStore) CreateAlertIncident(ctx context.Context, inc *AlertIncident) error {
	if inc.ID == uuid.Nil {
		inc.ID = uuid.New()
	}
	if inc.StartedAt.IsZero() {
		inc.StartedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO alert_incidents (id, server_id, server_name, rule_id, rule_name, severity, status, value, threshold, message, started_at, resolved_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := s.db.ExecContext(ctx, query,
		inc.ID, inc.ServerID, inc.ServerName, inc.RuleID, inc.RuleName,
		inc.Severity, inc.Status, inc.Value, inc.Threshold, inc.Message,
		inc.StartedAt, inc.ResolvedAt,
	)
	return err
}

func (s *PostgresStore) UpdateAlertIncident(ctx context.Context, inc *AlertIncident) error {
	query := `
		UPDATE alert_incidents
		SET status = $2, value = $3, resolved_at = $4
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, inc.ID, inc.Status, inc.Value, inc.ResolvedAt)
	return err
}

func (s *PostgresStore) ListAlertIncidents(ctx context.Context, limit int) ([]*AlertIncident, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, server_id, server_name, rule_id, rule_name, severity, status, value, threshold, message, started_at, resolved_at
		FROM alert_incidents
		ORDER BY started_at DESC
		LIMIT $1
	`
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*AlertIncident
	for rows.Next() {
		var inc AlertIncident
		var resolvedAt sql.NullTime
		if err := rows.Scan(
			&inc.ID, &inc.ServerID, &inc.ServerName, &inc.RuleID, &inc.RuleName,
			&inc.Severity, &inc.Status, &inc.Value, &inc.Threshold, &inc.Message,
			&inc.StartedAt, &resolvedAt,
		); err != nil {
			return nil, err
		}
		if resolvedAt.Valid {
			inc.ResolvedAt = &resolvedAt.Time
		}
		list = append(list, &inc)
	}
	return list, rows.Err()
}

func (s *PostgresStore) CreateNotificationChannel(ctx context.Context, ch *NotificationChannel) error {
	if ch.ID == uuid.Nil {
		ch.ID = uuid.New()
	}
	if ch.CreatedAt.IsZero() {
		ch.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO alert_notification_channels (id, organization_id, name, type, target, enabled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := s.db.ExecContext(ctx, query,
		ch.ID, ch.OrganizationID, ch.Name, ch.Type, ch.Target, ch.Enabled, ch.CreatedAt,
	)
	return err
}

func (s *PostgresStore) ListNotificationChannels(ctx context.Context, orgID uuid.UUID) ([]*NotificationChannel, error) {
	query := `
		SELECT id, organization_id, name, type, target, enabled, created_at
		FROM alert_notification_channels
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*NotificationChannel
	for rows.Next() {
		var ch NotificationChannel
		if err := rows.Scan(
			&ch.ID, &ch.OrganizationID, &ch.Name, &ch.Type, &ch.Target, &ch.Enabled, &ch.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &ch)
	}
	return list, rows.Err()
}

func (s *PostgresStore) CreateAPIKey(ctx context.Context, key *StoredAPIKey) error {
	if key.ID == uuid.Nil {
		key.ID = uuid.New()
	}
	if key.CreatedAt.IsZero() {
		key.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO api_keys (id, organization_id, user_id, name, key_prefix, key_hash, role, last_used_at, created_at, revoked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := s.db.ExecContext(ctx, query,
		key.ID, key.OrganizationID, key.UserID, key.Name, key.KeyPrefix, key.KeyHash,
		key.Role, key.LastUsedAt, key.CreatedAt, key.RevokedAt,
	)
	return err
}

func (s *PostgresStore) ListAPIKeys(ctx context.Context, orgID uuid.UUID) ([]*StoredAPIKey, error) {
	query := `
		SELECT id, organization_id, user_id, name, key_prefix, key_hash, role, last_used_at, created_at, revoked_at
		FROM api_keys
		WHERE organization_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*StoredAPIKey
	for rows.Next() {
		var k StoredAPIKey
		var userID *uuid.UUID
		var lastUsed, revoked sql.NullTime
		if err := rows.Scan(
			&k.ID, &k.OrganizationID, &userID, &k.Name, &k.KeyPrefix, &k.KeyHash,
			&k.Role, &lastUsed, &k.CreatedAt, &revoked,
		); err != nil {
			return nil, err
		}
		k.UserID = userID
		if lastUsed.Valid {
			k.LastUsedAt = &lastUsed.Time
		}
		if revoked.Valid {
			k.RevokedAt = &revoked.Time
		}
		list = append(list, &k)
	}
	return list, rows.Err()
}

func (s *PostgresStore) RevokeAPIKey(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `UPDATE api_keys SET revoked_at = NOW() WHERE id = $1`, id)
	return err
}
