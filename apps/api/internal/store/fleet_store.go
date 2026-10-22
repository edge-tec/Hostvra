package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrServerHasActiveWorkloads = errors.New("cannot delete server with active workloads: migrate or drain workloads first")
	ErrMigrationNotFound        = errors.New("cross-node migration job not found")
)

// ============================================================================
// MEMORY STORE IMPLEMENTATION
// ============================================================================

func (m *MemoryStore) UpdateServerMaintenance(ctx context.Context, id uuid.UUID, isMaintenance bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	server, ok := m.servers[id]
	if !ok {
		return ErrNotFound
	}

	server.IsMaintenance = isMaintenance
	if isMaintenance {
		server.Status = "maintenance"
	} else if server.Status == "maintenance" {
		server.Status = "online"
	}
	server.UpdatedAt = time.Now().UTC()
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) UpdateServerDraining(ctx context.Context, id uuid.UUID, draining bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	server, ok := m.servers[id]
	if !ok {
		return ErrNotFound
	}

	server.Draining = draining
	if draining {
		server.IsMaintenance = true
		server.Status = "draining"
	} else if server.Status == "draining" {
		server.Status = "online"
		server.IsMaintenance = false
	}
	server.UpdatedAt = time.Now().UTC()
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) UpdateServerCapabilities(ctx context.Context, id uuid.UUID, caps ServerCapabilities) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	server, ok := m.servers[id]
	if !ok {
		return ErrNotFound
	}

	server.Capabilities = caps
	server.UpdatedAt = time.Now().UTC()
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetServerWorkloadSummary(ctx context.Context, id uuid.UUID) (*ServerDrainSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, ok := m.servers[id]
	if !ok {
		return nil, ErrNotFound
	}

	websitesCount := 0
	for _, site := range m.websites {
		if site.ServerID == id && site.DeletedAt == nil {
			websitesCount++
		}
	}

	databasesCount := 0
	for _, db := range m.databases {
		if db.ServerID == id && db.DeletedAt == nil {
			databasesCount++
		}
	}

	mailboxesCount := 0
	for _, ms := range m.mailServers {
		if ms.NodeServerID == id && ms.DeletedAt == nil {
			for _, mb := range m.emailMailboxes {
				if mb.ServerID == ms.ID {
					mailboxesCount++
				}
			}
		}
	}

	activeWorkloads := websitesCount + databasesCount + mailboxesCount

	return &ServerDrainSummary{
		ServerID:        id,
		Status:          server.Status,
		WebsitesCount:   websitesCount,
		DatabasesCount:  databasesCount,
		MailboxesCount:  mailboxesCount,
		ActiveWorkloads: activeWorkloads,
		CanDecommission: activeWorkloads == 0,
	}, nil
}

func (m *MemoryStore) DeleteServer(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	server, ok := m.servers[id]
	if !ok {
		return ErrNotFound
	}

	// Verify workload draining
	websitesCount := 0
	for _, site := range m.websites {
		if site.ServerID == id && site.DeletedAt == nil {
			websitesCount++
		}
	}
	databasesCount := 0
	for _, db := range m.databases {
		if db.ServerID == id && db.DeletedAt == nil {
			databasesCount++
		}
	}

	if websitesCount > 0 || databasesCount > 0 {
		return ErrServerHasActiveWorkloads
	}

	delete(m.servers, id)
	delete(m.metrics, id)
	_ = server
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) ForceDeleteServer(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.servers[id]
	if !ok {
		return ErrNotFound
	}

	delete(m.servers, id)
	delete(m.metrics, id)
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) CreateCrossNodeMigration(ctx context.Context, job *CrossNodeMigration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.crossNodeMigrations == nil {
		m.crossNodeMigrations = make(map[uuid.UUID]*CrossNodeMigration)
	}

	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now
	if job.Logs == nil {
		job.Logs = make([]MigrationLogItem, 0)
	}

	m.crossNodeMigrations[job.ID] = job
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetCrossNodeMigration(ctx context.Context, id uuid.UUID) (*CrossNodeMigration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.crossNodeMigrations == nil {
		return nil, ErrMigrationNotFound
	}

	job, ok := m.crossNodeMigrations[id]
	if !ok {
		return nil, ErrMigrationNotFound
	}
	return job, nil
}

func (m *MemoryStore) ListCrossNodeMigrations(ctx context.Context, orgID uuid.UUID) ([]*CrossNodeMigration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*CrossNodeMigration
	for _, j := range m.crossNodeMigrations {
		if j.OrganizationID == orgID {
			list = append(list, j)
		}
	}
	return list, nil
}

func (m *MemoryStore) UpdateCrossNodeMigration(ctx context.Context, job *CrossNodeMigration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.crossNodeMigrations == nil {
		return ErrMigrationNotFound
	}

	existing, ok := m.crossNodeMigrations[job.ID]
	if !ok {
		return ErrMigrationNotFound
	}

	job.UpdatedAt = time.Now().UTC()
	*existing = *job
	m.saveToDiskLocked()
	return nil
}

// ============================================================================
// POSTGRES STORE IMPLEMENTATION
// ============================================================================

func (p *PostgresStore) UpdateServerMaintenance(ctx context.Context, id uuid.UUID, isMaintenance bool) error {
	status := "online"
	if isMaintenance {
		status = "maintenance"
	}

	query := `
		UPDATE servers
		SET is_maintenance = $1, status = $2, updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
	`
	res, err := p.db.ExecContext(ctx, query, isMaintenance, status, id)
	if err != nil {
		return fmt.Errorf("failed to update server maintenance: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) UpdateServerDraining(ctx context.Context, id uuid.UUID, draining bool) error {
	status := "online"
	isMaint := false
	if draining {
		status = "draining"
		isMaint = true
	}

	query := `
		UPDATE servers
		SET draining = $1, is_maintenance = $2, status = $3, updated_at = NOW()
		WHERE id = $4 AND deleted_at IS NULL
	`
	res, err := p.db.ExecContext(ctx, query, draining, isMaint, status, id)
	if err != nil {
		return fmt.Errorf("failed to update server draining: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) UpdateServerCapabilities(ctx context.Context, id uuid.UUID, caps ServerCapabilities) error {
	capsJSON, err := json.Marshal(caps)
	if err != nil {
		return err
	}

	query := `
		UPDATE servers
		SET capabilities = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`
	res, err := p.db.ExecContext(ctx, query, capsJSON, id)
	if err != nil {
		return fmt.Errorf("failed to update server capabilities: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) GetServerWorkloadSummary(ctx context.Context, id uuid.UUID) (*ServerDrainSummary, error) {
	var status string
	err := p.db.QueryRowContext(ctx, "SELECT status FROM servers WHERE id = $1 AND deleted_at IS NULL", id).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var sitesCount, dbsCount, mailCount int

	_ = p.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM websites WHERE server_id = $1 AND deleted_at IS NULL", id).Scan(&sitesCount)
	_ = p.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM databases WHERE server_id = $1 AND deleted_at IS NULL", id).Scan(&dbsCount)
	_ = p.db.QueryRowContext(ctx, `
		SELECT COUNT(em.id) 
		FROM email_mailboxes em
		JOIN mail_servers ms ON em.server_id = ms.id
		WHERE ms.server_id = $1 AND ms.deleted_at IS NULL
	`, id).Scan(&mailCount)

	active := sitesCount + dbsCount + mailCount

	return &ServerDrainSummary{
		ServerID:        id,
		Status:          status,
		WebsitesCount:   sitesCount,
		DatabasesCount:  dbsCount,
		MailboxesCount:  mailCount,
		ActiveWorkloads: active,
		CanDecommission: active == 0,
	}, nil
}

func (p *PostgresStore) DeleteServer(ctx context.Context, id uuid.UUID) error {
	summary, err := p.GetServerWorkloadSummary(ctx, id)
	if err != nil {
		return err
	}
	if summary.ActiveWorkloads > 0 {
		return ErrServerHasActiveWorkloads
	}

	query := `
		UPDATE servers
		SET deleted_at = NOW(), status = 'deleted', updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	res, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to soft-delete server: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) ForceDeleteServer(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE servers
		SET deleted_at = NOW(), status = 'deleted', updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	res, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to force soft-delete server: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) CreateCrossNodeMigration(ctx context.Context, job *CrossNodeMigration) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now

	logsJSON, err := json.Marshal(job.Logs)
	if err != nil {
		logsJSON = []byte("[]")
	}

	query := `
		INSERT INTO cross_node_migrations (
			id, organization_id, source_server_id, destination_server_id,
			website_id, database_id, status, progress, current_step,
			source_checksum, destination_checksum, transferred_bytes, total_bytes,
			error_message, logs, created_by, started_at, completed_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)
	`
	_, err = p.db.ExecContext(ctx, query,
		job.ID, job.OrganizationID, job.SourceServerID, job.DestinationServerID,
		job.WebsiteID, job.DatabaseID, job.Status, job.Progress, job.CurrentStep,
		job.SourceChecksum, job.DestinationChecksum, job.TransferredBytes, job.TotalBytes,
		job.ErrorMessage, logsJSON, job.CreatedBy, job.StartedAt, job.CompletedAt, job.CreatedAt, job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert cross_node_migration: %w", err)
	}
	return nil
}

func (p *PostgresStore) GetCrossNodeMigration(ctx context.Context, id uuid.UUID) (*CrossNodeMigration, error) {
	query := `
		SELECT id, organization_id, source_server_id, destination_server_id,
		       website_id, database_id, status, progress, current_step,
		       source_checksum, destination_checksum, transferred_bytes, total_bytes,
		       error_message, logs, created_by, started_at, completed_at, created_at, updated_at
		FROM cross_node_migrations
		WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)

	var m CrossNodeMigration
	var logsJSON []byte
	err := row.Scan(
		&m.ID, &m.OrganizationID, &m.SourceServerID, &m.DestinationServerID,
		&m.WebsiteID, &m.DatabaseID, &m.Status, &m.Progress, &m.CurrentStep,
		&m.SourceChecksum, &m.DestinationChecksum, &m.TransferredBytes, &m.TotalBytes,
		&m.ErrorMessage, &logsJSON, &m.CreatedBy, &m.StartedAt, &m.CompletedAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMigrationNotFound
		}
		return nil, err
	}

	if len(logsJSON) > 0 {
		_ = json.Unmarshal(logsJSON, &m.Logs)
	}
	return &m, nil
}

func (p *PostgresStore) ListCrossNodeMigrations(ctx context.Context, orgID uuid.UUID) ([]*CrossNodeMigration, error) {
	query := `
		SELECT id, organization_id, source_server_id, destination_server_id,
		       website_id, database_id, status, progress, current_step,
		       source_checksum, destination_checksum, transferred_bytes, total_bytes,
		       error_message, logs, created_by, started_at, completed_at, created_at, updated_at
		FROM cross_node_migrations
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*CrossNodeMigration
	for rows.Next() {
		var m CrossNodeMigration
		var logsJSON []byte
		if err := rows.Scan(
			&m.ID, &m.OrganizationID, &m.SourceServerID, &m.DestinationServerID,
			&m.WebsiteID, &m.DatabaseID, &m.Status, &m.Progress, &m.CurrentStep,
			&m.SourceChecksum, &m.DestinationChecksum, &m.TransferredBytes, &m.TotalBytes,
			&m.ErrorMessage, &logsJSON, &m.CreatedBy, &m.StartedAt, &m.CompletedAt, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if len(logsJSON) > 0 {
			_ = json.Unmarshal(logsJSON, &m.Logs)
		}
		list = append(list, &m)
	}
	return list, nil
}

func (p *PostgresStore) UpdateCrossNodeMigration(ctx context.Context, job *CrossNodeMigration) error {
	job.UpdatedAt = time.Now().UTC()
	logsJSON, err := json.Marshal(job.Logs)
	if err != nil {
		logsJSON = []byte("[]")
	}

	query := `
		UPDATE cross_node_migrations
		SET status = $1, progress = $2, current_step = $3,
		    source_checksum = $4, destination_checksum = $5,
		    transferred_bytes = $6, total_bytes = $7, error_message = $8,
		    logs = $9, completed_at = $10, updated_at = $11
		WHERE id = $12
	`
	res, err := p.db.ExecContext(ctx, query,
		job.Status, job.Progress, job.CurrentStep,
		job.SourceChecksum, job.DestinationChecksum,
		job.TransferredBytes, job.TotalBytes, job.ErrorMessage,
		logsJSON, job.CompletedAt, job.UpdatedAt, job.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update cross_node_migration: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrMigrationNotFound
	}
	return nil
}
