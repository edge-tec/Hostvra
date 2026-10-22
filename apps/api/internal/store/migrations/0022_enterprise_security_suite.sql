-- ============================================================================
-- 0022_enterprise_security_suite.sql
-- Enterprise Security Suite: TOTP 2FA Recovery Codes, Malware Scans & Quarantine
-- ============================================================================

-- 1. Two-Factor Authentication Recovery Codes Table
CREATE TABLE IF NOT EXISTS user_two_factor_recovery_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(64) NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_u2f_recovery_user ON user_two_factor_recovery_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_u2f_recovery_lookup ON user_two_factor_recovery_codes(user_id, code_hash);

-- 2. Security & Malware Scans History Table
CREATE TABLE IF NOT EXISTS security_scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    website_id UUID REFERENCES websites(id) ON DELETE SET NULL,
    scan_type VARCHAR(50) NOT NULL DEFAULT 'heuristic',
    target_path TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    total_files INT NOT NULL DEFAULT 0,
    infected_files INT NOT NULL DEFAULT 0,
    findings JSONB NOT NULL DEFAULT '[]'::jsonb,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_sec_scans_org ON security_scans(organization_id);
CREATE INDEX IF NOT EXISTS idx_sec_scans_created ON security_scans(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sec_scans_website ON security_scans(website_id);

-- 3. Security Quarantine Table
CREATE TABLE IF NOT EXISTS security_quarantine (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_id UUID REFERENCES security_scans(id) ON DELETE SET NULL,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    website_id UUID REFERENCES websites(id) ON DELETE SET NULL,
    original_path TEXT NOT NULL,
    quarantine_filename VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    file_mode VARCHAR(20) NOT NULL DEFAULT '0644',
    file_owner VARCHAR(50) NOT NULL DEFAULT 'root',
    file_group VARCHAR(50) NOT NULL DEFAULT 'root',
    sha256_hash VARCHAR(64) NOT NULL,
    threat_name TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'quarantined',
    restored_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    quarantined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    quarantined_by UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_sec_quarantine_org ON security_quarantine(organization_id);
CREATE INDEX IF NOT EXISTS idx_sec_quarantine_status ON security_quarantine(status);
CREATE INDEX IF NOT EXISTS idx_sec_quarantine_original ON security_quarantine(original_path);
