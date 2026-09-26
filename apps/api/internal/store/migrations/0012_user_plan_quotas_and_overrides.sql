-- ============================================================================
-- Migration 0012: User Plan Quotas, Feature Permissions & Admin Overrides
-- Production PostgreSQL DDL for Per-User Resource Limit & Permission Overrides
-- ============================================================================

-- 1. Create user_plan_overrides table for per-user custom quotas and permission toggles
CREATE TABLE IF NOT EXISTS user_plan_overrides (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    plan_id UUID REFERENCES hosting_plans(id) ON DELETE SET NULL,
    max_websites INT,
    max_databases INT,
    max_mailboxes INT,
    max_ftp INT,
    max_cron INT,
    max_subdomains INT,
    disk_space_mb BIGINT,
    bandwidth_mb BIGINT,
    permission_terminal BOOLEAN,
    permission_backups BOOLEAN,
    permission_dns BOOLEAN,
    permission_ssl BOOLEAN,
    permission_file_manager BOOLEAN,
    permission_cron BOOLEAN,
    permission_apps BOOLEAN,
    permission_php_selector BOOLEAN,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_plan_overrides_plan ON user_plan_overrides(plan_id);

-- 2. Ensure default Starter Cloud plan allows free trial for new customer self-service registration
UPDATE hosting_plans
SET trial_allowed = TRUE, trial_days = 14
WHERE slug = 'starter-cloud' AND trial_allowed IS FALSE;
