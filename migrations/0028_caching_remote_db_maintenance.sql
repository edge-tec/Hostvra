-- 0028_caching_remote_db_maintenance.sql
-- Phase Seven: Caching Engine (Redis/Memcached), Remote DB Access Grants & Database Maintenance

-- 1. Per-Website Caching Configuration (Redis & Memcached)
CREATE TABLE IF NOT EXISTS website_cache_configs (
    id UUID PRIMARY KEY,
    website_id UUID NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    cache_type VARCHAR(32) NOT NULL DEFAULT 'none',
    connection_type VARCHAR(32) NOT NULL DEFAULT 'socket',
    socket_path TEXT NOT NULL DEFAULT '',
    host VARCHAR(255) NOT NULL DEFAULT '127.0.0.1',
    port INTEGER NOT NULL DEFAULT 6379,
    password TEXT NOT NULL DEFAULT '',
    max_memory_mb INTEGER NOT NULL DEFAULT 128,
    eviction_policy VARCHAR(64) NOT NULL DEFAULT 'allkeys-lru',
    wp_object_cache_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(32) NOT NULL DEFAULT 'disabled',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_website_cache_configs_website UNIQUE (website_id)
);

CREATE INDEX IF NOT EXISTS idx_website_cache_configs_website ON website_cache_configs(website_id);
CREATE INDEX IF NOT EXISTS idx_website_cache_configs_org ON website_cache_configs(organization_id);

-- 2. Remote Database Access Grants with Authorized IP Allowlists
CREATE TABLE IF NOT EXISTS database_remote_access (
    id UUID PRIMARY KEY,
    database_id UUID NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    db_user VARCHAR(64) NOT NULL,
    authorized_ip VARCHAR(64) NOT NULL,
    granted_privileges VARCHAR(255) NOT NULL DEFAULT 'ALL PRIVILEGES',
    firewall_rule_applied BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    notes TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_database_remote_access_db ON database_remote_access(database_id);
CREATE INDEX IF NOT EXISTS idx_database_remote_access_org ON database_remote_access(organization_id);
CREATE INDEX IF NOT EXISTS idx_database_remote_access_ip ON database_remote_access(authorized_ip);

-- 3. Database Maintenance & Table Integrity Operations Log
CREATE TABLE IF NOT EXISTS database_maintenance_logs (
    id UUID PRIMARY KEY,
    database_id UUID NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    operation VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    tables_processed INTEGER NOT NULL DEFAULT 0,
    bytes_reclaimed BIGINT NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    details TEXT NOT NULL DEFAULT '',
    initiated_by VARCHAR(255) NOT NULL DEFAULT 'manual',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_database_maintenance_logs_db ON database_maintenance_logs(database_id);
CREATE INDEX IF NOT EXISTS idx_database_maintenance_logs_created ON database_maintenance_logs(created_at DESC);
