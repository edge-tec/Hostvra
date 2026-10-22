-- 0024_whmcs_blesta_reseller_provisioning.sql
-- Multi-Tenancy Reseller Engine, WHMCS/Blesta Webhook Provisioning & Activity Audits

-- 1. Reseller Profiles & White-Label Branding
CREATE TABLE IF NOT EXISTS reseller_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    max_accounts INT NOT NULL DEFAULT 10,
    allocated_disk_mb BIGINT NOT NULL DEFAULT 51200,
    allocated_bandwidth_mb BIGINT NOT NULL DEFAULT 512000,
    allow_overselling BOOLEAN NOT NULL DEFAULT FALSE,
    brand_name VARCHAR(255) NOT NULL DEFAULT 'Hostvra',
    brand_logo_url TEXT,
    custom_nameservers JSONB NOT NULL DEFAULT '["ns1.hostvra.com", "ns2.hostvra.com"]'::jsonb,
    custom_email_sender VARCHAR(255) NOT NULL DEFAULT 'noreply@hostvra.com',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_reseller_user UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_reseller_profiles_org ON reseller_profiles(organization_id);

-- 2. Link Sub-Accounts to Parent Reseller
ALTER TABLE users ADD COLUMN IF NOT EXISTS reseller_id UUID REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_users_reseller ON users(reseller_id);

-- 3. Provisioning Integration Adapters (WHMCS, Blesta)
CREATE TABLE IF NOT EXISTS provisioning_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- 'whmcs', 'blesta', 'custom'
    name VARCHAR(255) NOT NULL,
    secret_key_hash VARCHAR(255) NOT NULL,
    default_server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    default_plan_id UUID REFERENCES hosting_plans(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_provisioning_integrations_org ON provisioning_integrations(organization_id);

-- 4. Provisioning Webhook Activity Logs & Audit History
CREATE TABLE IF NOT EXISTS provisioning_webhook_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    integration_id UUID REFERENCES provisioning_integrations(id) ON DELETE SET NULL,
    provider VARCHAR(50) NOT NULL,
    action VARCHAR(100) NOT NULL,
    username VARCHAR(100),
    domain VARCHAR(255),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    http_status INT NOT NULL DEFAULT 200,
    response_body TEXT,
    error_message TEXT,
    ip_address VARCHAR(45),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhook_logs_org_created ON provisioning_webhook_logs(organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_status ON provisioning_webhook_logs(status);
