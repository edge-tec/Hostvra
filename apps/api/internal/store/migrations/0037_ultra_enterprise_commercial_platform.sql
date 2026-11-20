-- 0037_ultra_enterprise_commercial_platform.sql
-- Ultra-Enterprise Commercial Hosting Platform:
-- 1. Coupon & Promotion Engine
-- 2. Affiliate & Referral System
-- 3. Customer Credit Wallet (Immutable Double-Entry Ledger) & Auto-Pay
-- 4. Automated Billing Dunning & Subscription Lifecycle Tracking

CREATE TABLE IF NOT EXISTS coupons (
    id UUID PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    discount_type VARCHAR(20) NOT NULL DEFAULT 'percentage', -- percentage, fixed
    discount_value NUMERIC(16,4) NOT NULL,
    max_uses INT NOT NULL DEFAULT 0,                         -- 0 = unlimited
    current_uses INT NOT NULL DEFAULT 0,
    max_uses_per_user INT NOT NULL DEFAULT 1,
    min_order_usd NUMERIC(16,4) NOT NULL DEFAULT 0,
    first_order_only BOOLEAN NOT NULL DEFAULT FALSE,
    recurring_discount BOOLEAN NOT NULL DEFAULT FALSE,
    applicable_plan_ids TEXT NOT NULL DEFAULT '',            -- comma-separated or empty for all
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS coupon_redemptions (
    id UUID PRIMARY KEY,
    coupon_id UUID NOT NULL REFERENCES coupons(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    invoice_id UUID NOT NULL,
    discount_usd NUMERIC(16,4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (coupon_id, invoice_id)
);

CREATE TABLE IF NOT EXISTS affiliate_accounts (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE,
    referral_code VARCHAR(40) NOT NULL UNIQUE,
    commission_rate_pct NUMERIC(6,2) NOT NULL DEFAULT 15.00,
    pending_balance_usd NUMERIC(16,4) NOT NULL DEFAULT 0,
    approved_balance_usd NUMERIC(16,4) NOT NULL DEFAULT 0,
    total_paid_usd NUMERIC(16,4) NOT NULL DEFAULT 0,
    total_referrals INT NOT NULL DEFAULT 0,
    status VARCHAR(30) NOT NULL DEFAULT 'active',            -- active, suspended
    payout_method VARCHAR(50) NOT NULL DEFAULT 'credit_wallet',
    payout_details TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS affiliate_referrals (
    id UUID PRIMARY KEY,
    affiliate_id UUID NOT NULL REFERENCES affiliate_accounts(id) ON DELETE CASCADE,
    referred_user_id UUID NOT NULL UNIQUE,
    attribution_model VARCHAR(20) NOT NULL DEFAULT 'first_click',
    status VARCHAR(30) NOT NULL DEFAULT 'signed_up',         -- signed_up, converted, flagged_fraud
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS affiliate_commissions (
    id UUID PRIMARY KEY,
    affiliate_id UUID NOT NULL REFERENCES affiliate_accounts(id) ON DELETE CASCADE,
    referred_user_id UUID NOT NULL,
    invoice_id UUID NOT NULL UNIQUE,
    order_amount_usd NUMERIC(16,4) NOT NULL,
    commission_usd NUMERIC(16,4) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',           -- pending, approved, reversed, paid
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS affiliate_withdrawals (
    id UUID PRIMARY KEY,
    affiliate_id UUID NOT NULL REFERENCES affiliate_accounts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    amount_usd NUMERIC(16,4) NOT NULL,
    payout_method VARCHAR(50) NOT NULL,
    payout_details TEXT NOT NULL DEFAULT '',
    status VARCHAR(30) NOT NULL DEFAULT 'pending',           -- pending, approved, rejected, paid
    admin_notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS customer_wallets (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE,
    balance_usd NUMERIC(16,4) NOT NULL DEFAULT 0,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    auto_pay_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wallet_ledger_entries (
    id UUID PRIMARY KEY,
    wallet_id UUID NOT NULL REFERENCES customer_wallets(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    entry_type VARCHAR(40) NOT NULL,                         -- topup, invoice_payment, auto_renewal, refund, affiliate_payout, admin_adjustment
    direction VARCHAR(10) NOT NULL,                          -- credit, debit
    amount_usd NUMERIC(16,4) NOT NULL,
    balance_before_usd NUMERIC(16,4) NOT NULL,
    balance_after_usd NUMERIC(16,4) NOT NULL,
    reference_id VARCHAR(120) NOT NULL DEFAULT '',
    idempotency_key VARCHAR(160) NOT NULL UNIQUE,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS billing_dunning_events (
    id UUID PRIMARY KEY,
    subscription_id UUID NOT NULL,
    user_id UUID NOT NULL,
    invoice_id UUID,
    cycle_key VARCHAR(40) NOT NULL,                          -- e.g. 2026-12
    notice_stage VARCHAR(30) NOT NULL,                       -- renewal_7d, renewal_3d, renewal_1d, auto_pay_success, grace_warning, suspended
    details TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (subscription_id, cycle_key, notice_stage)
);

CREATE INDEX IF NOT EXISTS idx_coupons_code ON coupons(UPPER(code));
CREATE INDEX IF NOT EXISTS idx_coupon_redemptions_user ON coupon_redemptions(coupon_id, user_id);
CREATE INDEX IF NOT EXISTS idx_affiliate_ref_code ON affiliate_accounts(UPPER(referral_code));
CREATE INDEX IF NOT EXISTS idx_wallet_ledger_user ON wallet_ledger_entries(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_dunning_sub ON billing_dunning_events(subscription_id, created_at DESC);

-- Seed default welcome coupon
INSERT INTO coupons (id, code, description, discount_type, discount_value, max_uses, max_uses_per_user, min_order_usd, first_order_only, is_active)
VALUES ('11111111-2222-3333-4444-555555555555', 'WELCOME20', '20% Off First Hosting Order', 'percentage', 20.0000, 1000, 1, 1.0000, TRUE, TRUE)
ON CONFLICT (code) DO NOTHING;

-- Seed controlled EVM Gas Station settings
INSERT INTO crypto_payment_settings (key, value, description)
VALUES
('gas_station_enabled', 'false', 'Enable controlled native gas top-up from HD index 0 for ERC20/BEP20 sweeps'),
('max_gas_funding_native', '0.005', 'Hard safety cap per automated gas funding transaction in native coin (BNB/ETH)')
ON CONFLICT (key) DO NOTHING;
