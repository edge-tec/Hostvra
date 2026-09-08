-- ============================================================================
-- Hostvra.com — Migration 0003: PHP Management Subsystem Schema
-- Multi-version PHP, Extensions, PHP.ini Engine, FPM Pools, Health & Backups
-- ============================================================================

-- 1. Installed PHP Versions on Managed Servers
CREATE TABLE IF NOT EXISTS php_installed_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    version VARCHAR(16) NOT NULL, -- e.g. "8.1", "8.2", "8.3", "8.4"
    cli_binary_path VARCHAR(255) NOT NULL, -- e.g. "/usr/bin/php8.3"
    fpm_binary_path VARCHAR(255), -- e.g. "/usr/sbin/php-fpm8.3"
    fpm_service_name VARCHAR(64) NOT NULL, -- e.g. "php8.3-fpm"
    fpm_socket_path VARCHAR(255) NOT NULL, -- e.g. "/run/php/php8.3-fpm.sock"
    ini_path VARCHAR(255) NOT NULL, -- e.g. "/etc/php/8.3/fpm/php.ini"
    cli_ini_path VARCHAR(255), -- e.g. "/etc/php/8.3/cli/php.ini"
    fpm_pool_dir VARCHAR(255) NOT NULL, -- e.g. "/etc/php/8.3/fpm/pool.d"
    is_default_cli BOOLEAN NOT NULL DEFAULT FALSE,
    is_default_fpm BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(32) NOT NULL DEFAULT 'installed', -- 'installed', 'installing', 'failed', 'broken'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (server_id, version)
);

CREATE INDEX IF NOT EXISTS idx_php_versions_server ON php_installed_versions(server_id);

-- 2. PHP Extensions Tracking per Server & Version
CREATE TABLE IF NOT EXISTS php_extensions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    php_version VARCHAR(16) NOT NULL,
    name VARCHAR(64) NOT NULL, -- e.g. "curl", "mbstring", "redis", "imagick"
    package_name VARCHAR(128) NOT NULL, -- e.g. "php8.3-curl"
    version VARCHAR(32), -- module version e.g. "8.3.6"
    is_installed BOOLEAN NOT NULL DEFAULT TRUE,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    is_critical BOOLEAN NOT NULL DEFAULT FALSE, -- e.g. common, json, opcache
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (server_id, php_version, name)
);

CREATE INDEX IF NOT EXISTS idx_php_ext_lookup ON php_extensions(server_id, php_version, name);

-- 3. Isolated PHP-FPM Pools
CREATE TABLE IF NOT EXISTS php_fpm_pools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    website_id UUID REFERENCES websites(id) ON DELETE SET NULL,
    name VARCHAR(128) NOT NULL, -- e.g. "hostvra-example-com"
    php_version VARCHAR(16) NOT NULL,
    listen_socket VARCHAR(255) NOT NULL, -- e.g. "/run/php/php8.3-fpm-hostvra-example-com.sock"
    pool_user VARCHAR(64) NOT NULL DEFAULT 'www-data',
    pool_group VARCHAR(64) NOT NULL DEFAULT 'www-data',
    listen_owner VARCHAR(64) NOT NULL DEFAULT 'www-data',
    listen_group VARCHAR(64) NOT NULL DEFAULT 'www-data',
    pm_type VARCHAR(16) NOT NULL DEFAULT 'dynamic', -- 'dynamic', 'ondemand', 'static'
    pm_max_children INT NOT NULL DEFAULT 10,
    pm_start_servers INT NOT NULL DEFAULT 2,
    pm_min_spare_servers INT NOT NULL DEFAULT 2,
    pm_max_spare_servers INT NOT NULL DEFAULT 4,
    pm_max_requests INT NOT NULL DEFAULT 500,
    request_terminate_timeout INT NOT NULL DEFAULT 120,
    request_slowlog_timeout INT NOT NULL DEFAULT 10,
    slowlog_path VARCHAR(255),
    errorlog_path VARCHAR(255),
    status VARCHAR(32) NOT NULL DEFAULT 'active', -- 'active', 'stopped', 'failed'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (server_id, name)
);

CREATE INDEX IF NOT EXISTS idx_php_pools_server ON php_fpm_pools(server_id);
CREATE INDEX IF NOT EXISTS idx_php_pools_website ON php_fpm_pools(website_id);

-- 4. PHP.ini Directive Overrides (Global or Website scope)
CREATE TABLE IF NOT EXISTS php_ini_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    scope VARCHAR(16) NOT NULL DEFAULT 'global', -- 'global' or 'website'
    website_id UUID REFERENCES websites(id) ON DELETE CASCADE,
    php_version VARCHAR(16) NOT NULL,
    directive VARCHAR(128) NOT NULL, -- e.g. "memory_limit", "upload_max_filesize"
    value VARCHAR(255) NOT NULL, -- e.g. "512M", "Off", "300"
    directive_type VARCHAR(32) NOT NULL DEFAULT 'string', -- 'size', 'time', 'boolean', 'integer', 'string'
    category VARCHAR(64) NOT NULL DEFAULT 'Core', -- 'Core', 'Upload', 'Error Handling', 'Sessions', 'Security', 'OPcache'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (server_id, scope, website_id, php_version, directive)
);

CREATE INDEX IF NOT EXISTS idx_php_ini_overrides ON php_ini_overrides(server_id, php_version, scope);

-- 5. PHP Configuration Backups (Safe Rollback Transaction Model)
CREATE TABLE IF NOT EXISTS php_config_backups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    php_version VARCHAR(16) NOT NULL,
    backup_type VARCHAR(32) NOT NULL, -- 'ini', 'pool', 'master_fpm'
    file_path VARCHAR(255) NOT NULL,
    content_backup TEXT NOT NULL,
    reason VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_php_backups ON php_config_backups(server_id, php_version, created_at DESC);

-- 6. Link Websites table with custom isolated PHP-FPM pool if present
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'websites' AND column_name = 'php_fpm_pool_id'
    ) THEN
        ALTER TABLE websites ADD COLUMN php_fpm_pool_id UUID REFERENCES php_fpm_pools(id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'websites' AND column_name = 'php_settings_override'
    ) THEN
        ALTER TABLE websites ADD COLUMN php_settings_override JSONB DEFAULT '{}'::jsonb;
    END IF;
END $$;
