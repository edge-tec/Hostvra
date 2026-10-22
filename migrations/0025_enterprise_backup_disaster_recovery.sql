-- 0025_enterprise_backup_disaster_recovery.sql
-- Enterprise Backup & Disaster Recovery: Multi-Cloud Destinations, Integrity Verification, Preflight Safety & Restore Jobs

-- 1. Backup Destinations (AWS S3, Wasabi, GCS, Cloudflare R2, Backblaze B2, DigitalOcean Spaces, MinIO, SFTP, Local)
CREATE TABLE IF NOT EXISTS backup_destinations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 's3', 'wasabi', 'gcs', 'r2', 'b2', 'spaces', 'minio', 'sftp', 'local'
    endpoint VARCHAR(255),
    region VARCHAR(100),
    bucket VARCHAR(255),
    access_key TEXT,
    secret_key TEXT,
    prefix VARCHAR(255) NOT NULL DEFAULT 'hostvra-backups',
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_backup_destinations_org ON backup_destinations(organization_id);
CREATE INDEX IF NOT EXISTS idx_backup_destinations_type ON backup_destinations(type);

-- 2. Backup Snapshots & Automated Integrity Verification
CREATE TABLE IF NOT EXISTS backup_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    destination_id UUID REFERENCES backup_destinations(id) ON DELETE SET NULL,
    type VARCHAR(50) NOT NULL, -- 'website', 'database', 'full_config', 'full_account'
    target_name VARCHAR(255) NOT NULL,
    storage VARCHAR(50) NOT NULL DEFAULT 'local', -- 'local', 's3', 'wasabi', 'gcs', 'r2', 'b2', 'spaces', 'minio', 'sftp'
    size_bytes BIGINT NOT NULL DEFAULT 0,
    sha256_hash VARCHAR(64),
    item_count INT NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'completed', -- 'in_progress', 'completed', 'failed'
    file_name VARCHAR(255) NOT NULL,
    remote_key TEXT,
    verified_at TIMESTAMPTZ,
    integrity_status VARCHAR(50) NOT NULL DEFAULT 'unverified', -- 'unverified', 'verified', 'corrupted'
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_backup_snapshots_org_created ON backup_snapshots(organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_backup_snapshots_target ON backup_snapshots(target_name);
CREATE INDEX IF NOT EXISTS idx_backup_snapshots_status ON backup_snapshots(status);
CREATE INDEX IF NOT EXISTS idx_backup_snapshots_integrity ON backup_snapshots(integrity_status);

-- 3. Backup Schedules
CREATE TABLE IF NOT EXISTS backup_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    destination_id UUID REFERENCES backup_destinations(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    scope VARCHAR(50) NOT NULL DEFAULT 'website', -- 'website', 'database', 'full_config', 'full_account'
    target_name VARCHAR(255) NOT NULL,
    frequency VARCHAR(50) NOT NULL DEFAULT 'daily', -- 'daily', 'weekly', 'monthly', 'cron'
    cron_expr VARCHAR(100),
    retention INT NOT NULL DEFAULT 7,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_backup_schedules_org ON backup_schedules(organization_id);

-- 4. Disaster Recovery & Preflight Restore Jobs
CREATE TABLE IF NOT EXISTS backup_restore_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    snapshot_id UUID REFERENCES backup_snapshots(id) ON DELETE CASCADE,
    target_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'preflight', 'restoring', 'completed', 'failed'
    preflight_ok BOOLEAN NOT NULL DEFAULT FALSE,
    disk_space_required BIGINT NOT NULL DEFAULT 0,
    disk_space_available BIGINT NOT NULL DEFAULT 0,
    current_step VARCHAR(255),
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_backup_restore_jobs_org ON backup_restore_jobs(organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_backup_restore_jobs_snapshot ON backup_restore_jobs(snapshot_id);
