package store

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MEMORY STORE: CACHING ENGINE, REMOTE DB ACCESS, DATABASE MAINTENANCE
// ============================================================================

// Website Cache Config
func (m *MemoryStore) GetWebsiteCacheConfig(ctx context.Context, websiteID uuid.UUID) (*WebsiteCacheConfigRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cfg, ok := m.websiteCacheConfigs[websiteID]
	if !ok {
		return nil, ErrNotFound
	}
	copyCfg := *cfg
	return &copyCfg, nil
}

func (m *MemoryStore) SaveWebsiteCacheConfig(ctx context.Context, cfg *WebsiteCacheConfigRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.ID == uuid.Nil {
		cfg.ID = uuid.New()
	}
	now := time.Now().UTC()
	if existing, ok := m.websiteCacheConfigs[cfg.WebsiteID]; ok {
		cfg.CreatedAt = existing.CreatedAt
	} else if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now

	copyCfg := *cfg
	m.websiteCacheConfigs[cfg.WebsiteID] = &copyCfg
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteWebsiteCacheConfig(ctx context.Context, websiteID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.websiteCacheConfigs[websiteID]; !ok {
		return ErrNotFound
	}
	delete(m.websiteCacheConfigs, websiteID)
	m.saveToDiskLocked()
	return nil
}

// Remote Database Access Grants
func (m *MemoryStore) ListDatabaseRemoteAccess(ctx context.Context, databaseID uuid.UUID) ([]*DatabaseRemoteAccessRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*DatabaseRemoteAccessRecord
	for _, rec := range m.databaseRemoteAccess {
		if rec.DatabaseID == databaseID {
			copyRec := *rec
			list = append(list, &copyRec)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	return list, nil
}

func (m *MemoryStore) GetDatabaseRemoteAccessByID(ctx context.Context, id uuid.UUID) (*DatabaseRemoteAccessRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rec, ok := m.databaseRemoteAccess[id]
	if !ok {
		return nil, ErrNotFound
	}
	copyRec := *rec
	return &copyRec, nil
}

func (m *MemoryStore) CreateDatabaseRemoteAccess(ctx context.Context, rec *DatabaseRemoteAccessRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rec.ID == uuid.Nil {
		rec.ID = uuid.New()
	}
	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.UpdatedAt = now

	copyRec := *rec
	m.databaseRemoteAccess[rec.ID] = &copyRec
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) UpdateDatabaseRemoteAccess(ctx context.Context, rec *DatabaseRemoteAccessRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.databaseRemoteAccess[rec.ID]
	if !ok {
		return ErrNotFound
	}
	rec.CreatedAt = existing.CreatedAt
	rec.UpdatedAt = time.Now().UTC()

	copyRec := *rec
	m.databaseRemoteAccess[rec.ID] = &copyRec
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteDatabaseRemoteAccess(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.databaseRemoteAccess[id]; !ok {
		return ErrNotFound
	}
	delete(m.databaseRemoteAccess, id)
	m.saveToDiskLocked()
	return nil
}

// Database Maintenance Logs
func (m *MemoryStore) RecordDatabaseMaintenanceLog(ctx context.Context, log *DatabaseMaintenanceLogRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}

	copyLog := *log
	m.databaseMaintenanceLogs = append(m.databaseMaintenanceLogs, &copyLog)
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) ListDatabaseMaintenanceLogs(ctx context.Context, databaseID uuid.UUID, limit int) ([]*DatabaseMaintenanceLogRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matched []*DatabaseMaintenanceLogRecord
	for _, l := range m.databaseMaintenanceLogs {
		if l.DatabaseID == databaseID {
			copyLog := *l
			matched = append(matched, &copyLog)
		}
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].CreatedAt.After(matched[j].CreatedAt)
	})

	if limit > 0 && len(matched) > limit {
		matched = matched[:limit]
	}
	return matched, nil
}

// ============================================================================
// POSTGRES STORE: CACHING ENGINE, REMOTE DB ACCESS, DATABASE MAINTENANCE
// ============================================================================

func (s *PostgresStore) GetWebsiteCacheConfig(ctx context.Context, websiteID uuid.UUID) (*WebsiteCacheConfigRecord, error) {
	query := `
		SELECT id, website_id, organization_id, cache_type, connection_type, socket_path,
		       host, port, password, max_memory_mb, eviction_policy, wp_object_cache_enabled,
		       status, created_at, updated_at
		FROM website_cache_configs
		WHERE website_id = $1
		LIMIT 1
	`
	var cfg WebsiteCacheConfigRecord
	err := s.db.QueryRowContext(ctx, query, websiteID).Scan(
		&cfg.ID, &cfg.WebsiteID, &cfg.OrganizationID, &cfg.CacheType, &cfg.ConnectionType,
		&cfg.SocketPath, &cfg.Host, &cfg.Port, &cfg.Password, &cfg.MaxMemoryMB,
		&cfg.EvictionPolicy, &cfg.WPObjectCacheEnabled, &cfg.Status, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *PostgresStore) SaveWebsiteCacheConfig(ctx context.Context, cfg *WebsiteCacheConfigRecord) error {
	if cfg.ID == uuid.Nil {
		cfg.ID = uuid.New()
	}
	now := time.Now().UTC()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now

	query := `
		INSERT INTO website_cache_configs (
			id, website_id, organization_id, cache_type, connection_type, socket_path,
			host, port, password, max_memory_mb, eviction_policy, wp_object_cache_enabled,
			status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		ON CONFLICT (website_id) DO UPDATE SET
			cache_type = EXCLUDED.cache_type,
			connection_type = EXCLUDED.connection_type,
			socket_path = EXCLUDED.socket_path,
			host = EXCLUDED.host,
			port = EXCLUDED.port,
			password = EXCLUDED.password,
			max_memory_mb = EXCLUDED.max_memory_mb,
			eviction_policy = EXCLUDED.eviction_policy,
			wp_object_cache_enabled = EXCLUDED.wp_object_cache_enabled,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`
	_, err := s.db.ExecContext(ctx, query,
		cfg.ID, cfg.WebsiteID, cfg.OrganizationID, cfg.CacheType, cfg.ConnectionType,
		cfg.SocketPath, cfg.Host, cfg.Port, cfg.Password, cfg.MaxMemoryMB,
		cfg.EvictionPolicy, cfg.WPObjectCacheEnabled, cfg.Status, cfg.CreatedAt, cfg.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) DeleteWebsiteCacheConfig(ctx context.Context, websiteID uuid.UUID) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM website_cache_configs WHERE website_id = $1`, websiteID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ListDatabaseRemoteAccess(ctx context.Context, databaseID uuid.UUID) ([]*DatabaseRemoteAccessRecord, error) {
	query := `
		SELECT id, database_id, organization_id, db_user, authorized_ip,
		       granted_privileges, firewall_rule_applied, status, notes,
		       expires_at, created_at, updated_at
		FROM database_remote_access
		WHERE database_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, databaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DatabaseRemoteAccessRecord
	for rows.Next() {
		var rec DatabaseRemoteAccessRecord
		if err := rows.Scan(
			&rec.ID, &rec.DatabaseID, &rec.OrganizationID, &rec.DBUser, &rec.AuthorizedIP,
			&rec.GrantedPrivileges, &rec.FirewallRuleApplied, &rec.Status, &rec.Notes,
			&rec.ExpiresAt, &rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &rec)
	}
	return list, rows.Err()
}

func (s *PostgresStore) GetDatabaseRemoteAccessByID(ctx context.Context, id uuid.UUID) (*DatabaseRemoteAccessRecord, error) {
	query := `
		SELECT id, database_id, organization_id, db_user, authorized_ip,
		       granted_privileges, firewall_rule_applied, status, notes,
		       expires_at, created_at, updated_at
		FROM database_remote_access
		WHERE id = $1
		LIMIT 1
	`
	var rec DatabaseRemoteAccessRecord
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&rec.ID, &rec.DatabaseID, &rec.OrganizationID, &rec.DBUser, &rec.AuthorizedIP,
		&rec.GrantedPrivileges, &rec.FirewallRuleApplied, &rec.Status, &rec.Notes,
		&rec.ExpiresAt, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *PostgresStore) CreateDatabaseRemoteAccess(ctx context.Context, rec *DatabaseRemoteAccessRecord) error {
	if rec.ID == uuid.Nil {
		rec.ID = uuid.New()
	}
	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.UpdatedAt = now

	query := `
		INSERT INTO database_remote_access (
			id, database_id, organization_id, db_user, authorized_ip,
			granted_privileges, firewall_rule_applied, status, notes,
			expires_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`
	_, err := s.db.ExecContext(ctx, query,
		rec.ID, rec.DatabaseID, rec.OrganizationID, rec.DBUser, rec.AuthorizedIP,
		rec.GrantedPrivileges, rec.FirewallRuleApplied, rec.Status, rec.Notes,
		rec.ExpiresAt, rec.CreatedAt, rec.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) UpdateDatabaseRemoteAccess(ctx context.Context, rec *DatabaseRemoteAccessRecord) error {
	rec.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE database_remote_access SET
			db_user = $1,
			authorized_ip = $2,
			granted_privileges = $3,
			firewall_rule_applied = $4,
			status = $5,
			notes = $6,
			expires_at = $7,
			updated_at = $8
		WHERE id = $9
	`
	res, err := s.db.ExecContext(ctx, query,
		rec.DBUser, rec.AuthorizedIP, rec.GrantedPrivileges, rec.FirewallRuleApplied,
		rec.Status, rec.Notes, rec.ExpiresAt, rec.UpdatedAt, rec.ID,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteDatabaseRemoteAccess(ctx context.Context, id uuid.UUID) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM database_remote_access WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) RecordDatabaseMaintenanceLog(ctx context.Context, log *DatabaseMaintenanceLogRecord) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO database_maintenance_logs (
			id, database_id, organization_id, operation, status,
			tables_processed, bytes_reclaimed, duration_ms, details,
			initiated_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`
	_, err := s.db.ExecContext(ctx, query,
		log.ID, log.DatabaseID, log.OrganizationID, log.Operation, log.Status,
		log.TablesProcessed, log.BytesReclaimed, log.DurationMs, log.Details,
		log.InitiatedBy, log.CreatedAt,
	)
	return err
}

func (s *PostgresStore) ListDatabaseMaintenanceLogs(ctx context.Context, databaseID uuid.UUID, limit int) ([]*DatabaseMaintenanceLogRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, database_id, organization_id, operation, status,
		       tables_processed, bytes_reclaimed, duration_ms, details,
		       initiated_by, created_at
		FROM database_maintenance_logs
		WHERE database_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := s.db.QueryContext(ctx, query, databaseID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DatabaseMaintenanceLogRecord
	for rows.Next() {
		var l DatabaseMaintenanceLogRecord
		if err := rows.Scan(
			&l.ID, &l.DatabaseID, &l.OrganizationID, &l.Operation, &l.Status,
			&l.TablesProcessed, &l.BytesReclaimed, &l.DurationMs, &l.Details,
			&l.InitiatedBy, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &l)
	}
	return list, rows.Err()
}
