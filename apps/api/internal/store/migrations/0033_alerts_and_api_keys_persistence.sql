-- Migration 0033: Alerts, Incidents and API Keys Persistence

CREATE TABLE IF NOT EXISTS alert_rules (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    threshold DOUBLE PRECISION NOT NULL,
    duration_sec INT NOT NULL DEFAULT 60,
    severity VARCHAR(50) NOT NULL DEFAULT 'warning',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alert_rules_org ON alert_rules(organization_id);

CREATE TABLE IF NOT EXISTS alert_incidents (
    id UUID PRIMARY KEY,
    server_id UUID NOT NULL,
    server_name VARCHAR(255) NOT NULL,
    rule_id UUID NOT NULL,
    rule_name VARCHAR(255) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    threshold DOUBLE PRECISION NOT NULL,
    message TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_alert_incidents_started ON alert_incidents(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_incidents_status ON alert_incidents(status);

CREATE TABLE IF NOT EXISTS alert_notification_channels (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    target TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alert_channels_org ON alert_notification_channels(organization_id);

CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    user_id UUID,
    name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(50) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'manager',
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_api_keys_org ON api_keys(organization_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(key_prefix);
