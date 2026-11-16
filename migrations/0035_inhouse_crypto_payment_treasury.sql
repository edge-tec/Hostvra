-- 0035_inhouse_crypto_payment_treasury.sql
-- Production-Grade In-House Crypto Payment & Treasury System (Zero 3rd-Party Fees)

CREATE TABLE IF NOT EXISTS crypto_networks (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    chain_id BIGINT,
    rpc_url TEXT NOT NULL,
    fallback_rpc_url TEXT,
    explorer_tx_url TEXT NOT NULL,
    explorer_address_url TEXT NOT NULL,
    native_currency VARCHAR(20) NOT NULL,
    min_confirmations INT NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS crypto_currencies (
    id VARCHAR(50) PRIMARY KEY,
    network_id VARCHAR(50) NOT NULL REFERENCES crypto_networks(id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    is_token BOOLEAN NOT NULL DEFAULT FALSE,
    contract_address VARCHAR(255) NOT NULL DEFAULT '',
    decimals INT NOT NULL DEFAULT 18,
    min_confirmations INT NOT NULL DEFAULT 1,
    min_amount_usd NUMERIC(16,4) NOT NULL DEFAULT 1.0000,
    network_fee_buffer NUMERIC(16,4) NOT NULL DEFAULT 0.0000,
    icon VARCHAR(50) NOT NULL DEFAULT 'generic',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS crypto_treasury_wallets (
    id UUID PRIMARY KEY,
    network_id VARCHAR(50) NOT NULL REFERENCES crypto_networks(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    wallet_type VARCHAR(50) NOT NULL DEFAULT 'cold_storage',
    address VARCHAR(255) NOT NULL,
    xpub TEXT,
    encrypted_private_key TEXT,
    balance NUMERIC(32,18) NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    auto_sweep_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    sweep_threshold_usd NUMERIC(16,4) NOT NULL DEFAULT 50.0000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS crypto_deposit_addresses (
    id UUID PRIMARY KEY,
    network_id VARCHAR(50) NOT NULL REFERENCES crypto_networks(id) ON DELETE CASCADE,
    currency_id VARCHAR(50) NOT NULL REFERENCES crypto_currencies(id) ON DELETE CASCADE,
    address VARCHAR(255) NOT NULL,
    derivation_index INT NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'available',
    assigned_invoice_id UUID,
    assigned_at TIMESTAMPTZ,
    last_scanned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS crypto_payment_invoices (
    id UUID PRIMARY KEY,
    invoice_code VARCHAR(50) UNIQUE NOT NULL,
    user_id UUID NOT NULL,
    organization_id UUID,
    payable_type VARCHAR(50) NOT NULL DEFAULT 'invoice',
    payable_id VARCHAR(255) NOT NULL,
    fiat_amount_usd NUMERIC(16,4) NOT NULL,
    crypto_amount NUMERIC(32,18) NOT NULL,
    exchange_rate NUMERIC(32,18) NOT NULL,
    currency_id VARCHAR(50) NOT NULL REFERENCES crypto_currencies(id),
    network_id VARCHAR(50) NOT NULL REFERENCES crypto_networks(id),
    deposit_address VARCHAR(255) NOT NULL,
    tx_hash VARCHAR(255),
    confirmations INT NOT NULL DEFAULT 0,
    min_confirmations INT NOT NULL DEFAULT 1,
    amount_received NUMERIC(32,18) NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    qr_uri TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    detected_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS crypto_internal_ledger (
    id UUID PRIMARY KEY,
    invoice_id UUID REFERENCES crypto_payment_invoices(id) ON DELETE CASCADE,
    entry_type VARCHAR(50) NOT NULL,
    amount NUMERIC(32,18) NOT NULL,
    currency VARCHAR(50) NOT NULL,
    network VARCHAR(50) NOT NULL,
    from_address VARCHAR(255),
    to_address VARCHAR(255) NOT NULL,
    tx_hash VARCHAR(255),
    block_number BIGINT,
    confirmations INT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS crypto_payment_settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Performance & Idempotency Indexes
CREATE INDEX IF NOT EXISTS idx_crypto_currencies_net ON crypto_currencies(network_id, is_active);
CREATE INDEX IF NOT EXISTS idx_crypto_deposit_addresses_lookup ON crypto_deposit_addresses(network_id, currency_id, status);
CREATE INDEX IF NOT EXISTS idx_crypto_deposit_addresses_addr ON crypto_deposit_addresses(address);
CREATE INDEX IF NOT EXISTS idx_crypto_invoices_status_exp ON crypto_payment_invoices(status, expires_at);
CREATE INDEX IF NOT EXISTS idx_crypto_invoices_addr ON crypto_payment_invoices(deposit_address);
CREATE UNIQUE INDEX IF NOT EXISTS idx_crypto_invoices_tx_hash_unique ON crypto_payment_invoices(tx_hash) WHERE tx_hash IS NOT NULL AND tx_hash <> '';
CREATE INDEX IF NOT EXISTS idx_crypto_ledger_inv ON crypto_internal_ledger(invoice_id);

-- Seed Default Production Networks
INSERT INTO crypto_networks (id, name, chain_id, rpc_url, fallback_rpc_url, explorer_tx_url, explorer_address_url, native_currency, min_confirmations, is_active)
VALUES
('tron', 'TRON (TRC20)', 728126428, 'https://api.trongrid.io', 'https://api.tronstack.io', 'https://tronscan.org/#/transaction/%s', 'https://tronscan.org/#/address/%s', 'TRX', 19, TRUE),
('bsc', 'BNB Smart Chain (BEP20)', 56, 'https://bsc-dataseed.binance.org', 'https://rpc.ankr.com/bsc', 'https://bscscan.com/tx/%s', 'https://bscscan.com/address/%s', 'BNB', 3, TRUE),
('ethereum', 'Ethereum (ERC20)', 1, 'https://cloudflare-eth.com', 'https://rpc.ankr.com/eth', 'https://etherscan.io/tx/%s', 'https://etherscan.io/address/%s', 'ETH', 2, TRUE),
('polygon', 'Polygon (PoS)', 137, 'https://polygon-rpc.com', 'https://rpc.ankr.com/polygon', 'https://polygonscan.com/tx/%s', 'https://polygonscan.com/address/%s', 'POL', 5, TRUE),
('bitcoin', 'Bitcoin', 0, 'https://blockstream.info/api', 'https://mempool.space/api', 'https://mempool.space/tx/%s', 'https://mempool.space/address/%s', 'BTC', 2, TRUE),
('solana', 'Solana', 101, 'https://api.mainnet-beta.solana.com', 'https://rpc.ankr.com/solana', 'https://solscan.io/tx/%s', 'https://solscan.io/account/%s', 'SOL', 1, TRUE),
('litecoin', 'Litecoin', 0, 'https://litecoinspace.org/api', 'https://blockchair.com', 'https://blockchair.com/litecoin/transaction/%s', 'https://blockchair.com/litecoin/address/%s', 'LTC', 3, TRUE),
('dogecoin', 'Dogecoin', 0, 'https://dogechain.info/api/v1', 'https://blockchair.com', 'https://dogechain.info/tx/%s', 'https://dogechain.info/address/%s', 'DOGE', 3, TRUE)
ON CONFLICT (id) DO UPDATE SET
    rpc_url = EXCLUDED.rpc_url,
    explorer_tx_url = EXCLUDED.explorer_tx_url;

-- Seed Default Currencies & Token Contracts
INSERT INTO crypto_currencies (id, network_id, symbol, name, is_token, contract_address, decimals, min_confirmations, min_amount_usd, icon, is_active)
VALUES
('USDT_TRC20', 'tron', 'USDT', 'Tether USD (TRC20)', TRUE, 'TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t', 6, 19, 1.0000, 'usdt', TRUE),
('USDT_BEP20', 'bsc', 'USDT', 'Tether USD (BEP20)', TRUE, '0x55d398326f99059fF775485246999027B3197955', 18, 3, 1.0000, 'usdt', TRUE),
('USDT_ERC20', 'ethereum', 'USDT', 'Tether USD (ERC20)', TRUE, '0xdAC17F958D2ee523a2206206994597C13D831ec7', 6, 2, 5.0000, 'usdt', TRUE),
('USDC_BEP20', 'bsc', 'USDC', 'USD Coin (BEP20)', TRUE, '0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d', 18, 3, 1.0000, 'usdc', TRUE),
('USDC_ERC20', 'ethereum', 'USDC', 'USD Coin (ERC20)', TRUE, '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', 6, 2, 5.0000, 'usdc', TRUE),
('BTC', 'bitcoin', 'BTC', 'Bitcoin', FALSE, '', 8, 2, 5.0000, 'btc', TRUE),
('ETH', 'ethereum', 'ETH', 'Ethereum', FALSE, '', 18, 2, 5.0000, 'eth', TRUE),
('BNB', 'bsc', 'BNB', 'BNB Coin', FALSE, '', 18, 3, 1.0000, 'bnb', TRUE),
('TRX', 'tron', 'TRX', 'TRON', FALSE, '', 6, 19, 1.0000, 'trx', TRUE),
('SOL', 'solana', 'SOL', 'Solana', FALSE, '', 9, 1, 1.0000, 'sol', TRUE),
('LTC', 'litecoin', 'LTC', 'Litecoin', FALSE, '', 8, 3, 1.0000, 'ltc', TRUE),
('DOGE', 'dogecoin', 'DOGE', 'Dogecoin', FALSE, '', 8, 3, 1.0000, 'doge', TRUE)
ON CONFLICT (id) DO UPDATE SET
    contract_address = EXCLUDED.contract_address,
    decimals = EXCLUDED.decimals;

-- Seed Default Settings
INSERT INTO crypto_payment_settings (key, value, description)
VALUES
('invoice_expiry_minutes', '30', 'Standard duration in minutes before a pending crypto invoice expires'),
('payment_tolerance_percent', '1.0', 'Acceptable underpayment percentage margin (e.g. 1.0 allows 99% payment to be marked settled)'),
('auto_settle_confirmations', 'true', 'Automatically mark orders or balances settled once minimum confirmations are reached on-chain'),
('price_cache_seconds', '60', 'Live ticker price caching duration in seconds'),
('scanner_interval_seconds', '30', 'Interval between background on-chain scans')
ON CONFLICT (key) DO NOTHING;
