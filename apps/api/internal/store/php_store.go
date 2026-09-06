package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MEMORY STORE IMPLEMENTATION FOR PHP
// ============================================================================

func (m *MemoryStore) UpsertPHPVersion(ctx context.Context, v *PHPInstalledVersion) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", v.ServerID.String(), v.Version)
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	now := time.Now().UTC()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	v.UpdatedAt = now
	m.phpVersions[key] = v
	return nil
}

func (m *MemoryStore) GetPHPVersion(ctx context.Context, serverID uuid.UUID, version string) (*PHPInstalledVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", serverID.String(), version)
	v, ok := m.phpVersions[key]
	if !ok {
		return nil, ErrNotFound
	}
	return v, nil
}

func (m *MemoryStore) ListPHPVersionsByServer(ctx context.Context, serverID uuid.UUID) ([]*PHPInstalledVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*PHPInstalledVersion
	for _, v := range m.phpVersions {
		if v.ServerID == serverID {
			result = append(result, v)
		}
	}
	return result, nil
}

func (m *MemoryStore) DeletePHPVersion(ctx context.Context, serverID uuid.UUID, version string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", serverID.String(), version)
	delete(m.phpVersions, key)
	return nil
}

func (m *MemoryStore) SetDefaultPHPCli(ctx context.Context, serverID uuid.UUID, version string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, v := range m.phpVersions {
		if v.ServerID == serverID {
			v.IsDefaultCLI = (v.Version == version)
		}
	}
	return nil
}

func (m *MemoryStore) SetDefaultPHPFpm(ctx context.Context, serverID uuid.UUID, version string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, v := range m.phpVersions {
		if v.ServerID == serverID {
			v.IsDefaultFPM = (v.Version == version)
		}
	}
	return nil
}

func (m *MemoryStore) UpsertPHPExtension(ctx context.Context, ext *PHPExtension) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", ext.ServerID.String(), ext.PHPVersion, ext.Name)
	if ext.ID == uuid.Nil {
		ext.ID = uuid.New()
	}
	now := time.Now().UTC()
	if ext.CreatedAt.IsZero() {
		ext.CreatedAt = now
	}
	ext.UpdatedAt = now
	m.phpExtensions[key] = ext
	return nil
}

func (m *MemoryStore) ListPHPExtensions(ctx context.Context, serverID uuid.UUID, version string) ([]*PHPExtension, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*PHPExtension
	for _, ext := range m.phpExtensions {
		if ext.ServerID == serverID && (version == "" || ext.PHPVersion == version) {
			result = append(result, ext)
		}
	}
	return result, nil
}

func (m *MemoryStore) GetPHPExtension(ctx context.Context, serverID uuid.UUID, version string, name string) (*PHPExtension, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s:%s", serverID.String(), version, name)
	ext, ok := m.phpExtensions[key]
	if !ok {
		return nil, ErrNotFound
	}
	return ext, nil
}

func (m *MemoryStore) DeletePHPExtension(ctx context.Context, serverID uuid.UUID, version string, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", serverID.String(), version, name)
	delete(m.phpExtensions, key)
	return nil
}

func (m *MemoryStore) CreatePHPFPMPool(ctx context.Context, pool *PHPFPMPool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pool.ID == uuid.Nil {
		pool.ID = uuid.New()
	}
	now := time.Now().UTC()
	pool.CreatedAt = now
	pool.UpdatedAt = now
	m.phpPools[pool.ID] = pool
	return nil
}

func (m *MemoryStore) GetPHPFPMPool(ctx context.Context, id uuid.UUID) (*PHPFPMPool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.phpPools[id]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (m *MemoryStore) GetPHPFPMPoolByName(ctx context.Context, serverID uuid.UUID, name string) (*PHPFPMPool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.phpPools {
		if p.ServerID == serverID && p.Name == name {
			return p, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) GetPHPFPMPoolByWebsite(ctx context.Context, websiteID uuid.UUID) (*PHPFPMPool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.phpPools {
		if p.WebsiteID != nil && *p.WebsiteID == websiteID {
			return p, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) ListPHPFPMPoolsByServer(ctx context.Context, serverID uuid.UUID, version string) ([]*PHPFPMPool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*PHPFPMPool
	for _, p := range m.phpPools {
		if p.ServerID == serverID && (version == "" || p.PHPVersion == version) {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *MemoryStore) UpdatePHPFPMPool(ctx context.Context, pool *PHPFPMPool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.phpPools[pool.ID]
	if !ok {
		return ErrNotFound
	}
	pool.CreatedAt = existing.CreatedAt
	pool.UpdatedAt = time.Now().UTC()
	m.phpPools[pool.ID] = pool
	return nil
}

func (m *MemoryStore) DeletePHPFPMPool(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.phpPools, id)
	return nil
}

func (m *MemoryStore) UpsertPHPIniOverride(ctx context.Context, override *PHPIniOverride) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	siteIDStr := "none"
	if override.WebsiteID != nil {
		siteIDStr = override.WebsiteID.String()
	}
	key := fmt.Sprintf("%s:%s:%s:%s:%s", override.ServerID.String(), override.PHPVersion, override.Scope, siteIDStr, override.Directive)
	if override.ID == uuid.Nil {
		override.ID = uuid.New()
	}
	now := time.Now().UTC()
	if override.CreatedAt.IsZero() {
		override.CreatedAt = now
	}
	override.UpdatedAt = now
	m.phpIniOverrides[key] = override
	return nil
}

func (m *MemoryStore) ListPHPIniOverrides(ctx context.Context, serverID uuid.UUID, version string, scope string, websiteID *uuid.UUID) ([]*PHPIniOverride, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*PHPIniOverride
	for _, o := range m.phpIniOverrides {
		if o.ServerID != serverID || o.PHPVersion != version {
			continue
		}
		if scope != "" && o.Scope != scope {
			continue
		}
		if websiteID != nil {
			if o.WebsiteID == nil || *o.WebsiteID != *websiteID {
				continue
			}
		}
		result = append(result, o)
	}
	return result, nil
}

func (m *MemoryStore) DeletePHPIniOverride(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for k, o := range m.phpIniOverrides {
		if o.ID == id {
			delete(m.phpIniOverrides, k)
			break
		}
	}
	return nil
}

func (m *MemoryStore) CreatePHPConfigBackup(ctx context.Context, backup *PHPConfigBackup) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if backup.ID == uuid.Nil {
		backup.ID = uuid.New()
	}
	if backup.CreatedAt.IsZero() {
		backup.CreatedAt = time.Now().UTC()
	}
	m.phpConfigBackups = append(m.phpConfigBackups, backup)
	return nil
}

func (m *MemoryStore) ListPHPConfigBackups(ctx context.Context, serverID uuid.UUID, version string, limit int) ([]*PHPConfigBackup, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*PHPConfigBackup
	for i := len(m.phpConfigBackups) - 1; i >= 0; i-- {
		b := m.phpConfigBackups[i]
		if b.ServerID == serverID && (version == "" || b.PHPVersion == version) {
			result = append(result, b)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// ============================================================================
// POSTGRES STORE IMPLEMENTATION FOR PHP
// ============================================================================

func (p *PostgresStore) UpsertPHPVersion(ctx context.Context, v *PHPInstalledVersion) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	query := `
		INSERT INTO php_installed_versions (
			id, server_id, version, cli_binary_path, fpm_binary_path, fpm_service_name,
			fpm_socket_path, ini_path, cli_ini_path, fpm_pool_dir, is_default_cli, is_default_fpm, status, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
		ON CONFLICT (server_id, version) DO UPDATE SET
			cli_binary_path = EXCLUDED.cli_binary_path,
			fpm_binary_path = EXCLUDED.fpm_binary_path,
			fpm_service_name = EXCLUDED.fpm_service_name,
			fpm_socket_path = EXCLUDED.fpm_socket_path,
			ini_path = EXCLUDED.ini_path,
			cli_ini_path = EXCLUDED.cli_ini_path,
			fpm_pool_dir = EXCLUDED.fpm_pool_dir,
			is_default_cli = EXCLUDED.is_default_cli,
			is_default_fpm = EXCLUDED.is_default_fpm,
			status = EXCLUDED.status,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`
	return p.db.QueryRowContext(ctx, query,
		v.ID, v.ServerID, v.Version, v.CLIBinaryPath, v.FPMBinaryPath, v.FPMServiceName,
		v.FPMSocketPath, v.IniPath, v.CLIIniPath, v.FPMPoolDir, v.IsDefaultCLI, v.IsDefaultFPM, v.Status,
	).Scan(&v.CreatedAt, &v.UpdatedAt)
}

func (p *PostgresStore) GetPHPVersion(ctx context.Context, serverID uuid.UUID, version string) (*PHPInstalledVersion, error) {
	query := `
		SELECT id, server_id, version, cli_binary_path, COALESCE(fpm_binary_path, ''), fpm_service_name,
		       fpm_socket_path, ini_path, COALESCE(cli_ini_path, ''), fpm_pool_dir, is_default_cli, is_default_fpm, status, created_at, updated_at
		FROM php_installed_versions
		WHERE server_id = $1 AND version = $2
	`
	v := &PHPInstalledVersion{}
	err := p.db.QueryRowContext(ctx, query, serverID, version).Scan(
		&v.ID, &v.ServerID, &v.Version, &v.CLIBinaryPath, &v.FPMBinaryPath, &v.FPMServiceName,
		&v.FPMSocketPath, &v.IniPath, &v.CLIIniPath, &v.FPMPoolDir, &v.IsDefaultCLI, &v.IsDefaultFPM, &v.Status, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (p *PostgresStore) ListPHPVersionsByServer(ctx context.Context, serverID uuid.UUID) ([]*PHPInstalledVersion, error) {
	query := `
		SELECT id, server_id, version, cli_binary_path, COALESCE(fpm_binary_path, ''), fpm_service_name,
		       fpm_socket_path, ini_path, COALESCE(cli_ini_path, ''), fpm_pool_dir, is_default_cli, is_default_fpm, status, created_at, updated_at
		FROM php_installed_versions
		WHERE server_id = $1
		ORDER BY version DESC
	`
	rows, err := p.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*PHPInstalledVersion
	for rows.Next() {
		v := &PHPInstalledVersion{}
		err := rows.Scan(
			&v.ID, &v.ServerID, &v.Version, &v.CLIBinaryPath, &v.FPMBinaryPath, &v.FPMServiceName,
			&v.FPMSocketPath, &v.IniPath, &v.CLIIniPath, &v.FPMPoolDir, &v.IsDefaultCLI, &v.IsDefaultFPM, &v.Status, &v.CreatedAt, &v.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, nil
}

func (p *PostgresStore) DeletePHPVersion(ctx context.Context, serverID uuid.UUID, version string) error {
	query := `DELETE FROM php_installed_versions WHERE server_id = $1 AND version = $2`
	_, err := p.db.ExecContext(ctx, query, serverID, version)
	return err
}

func (p *PostgresStore) SetDefaultPHPCli(ctx context.Context, serverID uuid.UUID, version string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE php_installed_versions SET is_default_cli = FALSE WHERE server_id = $1`, serverID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE php_installed_versions SET is_default_cli = TRUE WHERE server_id = $1 AND version = $2`, serverID, version); err != nil {
		return err
	}
	return tx.Commit()
}

func (p *PostgresStore) SetDefaultPHPFpm(ctx context.Context, serverID uuid.UUID, version string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE php_installed_versions SET is_default_fpm = FALSE WHERE server_id = $1`, serverID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE php_installed_versions SET is_default_fpm = TRUE WHERE server_id = $1 AND version = $2`, serverID, version); err != nil {
		return err
	}
	return tx.Commit()
}

func (p *PostgresStore) UpsertPHPExtension(ctx context.Context, ext *PHPExtension) error {
	if ext.ID == uuid.Nil {
		ext.ID = uuid.New()
	}
	query := `
		INSERT INTO php_extensions (
			id, server_id, php_version, name, package_name, version, is_installed, is_enabled, is_critical, description, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (server_id, php_version, name) DO UPDATE SET
			package_name = EXCLUDED.package_name,
			version = EXCLUDED.version,
			is_installed = EXCLUDED.is_installed,
			is_enabled = EXCLUDED.is_enabled,
			is_critical = EXCLUDED.is_critical,
			description = EXCLUDED.description,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`
	return p.db.QueryRowContext(ctx, query,
		ext.ID, ext.ServerID, ext.PHPVersion, ext.Name, ext.PackageName, ext.Version,
		ext.IsInstalled, ext.IsEnabled, ext.IsCritical, ext.Description,
	).Scan(&ext.CreatedAt, &ext.UpdatedAt)
}

func (p *PostgresStore) ListPHPExtensions(ctx context.Context, serverID uuid.UUID, version string) ([]*PHPExtension, error) {
	query := `
		SELECT id, server_id, php_version, name, package_name, COALESCE(version, ''), is_installed, is_enabled, is_critical, COALESCE(description, ''), created_at, updated_at
		FROM php_extensions
		WHERE server_id = $1 AND ($2 = '' OR php_version = $2)
		ORDER BY name ASC
	`
	rows, err := p.db.QueryContext(ctx, query, serverID, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var extensions []*PHPExtension
	for rows.Next() {
		ext := &PHPExtension{}
		err := rows.Scan(
			&ext.ID, &ext.ServerID, &ext.PHPVersion, &ext.Name, &ext.PackageName, &ext.Version,
			&ext.IsInstalled, &ext.IsEnabled, &ext.IsCritical, &ext.Description, &ext.CreatedAt, &ext.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		extensions = append(extensions, ext)
	}
	return extensions, nil
}

func (p *PostgresStore) GetPHPExtension(ctx context.Context, serverID uuid.UUID, version string, name string) (*PHPExtension, error) {
	query := `
		SELECT id, server_id, php_version, name, package_name, COALESCE(version, ''), is_installed, is_enabled, is_critical, COALESCE(description, ''), created_at, updated_at
		FROM php_extensions
		WHERE server_id = $1 AND php_version = $2 AND name = $3
	`
	ext := &PHPExtension{}
	err := p.db.QueryRowContext(ctx, query, serverID, version, name).Scan(
		&ext.ID, &ext.ServerID, &ext.PHPVersion, &ext.Name, &ext.PackageName, &ext.Version,
		&ext.IsInstalled, &ext.IsEnabled, &ext.IsCritical, &ext.Description, &ext.CreatedAt, &ext.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return ext, nil
}

func (p *PostgresStore) DeletePHPExtension(ctx context.Context, serverID uuid.UUID, version string, name string) error {
	query := `DELETE FROM php_extensions WHERE server_id = $1 AND php_version = $2 AND name = $3`
	_, err := p.db.ExecContext(ctx, query, serverID, version, name)
	return err
}

func (p *PostgresStore) CreatePHPFPMPool(ctx context.Context, pool *PHPFPMPool) error {
	if pool.ID == uuid.Nil {
		pool.ID = uuid.New()
	}
	query := `
		INSERT INTO php_fpm_pools (
			id, server_id, website_id, name, php_version, listen_socket, pool_user, pool_group,
			listen_owner, listen_group, pm_type, pm_max_children, pm_start_servers, pm_min_spare_servers,
			pm_max_spare_servers, pm_max_requests, request_terminate_timeout, request_slowlog_timeout,
			slowlog_path, errorlog_path, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, NOW(), NOW()
		) RETURNING created_at, updated_at
	`
	return p.db.QueryRowContext(ctx, query,
		pool.ID, pool.ServerID, pool.WebsiteID, pool.Name, pool.PHPVersion, pool.ListenSocket,
		pool.PoolUser, pool.PoolGroup, pool.ListenOwner, pool.ListenGroup, pool.PMType,
		pool.PMMaxChildren, pool.PMStartServers, pool.PMMinSpareServers, pool.PMMaxSpareServers,
		pool.PMMaxRequests, pool.RequestTerminateTimeout, pool.RequestSlowlogTimeout,
		pool.SlowlogPath, pool.ErrorlogPath, pool.Status,
	).Scan(&pool.CreatedAt, &pool.UpdatedAt)
}

func (p *PostgresStore) GetPHPFPMPool(ctx context.Context, id uuid.UUID) (*PHPFPMPool, error) {
	query := `
		SELECT id, server_id, website_id, name, php_version, listen_socket, pool_user, pool_group,
		       listen_owner, listen_group, pm_type, pm_max_children, pm_start_servers, pm_min_spare_servers,
		       pm_max_spare_servers, pm_max_requests, request_terminate_timeout, request_slowlog_timeout,
		       COALESCE(slowlog_path, ''), COALESCE(errorlog_path, ''), status, created_at, updated_at
		FROM php_fpm_pools
		WHERE id = $1
	`
	pool := &PHPFPMPool{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&pool.ID, &pool.ServerID, &pool.WebsiteID, &pool.Name, &pool.PHPVersion, &pool.ListenSocket,
		&pool.PoolUser, &pool.PoolGroup, &pool.ListenOwner, &pool.ListenGroup, &pool.PMType,
		&pool.PMMaxChildren, &pool.PMStartServers, &pool.PMMinSpareServers, &pool.PMMaxSpareServers,
		&pool.PMMaxRequests, &pool.RequestTerminateTimeout, &pool.RequestSlowlogTimeout,
		&pool.SlowlogPath, &pool.ErrorlogPath, &pool.Status, &pool.CreatedAt, &pool.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func (p *PostgresStore) GetPHPFPMPoolByName(ctx context.Context, serverID uuid.UUID, name string) (*PHPFPMPool, error) {
	query := `
		SELECT id, server_id, website_id, name, php_version, listen_socket, pool_user, pool_group,
		       listen_owner, listen_group, pm_type, pm_max_children, pm_start_servers, pm_min_spare_servers,
		       pm_max_spare_servers, pm_max_requests, request_terminate_timeout, request_slowlog_timeout,
		       COALESCE(slowlog_path, ''), COALESCE(errorlog_path, ''), status, created_at, updated_at
		FROM php_fpm_pools
		WHERE server_id = $1 AND name = $2
	`
	pool := &PHPFPMPool{}
	err := p.db.QueryRowContext(ctx, query, serverID, name).Scan(
		&pool.ID, &pool.ServerID, &pool.WebsiteID, &pool.Name, &pool.PHPVersion, &pool.ListenSocket,
		&pool.PoolUser, &pool.PoolGroup, &pool.ListenOwner, &pool.ListenGroup, &pool.PMType,
		&pool.PMMaxChildren, &pool.PMStartServers, &pool.PMMinSpareServers, &pool.PMMaxSpareServers,
		&pool.PMMaxRequests, &pool.RequestTerminateTimeout, &pool.RequestSlowlogTimeout,
		&pool.SlowlogPath, &pool.ErrorlogPath, &pool.Status, &pool.CreatedAt, &pool.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func (p *PostgresStore) GetPHPFPMPoolByWebsite(ctx context.Context, websiteID uuid.UUID) (*PHPFPMPool, error) {
	query := `
		SELECT id, server_id, website_id, name, php_version, listen_socket, pool_user, pool_group,
		       listen_owner, listen_group, pm_type, pm_max_children, pm_start_servers, pm_min_spare_servers,
		       pm_max_spare_servers, pm_max_requests, request_terminate_timeout, request_slowlog_timeout,
		       COALESCE(slowlog_path, ''), COALESCE(errorlog_path, ''), status, created_at, updated_at
		FROM php_fpm_pools
		WHERE website_id = $1
	`
	pool := &PHPFPMPool{}
	err := p.db.QueryRowContext(ctx, query, websiteID).Scan(
		&pool.ID, &pool.ServerID, &pool.WebsiteID, &pool.Name, &pool.PHPVersion, &pool.ListenSocket,
		&pool.PoolUser, &pool.PoolGroup, &pool.ListenOwner, &pool.ListenGroup, &pool.PMType,
		&pool.PMMaxChildren, &pool.PMStartServers, &pool.PMMinSpareServers, &pool.PMMaxSpareServers,
		&pool.PMMaxRequests, &pool.RequestTerminateTimeout, &pool.RequestSlowlogTimeout,
		&pool.SlowlogPath, &pool.ErrorlogPath, &pool.Status, &pool.CreatedAt, &pool.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func (p *PostgresStore) ListPHPFPMPoolsByServer(ctx context.Context, serverID uuid.UUID, version string) ([]*PHPFPMPool, error) {
	query := `
		SELECT id, server_id, website_id, name, php_version, listen_socket, pool_user, pool_group,
		       listen_owner, listen_group, pm_type, pm_max_children, pm_start_servers, pm_min_spare_servers,
		       pm_max_spare_servers, pm_max_requests, request_terminate_timeout, request_slowlog_timeout,
		       COALESCE(slowlog_path, ''), COALESCE(errorlog_path, ''), status, created_at, updated_at
		FROM php_fpm_pools
		WHERE server_id = $1 AND ($2 = '' OR php_version = $2)
		ORDER BY name ASC
	`
	rows, err := p.db.QueryContext(ctx, query, serverID, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []*PHPFPMPool
	for rows.Next() {
		pool := &PHPFPMPool{}
		err := rows.Scan(
			&pool.ID, &pool.ServerID, &pool.WebsiteID, &pool.Name, &pool.PHPVersion, &pool.ListenSocket,
			&pool.PoolUser, &pool.PoolGroup, &pool.ListenOwner, &pool.ListenGroup, &pool.PMType,
			&pool.PMMaxChildren, &pool.PMStartServers, &pool.PMMinSpareServers, &pool.PMMaxSpareServers,
			&pool.PMMaxRequests, &pool.RequestTerminateTimeout, &pool.RequestSlowlogTimeout,
			&pool.SlowlogPath, &pool.ErrorlogPath, &pool.Status, &pool.CreatedAt, &pool.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		pools = append(pools, pool)
	}
	return pools, nil
}

func (p *PostgresStore) UpdatePHPFPMPool(ctx context.Context, pool *PHPFPMPool) error {
	query := `
		UPDATE php_fpm_pools SET
			website_id = $2,
			php_version = $3,
			listen_socket = $4,
			pool_user = $5,
			pool_group = $6,
			listen_owner = $7,
			listen_group = $8,
			pm_type = $9,
			pm_max_children = $10,
			pm_start_servers = $11,
			pm_min_spare_servers = $12,
			pm_max_spare_servers = $13,
			pm_max_requests = $14,
			request_terminate_timeout = $15,
			request_slowlog_timeout = $16,
			slowlog_path = $17,
			errorlog_path = $18,
			status = $19,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	return p.db.QueryRowContext(ctx, query,
		pool.ID, pool.WebsiteID, pool.PHPVersion, pool.ListenSocket, pool.PoolUser, pool.PoolGroup,
		pool.ListenOwner, pool.ListenGroup, pool.PMType, pool.PMMaxChildren, pool.PMStartServers,
		pool.PMMinSpareServers, pool.PMMaxSpareServers, pool.PMMaxRequests, pool.RequestTerminateTimeout,
		pool.RequestSlowlogTimeout, pool.SlowlogPath, pool.ErrorlogPath, pool.Status,
	).Scan(&pool.UpdatedAt)
}

func (p *PostgresStore) DeletePHPFPMPool(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM php_fpm_pools WHERE id = $1`
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgresStore) UpsertPHPIniOverride(ctx context.Context, override *PHPIniOverride) error {
	if override.ID == uuid.Nil {
		override.ID = uuid.New()
	}
	query := `
		INSERT INTO php_ini_overrides (
			id, server_id, scope, website_id, php_version, directive, value, directive_type, category, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (server_id, scope, website_id, php_version, directive) DO UPDATE SET
			value = EXCLUDED.value,
			directive_type = EXCLUDED.directive_type,
			category = EXCLUDED.category,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`
	return p.db.QueryRowContext(ctx, query,
		override.ID, override.ServerID, override.Scope, override.WebsiteID, override.PHPVersion,
		override.Directive, override.Value, override.DirectiveType, override.Category,
	).Scan(&override.CreatedAt, &override.UpdatedAt)
}

func (p *PostgresStore) ListPHPIniOverrides(ctx context.Context, serverID uuid.UUID, version string, scope string, websiteID *uuid.UUID) ([]*PHPIniOverride, error) {
	query := `
		SELECT id, server_id, scope, website_id, php_version, directive, value, directive_type, category, created_at, updated_at
		FROM php_ini_overrides
		WHERE server_id = $1 AND php_version = $2
		  AND ($3 = '' OR scope = $3)
		  AND ($4::uuid IS NULL OR website_id = $4)
		ORDER BY category, directive ASC
	`
	rows, err := p.db.QueryContext(ctx, query, serverID, version, scope, websiteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var overrides []*PHPIniOverride
	for rows.Next() {
		o := &PHPIniOverride{}
		err := rows.Scan(
			&o.ID, &o.ServerID, &o.Scope, &o.WebsiteID, &o.PHPVersion,
			&o.Directive, &o.Value, &o.DirectiveType, &o.Category, &o.CreatedAt, &o.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, o)
	}
	return overrides, nil
}

func (p *PostgresStore) DeletePHPIniOverride(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM php_ini_overrides WHERE id = $1`
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgresStore) CreatePHPConfigBackup(ctx context.Context, backup *PHPConfigBackup) error {
	if backup.ID == uuid.Nil {
		backup.ID = uuid.New()
	}
	query := `
		INSERT INTO php_config_backups (
			id, server_id, php_version, backup_type, file_path, content_backup, reason, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING created_at
	`
	return p.db.QueryRowContext(ctx, query,
		backup.ID, backup.ServerID, backup.PHPVersion, backup.BackupType, backup.FilePath, backup.ContentBackup, backup.Reason,
	).Scan(&backup.CreatedAt)
}

func (p *PostgresStore) ListPHPConfigBackups(ctx context.Context, serverID uuid.UUID, version string, limit int) ([]*PHPConfigBackup, error) {
	query := `
		SELECT id, server_id, php_version, backup_type, file_path, content_backup, COALESCE(reason, ''), created_at
		FROM php_config_backups
		WHERE server_id = $1 AND ($2 = '' OR php_version = $2)
		ORDER BY created_at DESC
		LIMIT $3
	`
	rows, err := p.db.QueryContext(ctx, query, serverID, version, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backups []*PHPConfigBackup
	for rows.Next() {
		b := &PHPConfigBackup{}
		err := rows.Scan(
			&b.ID, &b.ServerID, &b.PHPVersion, &b.BackupType, &b.FilePath, &b.ContentBackup, &b.Reason, &b.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		backups = append(backups, b)
	}
	return backups, nil
}
