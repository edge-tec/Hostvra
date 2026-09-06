-- ============================================================================
-- Hostvra.com — Migration 0004: Web Server Management Subsystem Schema
-- Nginx, Apache, OpenLiteSpeed, LiteSpeed Enterprise Instances, VHosts, & Backups
-- ============================================================================

-- 1. Web Server Instances installed on Managed Nodes
CREATE TABLE IF NOT EXISTS web_server_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    server_type VARCHAR(32) NOT NULL, -- 'nginx', 'apache', 'openlitespeed', 'litespeed'
    version VARCHAR(32),
    binary_path VARCHAR(255),
    config_path VARCHAR(255),
    service_name VARCHAR(64) NOT NULL,
    is_installed BOOLEAN NOT NULL DEFAULT FALSE,
    is_active_default BOOLEAN NOT NULL DEFAULT FALSE,
    http_port INT NOT NULL DEFAULT 80,
    https_port INT NOT NULL DEFAULT 443,
    status VARCHAR(32) NOT NULL DEFAULT 'stopped', -- 'running', 'stopped', 'failed', 'not_installed'
    license_status VARCHAR(32) NOT NULL DEFAULT 'n_a', -- 'licensed', 'trial', 'unlicensed', 'n_a'
    installed_modules JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (server_id, server_type)
);

CREATE INDEX IF NOT EXISTS idx_webservers_server ON web_server_instances(server_id);

-- 2. Virtual Hosts mapped to Domains and Websites
CREATE TABLE IF NOT EXISTS web_server_vhosts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    website_id UUID REFERENCES websites(id) ON DELETE CASCADE,
    web_server_type VARCHAR(32) NOT NULL DEFAULT 'nginx', -- 'nginx', 'apache', 'openlitespeed', 'litespeed'
    domain VARCHAR(255) NOT NULL,
    aliases JSONB NOT NULL DEFAULT '[]'::jsonb,
    document_root VARCHAR(255) NOT NULL,
    app_type VARCHAR(32) NOT NULL DEFAULT 'php', -- 'php', 'laravel', 'nodejs', 'python', 'static', 'proxy', 'docker'
    php_version VARCHAR(16), -- e.g. '8.3'
    php_handler_type VARCHAR(16) NOT NULL DEFAULT 'fpm', -- 'fpm' or 'lsphp'
    fpm_socket_path VARCHAR(255),
    proxy_target_url VARCHAR(255),
    websocket_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ssl_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    cert_path VARCHAR(255),
    key_path VARCHAR(255),
    config_file_path VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (server_id, web_server_type, domain)
);

CREATE INDEX IF NOT EXISTS idx_vhosts_server ON web_server_vhosts(server_id, web_server_type);
CREATE INDEX IF NOT EXISTS idx_vhosts_domain ON web_server_vhosts(domain);

-- 3. Reverse Proxies configuration table
CREATE TABLE IF NOT EXISTS web_server_reverse_proxies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vhost_id UUID NOT NULL REFERENCES web_server_vhosts(id) ON DELETE CASCADE,
    location_path VARCHAR(255) NOT NULL DEFAULT '/',
    target_protocol VARCHAR(16) NOT NULL DEFAULT 'http', -- 'http', 'https'
    target_host VARCHAR(255) NOT NULL DEFAULT '127.0.0.1',
    target_port INT NOT NULL,
    websocket_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    request_timeout_sec INT NOT NULL DEFAULT 60,
    custom_headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. Transactional Config Backups for Safe Rollback
CREATE TABLE IF NOT EXISTS web_server_config_backups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    web_server_type VARCHAR(32) NOT NULL,
    file_path VARCHAR(255) NOT NULL,
    content_backup TEXT NOT NULL,
    reason VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webserver_backups ON web_server_config_backups(server_id, web_server_type, created_at DESC);

-- 5. Extend websites table with web_server_type if not present
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'websites' AND column_name = 'web_server_type'
    ) THEN
        ALTER TABLE websites ADD COLUMN web_server_type VARCHAR(32) NOT NULL DEFAULT 'nginx';
    END IF;
END $$;
