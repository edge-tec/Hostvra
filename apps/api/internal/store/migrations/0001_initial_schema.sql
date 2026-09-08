-- Hostvra Database Migration 0001: Initial Core Schema
-- Production PostgreSQL DDL with UUID primary keys, relational integrity, and optimized indexing.

-- Enable pgcrypto for UUID generation if not already present
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- 1. MULTI-TENANCY & ORGANIZATIONS
-- ============================================================================
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    plan_tier VARCHAR(50) NOT NULL DEFAULT 'free', -- 'free', 'pro', 'enterprise'
    max_servers INT NOT NULL DEFAULT 1,
    max_websites INT NOT NULL DEFAULT 5,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug) WHERE deleted_at IS NULL;

-- ============================================================================
-- 2. USERS & IDENTITY
-- ============================================================================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_superadmin BOOLEAN NOT NULL DEFAULT FALSE,
    two_factor_secret VARCHAR(255),
    two_factor_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at TIMESTAMPTZ,
    last_login_ip INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE deleted_at IS NULL;

-- ============================================================================
-- 3. ROLE-BASED ACCESS CONTROL (RBAC)
-- ============================================================================
CREATE TABLE IF NOT EXISTS permissions (
    id VARCHAR(100) PRIMARY KEY, -- e.g. 'servers.view', 'websites.create'
    module VARCHAR(50) NOT NULL, -- e.g. 'servers', 'websites', 'databases', 'security'
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL, -- 'owner', 'admin', 'manager', 'developer', 'viewer'
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, name)
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id VARCHAR(100) NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS organization_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_org_members_user ON organization_members(user_id);
CREATE INDEX IF NOT EXISTS idx_org_members_org ON organization_members(organization_id);

-- User Sessions & Refresh Tokens
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) NOT NULL UNIQUE,
    user_agent TEXT,
    ip_address INET,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_hash ON user_sessions(refresh_token_hash) WHERE is_revoked IS FALSE;

-- ============================================================================
-- 4. SERVERS & HOSTVRA AGENTS
-- ============================================================================
CREATE TABLE IF NOT EXISTS servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    ip_address INET NOT NULL,
    os_name VARCHAR(100) NOT NULL, -- 'Ubuntu', 'Debian', 'AlmaLinux', etc.
    os_version VARCHAR(100) NOT NULL,
    architecture VARCHAR(50) NOT NULL DEFAULT 'x86_64',
    kernel_version VARCHAR(100),
    agent_version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    status VARCHAR(50) NOT NULL DEFAULT 'offline', -- 'online', 'offline', 'connecting', 'maintenance', 'error'
    cpu_cores INT NOT NULL DEFAULT 1,
    cpu_model VARCHAR(255),
    ram_total_mb BIGINT NOT NULL DEFAULT 0,
    disk_total_gb BIGINT NOT NULL DEFAULT 0,
    agent_token_hash VARCHAR(255) NOT NULL,
    last_heartbeat_at TIMESTAMPTZ,
    uptime_seconds BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_servers_org ON servers(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_servers_status ON servers(status);

-- Server Enrollment Tokens (Temporary, single-use onboarding tokens)
CREATE TABLE IF NOT EXISTS server_enrollment_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    label VARCHAR(255),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    used_by_ip INET,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_enrollment_tokens_hash ON server_enrollment_tokens(token_hash) WHERE used_at IS NULL;

-- Realtime & Historical Hardware Telemetry Metrics
CREATE TABLE IF NOT EXISTS server_metrics (
    id BIGSERIAL PRIMARY KEY,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    cpu_percent NUMERIC(5, 2) NOT NULL,
    ram_used_mb BIGINT NOT NULL,
    ram_total_mb BIGINT NOT NULL,
    disk_used_gb BIGINT NOT NULL,
    disk_total_gb BIGINT NOT NULL,
    load_1m NUMERIC(6, 2) NOT NULL,
    load_5m NUMERIC(6, 2) NOT NULL,
    load_15m NUMERIC(6, 2) NOT NULL,
    net_rx_bytes BIGINT NOT NULL DEFAULT 0,
    net_tx_bytes BIGINT NOT NULL DEFAULT 0,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_server_metrics_recorded ON server_metrics(server_id, recorded_at DESC);

-- ============================================================================
-- 5. WEBSITES, DOMAINS & SSL
-- ============================================================================
CREATE TABLE IF NOT EXISTS websites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    primary_domain VARCHAR(255) NOT NULL,
    document_root VARCHAR(512) NOT NULL,
    system_user VARCHAR(100) NOT NULL,
    php_version VARCHAR(20), -- e.g. '8.3', '8.2' or NULL
    app_type VARCHAR(50) NOT NULL DEFAULT 'php', -- 'php', 'nodejs', 'python', 'static', 'proxy'
    proxy_port INT,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- 'active', 'suspended', 'disabled'
    ssl_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(server_id, primary_domain)
);
CREATE INDEX IF NOT EXISTS idx_websites_server ON websites(server_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_websites_domain ON websites(primary_domain);

CREATE TABLE IF NOT EXISTS website_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    website_id UUID NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    domain VARCHAR(255) NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    redirect_target VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_website_domains_domain ON website_domains(domain);

CREATE TABLE IF NOT EXISTS ssl_certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    website_id UUID NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    domain_list TEXT[] NOT NULL,
    issuer VARCHAR(100) NOT NULL DEFAULT 'Let''s Encrypt',
    cert_path VARCHAR(512) NOT NULL,
    key_path VARCHAR(512) NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    auto_renew BOOLEAN NOT NULL DEFAULT TRUE,
    status VARCHAR(50) NOT NULL DEFAULT 'valid', -- 'valid', 'expired', 'renewing', 'failed'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- 6. DATABASES & DATABASE USERS
-- ============================================================================
CREATE TABLE IF NOT EXISTS databases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    db_type VARCHAR(50) NOT NULL DEFAULT 'mysql', -- 'mysql', 'mariadb', 'postgresql'
    name VARCHAR(100) NOT NULL,
    character_set VARCHAR(50) NOT NULL DEFAULT 'utf8mb4',
    collation VARCHAR(50) NOT NULL DEFAULT 'utf8mb4_unicode_ci',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(server_id, db_type, name)
);

CREATE TABLE IF NOT EXISTS database_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    db_type VARCHAR(50) NOT NULL DEFAULT 'mysql',
    username VARCHAR(100) NOT NULL,
    host_allow VARCHAR(100) NOT NULL DEFAULT 'localhost',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(server_id, db_type, username, host_allow)
);

-- ============================================================================
-- 7. FIREWALL & CRON JOBS
-- ============================================================================
CREATE TABLE IF NOT EXISTS firewall_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    port VARCHAR(50) NOT NULL, -- e.g. '80', '443', '22', '3000:3050'
    protocol VARCHAR(20) NOT NULL DEFAULT 'tcp', -- 'tcp', 'udp', 'both'
    action VARCHAR(20) NOT NULL DEFAULT 'allow', -- 'allow', 'deny'
    source_ip VARCHAR(100) NOT NULL DEFAULT 'any',
    comment VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cron_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    schedule VARCHAR(100) NOT NULL, -- e.g. '0 3 * * *'
    command TEXT NOT NULL,
    system_user VARCHAR(100) NOT NULL DEFAULT 'root',
    description VARCHAR(255),
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- 8. LICENSES & FEATURE ENTITLEMENTS
-- ============================================================================
CREATE TABLE IF NOT EXISTS licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID UNIQUE NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    license_key VARCHAR(255) UNIQUE NOT NULL,
    plan_tier VARCHAR(50) NOT NULL DEFAULT 'free', -- 'free', 'pro', 'enterprise'
    server_limit INT NOT NULL DEFAULT 1,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- 'active', 'expired', 'suspended', 'trial'
    features JSONB NOT NULL DEFAULT '{"multi_server": false, "advanced_monitoring": false, "backups": false, "docker": false, "team_management": false, "api_access": false}',
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    last_validated_at TIMESTAMPTZ
);

-- ============================================================================
-- 9. AUDIT LOGGING & SYSTEM NOTIFICATIONS
-- ============================================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL, -- e.g. 'auth.login', 'server.enroll', 'website.create'
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'success', -- 'success', 'failure'
    error_message TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_org_created ON audit_logs(organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    severity VARCHAR(50) NOT NULL DEFAULT 'info', -- 'info', 'warning', 'critical'
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user_read ON notifications(user_id, is_read, created_at DESC);

-- ============================================================================
-- 10. SEED DEFAULT PERMISSIONS & ROLES
-- ============================================================================
INSERT INTO permissions (id, module, description) VALUES
    ('servers.view', 'servers', 'View servers and hardware metrics'),
    ('servers.manage', 'servers', 'Enroll, edit, and reboot servers'),
    ('servers.delete', 'servers', 'Remove servers from fleet'),
    ('websites.view', 'websites', 'View websites and domain lists'),
    ('websites.create', 'websites', 'Create new virtual hosts and websites'),
    ('websites.manage', 'websites', 'Update PHP version, document roots, config'),
    ('websites.delete', 'websites', 'Delete websites and remove directories'),
    ('databases.view', 'databases', 'View databases and users'),
    ('databases.create', 'databases', 'Create databases and grant permissions'),
    ('databases.delete', 'databases', 'Drop databases and remove users'),
    ('ssl.manage', 'security', 'Issue, revoke, and renew SSL certificates'),
    ('firewall.view', 'security', 'View firewall rules and port statuses'),
    ('firewall.manage', 'security', 'Allow or deny network ports'),
    ('cron.manage', 'cron', 'Create, update, and trigger cron jobs'),
    ('docker.manage', 'docker', 'Manage containers, images, and compose files'),
    ('files.browse', 'files', 'Browse server filesystem sandbox'),
    ('files.edit', 'files', 'Upload, edit, and delete files'),
    ('terminal.access', 'terminal', 'Access interactive PTY browser terminal'),
    ('backups.create', 'backups', 'Create full, website, or database backups'),
    ('backups.restore', 'backups', 'Restore backups with verification check'),
    ('users.manage', 'organization', 'Invite, update roles, and remove team members'),
    ('licenses.manage', 'billing', 'Manage plan tier, license keys, and billing'),
    ('audit.view', 'security', 'Inspect organization audit logs')
ON CONFLICT (id) DO NOTHING;
