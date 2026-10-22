package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MEMORY STORE: BACKUP DESTINATIONS
// ============================================================================

func (m *MemoryStore) CreateBackupDestination(ctx context.Context, dest *BackupDestinationRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if dest.ID == uuid.Nil {
		dest.ID = uuid.New()
	}
	now := time.Now().UTC()
	dest.CreatedAt = now
	dest.UpdatedAt = now

	// If marked default, unset other defaults in the same org
	if dest.IsDefault {
		for _, d := range m.backupDestinations {
			if d.OrganizationID == dest.OrganizationID {
				d.IsDefault = false
			}
		}
	}

	m.backupDestinations[dest.ID] = dest
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetBackupDestinationByID(ctx context.Context, id uuid.UUID) (*BackupDestinationRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dest, ok := m.backupDestinations[id]
	if !ok {
		return nil, ErrNotFound
	}
	return dest, nil
}

func (m *MemoryStore) ListBackupDestinations(ctx context.Context, orgID uuid.UUID) ([]*BackupDestinationRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*BackupDestinationRecord
	for _, d := range m.backupDestinations {
		if d.OrganizationID == orgID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *MemoryStore) UpdateBackupDestination(ctx context.Context, dest *BackupDestinationRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.backupDestinations[dest.ID]
	if !ok {
		return ErrNotFound
	}

	dest.UpdatedAt = time.Now().UTC()
	dest.CreatedAt = existing.CreatedAt

	if dest.IsDefault {
		for _, d := range m.backupDestinations {
			if d.OrganizationID == dest.OrganizationID && d.ID != dest.ID {
				d.IsDefault = false
			}
		}
	}

	m.backupDestinations[dest.ID] = dest
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteBackupDestination(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.backupDestinations[id]; !ok {
		return ErrNotFound
	}
	delete(m.backupDestinations, id)
	m.saveToDiskLocked()
	return nil
}

// ============================================================================
// MEMORY STORE: BACKUP SNAPSHOTS
// ============================================================================

func (m *MemoryStore) CreateBackupSnapshot(ctx context.Context, snap *BackupSnapshotRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if snap.ID == uuid.Nil {
		snap.ID = uuid.New()
	}
	if snap.CreatedAt.IsZero() {
		snap.CreatedAt = time.Now().UTC()
	}
	if snap.IntegrityStatus == "" {
		snap.IntegrityStatus = "unverified"
	}
	if snap.Status == "" {
		snap.Status = "completed"
	}

	m.backupSnapshots[snap.ID] = snap
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetBackupSnapshotByID(ctx context.Context, id uuid.UUID) (*BackupSnapshotRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snap, ok := m.backupSnapshots[id]
	if !ok {
		return nil, ErrNotFound
	}
	return snap, nil
}

func (m *MemoryStore) ListBackupSnapshots(ctx context.Context, orgID uuid.UUID, serverID *uuid.UUID) ([]*BackupSnapshotRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*BackupSnapshotRecord
	for _, s := range m.backupSnapshots {
		if s.OrganizationID == orgID {
			if serverID != nil && s.ServerID != nil && *s.ServerID != *serverID {
				continue
			}
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *MemoryStore) UpdateBackupSnapshot(ctx context.Context, snap *BackupSnapshotRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.backupSnapshots[snap.ID]; !ok {
		return ErrNotFound
	}
	m.backupSnapshots[snap.ID] = snap
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteBackupSnapshot(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.backupSnapshots[id]; !ok {
		return ErrNotFound
	}
	delete(m.backupSnapshots, id)
	m.saveToDiskLocked()
	return nil
}

// ============================================================================
// MEMORY STORE: BACKUP SCHEDULES
// ============================================================================

func (m *MemoryStore) CreateBackupSchedule(ctx context.Context, sched *BackupScheduleRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sched.ID == uuid.Nil {
		sched.ID = uuid.New()
	}
	now := time.Now().UTC()
	sched.CreatedAt = now
	sched.UpdatedAt = now

	m.backupSchedules[sched.ID] = sched
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetBackupScheduleByID(ctx context.Context, id uuid.UUID) (*BackupScheduleRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sched, ok := m.backupSchedules[id]
	if !ok {
		return nil, ErrNotFound
	}
	return sched, nil
}

func (m *MemoryStore) ListBackupSchedules(ctx context.Context, orgID uuid.UUID) ([]*BackupScheduleRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*BackupScheduleRecord
	for _, s := range m.backupSchedules {
		if s.OrganizationID == orgID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *MemoryStore) UpdateBackupSchedule(ctx context.Context, sched *BackupScheduleRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.backupSchedules[sched.ID]
	if !ok {
		return ErrNotFound
	}
	sched.UpdatedAt = time.Now().UTC()
	sched.CreatedAt = existing.CreatedAt

	m.backupSchedules[sched.ID] = sched
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteBackupSchedule(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.backupSchedules[id]; !ok {
		return ErrNotFound
	}
	delete(m.backupSchedules, id)
	m.saveToDiskLocked()
	return nil
}

// ============================================================================
// MEMORY STORE: BACKUP RESTORE JOBS
// ============================================================================

func (m *MemoryStore) CreateBackupRestoreJob(ctx context.Context, job *BackupRestoreJobRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	m.backupRestoreJobs[job.ID] = job
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetBackupRestoreJobByID(ctx context.Context, id uuid.UUID) (*BackupRestoreJobRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.backupRestoreJobs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return job, nil
}

func (m *MemoryStore) ListBackupRestoreJobs(ctx context.Context, orgID uuid.UUID, limit int) ([]*BackupRestoreJobRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*BackupRestoreJobRecord
	for _, j := range m.backupRestoreJobs {
		if j.OrganizationID == orgID {
			result = append(result, j)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *MemoryStore) UpdateBackupRestoreJob(ctx context.Context, job *BackupRestoreJobRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.backupRestoreJobs[job.ID]; !ok {
		return ErrNotFound
	}
	m.backupRestoreJobs[job.ID] = job
	m.saveToDiskLocked()
	return nil
}

// ============================================================================
// POSTGRES STORE: BACKUP DESTINATIONS
// ============================================================================

func (p *PostgresStore) CreateBackupDestination(ctx context.Context, dest *BackupDestinationRecord) error {
	if dest.ID == uuid.Nil {
		dest.ID = uuid.New()
	}
	now := time.Now().UTC()
	dest.CreatedAt = now
	dest.UpdatedAt = now

	if dest.IsDefault {
		_, _ = p.db.ExecContext(ctx, `UPDATE backup_destinations SET is_default = FALSE WHERE organization_id = $1`, dest.OrganizationID)
	}

	query := `
		INSERT INTO backup_destinations (
			id, organization_id, server_id, name, type, endpoint, region,
			bucket, access_key, secret_key, prefix, is_default, is_active,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13,
			$14, $15
		)
	`
	_, err := p.db.ExecContext(ctx, query,
		dest.ID, dest.OrganizationID, dest.ServerID, dest.Name, dest.Type,
		dest.Endpoint, dest.Region, dest.Bucket, dest.AccessKey, dest.SecretKey,
		dest.Prefix, dest.IsDefault, dest.IsActive, dest.CreatedAt, dest.UpdatedAt,
	)
	return err
}

func (p *PostgresStore) GetBackupDestinationByID(ctx context.Context, id uuid.UUID) (*BackupDestinationRecord, error) {
	query := `
		SELECT id, organization_id, server_id, name, type, endpoint, region,
		       bucket, access_key, secret_key, prefix, is_default, is_active,
		       created_at, updated_at
		FROM backup_destinations
		WHERE id = $1
	`
	var d BackupDestinationRecord
	var serverID uuid.NullUUID
	var endpoint, region, bucket, accessKey, secretKey sql.NullString

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID, &d.OrganizationID, &serverID, &d.Name, &d.Type,
		&endpoint, &region, &bucket, &accessKey, &secretKey,
		&d.Prefix, &d.IsDefault, &d.IsActive, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if serverID.Valid {
		d.ServerID = &serverID.UUID
	}
	if endpoint.Valid {
		d.Endpoint = endpoint.String
	}
	if region.Valid {
		d.Region = region.String
	}
	if bucket.Valid {
		d.Bucket = bucket.String
	}
	if accessKey.Valid {
		d.AccessKey = accessKey.String
	}
	if secretKey.Valid {
		d.SecretKey = secretKey.String
	}
	return &d, nil
}

func (p *PostgresStore) ListBackupDestinations(ctx context.Context, orgID uuid.UUID) ([]*BackupDestinationRecord, error) {
	query := `
		SELECT id, organization_id, server_id, name, type, endpoint, region,
		       bucket, access_key, secret_key, prefix, is_default, is_active,
		       created_at, updated_at
		FROM backup_destinations
		WHERE organization_id = $1
		ORDER BY is_default DESC, name ASC
	`
	rows, err := p.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*BackupDestinationRecord
	for rows.Next() {
		var d BackupDestinationRecord
		var serverID uuid.NullUUID
		var endpoint, region, bucket, accessKey, secretKey sql.NullString

		if err := rows.Scan(
			&d.ID, &d.OrganizationID, &serverID, &d.Name, &d.Type,
			&endpoint, &region, &bucket, &accessKey, &secretKey,
			&d.Prefix, &d.IsDefault, &d.IsActive, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if serverID.Valid {
			d.ServerID = &serverID.UUID
		}
		if endpoint.Valid {
			d.Endpoint = endpoint.String
		}
		if region.Valid {
			d.Region = region.String
		}
		if bucket.Valid {
			d.Bucket = bucket.String
		}
		if accessKey.Valid {
			d.AccessKey = accessKey.String
		}
		if secretKey.Valid {
			d.SecretKey = secretKey.String
		}
		list = append(list, &d)
	}
	return list, nil
}

func (p *PostgresStore) UpdateBackupDestination(ctx context.Context, dest *BackupDestinationRecord) error {
	dest.UpdatedAt = time.Now().UTC()

	if dest.IsDefault {
		_, _ = p.db.ExecContext(ctx, `UPDATE backup_destinations SET is_default = FALSE WHERE organization_id = $1 AND id != $2`, dest.OrganizationID, dest.ID)
	}

	query := `
		UPDATE backup_destinations
		SET server_id = $1, name = $2, type = $3, endpoint = $4, region = $5,
		    bucket = $6, access_key = $7, secret_key = $8, prefix = $9,
		    is_default = $10, is_active = $11, updated_at = $12
		WHERE id = $13
	`
	res, err := p.db.ExecContext(ctx, query,
		dest.ServerID, dest.Name, dest.Type, dest.Endpoint, dest.Region,
		dest.Bucket, dest.AccessKey, dest.SecretKey, dest.Prefix,
		dest.IsDefault, dest.IsActive, dest.UpdatedAt, dest.ID,
	)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) DeleteBackupDestination(ctx context.Context, id uuid.UUID) error {
	res, err := p.db.ExecContext(ctx, `DELETE FROM backup_destinations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ============================================================================
// POSTGRES STORE: BACKUP SNAPSHOTS
// ============================================================================

func (p *PostgresStore) CreateBackupSnapshot(ctx context.Context, snap *BackupSnapshotRecord) error {
	if snap.ID == uuid.Nil {
		snap.ID = uuid.New()
	}
	if snap.CreatedAt.IsZero() {
		snap.CreatedAt = time.Now().UTC()
	}
	if snap.IntegrityStatus == "" {
		snap.IntegrityStatus = "unverified"
	}
	if snap.Status == "" {
		snap.Status = "completed"
	}

	query := `
		INSERT INTO backup_snapshots (
			id, organization_id, server_id, destination_id, type,
			target_name, storage, size_bytes, sha256_hash, item_count,
			status, file_name, remote_key, verified_at, integrity_status,
			error_message, created_at, completed_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18
		)
	`
	_, err := p.db.ExecContext(ctx, query,
		snap.ID, snap.OrganizationID, snap.ServerID, snap.DestinationID, snap.Type,
		snap.TargetName, snap.Storage, snap.SizeBytes, snap.SHA256Hash, snap.ItemCount,
		snap.Status, snap.FileName, snap.RemoteKey, snap.VerifiedAt, snap.IntegrityStatus,
		snap.ErrorMessage, snap.CreatedAt, snap.CompletedAt,
	)
	return err
}

func (p *PostgresStore) GetBackupSnapshotByID(ctx context.Context, id uuid.UUID) (*BackupSnapshotRecord, error) {
	query := `
		SELECT id, organization_id, server_id, destination_id, type,
		       target_name, storage, size_bytes, sha256_hash, item_count,
		       status, file_name, remote_key, verified_at, integrity_status,
		       error_message, created_at, completed_at
		FROM backup_snapshots
		WHERE id = $1
	`
	var s BackupSnapshotRecord
	var serverID, destinationID uuid.NullUUID
	var sha256, remoteKey, errMsg sql.NullString
	var verifiedAt, completedAt sql.NullTime

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.OrganizationID, &serverID, &destinationID, &s.Type,
		&s.TargetName, &s.Storage, &s.SizeBytes, &sha256, &s.ItemCount,
		&s.Status, &s.FileName, &remoteKey, &verifiedAt, &s.IntegrityStatus,
		&errMsg, &s.CreatedAt, &completedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if serverID.Valid {
		s.ServerID = &serverID.UUID
	}
	if destinationID.Valid {
		s.DestinationID = &destinationID.UUID
	}
	if sha256.Valid {
		s.SHA256Hash = sha256.String
	}
	if remoteKey.Valid {
		s.RemoteKey = remoteKey.String
	}
	if errMsg.Valid {
		s.ErrorMessage = errMsg.String
	}
	if verifiedAt.Valid {
		s.VerifiedAt = &verifiedAt.Time
	}
	if completedAt.Valid {
		s.CompletedAt = &completedAt.Time
	}
	return &s, nil
}

func (p *PostgresStore) ListBackupSnapshots(ctx context.Context, orgID uuid.UUID, serverID *uuid.UUID) ([]*BackupSnapshotRecord, error) {
	query := `
		SELECT id, organization_id, server_id, destination_id, type,
		       target_name, storage, size_bytes, sha256_hash, item_count,
		       status, file_name, remote_key, verified_at, integrity_status,
		       error_message, created_at, completed_at
		FROM backup_snapshots
		WHERE organization_id = $1 AND ($2::uuid IS NULL OR server_id = $2)
		ORDER BY created_at DESC
	`
	var filterServer interface{}
	if serverID != nil {
		filterServer = *serverID
	}

	rows, err := p.db.QueryContext(ctx, query, orgID, filterServer)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*BackupSnapshotRecord
	for rows.Next() {
		var s BackupSnapshotRecord
		var sID, dID uuid.NullUUID
		var sha256, remoteKey, errMsg sql.NullString
		var verifiedAt, completedAt sql.NullTime

		if err := rows.Scan(
			&s.ID, &s.OrganizationID, &sID, &dID, &s.Type,
			&s.TargetName, &s.Storage, &s.SizeBytes, &sha256, &s.ItemCount,
			&s.Status, &s.FileName, &remoteKey, &verifiedAt, &s.IntegrityStatus,
			&errMsg, &s.CreatedAt, &completedAt,
		); err != nil {
			return nil, err
		}
		if sID.Valid {
			s.ServerID = &sID.UUID
		}
		if dID.Valid {
			s.DestinationID = &dID.UUID
		}
		if sha256.Valid {
			s.SHA256Hash = sha256.String
		}
		if remoteKey.Valid {
			s.RemoteKey = remoteKey.String
		}
		if errMsg.Valid {
			s.ErrorMessage = errMsg.String
		}
		if verifiedAt.Valid {
			s.VerifiedAt = &verifiedAt.Time
		}
		if completedAt.Valid {
			s.CompletedAt = &completedAt.Time
		}
		list = append(list, &s)
	}
	return list, nil
}

func (p *PostgresStore) UpdateBackupSnapshot(ctx context.Context, snap *BackupSnapshotRecord) error {
	query := `
		UPDATE backup_snapshots
		SET status = $1, verified_at = $2, integrity_status = $3,
		    error_message = $4, completed_at = $5, sha256_hash = $6,
		    size_bytes = $7, item_count = $8
		WHERE id = $9
	`
	res, err := p.db.ExecContext(ctx, query,
		snap.Status, snap.VerifiedAt, snap.IntegrityStatus,
		snap.ErrorMessage, snap.CompletedAt, snap.SHA256Hash,
		snap.SizeBytes, snap.ItemCount, snap.ID,
	)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) DeleteBackupSnapshot(ctx context.Context, id uuid.UUID) error {
	res, err := p.db.ExecContext(ctx, `DELETE FROM backup_snapshots WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ============================================================================
// POSTGRES STORE: BACKUP SCHEDULES
// ============================================================================

func (p *PostgresStore) CreateBackupSchedule(ctx context.Context, sched *BackupScheduleRecord) error {
	if sched.ID == uuid.Nil {
		sched.ID = uuid.New()
	}
	now := time.Now().UTC()
	sched.CreatedAt = now
	sched.UpdatedAt = now

	query := `
		INSERT INTO backup_schedules (
			id, organization_id, server_id, destination_id, name,
			scope, target_name, frequency, cron_expr, retention,
			is_enabled, last_run_at, next_run_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15
		)
	`
	_, err := p.db.ExecContext(ctx, query,
		sched.ID, sched.OrganizationID, sched.ServerID, sched.DestinationID, sched.Name,
		sched.Scope, sched.TargetName, sched.Frequency, sched.CronExpr, sched.Retention,
		sched.IsEnabled, sched.LastRunAt, sched.NextRunAt, sched.CreatedAt, sched.UpdatedAt,
	)
	return err
}

func (p *PostgresStore) GetBackupScheduleByID(ctx context.Context, id uuid.UUID) (*BackupScheduleRecord, error) {
	query := `
		SELECT id, organization_id, server_id, destination_id, name,
		       scope, target_name, frequency, cron_expr, retention,
		       is_enabled, last_run_at, next_run_at, created_at, updated_at
		FROM backup_schedules
		WHERE id = $1
	`
	var s BackupScheduleRecord
	var serverID, destinationID uuid.NullUUID
	var cronExpr sql.NullString
	var lastRunAt, nextRunAt sql.NullTime

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.OrganizationID, &serverID, &destinationID, &s.Name,
		&s.Scope, &s.TargetName, &s.Frequency, &cronExpr, &s.Retention,
		&s.IsEnabled, &lastRunAt, &nextRunAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if serverID.Valid {
		s.ServerID = &serverID.UUID
	}
	if destinationID.Valid {
		s.DestinationID = &destinationID.UUID
	}
	if cronExpr.Valid {
		s.CronExpr = cronExpr.String
	}
	if lastRunAt.Valid {
		s.LastRunAt = &lastRunAt.Time
	}
	if nextRunAt.Valid {
		s.NextRunAt = &nextRunAt.Time
	}
	return &s, nil
}

func (p *PostgresStore) ListBackupSchedules(ctx context.Context, orgID uuid.UUID) ([]*BackupScheduleRecord, error) {
	query := `
		SELECT id, organization_id, server_id, destination_id, name,
		       scope, target_name, frequency, cron_expr, retention,
		       is_enabled, last_run_at, next_run_at, created_at, updated_at
		FROM backup_schedules
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*BackupScheduleRecord
	for rows.Next() {
		var s BackupScheduleRecord
		var serverID, destinationID uuid.NullUUID
		var cronExpr sql.NullString
		var lastRunAt, nextRunAt sql.NullTime

		if err := rows.Scan(
			&s.ID, &s.OrganizationID, &serverID, &destinationID, &s.Name,
			&s.Scope, &s.TargetName, &s.Frequency, &cronExpr, &s.Retention,
			&s.IsEnabled, &lastRunAt, &nextRunAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if serverID.Valid {
			s.ServerID = &serverID.UUID
		}
		if destinationID.Valid {
			s.DestinationID = &destinationID.UUID
		}
		if cronExpr.Valid {
			s.CronExpr = cronExpr.String
		}
		if lastRunAt.Valid {
			s.LastRunAt = &lastRunAt.Time
		}
		if nextRunAt.Valid {
			s.NextRunAt = &nextRunAt.Time
		}
		list = append(list, &s)
	}
	return list, nil
}

func (p *PostgresStore) UpdateBackupSchedule(ctx context.Context, sched *BackupScheduleRecord) error {
	sched.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE backup_schedules
		SET server_id = $1, destination_id = $2, name = $3, scope = $4,
		    target_name = $5, frequency = $6, cron_expr = $7, retention = $8,
		    is_enabled = $9, last_run_at = $10, next_run_at = $11, updated_at = $12
		WHERE id = $13
	`
	res, err := p.db.ExecContext(ctx, query,
		sched.ServerID, sched.DestinationID, sched.Name, sched.Scope,
		sched.TargetName, sched.Frequency, sched.CronExpr, sched.Retention,
		sched.IsEnabled, sched.LastRunAt, sched.NextRunAt, sched.UpdatedAt, sched.ID,
	)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) DeleteBackupSchedule(ctx context.Context, id uuid.UUID) error {
	res, err := p.db.ExecContext(ctx, `DELETE FROM backup_schedules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ============================================================================
// POSTGRES STORE: BACKUP RESTORE JOBS
// ============================================================================

func (p *PostgresStore) CreateBackupRestoreJob(ctx context.Context, job *BackupRestoreJobRecord) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO backup_restore_jobs (
			id, organization_id, server_id, snapshot_id, target_name,
			status, preflight_ok, disk_space_required, disk_space_available,
			current_step, error_message, started_at, completed_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13, $14
		)
	`
	_, err := p.db.ExecContext(ctx, query,
		job.ID, job.OrganizationID, job.ServerID, job.SnapshotID, job.TargetName,
		job.Status, job.PreflightOK, job.DiskSpaceRequired, job.DiskSpaceAvailable,
		job.CurrentStep, job.ErrorMessage, job.StartedAt, job.CompletedAt, job.CreatedAt,
	)
	return err
}

func (p *PostgresStore) GetBackupRestoreJobByID(ctx context.Context, id uuid.UUID) (*BackupRestoreJobRecord, error) {
	query := `
		SELECT id, organization_id, server_id, snapshot_id, target_name,
		       status, preflight_ok, disk_space_required, disk_space_available,
		       current_step, error_message, started_at, completed_at, created_at
		FROM backup_restore_jobs
		WHERE id = $1
	`
	var j BackupRestoreJobRecord
	var serverID uuid.NullUUID
	var currentStep, errMsg sql.NullString
	var startedAt, completedAt sql.NullTime

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&j.ID, &j.OrganizationID, &serverID, &j.SnapshotID, &j.TargetName,
		&j.Status, &j.PreflightOK, &j.DiskSpaceRequired, &j.DiskSpaceAvailable,
		&currentStep, &errMsg, &startedAt, &completedAt, &j.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if serverID.Valid {
		j.ServerID = &serverID.UUID
	}
	if currentStep.Valid {
		j.CurrentStep = currentStep.String
	}
	if errMsg.Valid {
		j.ErrorMessage = errMsg.String
	}
	if startedAt.Valid {
		j.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		j.CompletedAt = &completedAt.Time
	}
	return &j, nil
}

func (p *PostgresStore) ListBackupRestoreJobs(ctx context.Context, orgID uuid.UUID, limit int) ([]*BackupRestoreJobRecord, error) {
	query := `
		SELECT id, organization_id, server_id, snapshot_id, target_name,
		       status, preflight_ok, disk_space_required, disk_space_available,
		       current_step, error_message, started_at, completed_at, created_at
		FROM backup_restore_jobs
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`
	if limit > 0 {
		query += " LIMIT $2"
	}

	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = p.db.QueryContext(ctx, query, orgID, limit)
	} else {
		rows, err = p.db.QueryContext(ctx, query, orgID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*BackupRestoreJobRecord
	for rows.Next() {
		var j BackupRestoreJobRecord
		var serverID uuid.NullUUID
		var currentStep, errMsg sql.NullString
		var startedAt, completedAt sql.NullTime

		if err := rows.Scan(
			&j.ID, &j.OrganizationID, &serverID, &j.SnapshotID, &j.TargetName,
			&j.Status, &j.PreflightOK, &j.DiskSpaceRequired, &j.DiskSpaceAvailable,
			&currentStep, &errMsg, &startedAt, &completedAt, &j.CreatedAt,
		); err != nil {
			return nil, err
		}
		if serverID.Valid {
			j.ServerID = &serverID.UUID
		}
		if currentStep.Valid {
			j.CurrentStep = currentStep.String
		}
		if errMsg.Valid {
			j.ErrorMessage = errMsg.String
		}
		if startedAt.Valid {
			j.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			j.CompletedAt = &completedAt.Time
		}
		list = append(list, &j)
	}
	return list, nil
}

func (p *PostgresStore) UpdateBackupRestoreJob(ctx context.Context, job *BackupRestoreJobRecord) error {
	query := `
		UPDATE backup_restore_jobs
		SET status = $1, preflight_ok = $2, disk_space_required = $3,
		    disk_space_available = $4, current_step = $5, error_message = $6,
		    started_at = $7, completed_at = $8
		WHERE id = $9
	`
	res, err := p.db.ExecContext(ctx, query,
		job.Status, job.PreflightOK, job.DiskSpaceRequired,
		job.DiskSpaceAvailable, job.CurrentStep, job.ErrorMessage,
		job.StartedAt, job.CompletedAt, job.ID,
	)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
