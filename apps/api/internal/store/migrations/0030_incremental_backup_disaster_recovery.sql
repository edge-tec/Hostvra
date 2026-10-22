-- 0030_incremental_backup_disaster_recovery.sql
-- Phase Nine: Incremental Backup & Disaster Recovery (Restic/Borg Provider Abstraction,
-- Granular File-Level Restoration, Application-Consistent Database Snapshots & Retention Pruning)

-- 1. Backup Repositories (Restic, Borg, or Native Deduplicated Repositories)
CREATE TABLE IF NOT EXISTS backup_repositories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    destination_id UUID REFERENCES backup_destinations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL DEFAULT 'restic', -- 'restic', 'borg', 'native'
    repo_uri TEXT NOT NULL,
    encryption_algorithm VARCHAR(50) NOT NULL DEFAULT 'AES-256',
    password_encrypted TEXT NOT NULL,
    key_id VARCHAR(100),
    is_initialized BOOLEAN NOT NULL DEFAULT FALSE,
    last_checked_at TIMESTAMPTZ,
    integrity_status VARCHAR(50) NOT NULL DEFAULT 'healthy', -- 'healthy', 'checking', 'error'
    total_snapshots INT NOT NULL DEFAULT 0,
    total_size_bytes BIGINT NOT NULL DEFAULT 0,
    deduplicated_size_bytes BIGINT NOT NULL DEFAULT 0,
    compression_ratio NUMERIC(5,2) NOT NULL DEFAULT 1.0,
    deletion_protection BOOLEAN NOT NULL DEFAULT TRUE, -- protects against unauthorized/accidental deletion
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_backup_repositories_org ON backup_repositories(organization_id);
CREATE INDEX IF NOT EXISTS idx_backup_repositories_dest ON backup_repositories(destination_id);

-- 2. Incremental Backup Snapshots
CREATE TABLE IF NOT EXISTS backup_incremental_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES backup_repositories(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    snapshot_hash VARCHAR(128) NOT NULL,
    parent_snapshot_id UUID REFERENCES backup_incremental_snapshots(id) ON DELETE SET NULL,
    scope VARCHAR(50) NOT NULL DEFAULT 'full_account', -- 'website', 'database', 'mail', 'full_account'
    target_name VARCHAR(255) NOT NULL,
    paths_included JSONB NOT NULL DEFAULT '[]'::jsonb,
    paths_excluded JSONB NOT NULL DEFAULT '[]'::jsonb,
    files_new INT NOT NULL DEFAULT 0,
    files_changed INT NOT NULL DEFAULT 0,
    files_unmodified INT NOT NULL DEFAULT 0,
    total_files INT NOT NULL DEFAULT 0,
    raw_size_bytes BIGINT NOT NULL DEFAULT 0,
    stored_size_bytes BIGINT NOT NULL DEFAULT 0,
    database_snapshot_consistent BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(50) NOT NULL DEFAULT 'completed', -- 'pending', 'running', 'completed', 'failed', 'pruned'
    integrity_status VARCHAR(50) NOT NULL DEFAULT 'verified',
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_backup_inc_snapshots_repo ON backup_incremental_snapshots(repository_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_backup_inc_snapshots_org ON backup_incremental_snapshots(organization_id);
CREATE INDEX IF NOT EXISTS idx_backup_inc_snapshots_hash ON backup_incremental_snapshots(snapshot_hash);

-- 3. Retention Policies & Automated Pruning Safeguards
CREATE TABLE IF NOT EXISTS backup_retention_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES backup_repositories(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    keep_last INT NOT NULL DEFAULT 5,
    keep_hourly INT NOT NULL DEFAULT 24,
    keep_daily INT NOT NULL DEFAULT 7,
    keep_weekly INT NOT NULL DEFAULT 4,
    keep_monthly INT NOT NULL DEFAULT 12,
    keep_yearly INT NOT NULL DEFAULT 1,
    prune_schedule VARCHAR(100) NOT NULL DEFAULT '0 3 * * 0',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_pruned_at TIMESTAMPTZ,
    pruned_snapshots_count INT NOT NULL DEFAULT 0,
    bytes_reclaimed BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_backup_retention_repo ON backup_retention_policies(repository_id);
CREATE INDEX IF NOT EXISTS idx_backup_retention_org ON backup_retention_policies(organization_id);

-- 4. Granular File-Level & Full Account Restore Requests
CREATE TABLE IF NOT EXISTS backup_file_restore_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    snapshot_id UUID NOT NULL REFERENCES backup_incremental_snapshots(id) ON DELETE CASCADE,
    selected_paths JSONB NOT NULL DEFAULT '[]'::jsonb,
    target_directory TEXT NOT NULL,
    restore_mode VARCHAR(50) NOT NULL DEFAULT 'in_place', -- 'in_place', 'alternate_dir'
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed'
    files_restored INT NOT NULL DEFAULT 0,
    bytes_restored BIGINT NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_backup_restore_req_snapshot ON backup_file_restore_requests(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_backup_restore_req_org ON backup_file_restore_requests(organization_id, created_at DESC);
