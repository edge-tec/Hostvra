-- Migration 0026: Advanced Resource Quotas, Linux Cgroups v2 & Systemd Process Isolation Engine

CREATE TABLE IF NOT EXISTS website_resource_quotas (
    website_id UUID PRIMARY KEY REFERENCES websites(id) ON DELETE CASCADE,
    username VARCHAR(64) NOT NULL,
    memory_limit_mb INT NOT NULL DEFAULT 512,
    memory_high_mb INT NOT NULL DEFAULT 460,
    cpu_quota_percent INT NOT NULL DEFAULT 100,
    cpu_weight INT NOT NULL DEFAULT 100,
    tasks_max INT NOT NULL DEFAULT 100,
    io_read_mbps INT NOT NULL DEFAULT 50,
    io_write_mbps INT NOT NULL DEFAULT 30,
    io_read_iops INT NOT NULL DEFAULT 1000,
    io_write_iops INT NOT NULL DEFAULT 500,
    oom_policy VARCHAR(16) NOT NULL DEFAULT 'kill',
    open_basedir_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_website_resource_quotas_username ON website_resource_quotas(username);

CREATE TABLE IF NOT EXISTS resource_quota_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    website_id UUID NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    username VARCHAR(64) NOT NULL,
    event_type VARCHAR(32) NOT NULL,
    severity VARCHAR(16) NOT NULL DEFAULT 'warning',
    details TEXT NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    threshold_value DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_quota_events_website_id ON resource_quota_events(website_id);
CREATE INDEX IF NOT EXISTS idx_quota_events_created_at ON resource_quota_events(created_at DESC);
