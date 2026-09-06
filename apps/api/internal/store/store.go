package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var (
	ErrNotFound       = errors.New("record not found")
	ErrAlreadyExists  = errors.New("record already exists")
	ErrTokenExpired   = errors.New("enrollment token has expired")
	ErrTokenUsed      = errors.New("enrollment token has already been used")
)

type Store interface {
	// Organizations
	CreateOrganization(ctx context.Context, org *Organization) error
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error)

	// Users & Members
	CreateUser(ctx context.Context, user *User, orgID uuid.UUID, role string) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateUserLastLogin(ctx context.Context, id uuid.UUID, ip string) error

	// Servers & Enrollment
	CreateServer(ctx context.Context, server *Server) error
	GetServerByID(ctx context.Context, id uuid.UUID) (*Server, error)
	ListServersByOrg(ctx context.Context, orgID uuid.UUID) ([]*Server, error)
	UpdateServerHeartbeat(ctx context.Context, id uuid.UUID, uptime int64) error

	// Enrollment Tokens
	CreateEnrollmentToken(ctx context.Context, token *ServerEnrollmentToken) error
	ConsumeEnrollmentToken(ctx context.Context, tokenHash string, clientIP string) (*ServerEnrollmentToken, error)

	// Telemetry & Metrics
	RecordServerMetric(ctx context.Context, metric *ServerMetric) error
	GetLatestServerMetrics(ctx context.Context, serverID uuid.UUID, limit int) ([]*ServerMetric, error)

	// Audit Logs
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	ListAuditLogsByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]*AuditLog, error)

	// Websites
	CreateWebsite(ctx context.Context, site *Website) error
	GetWebsiteByID(ctx context.Context, id uuid.UUID) (*Website, error)
	ListWebsitesByOrg(ctx context.Context, orgID uuid.UUID) ([]*Website, error)
	UpdateWebsite(ctx context.Context, site *Website) error
	UpdateWebsiteStatus(ctx context.Context, id uuid.UUID, status string) error
	DeleteWebsite(ctx context.Context, id uuid.UUID) error
	UpdateWebsiteSSL(ctx context.Context, id uuid.UUID, sslEnabled bool) error

	// Databases
	CreateDatabase(ctx context.Context, db *Database) error
	ListDatabasesByServer(ctx context.Context, serverID uuid.UUID) ([]*Database, error)
	DeleteDatabase(ctx context.Context, id uuid.UUID) error
	CreateDatabaseUser(ctx context.Context, user *DatabaseUser) error
	ListDatabaseUsersByServer(ctx context.Context, serverID uuid.UUID) ([]*DatabaseUser, error)

	// SSL
	CreateOrUpdateSSL(ctx context.Context, cert *SSLCertificate) error
	GetSSLByWebsiteID(ctx context.Context, websiteID uuid.UUID) (*SSLCertificate, error)

	// Email Domains
	CreateEmailDomain(ctx context.Context, domain *EmailDomain) error
	GetEmailDomainByID(ctx context.Context, id uuid.UUID) (*EmailDomain, error)
	GetEmailDomainByName(ctx context.Context, serverID uuid.UUID, domainName string) (*EmailDomain, error)
	ListEmailDomainsByOrg(ctx context.Context, orgID uuid.UUID) ([]*EmailDomain, error)
	ListEmailDomainsByServer(ctx context.Context, serverID uuid.UUID) ([]*EmailDomain, error)
	UpdateEmailDomain(ctx context.Context, domain *EmailDomain) error
	DeleteEmailDomain(ctx context.Context, id uuid.UUID) error

	// Email Mailboxes
	CreateEmailMailbox(ctx context.Context, mb *EmailMailbox) error
	GetEmailMailboxByID(ctx context.Context, id uuid.UUID) (*EmailMailbox, error)
	GetEmailMailboxByEmail(ctx context.Context, email string) (*EmailMailbox, error)
	ListEmailMailboxesByDomain(ctx context.Context, domainID uuid.UUID) ([]*EmailMailbox, error)
	ListEmailMailboxesByServer(ctx context.Context, serverID uuid.UUID) ([]*EmailMailbox, error)
	UpdateEmailMailbox(ctx context.Context, mb *EmailMailbox) error
	UpdateEmailMailboxPassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	DeleteEmailMailbox(ctx context.Context, id uuid.UUID) error

	// Email Aliases & Forwarders
	CreateEmailAlias(ctx context.Context, alias *EmailAlias) error
	ListEmailAliasesByDomain(ctx context.Context, domainID uuid.UUID) ([]*EmailAlias, error)
	DeleteEmailAlias(ctx context.Context, id uuid.UUID) error
	CreateEmailForwarder(ctx context.Context, fwd *EmailForwarder) error
	ListEmailForwardersByDomain(ctx context.Context, domainID uuid.UUID) ([]*EmailForwarder, error)
	DeleteEmailForwarder(ctx context.Context, id uuid.UUID) error

	// Email Autoresponders & DKIM
	SetEmailAutoresponder(ctx context.Context, ar *EmailAutoresponder) error
	GetEmailAutoresponderByMailbox(ctx context.Context, mailboxID uuid.UUID) (*EmailAutoresponder, error)
	DeleteEmailAutoresponder(ctx context.Context, mailboxID uuid.UUID) error
	SaveEmailDKIMKey(ctx context.Context, dkim *EmailDKIMKey) error
	GetEmailDKIMKeyByDomain(ctx context.Context, domainID uuid.UUID) (*EmailDKIMKey, error)

	// Email Delivery Logs
	RecordEmailDeliveryLog(ctx context.Context, log *EmailDeliveryLog) error
	ListEmailDeliveryLogs(ctx context.Context, serverID uuid.UUID, limit int) ([]*EmailDeliveryLog, error)

	// PHP Management
	UpsertPHPVersion(ctx context.Context, v *PHPInstalledVersion) error
	GetPHPVersion(ctx context.Context, serverID uuid.UUID, version string) (*PHPInstalledVersion, error)
	ListPHPVersionsByServer(ctx context.Context, serverID uuid.UUID) ([]*PHPInstalledVersion, error)
	DeletePHPVersion(ctx context.Context, serverID uuid.UUID, version string) error
	SetDefaultPHPCli(ctx context.Context, serverID uuid.UUID, version string) error
	SetDefaultPHPFpm(ctx context.Context, serverID uuid.UUID, version string) error

	UpsertPHPExtension(ctx context.Context, ext *PHPExtension) error
	ListPHPExtensions(ctx context.Context, serverID uuid.UUID, version string) ([]*PHPExtension, error)
	GetPHPExtension(ctx context.Context, serverID uuid.UUID, version string, name string) (*PHPExtension, error)
	DeletePHPExtension(ctx context.Context, serverID uuid.UUID, version string, name string) error

	CreatePHPFPMPool(ctx context.Context, pool *PHPFPMPool) error
	GetPHPFPMPool(ctx context.Context, id uuid.UUID) (*PHPFPMPool, error)
	GetPHPFPMPoolByName(ctx context.Context, serverID uuid.UUID, name string) (*PHPFPMPool, error)
	GetPHPFPMPoolByWebsite(ctx context.Context, websiteID uuid.UUID) (*PHPFPMPool, error)
	ListPHPFPMPoolsByServer(ctx context.Context, serverID uuid.UUID, version string) ([]*PHPFPMPool, error)
	UpdatePHPFPMPool(ctx context.Context, pool *PHPFPMPool) error
	DeletePHPFPMPool(ctx context.Context, id uuid.UUID) error

	UpsertPHPIniOverride(ctx context.Context, override *PHPIniOverride) error
	ListPHPIniOverrides(ctx context.Context, serverID uuid.UUID, version string, scope string, websiteID *uuid.UUID) ([]*PHPIniOverride, error)
	DeletePHPIniOverride(ctx context.Context, id uuid.UUID) error

	CreatePHPConfigBackup(ctx context.Context, backup *PHPConfigBackup) error
	ListPHPConfigBackups(ctx context.Context, serverID uuid.UUID, version string, limit int) ([]*PHPConfigBackup, error)

	// Web Server Management
	UpsertWebServerInstance(ctx context.Context, instance *WebServerInstance) error
	GetWebServerInstance(ctx context.Context, serverID uuid.UUID, serverType string) (*WebServerInstance, error)
	ListWebServerInstances(ctx context.Context, serverID uuid.UUID) ([]*WebServerInstance, error)
	SetActiveDefaultWebServer(ctx context.Context, serverID uuid.UUID, serverType string) error

	UpsertWebServerVHost(ctx context.Context, vhost *WebServerVHost) error
	GetWebServerVHost(ctx context.Context, serverID uuid.UUID, serverType string, domain string) (*WebServerVHost, error)
	ListWebServerVHosts(ctx context.Context, serverID uuid.UUID, serverType string) ([]*WebServerVHost, error)
	DeleteWebServerVHost(ctx context.Context, serverID uuid.UUID, serverType string, domain string) error

	CreateWebServerConfigBackup(ctx context.Context, backup *WebServerConfigBackup) error
	ListWebServerConfigBackups(ctx context.Context, serverID uuid.UUID, serverType string, limit int) ([]*WebServerConfigBackup, error)

	// Close
	Close() error
}

// ============================================================================
// IN-MEMORY STORE (Production-grade fallback & unit test engine)
// ============================================================================
type MemoryStore struct {
	mu                  sync.RWMutex
	orgs                map[uuid.UUID]*Organization
	orgsBySlug          map[string]uuid.UUID
	users               map[uuid.UUID]*User
	usersByEmail        map[string]uuid.UUID
	servers             map[uuid.UUID]*Server
	tokens              map[string]*ServerEnrollmentToken
	metrics             map[uuid.UUID][]*ServerMetric
	auditLogs           []*AuditLog
	websites            map[uuid.UUID]*Website
	databases           map[uuid.UUID]*Database
	databaseUsers       map[uuid.UUID]*DatabaseUser
	sslCerts            map[uuid.UUID]*SSLCertificate
	emailDomains        map[uuid.UUID]*EmailDomain
	emailMailboxes      map[uuid.UUID]*EmailMailbox
	emailAliases        map[uuid.UUID]*EmailAlias
	emailForwarders     map[uuid.UUID]*EmailForwarder
	emailAutoresponders map[uuid.UUID]*EmailAutoresponder
	emailDKIMKeys       map[uuid.UUID]*EmailDKIMKey
	emailDeliveryLogs   []*EmailDeliveryLog
	phpVersions         map[string]*PHPInstalledVersion // key: serverID:version
	phpExtensions       map[string]*PHPExtension        // key: serverID:version:name
	phpPools            map[uuid.UUID]*PHPFPMPool
	phpIniOverrides     map[string]*PHPIniOverride // key: serverID:version:scope:siteID:directive
	phpConfigBackups    []*PHPConfigBackup
	webServerInstances  map[string]*WebServerInstance // key: serverID:type
	webServerVHosts     map[string]*WebServerVHost    // key: serverID:type:domain
	webServerBackups    []*WebServerConfigBackup
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		orgs:                make(map[uuid.UUID]*Organization),
		orgsBySlug:          make(map[string]uuid.UUID),
		users:               make(map[uuid.UUID]*User),
		usersByEmail:        make(map[string]uuid.UUID),
		servers:             make(map[uuid.UUID]*Server),
		tokens:              make(map[string]*ServerEnrollmentToken),
		metrics:             make(map[uuid.UUID][]*ServerMetric),
		auditLogs:           make([]*AuditLog, 0),
		websites:            make(map[uuid.UUID]*Website),
		databases:           make(map[uuid.UUID]*Database),
		databaseUsers:       make(map[uuid.UUID]*DatabaseUser),
		sslCerts:            make(map[uuid.UUID]*SSLCertificate),
		emailDomains:        make(map[uuid.UUID]*EmailDomain),
		emailMailboxes:      make(map[uuid.UUID]*EmailMailbox),
		emailAliases:        make(map[uuid.UUID]*EmailAlias),
		emailForwarders:     make(map[uuid.UUID]*EmailForwarder),
		emailAutoresponders: make(map[uuid.UUID]*EmailAutoresponder),
		emailDKIMKeys:       make(map[uuid.UUID]*EmailDKIMKey),
		emailDeliveryLogs:   make([]*EmailDeliveryLog, 0),
		phpVersions:         make(map[string]*PHPInstalledVersion),
		phpExtensions:       make(map[string]*PHPExtension),
		phpPools:            make(map[uuid.UUID]*PHPFPMPool),
		phpIniOverrides:     make(map[string]*PHPIniOverride),
		phpConfigBackups:    make([]*PHPConfigBackup, 0),
		webServerInstances:  make(map[string]*WebServerInstance),
		webServerVHosts:     make(map[string]*WebServerVHost),
		webServerBackups:    make([]*WebServerConfigBackup, 0),
	}
}

func (m *MemoryStore) Close() error {
	return nil
}

func (m *MemoryStore) CreateOrganization(ctx context.Context, org *Organization) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.orgsBySlug[org.Slug]; exists {
		return ErrAlreadyExists
	}
	if org.ID == uuid.Nil {
		org.ID = uuid.New()
	}
	org.CreatedAt = time.Now().UTC()
	org.UpdatedAt = org.CreatedAt

	m.orgs[org.ID] = org
	m.orgsBySlug[org.Slug] = org.ID
	return nil
}

func (m *MemoryStore) GetOrganizationByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	org, exists := m.orgs[id]
	if !exists {
		return nil, ErrNotFound
	}
	return org, nil
}

func (m *MemoryStore) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id, exists := m.orgsBySlug[slug]
	if !exists {
		return nil, ErrNotFound
	}
	return m.orgs[id], nil
}

func (m *MemoryStore) CreateUser(ctx context.Context, user *User, orgID uuid.UUID, role string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.usersByEmail[user.Email]; exists {
		return ErrAlreadyExists
	}
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now
	user.DefaultOrgID = orgID
	user.Role = role

	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user.ID
	return nil
}

func (m *MemoryStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id, exists := m.usersByEmail[email]
	if !exists {
		return nil, ErrNotFound
	}
	return m.users[id], nil
}

func (m *MemoryStore) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[id]
	if !exists {
		return nil, ErrNotFound
	}
	return user, nil
}

func (m *MemoryStore) UpdateUserLastLogin(ctx context.Context, id uuid.UUID, ip string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		return ErrNotFound
	}
	now := time.Now().UTC()
	user.LastLoginAt = &now
	user.LastLoginIP = ip
	return nil
}

func (m *MemoryStore) CreateServer(ctx context.Context, server *Server) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if server.ID == uuid.Nil {
		server.ID = uuid.New()
	}
	now := time.Now().UTC()
	server.CreatedAt = now
	server.UpdatedAt = now
	m.servers[server.ID] = server
	return nil
}

func (m *MemoryStore) GetServerByID(ctx context.Context, id uuid.UUID) (*Server, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, exists := m.servers[id]
	if !exists {
		return nil, ErrNotFound
	}
	return server, nil
}

func (m *MemoryStore) ListServersByOrg(ctx context.Context, orgID uuid.UUID) ([]*Server, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]*Server, 0)
	for _, s := range m.servers {
		if s.OrganizationID == orgID {
			results = append(results, s)
		}
	}
	return results, nil
}

func (m *MemoryStore) UpdateServerHeartbeat(ctx context.Context, id uuid.UUID, uptime int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	server, exists := m.servers[id]
	if !exists {
		return ErrNotFound
	}
	now := time.Now().UTC()
	server.LastHeartbeatAt = &now
	server.UptimeSeconds = uptime
	server.Status = "online"
	server.UpdatedAt = now
	return nil
}

func (m *MemoryStore) CreateEnrollmentToken(ctx context.Context, token *ServerEnrollmentToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	token.CreatedAt = time.Now().UTC()
	m.tokens[token.TokenHash] = token
	return nil
}

func (m *MemoryStore) ConsumeEnrollmentToken(ctx context.Context, tokenHash string, clientIP string) (*ServerEnrollmentToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	token, exists := m.tokens[tokenHash]
	if !exists {
		return nil, ErrNotFound
	}
	if token.UsedAt != nil {
		return nil, ErrTokenUsed
	}
	if time.Now().UTC().After(token.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	now := time.Now().UTC()
	token.UsedAt = &now
	token.UsedByIP = clientIP
	return token, nil
}

func (m *MemoryStore) RecordServerMetric(ctx context.Context, metric *ServerMetric) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if metric.RecordedAt.IsZero() {
		metric.RecordedAt = time.Now().UTC()
	}
	m.metrics[metric.ServerID] = append(m.metrics[metric.ServerID], metric)
	return nil
}

func (m *MemoryStore) GetLatestServerMetrics(ctx context.Context, serverID uuid.UUID, limit int) ([]*ServerMetric, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := m.metrics[serverID]
	if len(metrics) == 0 {
		return []*ServerMetric{}, nil
	}
	if limit <= 0 || limit > len(metrics) {
		limit = len(metrics)
	}
	start := len(metrics) - limit
	return metrics[start:], nil
}

func (m *MemoryStore) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}
	m.auditLogs = append([]*AuditLog{log}, m.auditLogs...) // prepend for recency
	return nil
}

func (m *MemoryStore) ListAuditLogsByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]*AuditLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]*AuditLog, 0)
	for _, l := range m.auditLogs {
		if l.OrganizationID != nil && *l.OrganizationID == orgID {
			results = append(results, l)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

// ============================================================================
// POSTGRESQL STORE
// ============================================================================
type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgresStore{db: db}, nil
}

func (p *PostgresStore) Close() error {
	return p.db.Close()
}

func (p *PostgresStore) CreateOrganization(ctx context.Context, org *Organization) error {
	query := `
		INSERT INTO organizations (id, name, slug, plan_tier, max_servers, max_websites)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`
	if org.ID == uuid.Nil {
		org.ID = uuid.New()
	}
	return p.db.QueryRowContext(ctx, query,
		org.ID, org.Name, org.Slug, org.PlanTier, org.MaxServers, org.MaxWebsites,
	).Scan(&org.CreatedAt, &org.UpdatedAt)
}

func (p *PostgresStore) GetOrganizationByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	query := `
		SELECT id, name, slug, plan_tier, max_servers, max_websites, created_at, updated_at
		FROM organizations WHERE id = $1 AND deleted_at IS NULL
	`
	org := &Organization{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Slug, &org.PlanTier, &org.MaxServers, &org.MaxWebsites, &org.CreatedAt, &org.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return org, err
}

func (p *PostgresStore) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	query := `
		SELECT id, name, slug, plan_tier, max_servers, max_websites, created_at, updated_at
		FROM organizations WHERE slug = $1 AND deleted_at IS NULL
	`
	org := &Organization{}
	err := p.db.QueryRowContext(ctx, query, slug).Scan(
		&org.ID, &org.Name, &org.Slug, &org.PlanTier, &org.MaxServers, &org.MaxWebsites, &org.CreatedAt, &org.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return org, err
}

func (p *PostgresStore) CreateUser(ctx context.Context, user *User, orgID uuid.UUID, role string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	userQuery := `
		INSERT INTO users (id, email, password_hash, full_name, is_active, is_superadmin)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`
	err = tx.QueryRowContext(ctx, userQuery,
		user.ID, user.Email, user.PasswordHash, user.FullName, user.IsActive, user.IsSuperAdmin,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return err
	}

	// Fetch or create role ID
	var roleID uuid.UUID
	roleQuery := `SELECT id FROM roles WHERE name = $1 LIMIT 1`
	err = tx.QueryRowContext(ctx, roleQuery, role).Scan(&roleID)
	if err != nil {
		// Insert default role for org
		insertRole := `INSERT INTO roles (organization_id, name) VALUES ($1, $2) RETURNING id`
		if err := tx.QueryRowContext(ctx, insertRole, orgID, role).Scan(&roleID); err != nil {
			return err
		}
	}

	memberQuery := `
		INSERT INTO organization_members (organization_id, user_id, role_id)
		VALUES ($1, $2, $3)
	`
	if _, err := tx.ExecContext(ctx, memberQuery, orgID, user.ID, roleID); err != nil {
		return err
	}

	user.DefaultOrgID = orgID
	user.Role = role
	return tx.Commit()
}

func (p *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.full_name, u.is_active, u.is_superadmin, u.two_factor_enabled,
		       om.organization_id, r.name
		FROM users u
		LEFT JOIN organization_members om ON om.user_id = u.id
		LEFT JOIN roles r ON r.id = om.role_id
		WHERE u.email = $1 AND u.deleted_at IS NULL
		LIMIT 1
	`
	user := &User{}
	err := p.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.IsActive, &user.IsSuperAdmin, &user.TwoFactorEnabled,
		&user.DefaultOrgID, &user.Role,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return user, err
}

func (p *PostgresStore) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.full_name, u.is_active, u.is_superadmin, u.two_factor_enabled,
		       om.organization_id, COALESCE(r.name, 'owner')
		FROM users u
		LEFT JOIN organization_members om ON om.user_id = u.id
		LEFT JOIN roles r ON r.id = om.role_id
		WHERE u.id = $1 AND u.deleted_at IS NULL
		LIMIT 1
	`
	user := &User{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.IsActive, &user.IsSuperAdmin, &user.TwoFactorEnabled,
		&user.DefaultOrgID, &user.Role,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return user, err
}

func (p *PostgresStore) UpdateUserLastLogin(ctx context.Context, id uuid.UUID, ip string) error {
	query := `UPDATE users SET last_login_at = NOW(), last_login_ip = $2 WHERE id = $1`
	_, err := p.db.ExecContext(ctx, query, id, ip)
	return err
}

func (p *PostgresStore) CreateServer(ctx context.Context, server *Server) error {
	query := `
		INSERT INTO servers (id, organization_id, name, hostname, ip_address, os_name, os_version, architecture, kernel_version, agent_version, status, cpu_cores, cpu_model, ram_total_mb, disk_total_gb, agent_token_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING created_at, updated_at
	`
	if server.ID == uuid.Nil {
		server.ID = uuid.New()
	}
	return p.db.QueryRowContext(ctx, query,
		server.ID, server.OrganizationID, server.Name, server.Hostname, server.IPAddress,
		server.OSName, server.OSVersion, server.Architecture, server.KernelVersion,
		server.AgentVersion, server.Status, server.CPUCores, server.CPUModel,
		server.RAMTotalMB, server.DiskTotalGB, server.AgentTokenHash,
	).Scan(&server.CreatedAt, &server.UpdatedAt)
}

func (p *PostgresStore) GetServerByID(ctx context.Context, id uuid.UUID) (*Server, error) {
	query := `
		SELECT id, organization_id, name, hostname, ip_address, os_name, os_version, architecture, kernel_version, agent_version, status, cpu_cores, cpu_model, ram_total_mb, disk_total_gb, last_heartbeat_at, uptime_seconds, created_at, updated_at
		FROM servers WHERE id = $1 AND deleted_at IS NULL
	`
	s := &Server{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.OrganizationID, &s.Name, &s.Hostname, &s.IPAddress, &s.OSName, &s.OSVersion,
		&s.Architecture, &s.KernelVersion, &s.AgentVersion, &s.Status, &s.CPUCores, &s.CPUModel,
		&s.RAMTotalMB, &s.DiskTotalGB, &s.LastHeartbeatAt, &s.UptimeSeconds, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (p *PostgresStore) ListServersByOrg(ctx context.Context, orgID uuid.UUID) ([]*Server, error) {
	query := `
		SELECT id, organization_id, name, hostname, ip_address, os_name, os_version, architecture, kernel_version, agent_version, status, cpu_cores, cpu_model, ram_total_mb, disk_total_gb, last_heartbeat_at, uptime_seconds, created_at, updated_at
		FROM servers WHERE organization_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	servers := make([]*Server, 0)
	for rows.Next() {
		s := &Server{}
		err := rows.Scan(
			&s.ID, &s.OrganizationID, &s.Name, &s.Hostname, &s.IPAddress, &s.OSName, &s.OSVersion,
			&s.Architecture, &s.KernelVersion, &s.AgentVersion, &s.Status, &s.CPUCores, &s.CPUModel,
			&s.RAMTotalMB, &s.DiskTotalGB, &s.LastHeartbeatAt, &s.UptimeSeconds, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		servers = append(servers, s)
	}
	return servers, nil
}

func (p *PostgresStore) UpdateServerHeartbeat(ctx context.Context, id uuid.UUID, uptime int64) error {
	query := `
		UPDATE servers
		SET last_heartbeat_at = NOW(), uptime_seconds = $2, status = 'online', updated_at = NOW()
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query, id, uptime)
	return err
}

func (p *PostgresStore) CreateEnrollmentToken(ctx context.Context, token *ServerEnrollmentToken) error {
	query := `
		INSERT INTO server_enrollment_tokens (id, organization_id, token_hash, label, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	return p.db.QueryRowContext(ctx, query,
		token.ID, token.OrganizationID, token.TokenHash, token.Label, token.ExpiresAt, token.CreatedBy,
	).Scan(&token.CreatedAt)
}

func (p *PostgresStore) ConsumeEnrollmentToken(ctx context.Context, tokenHash string, clientIP string) (*ServerEnrollmentToken, error) {
	query := `
		UPDATE server_enrollment_tokens
		SET used_at = NOW(), used_by_ip = $2
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
		RETURNING id, organization_id, token_hash, label, expires_at, used_at, used_by_ip, created_by, created_at
	`
	token := &ServerEnrollmentToken{}
	err := p.db.QueryRowContext(ctx, query, tokenHash, clientIP).Scan(
		&token.ID, &token.OrganizationID, &token.TokenHash, &token.Label, &token.ExpiresAt,
		&token.UsedAt, &token.UsedByIP, &token.CreatedBy, &token.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return token, err
}

func (p *PostgresStore) RecordServerMetric(ctx context.Context, metric *ServerMetric) error {
	query := `
		INSERT INTO server_metrics (server_id, cpu_percent, ram_used_mb, ram_total_mb, disk_used_gb, disk_total_gb, load_1m, load_5m, load_15m, net_rx_bytes, net_tx_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, recorded_at
	`
	return p.db.QueryRowContext(ctx, query,
		metric.ServerID, metric.CPUPercent, metric.RAMUsedMB, metric.RAMTotalMB,
		metric.DiskUsedGB, metric.DiskTotalGB, metric.Load1m, metric.Load5m, metric.Load15m,
		metric.NetRxBytes, metric.NetTxBytes,
	).Scan(&metric.ID, &metric.RecordedAt)
}

func (p *PostgresStore) GetLatestServerMetrics(ctx context.Context, serverID uuid.UUID, limit int) ([]*ServerMetric, error) {
	query := `
		SELECT id, server_id, cpu_percent, ram_used_mb, ram_total_mb, disk_used_gb, disk_total_gb, load_1m, load_5m, load_15m, net_rx_bytes, net_tx_bytes, recorded_at
		FROM server_metrics
		WHERE server_id = $1
		ORDER BY recorded_at DESC
		LIMIT $2
	`
	rows, err := p.db.QueryContext(ctx, query, serverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics := make([]*ServerMetric, 0)
	for rows.Next() {
		m := &ServerMetric{}
		err := rows.Scan(
			&m.ID, &m.ServerID, &m.CPUPercent, &m.RAMUsedMB, &m.RAMTotalMB,
			&m.DiskUsedGB, &m.DiskTotalGB, &m.Load1m, &m.Load5m, &m.Load15m,
			&m.NetRxBytes, &m.NetTxBytes, &m.RecordedAt,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}

func (p *PostgresStore) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	metaJSON, _ := json.Marshal(log.Metadata)
	query := `
		INSERT INTO audit_logs (id, organization_id, user_id, server_id, action, resource_type, resource_id, ip_address, user_agent, status, error_message, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at
	`
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	return p.db.QueryRowContext(ctx, query,
		log.ID, log.OrganizationID, log.UserID, log.ServerID, log.Action, log.ResourceType,
		log.ResourceID, log.IPAddress, log.UserAgent, log.Status, log.ErrorMessage, metaJSON,
	).Scan(&log.CreatedAt)
}

func (p *PostgresStore) ListAuditLogsByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]*AuditLog, error) {
	query := `
		SELECT id, organization_id, user_id, server_id, action, resource_type, resource_id, ip_address, user_agent, status, error_message, metadata, created_at
		FROM audit_logs
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := p.db.QueryContext(ctx, query, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]*AuditLog, 0)
	for rows.Next() {
		l := &AuditLog{}
		var metaBytes []byte
		err := rows.Scan(
			&l.ID, &l.OrganizationID, &l.UserID, &l.ServerID, &l.Action, &l.ResourceType,
			&l.ResourceID, &l.IPAddress, &l.UserAgent, &l.Status, &l.ErrorMessage, &metaBytes, &l.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if len(metaBytes) > 0 {
			_ = json.Unmarshal(metaBytes, &l.Metadata)
		}
		logs = append(logs, l)
	}
	return logs, nil
}
