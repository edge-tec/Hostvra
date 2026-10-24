-- Migration 0034: Enterprise WHM Hosting Enhancement
-- Per-client resource limits, contact info, quota status, impersonation sessions, and background jobs.

-- 1. Extend hosting_accounts with additional limits and metadata
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS contact_email VARCHAR(255);
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS contact_phone VARCHAR(50);
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS notes TEXT;
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS subdomains_limit INT NOT NULL DEFAULT 5;
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS ftp_users_limit INT NOT NULL DEFAULT 5;
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS cron_jobs_limit INT NOT NULL DEFAULT 5;
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS max_email_send_rate INT NOT NULL DEFAULT 100;
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS quota_enforced BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS quota_status VARCHAR(50) NOT NULL DEFAULT 'synced';
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS last_sync_at TIMESTAMPTZ DEFAULT NOW();
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS target_server_id UUID REFERENCES servers(id) ON DELETE SET NULL;
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS migration_status VARCHAR(50) DEFAULT 'none';
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS migration_log TEXT;
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

-- 2. Audit and Active Impersonation Sessions Registry
CREATE TABLE IF NOT EXISTS hosting_impersonation_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES hosting_accounts(id) ON DELETE CASCADE,
    session_token_hash VARCHAR(255) NOT NULL UNIQUE,
    reason TEXT,
    ip_address VARCHAR(100),
    user_agent TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_impersonation_admin ON hosting_impersonation_sessions(admin_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_target ON hosting_impersonation_sessions(target_user_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_account ON hosting_impersonation_sessions(account_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_token ON hosting_impersonation_sessions(session_token_hash);

-- 3. Asynchronous WHM Provisioning & Migration Jobs
CREATE TABLE IF NOT EXISTS hosting_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    account_id UUID REFERENCES hosting_accounts(id) ON DELETE CASCADE,
    job_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    progress_percent INT NOT NULL DEFAULT 0,
    payload JSONB,
    result JSONB,
    error_message TEXT,
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_hosting_jobs_org ON hosting_jobs(organization_id);
CREATE INDEX IF NOT EXISTS idx_hosting_jobs_account ON hosting_jobs(account_id);
CREATE INDEX IF NOT EXISTS idx_hosting_jobs_status ON hosting_jobs(status);
