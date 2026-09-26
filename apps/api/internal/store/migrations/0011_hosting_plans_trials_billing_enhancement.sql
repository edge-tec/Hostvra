-- ============================================================================
-- Migration 0011: Hosting Plans, Free Trials & Billing Enhancements
-- Production PostgreSQL DDL for Free Trial Engine, Resource Limits & Packages
-- ============================================================================

-- 1. Enhance hosting_plans with trial and resource limits
ALTER TABLE hosting_plans ADD COLUMN IF NOT EXISTS setup_fee NUMERIC(12, 2) NOT NULL DEFAULT 0.00;
ALTER TABLE hosting_plans ADD COLUMN IF NOT EXISTS trial_allowed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE hosting_plans ADD COLUMN IF NOT EXISTS trial_days INT NOT NULL DEFAULT 14;
ALTER TABLE hosting_plans ADD COLUMN IF NOT EXISTS is_featured BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE hosting_plans ADD COLUMN IF NOT EXISTS cpu_limit NUMERIC(4, 2) NOT NULL DEFAULT 1.0;
ALTER TABLE hosting_plans ADD COLUMN IF NOT EXISTS ram_limit_mb INT NOT NULL DEFAULT 1024;
ALTER TABLE hosting_plans ADD COLUMN IF NOT EXISTS max_cron INT NOT NULL DEFAULT 5;
ALTER TABLE hosting_plans ADD COLUMN IF NOT EXISTS max_subdomains INT NOT NULL DEFAULT 10;

-- 2. Enhance subscriptions with trial lifecycle timestamps
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS trial_started_at TIMESTAMPTZ;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_subscriptions_trial_ends ON subscriptions(trial_ends_at) WHERE status = 'trial';

-- 3. Create trial_settings singleton table
CREATE TABLE IF NOT EXISTS trial_settings (
    id INT PRIMARY KEY DEFAULT 1,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    default_days INT NOT NULL DEFAULT 14,
    require_payment_method BOOLEAN NOT NULL DEFAULT FALSE,
    one_trial_per_customer BOOLEAN NOT NULL DEFAULT TRUE,
    auto_suspend_on_expiry BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT trial_settings_singleton CHECK (id = 1)
);

INSERT INTO trial_settings (id, enabled, default_days, require_payment_method, one_trial_per_customer, auto_suspend_on_expiry, updated_at)
VALUES (1, TRUE, 14, FALSE, TRUE, TRUE, NOW())
ON CONFLICT (id) DO NOTHING;
