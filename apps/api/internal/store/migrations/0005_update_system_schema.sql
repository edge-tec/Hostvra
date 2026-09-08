-- Migration: 0005_update_system_schema.sql
-- Description: Production-grade schema for Hostvra Live Update System

-- 1. System Versions Table (Tracks active installed versions across nodes & control plane)
CREATE TABLE IF NOT EXISTS system_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    component VARCHAR(50) NOT NULL, -- 'api', 'agent', 'web', 'database'
    node_id UUID REFERENCES servers(id) ON DELETE CASCADE, -- NULL for control plane
    current_version VARCHAR(50) NOT NULL,
    git_commit VARCHAR(100),
    build_date TIMESTAMP WITH TIME ZONE,
    channel VARCHAR(30) NOT NULL DEFAULT 'stable', -- 'stable', 'beta', 'nightly'
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_component_node UNIQUE (component, node_id)
);

-- 2. Update Releases Catalog (Signed metadata of releases fetched from update server)
CREATE TABLE IF NOT EXISTS update_releases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version VARCHAR(50) NOT NULL,
    channel VARCHAR(30) NOT NULL DEFAULT 'stable',
    component VARCHAR(50) NOT NULL, -- 'api', 'agent', 'web', 'bundle'
    release_notes TEXT,
    min_supported_version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    package_url TEXT NOT NULL,
    sha256_checksum VARCHAR(64) NOT NULL,
    ed25519_signature TEXT NOT NULL,
    package_size_bytes BIGINT NOT NULL DEFAULT 0,
    os_compatibility VARCHAR(50)[] DEFAULT ARRAY['ubuntu-22.04', 'ubuntu-24.04', 'debian-12'],
    arch_compatibility VARCHAR(20)[] DEFAULT ARRAY['amd64', 'arm64'],
    requires_db_migration BOOLEAN NOT NULL DEFAULT false,
    requires_reboot BOOLEAN NOT NULL DEFAULT false,
    is_revoked BOOLEAN NOT NULL DEFAULT false,
    released_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_version_channel_component UNIQUE (version, channel, component)
);

-- 3. Update Jobs (Persistent state machine for background updates)
CREATE TABLE IF NOT EXISTS update_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_version VARCHAR(50) NOT NULL,
    channel VARCHAR(30) NOT NULL DEFAULT 'stable',
    component VARCHAR(50) NOT NULL DEFAULT 'bundle',
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    -- PENDING, PRECHECKING, BACKING_UP, DOWNLOADING, VERIFYING, PREPARING, MIGRATING, INSTALLING, ACTIVATING, HEALTH_CHECKING, COMPLETED, FAILED, ROLLING_BACK, ROLLED_BACK
    previous_version VARCHAR(50) NOT NULL,
    node_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    backup_snapshot_id UUID,
    initiated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    error_message TEXT,
    error_details JSONB,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. Update Steps (Fine-grained tracking of each step in the state machine)
CREATE TABLE IF NOT EXISTS update_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES update_jobs(id) ON DELETE CASCADE,
    step_name VARCHAR(50) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING', -- PENDING, RUNNING, COMPLETED, FAILED, SKIPPED, ROLLED_BACK
    details TEXT,
    logs TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER DEFAULT 0
);

-- 5. Database Migrations Framework Table (Deterministic migration lock and history)
CREATE TABLE IF NOT EXISTS database_migrations (
    id SERIAL PRIMARY KEY,
    version VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    checksum VARCHAR(64) NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    execution_time_ms INTEGER NOT NULL,
    rollback_sql TEXT,
    is_success BOOLEAN NOT NULL DEFAULT true
);

-- Indexes for lightning fast lookups and state queries
CREATE INDEX IF NOT EXISTS idx_update_releases_channel_version ON update_releases(channel, version);
CREATE INDEX IF NOT EXISTS idx_update_jobs_status ON update_jobs(status);
CREATE INDEX IF NOT EXISTS idx_update_steps_job_id ON update_steps(job_id);
