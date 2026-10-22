package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MEMORY STORE: RESOURCE QUOTAS & CGROUPS ISOLATION
// ============================================================================

func (m *MemoryStore) GetWebsiteResourceQuota(ctx context.Context, websiteID uuid.UUID) (*WebsiteResourceQuotaRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	quota, ok := m.websiteResourceQuotas[websiteID]
	if !ok {
		return nil, ErrNotFound
	}
	// Return a copy
	qCopy := *quota
	return &qCopy, nil
}

func (m *MemoryStore) SaveWebsiteResourceQuota(ctx context.Context, quota *WebsiteResourceQuotaRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if existing, ok := m.websiteResourceQuotas[quota.WebsiteID]; ok {
		quota.CreatedAt = existing.CreatedAt
	} else if quota.CreatedAt.IsZero() {
		quota.CreatedAt = now
	}
	quota.UpdatedAt = now

	// Defaults if unset
	if quota.MemoryLimitMB <= 0 {
		quota.MemoryLimitMB = 512
	}
	if quota.MemoryHighMB <= 0 {
		quota.MemoryHighMB = int(float64(quota.MemoryLimitMB) * 0.90)
	}
	if quota.CPUQuotaPercent <= 0 {
		quota.CPUQuotaPercent = 100
	}
	if quota.CPUWeight <= 0 {
		quota.CPUWeight = 100
	}
	if quota.TasksMax <= 0 {
		quota.TasksMax = 100
	}
	if quota.IOReadMBps <= 0 {
		quota.IOReadMBps = 50
	}
	if quota.IOWriteMBps <= 0 {
		quota.IOWriteMBps = 30
	}
	if quota.IOReadIOPS <= 0 {
		quota.IOReadIOPS = 1000
	}
	if quota.IOWriteIOPS <= 0 {
		quota.IOWriteIOPS = 500
	}
	if quota.OOMPolicy == "" {
		quota.OOMPolicy = "kill"
	}

	qCopy := *quota
	m.websiteResourceQuotas[quota.WebsiteID] = &qCopy
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) RecordQuotaEvent(ctx context.Context, event *ResourceQuotaEventRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.Severity == "" {
		event.Severity = "warning"
	}

	eCopy := *event
	m.resourceQuotaEvents = append(m.resourceQuotaEvents, &eCopy)
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) ListQuotaEvents(ctx context.Context, websiteID uuid.UUID, limit int) ([]*ResourceQuotaEventRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var matched []*ResourceQuotaEventRecord
	// iterate in reverse (newest first)
	for i := len(m.resourceQuotaEvents) - 1; i >= 0; i-- {
		ev := m.resourceQuotaEvents[i]
		if websiteID == uuid.Nil || ev.WebsiteID == websiteID {
			eCopy := *ev
			matched = append(matched, &eCopy)
			if len(matched) >= limit {
				break
			}
		}
	}
	return matched, nil
}

// ============================================================================
// POSTGRES STORE: RESOURCE QUOTAS & CGROUPS ISOLATION
// ============================================================================

func (s *PostgresStore) GetWebsiteResourceQuota(ctx context.Context, websiteID uuid.UUID) (*WebsiteResourceQuotaRecord, error) {
	query := `
		SELECT website_id, username, memory_limit_mb, memory_high_mb, cpu_quota_percent, cpu_weight,
		       tasks_max, io_read_mbps, io_write_mbps, io_read_iops, io_write_iops, oom_policy,
		       open_basedir_enabled, created_at, updated_at
		FROM website_resource_quotas
		WHERE website_id = $1`

	var q WebsiteResourceQuotaRecord
	err := s.db.QueryRowContext(ctx, query, websiteID).Scan(
		&q.WebsiteID,
		&q.Username,
		&q.MemoryLimitMB,
		&q.MemoryHighMB,
		&q.CPUQuotaPercent,
		&q.CPUWeight,
		&q.TasksMax,
		&q.IOReadMBps,
		&q.IOWriteMBps,
		&q.IOReadIOPS,
		&q.IOWriteIOPS,
		&q.OOMPolicy,
		&q.OpenBaseDirEnabled,
		&q.CreatedAt,
		&q.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (s *PostgresStore) SaveWebsiteResourceQuota(ctx context.Context, quota *WebsiteResourceQuotaRecord) error {
	now := time.Now().UTC()
	quota.UpdatedAt = now
	if quota.CreatedAt.IsZero() {
		quota.CreatedAt = now
	}

	query := `
		INSERT INTO website_resource_quotas (
			website_id, username, memory_limit_mb, memory_high_mb, cpu_quota_percent, cpu_weight,
			tasks_max, io_read_mbps, io_write_mbps, io_read_iops, io_write_iops, oom_policy,
			open_basedir_enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (website_id) DO UPDATE SET
			username = EXCLUDED.username,
			memory_limit_mb = EXCLUDED.memory_limit_mb,
			memory_high_mb = EXCLUDED.memory_high_mb,
			cpu_quota_percent = EXCLUDED.cpu_quota_percent,
			cpu_weight = EXCLUDED.cpu_weight,
			tasks_max = EXCLUDED.tasks_max,
			io_read_mbps = EXCLUDED.io_read_mbps,
			io_write_mbps = EXCLUDED.io_write_mbps,
			io_read_iops = EXCLUDED.io_read_iops,
			io_write_iops = EXCLUDED.io_write_iops,
			oom_policy = EXCLUDED.oom_policy,
			open_basedir_enabled = EXCLUDED.open_basedir_enabled,
			updated_at = EXCLUDED.updated_at`

	_, err := s.db.ExecContext(ctx, query,
		quota.WebsiteID,
		quota.Username,
		quota.MemoryLimitMB,
		quota.MemoryHighMB,
		quota.CPUQuotaPercent,
		quota.CPUWeight,
		quota.TasksMax,
		quota.IOReadMBps,
		quota.IOWriteMBps,
		quota.IOReadIOPS,
		quota.IOWriteIOPS,
		quota.OOMPolicy,
		quota.OpenBaseDirEnabled,
		quota.CreatedAt,
		quota.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) RecordQuotaEvent(ctx context.Context, event *ResourceQuotaEventRecord) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.Severity == "" {
		event.Severity = "warning"
	}

	query := `
		INSERT INTO resource_quota_events (
			id, website_id, username, event_type, severity, details, metric_value, threshold_value, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := s.db.ExecContext(ctx, query,
		event.ID,
		event.WebsiteID,
		event.Username,
		event.EventType,
		event.Severity,
		event.Details,
		event.MetricValue,
		event.ThresholdValue,
		event.CreatedAt,
	)
	return err
}

func (s *PostgresStore) ListQuotaEvents(ctx context.Context, websiteID uuid.UUID, limit int) ([]*ResourceQuotaEventRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var rows *sql.Rows
	var err error

	if websiteID != uuid.Nil {
		query := `
			SELECT id, website_id, username, event_type, severity, details, metric_value, threshold_value, created_at
			FROM resource_quota_events
			WHERE website_id = $1
			ORDER BY created_at DESC
			LIMIT $2`
		rows, err = s.db.QueryContext(ctx, query, websiteID, limit)
	} else {
		query := `
			SELECT id, website_id, username, event_type, severity, details, metric_value, threshold_value, created_at
			FROM resource_quota_events
			ORDER BY created_at DESC
			LIMIT $1`
		rows, err = s.db.QueryContext(ctx, query, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*ResourceQuotaEventRecord
	for rows.Next() {
		var ev ResourceQuotaEventRecord
		if err := rows.Scan(
			&ev.ID,
			&ev.WebsiteID,
			&ev.Username,
			&ev.EventType,
			&ev.Severity,
			&ev.Details,
			&ev.MetricValue,
			&ev.ThresholdValue,
			&ev.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, &ev)
	}
	return events, rows.Err()
}
