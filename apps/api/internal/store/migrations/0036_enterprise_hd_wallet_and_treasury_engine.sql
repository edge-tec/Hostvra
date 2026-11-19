-- 0036_enterprise_hd_wallet_and_treasury_engine.sql
-- Phase 5: Enterprise HD Wallet, Unique User Deposit Addresses & Automated Treasury Engine
-- Backward-compatible migration: never stores plaintext seeds or private keys.

CREATE TABLE IF NOT EXISTS hd_wallet_accounts (
    id UUID PRIMARY KEY,
    account_name VARCHAR(100) NOT NULL,
    wallet_type VARCHAR(50) NOT NULL DEFAULT 'watch_only_xpub', -- watch_only_xpub, isolated_signer
    network_id VARCHAR(50) NOT NULL REFERENCES crypto_networks(id) ON DELETE CASCADE,
    derivation_scheme VARCHAR(30) NOT NULL,                     -- BIP-44, BIP-84, SLIP-0010
    derivation_path VARCHAR(100) NOT NULL,                      -- e.g. m/44'/60'/0'/0
    public_derivation_reference TEXT NOT NULL DEFAULT '',       -- Watch-only xpub (NEVER xprv/seed)
    next_derivation_index INT NOT NULL DEFAULT 0,
    status VARCHAR(30) NOT NULL DEFAULT 'active',               -- active, rotated, paused
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (network_id, derivation_path)
);

CREATE TABLE IF NOT EXISTS hd_wallet_addresses (
    id UUID PRIMARY KEY,
    account_id UUID REFERENCES hd_wallet_accounts(id) ON DELETE SET NULL,
    user_id UUID NOT NULL,
    network_id VARCHAR(50) NOT NULL REFERENCES crypto_networks(id) ON DELETE CASCADE,
    asset_id VARCHAR(50) NOT NULL REFERENCES crypto_currencies(id) ON DELETE CASCADE,
    derivation_index INT NOT NULL,
    derivation_path VARCHAR(120) NOT NULL,
    deposit_address VARCHAR(255) NOT NULL,
    address_status VARCHAR(30) NOT NULL DEFAULT 'active',       -- active, funded, sweeping, swept, retired
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (network_id, derivation_index),
    UNIQUE (network_id, deposit_address)
);

CREATE TABLE IF NOT EXISTS deposit_address_allocations (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    invoice_id UUID NOT NULL REFERENCES crypto_payment_invoices(id) ON DELETE CASCADE,
    address_id UUID NOT NULL REFERENCES hd_wallet_addresses(id) ON DELETE CASCADE,
    network_id VARCHAR(50) NOT NULL,
    asset_id VARCHAR(50) NOT NULL,
    allocation_status VARCHAR(30) NOT NULL DEFAULT 'allocated', -- allocated, paid, expired, swept
    allocated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    UNIQUE (invoice_id)
);

CREATE TABLE IF NOT EXISTS treasury_sweep_jobs (
    id UUID PRIMARY KEY,
    invoice_id UUID REFERENCES crypto_payment_invoices(id) ON DELETE SET NULL,
    source_address_id UUID REFERENCES hd_wallet_addresses(id) ON DELETE SET NULL,
    source_address VARCHAR(255) NOT NULL,
    derivation_index INT NOT NULL DEFAULT 0,
    destination_wallet_id UUID REFERENCES crypto_treasury_wallets(id) ON DELETE SET NULL,
    destination_address VARCHAR(255) NOT NULL,
    asset_id VARCHAR(50) NOT NULL,
    network_id VARCHAR(50) NOT NULL,
    requested_amount NUMERIC(32,18) NOT NULL DEFAULT 0,
    actual_amount NUMERIC(32,18) NOT NULL DEFAULT 0,
    amount_usd NUMERIC(16,4) NOT NULL DEFAULT 0,
    estimated_network_fee NUMERIC(32,18) NOT NULL DEFAULT 0,
    actual_network_fee NUMERIC(32,18) NOT NULL DEFAULT 0,
    transaction_hash VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(30) NOT NULL DEFAULT 'queued',               -- queued, signing, broadcast, confirming, completed, failed, paused
    idempotency_key VARCHAR(150) NOT NULL UNIQUE,
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 5,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS treasury_sweep_logs (
    id UUID PRIMARY KEY,
    sweep_job_id UUID NOT NULL REFERENCES treasury_sweep_jobs(id) ON DELETE CASCADE,
    event_type VARCHAR(60) NOT NULL,
    event_details TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wallet_derivation_audit_logs (
    id UUID PRIMARY KEY,
    account_id UUID REFERENCES hd_wallet_accounts(id) ON DELETE SET NULL,
    address_id UUID REFERENCES hd_wallet_addresses(id) ON DELETE SET NULL,
    event_type VARCHAR(60) NOT NULL,
    actor_id VARCHAR(100) NOT NULL DEFAULT 'system',
    event_details TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for fast lookup, monitoring & reconciliation
CREATE INDEX IF NOT EXISTS idx_hd_wallet_addr_user_net ON hd_wallet_addresses(user_id, network_id);
CREATE INDEX IF NOT EXISTS idx_hd_wallet_addr_status ON hd_wallet_addresses(address_status);
CREATE INDEX IF NOT EXISTS idx_deposit_alloc_user ON deposit_address_allocations(user_id, network_id);
CREATE INDEX IF NOT EXISTS idx_sweep_jobs_status ON treasury_sweep_jobs(status, created_at);
CREATE INDEX IF NOT EXISTS idx_sweep_jobs_network ON treasury_sweep_jobs(network_id, status);
CREATE INDEX IF NOT EXISTS idx_sweep_logs_job ON treasury_sweep_logs(sweep_job_id, created_at);
CREATE INDEX IF NOT EXISTS idx_wallet_audit_time ON wallet_derivation_audit_logs(created_at DESC);

-- Ensure Polygon POL is seeded alongside all 9 enterprise assets
INSERT INTO crypto_currencies (id, network_id, symbol, name, is_token, contract_address, decimals, min_confirmations, min_amount_usd, icon, is_active)
VALUES
('POL', 'polygon', 'POL', 'Polygon (POL/MATIC)', FALSE, '', 18, 5, 1.0000, 'pol', TRUE)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    is_active = TRUE;

-- Seed default Phase 5 Treasury Engine settings
INSERT INTO crypto_payment_settings (key, value, description)
VALUES
('hd_wallet_enabled', 'true', 'Generate unique deterministic HD deposit addresses per user/invoice'),
('auto_sweep_enabled', 'true', 'Automatically sweep confirmed HD deposit balances to Trust Wallet treasury'),
('emergency_sweep_pause', 'false', 'Emergency kill-switch halting all automated and manual treasury sweeps'),
('min_sweep_usd', '0.5', 'Minimum USD balance threshold before sweeping a deposit address'),
('max_single_sweep_usd', '50000.0', 'Maximum USD value permitted in a single automated sweep transaction'),
('max_daily_sweep_usd', '250000.0', 'Maximum cumulative USD value permitted to sweep per 24 hours')
ON CONFLICT (key) DO NOTHING;
