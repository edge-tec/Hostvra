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
// MEMORY STORE: INCREMENTAL BACKUPS, REPOSITORIES & RESTORATION
// ============================================================================

// 1. Repositories
func (m *MemoryStore) CreateBackupRepository(ctx context.Context, repo *BackupRepositoryRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if repo.ID == uuid.Nil {
		repo.ID = uuid.New()
	}
	now := time.Now().UTC()
	repo.CreatedAt = now
	repo.UpdatedAt = now

	copyR := *repo
	m.backupRepositories[repo.ID] = &copyR
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetBackupRepositoryByID(ctx context.Context, id uuid.UUID) (*BackupRepositoryRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	r, ok := m.backupRepositories[id]
	if !ok {
		return nil, ErrNotFound
	}
	copyR := *r
	return &copyR, nil
}

func (m *MemoryStore) ListBackupRepositories(ctx context.Context, orgID uuid.UUID) ([]*BackupRepositoryRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*BackupRepositoryRecord
	for _, r := range m.backupRepositories {
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

func (m *MemoryStore) UpdateBackupRepository(ctx context.Context, repo *BackupRepositoryRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.backupRepositories[repo.ID]
	if !ok {
		return ErrNotFound
	}

	repo.CreatedAt = existing.CreatedAt
	repo.UpdatedAt = time.Now().UTC()

	copyR := *repo
	m.backupRepositories[repo.ID] = &copyR
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteBackupRepository(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	r, ok := m.backupRepositories[id]
	if !ok {
		return ErrNotFound
	}
	if r.DeletionProtection {
		return errors.New("cannot delete repository: deletion protection safeguard is enabled")
	}

	delete(m.backupRepositories, id)

	// Clean up related snapshots
	for sID, s := range m.backupIncrementalSnapshots {
		if s.RepositoryID == id {
			delete(m.backupIncrementalSnapshots, sID)
		}
	}
	// Clean up related retention policies
	for pID, p := range m.backupRetentionPolicies {
		if p.RepositoryID == id {
			delete(m.backupRetentionPolicies, pID)
		}
	}

	m.saveToDiskLocked()
	return nil
}

// 2. Incremental Snapshots
func (m *MemoryStore) CreateIncrementalSnapshot(ctx context.Context, s *BackupIncrementalSnapshotRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	now := time.Now().UTC()
	s.CreatedAt = now
	if s.StartedAt.IsZero() {
		s.StartedAt = now
	}

	copyS := *s
	m.backupIncrementalSnapshots[s.ID] = &copyS

	// Update repository statistics
	if repo, ok := m.backupRepositories[s.RepositoryID]; ok {
		repo.TotalSnapshots++
		repo.TotalSizeBytes += s.RawSizeBytes
		repo.DeduplicatedSizeBytes += s.StoredSizeBytes
		if repo.DeduplicatedSizeBytes > 0 && repo.TotalSizeBytes > 0 {
			repo.CompressionRatio = float64(repo.TotalSizeBytes) / float64(repo.DeduplicatedSizeBytes)
		}
		repo.UpdatedAt = now
	}

	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetIncrementalSnapshotByID(ctx context.Context, id uuid.UUID) (*BackupIncrementalSnapshotRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.backupIncrementalSnapshots[id]
	if !ok {
		return nil, ErrNotFound
	}
	copyS := *s
	return &copyS, nil
}

func (m *MemoryStore) ListIncrementalSnapshots(ctx context.Context, orgID uuid.UUID, repoID *uuid.UUID, limit int) ([]*BackupIncrementalSnapshotRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*BackupIncrementalSnapshotRecord
	for _, s := range m.backupIncrementalSnapshots {
		if s.OrganizationID == orgID {
			if repoID != nil && s.RepositoryID != *repoID {
				continue
			}
			copyS := *s
			list = append(list, &copyS)
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

func (m *MemoryStore) UpdateIncrementalSnapshot(ctx context.Context, s *BackupIncrementalSnapshotRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.backupIncrementalSnapshots[s.ID]
	if !ok {
		return ErrNotFound
	}

	s.CreatedAt = existing.CreatedAt
	copyS := *s
	m.backupIncrementalSnapshots[s.ID] = &copyS
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteIncrementalSnapshot(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.backupIncrementalSnapshots[id]
	if !ok {
		return ErrNotFound
	}

	// Decrement repo count
	if repo, rOk := m.backupRepositories[s.RepositoryID]; rOk && repo.TotalSnapshots > 0 {
		repo.TotalSnapshots--
	}

	delete(m.backupIncrementalSnapshots, id)
	m.saveToDiskLocked()
	return nil
}

// 3. Retention Policies
func (m *MemoryStore) CreateBackupRetentionPolicy(ctx context.Context, p *BackupRetentionPolicyRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	copyP := *p
	m.backupRetentionPolicies[p.ID] = &copyP
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetBackupRetentionPolicyByRepoID(ctx context.Context, repoID uuid.UUID) (*BackupRetentionPolicyRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.backupRetentionPolicies {
		if p.RepositoryID == repoID {
			copyP := *p
			return &copyP, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) UpdateBackupRetentionPolicy(ctx context.Context, p *BackupRetentionPolicyRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.backupRetentionPolicies[p.ID]
	if !ok {
		return ErrNotFound
	}

	p.CreatedAt = existing.CreatedAt
	p.UpdatedAt = time.Now().UTC()

	copyP := *p
	m.backupRetentionPolicies[p.ID] = &copyP
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) ListBackupRetentionPolicies(ctx context.Context, orgID uuid.UUID) ([]*BackupRetentionPolicyRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*BackupRetentionPolicyRecord
	for _, p := range m.backupRetentionPolicies {
		if p.OrganizationID == orgID {
			copyP := *p
			list = append(list, &copyP)
		}
	}
	return list, nil
}

// 4. File-Level & Full Account Restore Requests
func (m *MemoryStore) CreateFileRestoreRequest(ctx context.Context, req *BackupFileRestoreRequestRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	req.CreatedAt = time.Now().UTC()

	copyR := *req
	m.backupFileRestoreRequests[req.ID] = &copyR
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetFileRestoreRequestByID(ctx context.Context, id uuid.UUID) (*BackupFileRestoreRequestRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	req, ok := m.backupFileRestoreRequests[id]
	if !ok {
		return nil, ErrNotFound
	}
	copyR := *req
	return &copyR, nil
}

func (m *MemoryStore) ListFileRestoreRequests(ctx context.Context, orgID uuid.UUID, limit int) ([]*BackupFileRestoreRequestRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*BackupFileRestoreRequestRecord
	for _, req := range m.backupFileRestoreRequests {
		if req.OrganizationID == orgID {
			copyR := *req
			list = append(list, &copyR)
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

func (m *MemoryStore) UpdateFileRestoreRequest(ctx context.Context, req *BackupFileRestoreRequestRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.backupFileRestoreRequests[req.ID]
	if !ok {
		return ErrNotFound
	}

	req.CreatedAt = existing.CreatedAt
	copyR := *req
	m.backupFileRestoreRequests[req.ID] = &copyR
	m.saveToDiskLocked()
	return nil
}

// ============================================================================
// POSTGRES STORE: INCREMENTAL BACKUPS, REPOSITORIES & RESTORATION
// ============================================================================

// 1. Repositories
func (s *PostgresStore) CreateBackupRepository(ctx context.Context, repo *BackupRepositoryRecord) error {
	if repo.ID == uuid.Nil {
		repo.ID = uuid.New()
	}
	now := time.Now().UTC()
	repo.CreatedAt = now
	repo.UpdatedAt = now

	query := `
		INSERT INTO backup_repositories (
			id, organization_id, destination_id, name, provider, repo_uri, encryption_algorithm,
			password_encrypted, key_id, is_initialized, last_checked_at, integrity_status,
			total_snapshots, total_size_bytes, deduplicated_size_bytes, compression_ratio,
			deletion_protection, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`
	_, err := s.db.ExecContext(ctx, query,
		repo.ID, repo.OrganizationID, repo.DestinationID, repo.Name, repo.Provider, repo.RepoURI,
		repo.EncryptionAlgorithm, repo.PasswordEncrypted, repo.KeyID, repo.IsInitialized,
		repo.LastCheckedAt, repo.IntegrityStatus, repo.TotalSnapshots, repo.TotalSizeBytes,
		repo.DeduplicatedSizeBytes, repo.CompressionRatio, repo.DeletionProtection,
		repo.CreatedAt, repo.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) GetBackupRepositoryByID(ctx context.Context, id uuid.UUID) (*BackupRepositoryRecord, error) {
	query := `
		SELECT id, organization_id, destination_id, name, provider, repo_uri, encryption_algorithm,
		       password_encrypted, key_id, is_initialized, last_checked_at, integrity_status,
		       total_snapshots, total_size_bytes, deduplicated_size_bytes, compression_ratio,
		       deletion_protection, created_at, updated_at
		FROM backup_repositories
		WHERE id = $1
		LIMIT 1
	`
	var r BackupRepositoryRecord
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&r.ID, &r.OrganizationID, &r.DestinationID, &r.Name, &r.Provider, &r.RepoURI,
		&r.EncryptionAlgorithm, &r.PasswordEncrypted, &r.KeyID, &r.IsInitialized,
		&r.LastCheckedAt, &r.IntegrityStatus, &r.TotalSnapshots, &r.TotalSizeBytes,
		&r.DeduplicatedSizeBytes, &r.CompressionRatio, &r.DeletionProtection,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &r, nil
}

func (s *PostgresStore) ListBackupRepositories(ctx context.Context, orgID uuid.UUID) ([]*BackupRepositoryRecord, error) {
	query := `
		SELECT id, organization_id, destination_id, name, provider, repo_uri, encryption_algorithm,
		       password_encrypted, key_id, is_initialized, last_checked_at, integrity_status,
		       total_snapshots, total_size_bytes, deduplicated_size_bytes, compression_ratio,
		       deletion_protection, created_at, updated_at
		FROM backup_repositories
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*BackupRepositoryRecord
	for rows.Next() {
		var r BackupRepositoryRecord
		err := rows.Scan(
			&r.ID, &r.OrganizationID, &r.DestinationID, &r.Name, &r.Provider, &r.RepoURI,
			&r.EncryptionAlgorithm, &r.PasswordEncrypted, &r.KeyID, &r.IsInitialized,
			&r.LastCheckedAt, &r.IntegrityStatus, &r.TotalSnapshots, &r.TotalSizeBytes,
			&r.DeduplicatedSizeBytes, &r.CompressionRatio, &r.DeletionProtection,
			&r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, &r)
	}
	return list, nil
}

func (s *PostgresStore) UpdateBackupRepository(ctx context.Context, repo *BackupRepositoryRecord) error {
	repo.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE backup_repositories
		SET name = $2, provider = $3, repo_uri = $4, encryption_algorithm = $5, password_encrypted = $6,
		    key_id = $7, is_initialized = $8, last_checked_at = $9, integrity_status = $10,
		    total_snapshots = $11, total_size_bytes = $12, deduplicated_size_bytes = $13,
		    compression_ratio = $14, deletion_protection = $15, updated_at = $16
		WHERE id = $1
	`
	res, err := s.db.ExecContext(ctx, query,
		repo.ID, repo.Name, repo.Provider, repo.RepoURI, repo.EncryptionAlgorithm, repo.PasswordEncrypted,
		repo.KeyID, repo.IsInitialized, repo.LastCheckedAt, repo.IntegrityStatus, repo.TotalSnapshots,
		repo.TotalSizeBytes, repo.DeduplicatedSizeBytes, repo.CompressionRatio, repo.DeletionProtection,
		repo.UpdatedAt,
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

func (s *PostgresStore) DeleteBackupRepository(ctx context.Context, id uuid.UUID) error {
	// Enforce deletion protection
	repo, err := s.GetBackupRepositoryByID(ctx, id)
	if err != nil {
		return err
	}
	if repo.DeletionProtection {
		return errors.New("cannot delete repository: deletion protection safeguard is enabled")
	}

	query := `DELETE FROM backup_repositories WHERE id = $1`
	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// 2. Incremental Snapshots
func (s *PostgresStore) CreateIncrementalSnapshot(ctx context.Context, sn *BackupIncrementalSnapshotRecord) error {
	if sn.ID == uuid.Nil {
		sn.ID = uuid.New()
	}
	now := time.Now().UTC()
	sn.CreatedAt = now
	if sn.StartedAt.IsZero() {
		sn.StartedAt = now
	}

	query := `
		INSERT INTO backup_incremental_snapshots (
			id, repository_id, organization_id, snapshot_hash, parent_snapshot_id, scope,
			target_name, paths_included, paths_excluded, files_new, files_changed, files_unmodified,
			total_files, raw_size_bytes, stored_size_bytes, database_snapshot_consistent,
			status, integrity_status, error_message, started_at, completed_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
	`
	_, err := s.db.ExecContext(ctx, query,
		sn.ID, sn.RepositoryID, sn.OrganizationID, sn.SnapshotHash, sn.ParentSnapshotID, sn.Scope,
		sn.TargetName, sn.PathsIncluded, sn.PathsExcluded, sn.FilesNew, sn.FilesChanged, sn.FilesUnmodified,
		sn.TotalFiles, sn.RawSizeBytes, sn.StoredSizeBytes, sn.DatabaseSnapshotConsistent,
		sn.Status, sn.IntegrityStatus, sn.ErrorMessage, sn.StartedAt, sn.CompletedAt, sn.CreatedAt,
	)
	return err
}

func (s *PostgresStore) GetIncrementalSnapshotByID(ctx context.Context, id uuid.UUID) (*BackupIncrementalSnapshotRecord, error) {
	query := `
		SELECT id, repository_id, organization_id, snapshot_hash, parent_snapshot_id, scope,
		       target_name, paths_included::text, paths_excluded::text, files_new, files_changed, files_unmodified,
		       total_files, raw_size_bytes, stored_size_bytes, database_snapshot_consistent,
		       status, integrity_status, error_message, started_at, completed_at, created_at
		FROM backup_incremental_snapshots
		WHERE id = $1
		LIMIT 1
	`
	var sn BackupIncrementalSnapshotRecord
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&sn.ID, &sn.RepositoryID, &sn.OrganizationID, &sn.SnapshotHash, &sn.ParentSnapshotID, &sn.Scope,
		&sn.TargetName, &sn.PathsIncluded, &sn.PathsExcluded, &sn.FilesNew, &sn.FilesChanged, &sn.FilesUnmodified,
		&sn.TotalFiles, &sn.RawSizeBytes, &sn.StoredSizeBytes, &sn.DatabaseSnapshotConsistent,
		&sn.Status, &sn.IntegrityStatus, &sn.ErrorMessage, &sn.StartedAt, &sn.CompletedAt, &sn.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &sn, nil
}

func (s *PostgresStore) ListIncrementalSnapshots(ctx context.Context, orgID uuid.UUID, repoID *uuid.UUID, limit int) ([]*BackupIncrementalSnapshotRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT id, repository_id, organization_id, snapshot_hash, parent_snapshot_id, scope,
		       target_name, paths_included::text, paths_excluded::text, files_new, files_changed, files_unmodified,
		       total_files, raw_size_bytes, stored_size_bytes, database_snapshot_consistent,
		       status, integrity_status, error_message, started_at, completed_at, created_at
		FROM backup_incremental_snapshots
		WHERE organization_id = $1 AND ($2::uuid IS NULL OR repository_id = $2)
		ORDER BY created_at DESC
		LIMIT $3
	`
	rows, err := s.db.QueryContext(ctx, query, orgID, repoID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*BackupIncrementalSnapshotRecord
	for rows.Next() {
		var sn BackupIncrementalSnapshotRecord
		err := rows.Scan(
			&sn.ID, &sn.RepositoryID, &sn.OrganizationID, &sn.SnapshotHash, &sn.ParentSnapshotID, &sn.Scope,
			&sn.TargetName, &sn.PathsIncluded, &sn.PathsExcluded, &sn.FilesNew, &sn.FilesChanged, &sn.FilesUnmodified,
			&sn.TotalFiles, &sn.RawSizeBytes, &sn.StoredSizeBytes, &sn.DatabaseSnapshotConsistent,
			&sn.Status, &sn.IntegrityStatus, &sn.ErrorMessage, &sn.StartedAt, &sn.CompletedAt, &sn.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, &sn)
	}
	return list, nil
}

func (s *PostgresStore) UpdateIncrementalSnapshot(ctx context.Context, sn *BackupIncrementalSnapshotRecord) error {
	query := `
		UPDATE backup_incremental_snapshots
		SET status = $2, integrity_status = $3, files_new = $4, files_changed = $5,
		    files_unmodified = $6, total_files = $7, raw_size_bytes = $8, stored_size_bytes = $9,
		    database_snapshot_consistent = $10, error_message = $11, completed_at = $12
		WHERE id = $1
	`
	res, err := s.db.ExecContext(ctx, query,
		sn.ID, sn.Status, sn.IntegrityStatus, sn.FilesNew, sn.FilesChanged,
		sn.FilesUnmodified, sn.TotalFiles, sn.RawSizeBytes, sn.StoredSizeBytes,
		sn.DatabaseSnapshotConsistent, sn.ErrorMessage, sn.CompletedAt,
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

func (s *PostgresStore) DeleteIncrementalSnapshot(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM backup_incremental_snapshots WHERE id = $1`
	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// 3. Retention Policies
func (s *PostgresStore) CreateBackupRetentionPolicy(ctx context.Context, p *BackupRetentionPolicyRecord) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	query := `
		INSERT INTO backup_retention_policies (
			id, repository_id, organization_id, name, keep_last, keep_hourly, keep_daily,
			keep_weekly, keep_monthly, keep_yearly, prune_schedule, is_active, last_pruned_at,
			pruned_snapshots_count, bytes_reclaimed, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	_, err := s.db.ExecContext(ctx, query,
		p.ID, p.RepositoryID, p.OrganizationID, p.Name, p.KeepLast, p.KeepHourly, p.KeepDaily,
		p.KeepWeekly, p.KeepMonthly, p.KeepYearly, p.PruneSchedule, p.IsActive, p.LastPrunedAt,
		p.PrunedSnapshotsCount, p.BytesReclaimed, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) GetBackupRetentionPolicyByRepoID(ctx context.Context, repoID uuid.UUID) (*BackupRetentionPolicyRecord, error) {
	query := `
		SELECT id, repository_id, organization_id, name, keep_last, keep_hourly, keep_daily,
		       keep_weekly, keep_monthly, keep_yearly, prune_schedule, is_active, last_pruned_at,
		       pruned_snapshots_count, bytes_reclaimed, created_at, updated_at
		FROM backup_retention_policies
		WHERE repository_id = $1
		LIMIT 1
	`
	var p BackupRetentionPolicyRecord
	err := s.db.QueryRowContext(ctx, query, repoID).Scan(
		&p.ID, &p.RepositoryID, &p.OrganizationID, &p.Name, &p.KeepLast, &p.KeepHourly, &p.KeepDaily,
		&p.KeepWeekly, &p.KeepMonthly, &p.KeepYearly, &p.PruneSchedule, &p.IsActive, &p.LastPrunedAt,
		&p.PrunedSnapshotsCount, &p.BytesReclaimed, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (s *PostgresStore) UpdateBackupRetentionPolicy(ctx context.Context, p *BackupRetentionPolicyRecord) error {
	p.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE backup_retention_policies
		SET name = $2, keep_last = $3, keep_hourly = $4, keep_daily = $5, keep_weekly = $6,
		    keep_monthly = $7, keep_yearly = $8, prune_schedule = $9, is_active = $10,
		    last_pruned_at = $11, pruned_snapshots_count = $12, bytes_reclaimed = $13, updated_at = $14
		WHERE id = $1
	`
	res, err := s.db.ExecContext(ctx, query,
		p.ID, p.Name, p.KeepLast, p.KeepHourly, p.KeepDaily, p.KeepWeekly,
		p.KeepMonthly, p.KeepYearly, p.PruneSchedule, p.IsActive, p.LastPrunedAt,
		p.PrunedSnapshotsCount, p.BytesReclaimed, p.UpdatedAt,
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

func (s *PostgresStore) ListBackupRetentionPolicies(ctx context.Context, orgID uuid.UUID) ([]*BackupRetentionPolicyRecord, error) {
	query := `
		SELECT id, repository_id, organization_id, name, keep_last, keep_hourly, keep_daily,
		       keep_weekly, keep_monthly, keep_yearly, prune_schedule, is_active, last_pruned_at,
		       pruned_snapshots_count, bytes_reclaimed, created_at, updated_at
		FROM backup_retention_policies
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*BackupRetentionPolicyRecord
	for rows.Next() {
		var p BackupRetentionPolicyRecord
		err := rows.Scan(
			&p.ID, &p.RepositoryID, &p.OrganizationID, &p.Name, &p.KeepLast, &p.KeepHourly, &p.KeepDaily,
			&p.KeepWeekly, &p.KeepMonthly, &p.KeepYearly, &p.PruneSchedule, &p.IsActive, &p.LastPrunedAt,
			&p.PrunedSnapshotsCount, &p.BytesReclaimed, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, &p)
	}
	return list, nil
}

// 4. File-Level & Full Account Restore Requests
func (s *PostgresStore) CreateFileRestoreRequest(ctx context.Context, req *BackupFileRestoreRequestRecord) error {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	req.CreatedAt = time.Now().UTC()

	query := `
		INSERT INTO backup_file_restore_requests (
			id, organization_id, snapshot_id, selected_paths, target_directory, restore_mode,
			status, files_restored, bytes_restored, error_message, created_at, completed_at
		) VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := s.db.ExecContext(ctx, query,
		req.ID, req.OrganizationID, req.SnapshotID, req.SelectedPaths, req.TargetDirectory,
		req.RestoreMode, req.Status, req.FilesRestored, req.BytesRestored, req.ErrorMessage,
		req.CreatedAt, req.CompletedAt,
	)
	return err
}

func (s *PostgresStore) GetFileRestoreRequestByID(ctx context.Context, id uuid.UUID) (*BackupFileRestoreRequestRecord, error) {
	query := `
		SELECT id, organization_id, snapshot_id, selected_paths::text, target_directory, restore_mode,
		       status, files_restored, bytes_restored, error_message, created_at, completed_at
		FROM backup_file_restore_requests
		WHERE id = $1
		LIMIT 1
	`
	var req BackupFileRestoreRequestRecord
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&req.ID, &req.OrganizationID, &req.SnapshotID, &req.SelectedPaths, &req.TargetDirectory,
		&req.RestoreMode, &req.Status, &req.FilesRestored, &req.BytesRestored, &req.ErrorMessage,
		&req.CreatedAt, &req.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &req, nil
}

func (s *PostgresStore) ListFileRestoreRequests(ctx context.Context, orgID uuid.UUID, limit int) ([]*BackupFileRestoreRequestRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, organization_id, snapshot_id, selected_paths::text, target_directory, restore_mode,
		       status, files_restored, bytes_restored, error_message, created_at, completed_at
		FROM backup_file_restore_requests
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := s.db.QueryContext(ctx, query, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*BackupFileRestoreRequestRecord
	for rows.Next() {
		var req BackupFileRestoreRequestRecord
		err := rows.Scan(
			&req.ID, &req.OrganizationID, &req.SnapshotID, &req.SelectedPaths, &req.TargetDirectory,
			&req.RestoreMode, &req.Status, &req.FilesRestored, &req.BytesRestored, &req.ErrorMessage,
			&req.CreatedAt, &req.CompletedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, &req)
	}
	return list, nil
}

func (s *PostgresStore) UpdateFileRestoreRequest(ctx context.Context, req *BackupFileRestoreRequestRecord) error {
	query := `
		UPDATE backup_file_restore_requests
		SET status = $2, files_restored = $3, bytes_restored = $4, error_message = $5, completed_at = $6
		WHERE id = $1
	`
	res, err := s.db.ExecContext(ctx, query,
		req.ID, req.Status, req.FilesRestored, req.BytesRestored, req.ErrorMessage, req.CompletedAt,
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
