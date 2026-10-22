-- Migration 0027: Developer and Application Management (Node.js & Python, Git-Based CI/CD, WordPress Staging)

-- 1. Managed Applications (Node.js & Python Process Supervisor)
CREATE TABLE IF NOT EXISTS managed_apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    website_id UUID NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    app_type VARCHAR(32) NOT NULL, -- 'nodejs', 'python'
    runtime_version VARCHAR(32) NOT NULL, -- '20', '18', '22', '3.11', '3.12'
    app_root VARCHAR(512) NOT NULL,
    entrypoint VARCHAR(256) NOT NULL DEFAULT 'index.js',
    start_command VARCHAR(512) NOT NULL DEFAULT 'npm start',
    port INT NOT NULL DEFAULT 3000,
    system_user VARCHAR(64) NOT NULL,
    env_vars JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'stopped', -- 'running', 'stopped', 'failed'
    restart_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_managed_apps_website ON managed_apps(website_id);
CREATE INDEX IF NOT EXISTS idx_managed_apps_org ON managed_apps(organization_id);

-- 2. Git Repositories (CI/CD Configuration)
CREATE TABLE IF NOT EXISTS git_repositories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    website_id UUID NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL DEFAULT 'github', -- 'github', 'gitlab', 'custom'
    repo_url VARCHAR(512) NOT NULL,
    branch VARCHAR(128) NOT NULL DEFAULT 'main',
    webhook_secret VARCHAR(128) NOT NULL,
    deploy_key_pub TEXT NOT NULL DEFAULT '',
    deploy_key_priv TEXT NOT NULL DEFAULT '',
    auto_deploy BOOLEAN NOT NULL DEFAULT TRUE,
    build_command VARCHAR(512) NOT NULL DEFAULT '',
    deploy_path VARCHAR(512) NOT NULL,
    release_keep_count INT NOT NULL DEFAULT 5,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_git_repositories_website ON git_repositories(website_id);

-- 3. Git Deployments (Atomic Releases History)
CREATE TABLE IF NOT EXISTS git_deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES git_repositories(id) ON DELETE CASCADE,
    website_id UUID NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    commit_hash VARCHAR(64) NOT NULL DEFAULT '',
    commit_message TEXT NOT NULL DEFAULT '',
    author VARCHAR(128) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'queued', -- 'queued', 'cloning', 'building', 'success', 'failed', 'rolled_back'
    build_logs TEXT NOT NULL DEFAULT '',
    release_path VARCHAR(512) NOT NULL DEFAULT '',
    duration_seconds INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_git_deployments_repo ON git_deployments(repository_id);
CREATE INDEX IF NOT EXISTS idx_git_deployments_website ON git_deployments(website_id);
CREATE INDEX IF NOT EXISTS idx_git_deployments_created ON git_deployments(created_at DESC);

-- 4. WordPress Staging Environments
CREATE TABLE IF NOT EXISTS wordpress_stagings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    website_id UUID NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    staging_domain VARCHAR(255) NOT NULL,
    staging_doc_root VARCHAR(512) NOT NULL,
    staging_db_name VARCHAR(64) NOT NULL,
    staging_db_user VARCHAR(64) NOT NULL,
    auth_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    auth_user VARCHAR(64) NOT NULL DEFAULT 'staging',
    auth_password VARCHAR(128) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active', -- 'active', 'syncing', 'deleted'
    last_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_wordpress_stagings_website ON wordpress_stagings(website_id);
CREATE INDEX IF NOT EXISTS idx_wordpress_stagings_domain ON wordpress_stagings(staging_domain);
