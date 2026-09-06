package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ============================================================================
// MEMORY STORE IMPLEMENTATION FOR WEB SERVERS
// ============================================================================

func (m *MemoryStore) UpsertWebServerInstance(ctx context.Context, instance *WebServerInstance) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", instance.ServerID.String(), instance.ServerType)
	if instance.ID == uuid.Nil {
		instance.ID = uuid.New()
	}
	now := time.Now().UTC()
	if instance.CreatedAt.IsZero() {
		instance.CreatedAt = now
	}
	instance.UpdatedAt = now
	m.webServerInstances[key] = instance
	return nil
}

func (m *MemoryStore) GetWebServerInstance(ctx context.Context, serverID uuid.UUID, serverType string) (*WebServerInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", serverID.String(), serverType)
	inst, ok := m.webServerInstances[key]
	if !ok {
		return nil, ErrNotFound
	}
	return inst, nil
}

func (m *MemoryStore) ListWebServerInstances(ctx context.Context, serverID uuid.UUID) ([]*WebServerInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*WebServerInstance
	for _, inst := range m.webServerInstances {
		if inst.ServerID == serverID {
			result = append(result, inst)
		}
	}
	return result, nil
}

func (m *MemoryStore) SetActiveDefaultWebServer(ctx context.Context, serverID uuid.UUID, serverType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, inst := range m.webServerInstances {
		if inst.ServerID == serverID {
			inst.IsActiveDefault = (inst.ServerType == serverType)
		}
	}
	return nil
}

func (m *MemoryStore) UpsertWebServerVHost(ctx context.Context, vhost *WebServerVHost) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", vhost.ServerID.String(), vhost.WebServerType, vhost.Domain)
	if vhost.ID == uuid.Nil {
		vhost.ID = uuid.New()
	}
	now := time.Now().UTC()
	if vhost.CreatedAt.IsZero() {
		vhost.CreatedAt = now
	}
	vhost.UpdatedAt = now
	m.webServerVHosts[key] = vhost
	return nil
}

func (m *MemoryStore) GetWebServerVHost(ctx context.Context, serverID uuid.UUID, serverType string, domain string) (*WebServerVHost, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s:%s", serverID.String(), serverType, domain)
	vhost, ok := m.webServerVHosts[key]
	if !ok {
		return nil, ErrNotFound
	}
	return vhost, nil
}

func (m *MemoryStore) ListWebServerVHosts(ctx context.Context, serverID uuid.UUID, serverType string) ([]*WebServerVHost, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*WebServerVHost
	for _, vhost := range m.webServerVHosts {
		if vhost.ServerID == serverID && (serverType == "" || vhost.WebServerType == serverType) {
			result = append(result, vhost)
		}
	}
	return result, nil
}

func (m *MemoryStore) DeleteWebServerVHost(ctx context.Context, serverID uuid.UUID, serverType string, domain string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", serverID.String(), serverType, domain)
	delete(m.webServerVHosts, key)
	return nil
}

func (m *MemoryStore) CreateWebServerConfigBackup(ctx context.Context, backup *WebServerConfigBackup) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if backup.ID == uuid.Nil {
		backup.ID = uuid.New()
	}
	if backup.CreatedAt.IsZero() {
		backup.CreatedAt = time.Now().UTC()
	}
	m.webServerBackups = append(m.webServerBackups, backup)
	return nil
}

func (m *MemoryStore) ListWebServerConfigBackups(ctx context.Context, serverID uuid.UUID, serverType string, limit int) ([]*WebServerConfigBackup, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*WebServerConfigBackup
	for i := len(m.webServerBackups) - 1; i >= 0; i-- {
		b := m.webServerBackups[i]
		if b.ServerID == serverID && (serverType == "" || b.WebServerType == serverType) {
			result = append(result, b)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// ============================================================================
// POSTGRESQL STORE IMPLEMENTATION FOR WEB SERVERS
// ============================================================================

func (p *PostgresStore) UpsertWebServerInstance(ctx context.Context, instance *WebServerInstance) error {
	if instance.ID == uuid.Nil {
		instance.ID = uuid.New()
	}
	modulesJSON, _ := json.Marshal(instance.InstalledModules)
	query := `
		INSERT INTO web_server_instances (
			id, server_id, server_type, version, binary_path, config_path, service_name,
			is_installed, is_active_default, http_port, https_port, status, license_status, installed_modules, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
		ON CONFLICT (server_id, server_type) DO UPDATE SET
			version = EXCLUDED.version,
			binary_path = EXCLUDED.binary_path,
			config_path = EXCLUDED.config_path,
			service_name = EXCLUDED.service_name,
			is_installed = EXCLUDED.is_installed,
			is_active_default = EXCLUDED.is_active_default,
			http_port = EXCLUDED.http_port,
			https_port = EXCLUDED.https_port,
			status = EXCLUDED.status,
			license_status = EXCLUDED.license_status,
			installed_modules = EXCLUDED.installed_modules,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`
	return p.db.QueryRowContext(ctx, query,
		instance.ID, instance.ServerID, instance.ServerType, instance.Version, instance.BinaryPath,
		instance.ConfigPath, instance.ServiceName, instance.IsInstalled, instance.IsActiveDefault,
		instance.HTTPPort, instance.HTTPSPort, instance.Status, instance.LicenseStatus, modulesJSON,
	).Scan(&instance.CreatedAt, &instance.UpdatedAt)
}

func (p *PostgresStore) GetWebServerInstance(ctx context.Context, serverID uuid.UUID, serverType string) (*WebServerInstance, error) {
	query := `
		SELECT id, server_id, server_type, COALESCE(version, ''), COALESCE(binary_path, ''), COALESCE(config_path, ''),
		       service_name, is_installed, is_active_default, http_port, https_port, status, license_status, installed_modules, created_at, updated_at
		FROM web_server_instances
		WHERE server_id = $1 AND server_type = $2
	`
	inst := &WebServerInstance{}
	var modulesBytes []byte
	err := p.db.QueryRowContext(ctx, query, serverID, serverType).Scan(
		&inst.ID, &inst.ServerID, &inst.ServerType, &inst.Version, &inst.BinaryPath, &inst.ConfigPath,
		&inst.ServiceName, &inst.IsInstalled, &inst.IsActiveDefault, &inst.HTTPPort, &inst.HTTPSPort,
		&inst.Status, &inst.LicenseStatus, &modulesBytes, &inst.CreatedAt, &inst.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(modulesBytes) > 0 {
		_ = json.Unmarshal(modulesBytes, &inst.InstalledModules)
	}
	return inst, nil
}

func (p *PostgresStore) ListWebServerInstances(ctx context.Context, serverID uuid.UUID) ([]*WebServerInstance, error) {
	query := `
		SELECT id, server_id, server_type, COALESCE(version, ''), COALESCE(binary_path, ''), COALESCE(config_path, ''),
		       service_name, is_installed, is_active_default, http_port, https_port, status, license_status, installed_modules, created_at, updated_at
		FROM web_server_instances
		WHERE server_id = $1
		ORDER BY is_active_default DESC, server_type ASC
	`
	rows, err := p.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*WebServerInstance
	for rows.Next() {
		inst := &WebServerInstance{}
		var modulesBytes []byte
		err := rows.Scan(
			&inst.ID, &inst.ServerID, &inst.ServerType, &inst.Version, &inst.BinaryPath, &inst.ConfigPath,
			&inst.ServiceName, &inst.IsInstalled, &inst.IsActiveDefault, &inst.HTTPPort, &inst.HTTPSPort,
			&inst.Status, &inst.LicenseStatus, &modulesBytes, &inst.CreatedAt, &inst.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if len(modulesBytes) > 0 {
			_ = json.Unmarshal(modulesBytes, &inst.InstalledModules)
		}
		result = append(result, inst)
	}
	return result, nil
}

func (p *PostgresStore) SetActiveDefaultWebServer(ctx context.Context, serverID uuid.UUID, serverType string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE web_server_instances SET is_active_default = FALSE WHERE server_id = $1`, serverID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE web_server_instances SET is_active_default = TRUE WHERE server_id = $1 AND server_type = $2`, serverID, serverType); err != nil {
		return err
	}
	return tx.Commit()
}

func (p *PostgresStore) UpsertWebServerVHost(ctx context.Context, vhost *WebServerVHost) error {
	if vhost.ID == uuid.Nil {
		vhost.ID = uuid.New()
	}
	aliasesJSON, _ := json.Marshal(vhost.Aliases)
	query := `
		INSERT INTO web_server_vhosts (
			id, server_id, website_id, web_server_type, domain, aliases, document_root,
			app_type, php_version, php_handler_type, fpm_socket_path, proxy_target_url,
			websocket_enabled, ssl_enabled, cert_path, key_path, config_file_path, is_active, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, NOW()
		) ON CONFLICT (server_id, web_server_type, domain) DO UPDATE SET
			aliases = EXCLUDED.aliases,
			document_root = EXCLUDED.document_root,
			app_type = EXCLUDED.app_type,
			php_version = EXCLUDED.php_version,
			php_handler_type = EXCLUDED.php_handler_type,
			fpm_socket_path = EXCLUDED.fpm_socket_path,
			proxy_target_url = EXCLUDED.proxy_target_url,
			websocket_enabled = EXCLUDED.websocket_enabled,
			ssl_enabled = EXCLUDED.ssl_enabled,
			cert_path = EXCLUDED.cert_path,
			key_path = EXCLUDED.key_path,
			config_file_path = EXCLUDED.config_file_path,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`
	return p.db.QueryRowContext(ctx, query,
		vhost.ID, vhost.ServerID, vhost.WebsiteID, vhost.WebServerType, vhost.Domain, aliasesJSON,
		vhost.DocumentRoot, vhost.AppType, vhost.PHPVersion, vhost.PHPHandlerType, vhost.FPMSocketPath,
		vhost.ProxyTargetURL, vhost.WebSocketEnabled, vhost.SSLEnabled, vhost.CertPath, vhost.KeyPath,
		vhost.ConfigFilePath, vhost.IsActive,
	).Scan(&vhost.CreatedAt, &vhost.UpdatedAt)
}

func (p *PostgresStore) GetWebServerVHost(ctx context.Context, serverID uuid.UUID, serverType string, domain string) (*WebServerVHost, error) {
	query := `
		SELECT id, server_id, website_id, web_server_type, domain, aliases, document_root,
		       app_type, php_version, php_handler_type, COALESCE(fpm_socket_path, ''), COALESCE(proxy_target_url, ''),
		       websocket_enabled, ssl_enabled, COALESCE(cert_path, ''), COALESCE(key_path, ''), config_file_path, is_active, created_at, updated_at
		FROM web_server_vhosts
		WHERE server_id = $1 AND web_server_type = $2 AND domain = $3
	`
	vhost := &WebServerVHost{}
	var aliasesBytes []byte
	err := p.db.QueryRowContext(ctx, query, serverID, serverType, domain).Scan(
		&vhost.ID, &vhost.ServerID, &vhost.WebsiteID, &vhost.WebServerType, &vhost.Domain, &aliasesBytes,
		&vhost.DocumentRoot, &vhost.AppType, &vhost.PHPVersion, &vhost.PHPHandlerType, &vhost.FPMSocketPath,
		&vhost.ProxyTargetURL, &vhost.WebSocketEnabled, &vhost.SSLEnabled, &vhost.CertPath, &vhost.KeyPath,
		&vhost.ConfigFilePath, &vhost.IsActive, &vhost.CreatedAt, &vhost.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(aliasesBytes) > 0 {
		_ = json.Unmarshal(aliasesBytes, &vhost.Aliases)
	}
	return vhost, nil
}

func (p *PostgresStore) ListWebServerVHosts(ctx context.Context, serverID uuid.UUID, serverType string) ([]*WebServerVHost, error) {
	query := `
		SELECT id, server_id, website_id, web_server_type, domain, aliases, document_root,
		       app_type, php_version, php_handler_type, COALESCE(fpm_socket_path, ''), COALESCE(proxy_target_url, ''),
		       websocket_enabled, ssl_enabled, COALESCE(cert_path, ''), COALESCE(key_path, ''), config_file_path, is_active, created_at, updated_at
		FROM web_server_vhosts
		WHERE server_id = $1 AND ($2 = '' OR web_server_type = $2)
		ORDER BY domain ASC
	`
	rows, err := p.db.QueryContext(ctx, query, serverID, serverType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*WebServerVHost
	for rows.Next() {
		vhost := &WebServerVHost{}
		var aliasesBytes []byte
		err := rows.Scan(
			&vhost.ID, &vhost.ServerID, &vhost.WebsiteID, &vhost.WebServerType, &vhost.Domain, &aliasesBytes,
			&vhost.DocumentRoot, &vhost.AppType, &vhost.PHPVersion, &vhost.PHPHandlerType, &vhost.FPMSocketPath,
			&vhost.ProxyTargetURL, &vhost.WebSocketEnabled, &vhost.SSLEnabled, &vhost.CertPath, &vhost.KeyPath,
			&vhost.ConfigFilePath, &vhost.IsActive, &vhost.CreatedAt, &vhost.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if len(aliasesBytes) > 0 {
			_ = json.Unmarshal(aliasesBytes, &vhost.Aliases)
		}
		result = append(result, vhost)
	}
	return result, nil
}

func (p *PostgresStore) DeleteWebServerVHost(ctx context.Context, serverID uuid.UUID, serverType string, domain string) error {
	query := `DELETE FROM web_server_vhosts WHERE server_id = $1 AND web_server_type = $2 AND domain = $3`
	_, err := p.db.ExecContext(ctx, query, serverID, serverType, domain)
	return err
}

func (p *PostgresStore) CreateWebServerConfigBackup(ctx context.Context, backup *WebServerConfigBackup) error {
	if backup.ID == uuid.Nil {
		backup.ID = uuid.New()
	}
	query := `
		INSERT INTO web_server_config_backups (
			id, server_id, web_server_type, file_path, content_backup, reason, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING created_at
	`
	return p.db.QueryRowContext(ctx, query,
		backup.ID, backup.ServerID, backup.WebServerType, backup.FilePath, backup.ContentBackup, backup.Reason,
	).Scan(&backup.CreatedAt)
}

func (p *PostgresStore) ListWebServerConfigBackups(ctx context.Context, serverID uuid.UUID, serverType string, limit int) ([]*WebServerConfigBackup, error) {
	query := `
		SELECT id, server_id, web_server_type, file_path, content_backup, COALESCE(reason, ''), created_at
		FROM web_server_config_backups
		WHERE server_id = $1 AND ($2 = '' OR web_server_type = $2)
		ORDER BY created_at DESC
		LIMIT $3
	`
	rows, err := p.db.QueryContext(ctx, query, serverID, serverType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*WebServerConfigBackup
	for rows.Next() {
		b := &WebServerConfigBackup{}
		err := rows.Scan(
			&b.ID, &b.ServerID, &b.WebServerType, &b.FilePath, &b.ContentBackup, &b.Reason, &b.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, nil
}

var _ = pq.Array
