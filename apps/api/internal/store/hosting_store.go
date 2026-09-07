package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ============================================================================
// MEMORY STORE HOSTING IMPLEMENTATION
// ============================================================================

func (m *MemoryStore) CreateWebsite(ctx context.Context, site *Website) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, s := range m.websites {
		if s.ServerID == site.ServerID && s.PrimaryDomain == site.PrimaryDomain && s.DeletedAt == nil {
			return ErrAlreadyExists
		}
	}

	if site.ID == uuid.Nil {
		site.ID = uuid.New()
	}
	now := time.Now().UTC()
	site.CreatedAt = now
	site.UpdatedAt = now
	site.Status = "active"

	m.websites[site.ID] = site
	return nil
}

func (m *MemoryStore) GetWebsiteByID(ctx context.Context, id uuid.UUID) (*Website, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	site, exists := m.websites[id]
	if !exists || site.DeletedAt != nil {
		return nil, ErrNotFound
	}
	return site, nil
}

func (m *MemoryStore) ListWebsitesByOrg(ctx context.Context, orgID uuid.UUID) ([]*Website, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sites := make([]*Website, 0)
	for _, s := range m.websites {
		if (orgID == uuid.Nil || s.OrganizationID == orgID) && s.DeletedAt == nil {
			sites = append(sites, s)
		}
	}
	return sites, nil
}

func (m *MemoryStore) UpdateWebsiteStatus(ctx context.Context, id uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	site, exists := m.websites[id]
	if !exists || site.DeletedAt != nil {
		return ErrNotFound
	}
	site.Status = status
	site.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *MemoryStore) UpdateWebsite(ctx context.Context, site *Website) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.websites[site.ID]
	if !exists || existing.DeletedAt != nil {
		return ErrNotFound
	}
	site.CreatedAt = existing.CreatedAt
	site.UpdatedAt = time.Now().UTC()
	m.websites[site.ID] = site
	return nil
}

func (m *MemoryStore) DeleteWebsite(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	site, exists := m.websites[id]
	if !exists || site.DeletedAt != nil {
		return ErrNotFound
	}
	now := time.Now().UTC()
	site.DeletedAt = &now
	return nil
}

func (m *MemoryStore) UpdateWebsiteSSL(ctx context.Context, id uuid.UUID, sslEnabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	site, exists := m.websites[id]
	if !exists || site.DeletedAt != nil {
		return ErrNotFound
	}
	site.SSLEnabled = sslEnabled
	site.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *MemoryStore) CreateDatabase(ctx context.Context, db *Database) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, d := range m.databases {
		if d.ServerID == db.ServerID && d.DBType == db.DBType && d.Name == db.Name && d.DeletedAt == nil {
			return ErrAlreadyExists
		}
	}

	if db.ID == uuid.Nil {
		db.ID = uuid.New()
	}
	db.CreatedAt = time.Now().UTC()
	m.databases[db.ID] = db
	return nil
}

func (m *MemoryStore) GetDatabaseByID(ctx context.Context, id uuid.UUID) (*Database, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	db, exists := m.databases[id]
	if !exists || db.DeletedAt != nil {
		return nil, ErrNotFound
	}
	return db, nil
}

func (m *MemoryStore) ListDatabasesByServer(ctx context.Context, serverID uuid.UUID) ([]*Database, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dbs := make([]*Database, 0)
	for _, d := range m.databases {
		if (serverID == uuid.Nil || d.ServerID == serverID) && d.DeletedAt == nil {
			dbs = append(dbs, d)
		}
	}
	return dbs, nil
}

func (m *MemoryStore) UpdateDatabase(ctx context.Context, db *Database) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.databases[db.ID]
	if !exists || existing.DeletedAt != nil {
		return ErrNotFound
	}
	db.CreatedAt = existing.CreatedAt
	m.databases[db.ID] = db
	return nil
}

func (m *MemoryStore) DeleteDatabase(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	db, exists := m.databases[id]
	if !exists || db.DeletedAt != nil {
		return ErrNotFound
	}
	now := time.Now().UTC()
	db.DeletedAt = &now
	db.InRecycleBin = true
	return nil
}

func (m *MemoryStore) RestoreDatabase(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	db, exists := m.databases[id]
	if !exists {
		return ErrNotFound
	}
	db.DeletedAt = nil
	db.InRecycleBin = false
	return nil
}

func (m *MemoryStore) CreateDatabaseUser(ctx context.Context, user *DatabaseUser) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, u := range m.databaseUsers {
		if u.ServerID == user.ServerID && u.DBType == user.DBType && u.Username == user.Username && u.DeletedAt == nil {
			return ErrAlreadyExists
		}
	}

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	user.CreatedAt = time.Now().UTC()
	m.databaseUsers[user.ID] = user
	return nil
}

func (m *MemoryStore) ListDatabaseUsersByServer(ctx context.Context, serverID uuid.UUID) ([]*DatabaseUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]*DatabaseUser, 0)
	for _, u := range m.databaseUsers {
		if u.ServerID == serverID && u.DeletedAt == nil {
			users = append(users, u)
		}
	}
	return users, nil
}

func (m *MemoryStore) CreateOrUpdateSSL(ctx context.Context, cert *SSLCertificate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cert.ID == uuid.Nil {
		cert.ID = uuid.New()
	}
	now := time.Now().UTC()
	if cert.CreatedAt.IsZero() {
		cert.CreatedAt = now
	}
	cert.UpdatedAt = now

	m.sslCerts[cert.ID] = cert
	return nil
}

func (m *MemoryStore) GetSSLByWebsiteID(ctx context.Context, websiteID uuid.UUID) (*SSLCertificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, cert := range m.sslCerts {
		if cert.WebsiteID == websiteID {
			return cert, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) GetSSLByID(ctx context.Context, id uuid.UUID) (*SSLCertificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cert, exists := m.sslCerts[id]
	if !exists {
		return nil, ErrNotFound
	}
	return cert, nil
}

func (m *MemoryStore) ListSSLCertificates(ctx context.Context, orgID uuid.UUID) ([]*SSLCertificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	siteIDs := make(map[uuid.UUID]bool)
	for _, site := range m.websites {
		if orgID == uuid.Nil || site.OrganizationID == orgID {
			siteIDs[site.ID] = true
		}
	}

	var results []*SSLCertificate
	for _, cert := range m.sslCerts {
		if cert.WebsiteID == uuid.Nil || siteIDs[cert.WebsiteID] || orgID == uuid.Nil {
			results = append(results, cert)
		}
	}
	return results, nil
}

func (m *MemoryStore) DeleteSSL(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sslCerts[id]; !exists {
		return ErrNotFound
	}
	delete(m.sslCerts, id)
	return nil
}


// ============================================================================
// POSTGRESQL STORE HOSTING IMPLEMENTATION
// ============================================================================

func (p *PostgresStore) CreateWebsite(ctx context.Context, site *Website) error {
	query := `
		INSERT INTO websites (id, server_id, organization_id, primary_domain, document_root, system_user, php_version, app_type, proxy_port, status, ssl_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at
	`
	if site.ID == uuid.Nil {
		site.ID = uuid.New()
	}
	return p.db.QueryRowContext(ctx, query,
		site.ID, site.ServerID, site.OrganizationID, site.PrimaryDomain,
		site.DocumentRoot, site.SystemUser, site.PHPVersion, site.AppType,
		site.ProxyPort, site.Status, site.SSLEnabled,
	).Scan(&site.CreatedAt, &site.UpdatedAt)
}

func (p *PostgresStore) GetWebsiteByID(ctx context.Context, id uuid.UUID) (*Website, error) {
	query := `
		SELECT id, server_id, organization_id, primary_domain, document_root, system_user, php_version, app_type, proxy_port, status, ssl_enabled, created_at, updated_at
		FROM websites
		WHERE id = $1 AND deleted_at IS NULL
	`
	site := &Website{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&site.ID, &site.ServerID, &site.OrganizationID, &site.PrimaryDomain,
		&site.DocumentRoot, &site.SystemUser, &site.PHPVersion, &site.AppType,
		&site.ProxyPort, &site.Status, &site.SSLEnabled, &site.CreatedAt, &site.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return site, err
}

func (p *PostgresStore) ListWebsitesByOrg(ctx context.Context, orgID uuid.UUID) ([]*Website, error) {
	query := `
		SELECT id, server_id, organization_id, primary_domain, document_root, system_user, php_version, app_type, proxy_port, status, ssl_enabled, created_at, updated_at
		FROM websites
		WHERE organization_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sites := make([]*Website, 0)
	for rows.Next() {
		s := &Website{}
		err := rows.Scan(
			&s.ID, &s.ServerID, &s.OrganizationID, &s.PrimaryDomain,
			&s.DocumentRoot, &s.SystemUser, &s.PHPVersion, &s.AppType,
			&s.ProxyPort, &s.Status, &s.SSLEnabled, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		sites = append(sites, s)
	}
	return sites, nil
}

func (p *PostgresStore) UpdateWebsiteStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE websites SET status = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := p.db.ExecContext(ctx, query, id, status)
	return err
}

func (p *PostgresStore) UpdateWebsite(ctx context.Context, site *Website) error {
	query := `
		UPDATE websites SET
			php_version = $2,
			php_fpm_pool_id = $3,
			status = $4,
			ssl_enabled = $5,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := p.db.ExecContext(ctx, query, site.ID, site.PHPVersion, site.PHPFPMPoolID, site.Status, site.SSLEnabled)
	return err
}

func (p *PostgresStore) DeleteWebsite(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE websites SET deleted_at = NOW() WHERE id = $1`
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgresStore) UpdateWebsiteSSL(ctx context.Context, id uuid.UUID, sslEnabled bool) error {
	query := `UPDATE websites SET ssl_enabled = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := p.db.ExecContext(ctx, query, id, sslEnabled)
	return err
}

func (p *PostgresStore) CreateDatabase(ctx context.Context, db *Database) error {
	query := `
		INSERT INTO databases (id, server_id, db_type, name, character_set, collation)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`
	if db.ID == uuid.Nil {
		db.ID = uuid.New()
	}
	return p.db.QueryRowContext(ctx, query,
		db.ID, db.ServerID, db.DBType, db.Name, db.CharacterSet, db.Collation,
	).Scan(&db.CreatedAt)
}

func (p *PostgresStore) GetDatabaseByID(ctx context.Context, id uuid.UUID) (*Database, error) {
	query := `
		SELECT id, server_id, db_type, name, character_set, collation, size_bytes, created_at
		FROM databases
		WHERE id = $1 AND deleted_at IS NULL
	`
	d := &Database{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID, &d.ServerID, &d.DBType, &d.Name, &d.CharacterSet, &d.Collation, &d.SizeBytes, &d.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return d, err
}

func (p *PostgresStore) ListDatabasesByServer(ctx context.Context, serverID uuid.UUID) ([]*Database, error) {
	query := `
		SELECT id, server_id, db_type, name, character_set, collation, size_bytes, created_at
		FROM databases
		WHERE ($1 = '00000000-0000-0000-0000-000000000000'::uuid OR server_id = $1) AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dbs := make([]*Database, 0)
	for rows.Next() {
		d := &Database{}
		err := rows.Scan(&d.ID, &d.ServerID, &d.DBType, &d.Name, &d.CharacterSet, &d.Collation, &d.SizeBytes, &d.CreatedAt)
		if err != nil {
			return nil, err
		}
		dbs = append(dbs, d)
	}
	return dbs, nil
}

func (p *PostgresStore) UpdateDatabase(ctx context.Context, db *Database) error {
	query := `
		UPDATE databases
		SET name = $2, character_set = $3, collation = $4, size_bytes = $5
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := p.db.ExecContext(ctx, query, db.ID, db.Name, db.CharacterSet, db.Collation, db.SizeBytes)
	return err
}

func (p *PostgresStore) DeleteDatabase(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE databases SET deleted_at = NOW() WHERE id = $1`
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgresStore) RestoreDatabase(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE databases SET deleted_at = NULL WHERE id = $1`
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgresStore) CreateDatabaseUser(ctx context.Context, user *DatabaseUser) error {
	query := `
		INSERT INTO database_users (id, server_id, db_type, username, host_allow)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	return p.db.QueryRowContext(ctx, query,
		user.ID, user.ServerID, user.DBType, user.Username, user.HostAllow,
	).Scan(&user.CreatedAt)
}

func (p *PostgresStore) ListDatabaseUsersByServer(ctx context.Context, serverID uuid.UUID) ([]*DatabaseUser, error) {
	query := `
		SELECT id, server_id, db_type, username, host_allow, created_at
		FROM database_users
		WHERE server_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*DatabaseUser, 0)
	for rows.Next() {
		u := &DatabaseUser{}
		err := rows.Scan(&u.ID, &u.ServerID, &u.DBType, &u.Username, &u.HostAllow, &u.CreatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (p *PostgresStore) CreateOrUpdateSSL(ctx context.Context, cert *SSLCertificate) error {
	query := `
		INSERT INTO ssl_certificates (id, website_id, domain_list, issuer, cert_path, key_path, issued_at, expires_at, auto_renew, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			domain_list = EXCLUDED.domain_list,
			expires_at = EXCLUDED.expires_at,
			status = EXCLUDED.status,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`
	if cert.ID == uuid.Nil {
		cert.ID = uuid.New()
	}
	return p.db.QueryRowContext(ctx, query,
		cert.ID, cert.WebsiteID, pq.Array(cert.DomainList), cert.Issuer,
		cert.CertPath, cert.KeyPath, cert.IssuedAt, cert.ExpiresAt, cert.AutoRenew, cert.Status,
	).Scan(&cert.CreatedAt, &cert.UpdatedAt)
}

func (p *PostgresStore) GetSSLByWebsiteID(ctx context.Context, websiteID uuid.UUID) (*SSLCertificate, error) {
	query := `
		SELECT id, website_id, domain_list, issuer, cert_path, key_path, issued_at, expires_at, auto_renew, status, created_at, updated_at
		FROM ssl_certificates
		WHERE website_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	cert := &SSLCertificate{}
	err := p.db.QueryRowContext(ctx, query, websiteID).Scan(
		&cert.ID, &cert.WebsiteID, pq.Array(&cert.DomainList), &cert.Issuer,
		&cert.CertPath, &cert.KeyPath, &cert.IssuedAt, &cert.ExpiresAt, &cert.AutoRenew,
		&cert.Status, &cert.CreatedAt, &cert.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return cert, err
}

func (p *PostgresStore) GetSSLByID(ctx context.Context, id uuid.UUID) (*SSLCertificate, error) {
	query := `
		SELECT id, website_id, domain_list, issuer, cert_path, key_path, issued_at, expires_at, auto_renew, status, created_at, updated_at
		FROM ssl_certificates
		WHERE id = $1
	`
	cert := &SSLCertificate{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&cert.ID, &cert.WebsiteID, pq.Array(&cert.DomainList), &cert.Issuer,
		&cert.CertPath, &cert.KeyPath, &cert.IssuedAt, &cert.ExpiresAt, &cert.AutoRenew,
		&cert.Status, &cert.CreatedAt, &cert.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return cert, err
}

func (p *PostgresStore) ListSSLCertificates(ctx context.Context, orgID uuid.UUID) ([]*SSLCertificate, error) {
	query := `
		SELECT c.id, c.website_id, c.domain_list, c.issuer, c.cert_path, c.key_path, c.issued_at, c.expires_at, c.auto_renew, c.status, c.created_at, c.updated_at
		FROM ssl_certificates c
		LEFT JOIN websites w ON c.website_id = w.id
		WHERE ($1 = '00000000-0000-0000-0000-000000000000'::uuid OR w.organization_id = $1)
		ORDER BY c.created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []*SSLCertificate
	for rows.Next() {
		cert := &SSLCertificate{}
		if err := rows.Scan(
			&cert.ID, &cert.WebsiteID, pq.Array(&cert.DomainList), &cert.Issuer,
			&cert.CertPath, &cert.KeyPath, &cert.IssuedAt, &cert.ExpiresAt, &cert.AutoRenew,
			&cert.Status, &cert.CreatedAt, &cert.UpdatedAt,
		); err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

func (p *PostgresStore) DeleteSSL(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ssl_certificates WHERE id = $1`
	res, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

