-- ============================================================================
-- Migration: 0023_multi_server_fleet_management.sql
-- Description: Multi-Server Fleet Capabilities, Node Draining & Cross-Node Migration
-- ============================================================================

-- 1. Extend servers table with fleet capabilities, maintenance mode, and quotas
ALTER TABLE servers
    ADD COLUMN IF NOT EXISTS capabilities JSONB NOT NULL DEFAULT '{"web_servers":["nginx"],"php_versions":["8.2","8.3"],"databases":["mariadb"],"services":["ufw","fail2ban"]}'::jsonb,
    ADD COLUMN IF NOT EXISTS is_maintenance BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS draining BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS max_websites INT NOT NULL DEFAULT 100,
    ADD COLUMN IF NOT EXISTS max_databases INT NOT NULL DEFAULT 100;

CREATE INDEX IF NOT EXISTS idx_servers_maintenance ON servers(is_maintenance) WHERE is_maintenance = TRUE;
CREATE INDEX IF NOT EXISTS idx_servers_draining ON servers(draining) WHERE draining = TRUE;

-- 2. Cross-Node Migrations Table
CREATE TABLE IF NOT EXISTS cross_node_migrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    source_server_id UUID NOT NULL REFERENCES servers(id) ON DELETE RESTRICT,
    destination_server_id UUID NOT NULL REFERENCES servers(id) ON DELETE RESTRICT,
    website_id UUID REFERENCES websites(id) ON DELETE SET NULL,
    database_id UUID REFERENCES databases(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    progress INT NOT NULL DEFAULT 0,
    current_step VARCHAR(255) NOT NULL DEFAULT 'Initialized',
    source_checksum VARCHAR(64) DEFAULT '',
    destination_checksum VARCHAR(64) DEFAULT '',
    transferred_bytes BIGINT NOT NULL DEFAULT 0,
    total_bytes BIGINT NOT NULL DEFAULT 0,
    error_message TEXT DEFAULT '',
    logs JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cross_node_migrations_org ON cross_node_migrations(organization_id);
CREATE INDEX IF NOT EXISTS idx_cross_node_migrations_source ON cross_node_migrations(source_server_id);
CREATE INDEX IF NOT EXISTS idx_cross_node_migrations_dest ON cross_node_migrations(destination_server_id);
CREATE INDEX IF NOT EXISTS idx_cross_node_migrations_status ON cross_node_migrations(status);
