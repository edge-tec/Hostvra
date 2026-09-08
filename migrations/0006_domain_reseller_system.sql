-- ============================================================================
-- 0006: HOSTVRA DOMAIN RESELLER & MANAGEMENT SUBSYSTEM SCHEMA
-- ============================================================================

-- 1. Supported Domain TLDs
CREATE TABLE IF NOT EXISTS domain_tlds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tld VARCHAR(64) NOT NULL UNIQUE, -- e.g. 'com', 'net', 'org', 'xyz'
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    registration_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    transfer_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    renewal_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    min_years INT NOT NULL DEFAULT 1,
    max_years INT NOT NULL DEFAULT 10,
    provider VARCHAR(64) NOT NULL DEFAULT 'resellerclub',
    is_popular BOOLEAN NOT NULL DEFAULT FALSE,
    category VARCHAR(64) NOT NULL DEFAULT 'popular', -- 'popular', 'tech', 'business', 'country'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_domain_tlds_enabled ON domain_tlds(enabled);
CREATE INDEX IF NOT EXISTS idx_domain_tlds_tld ON domain_tlds(tld);

-- 2. Domain Prices (Customer Selling Prices vs Wholesale Registrar Costs)
CREATE TABLE IF NOT EXISTS domain_prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tld VARCHAR(64) NOT NULL UNIQUE REFERENCES domain_tlds(tld) ON DELETE CASCADE,
    registration_cost NUMERIC(12, 2) NOT NULL DEFAULT 0.00,  -- Internal wholesale provider cost
    registration_price NUMERIC(12, 2) NOT NULL DEFAULT 0.00, -- Customer selling price
    renewal_cost NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    renewal_price NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    transfer_cost NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    transfer_price NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_domain_prices_tld ON domain_prices(tld);

-- 3. Customer Registered Domains
CREATE TABLE IF NOT EXISTS domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    order_id UUID,
    domain_name VARCHAR(255) NOT NULL UNIQUE,
    tld VARCHAR(64) NOT NULL,
    registrar VARCHAR(64) NOT NULL DEFAULT 'resellerclub',
    provider_order_id VARCHAR(128),
    provider_domain_id VARCHAR(128),
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'provisioning', 'active', 'suspended', 'expired', 'transferred_out', 'cancelled'
    registration_date TIMESTAMPTZ,
    expiry_date TIMESTAMPTZ,
    transfer_status VARCHAR(50) NOT NULL DEFAULT 'none', -- 'none', 'pending', 'completed', 'failed'
    auto_renew BOOLEAN NOT NULL DEFAULT FALSE,
    registrar_lock BOOLEAN NOT NULL DEFAULT TRUE,
    privacy_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    website_id UUID REFERENCES websites(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_domains_user_id ON domains(user_id);
CREATE INDEX IF NOT EXISTS idx_domains_org_id ON domains(organization_id);
CREATE INDEX IF NOT EXISTS idx_domains_domain_name ON domains(domain_name);
CREATE INDEX IF NOT EXISTS idx_domains_status ON domains(status);
CREATE INDEX IF NOT EXISTS idx_domains_expiry ON domains(expiry_date);

-- 4. Domain Orders (Registration, Renewal, Transfer)
CREATE TABLE IF NOT EXISTS domain_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    domain_id UUID REFERENCES domains(id) ON DELETE SET NULL,
    domain_name VARCHAR(255) NOT NULL,
    order_type VARCHAR(32) NOT NULL, -- 'registration', 'renewal', 'transfer'
    years INT NOT NULL DEFAULT 1,
    amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    cost NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    payment_status VARCHAR(32) NOT NULL DEFAULT 'pending', -- 'pending', 'paid', 'refunded', 'failed'
    provisioning_status VARCHAR(32) NOT NULL DEFAULT 'pending', -- 'pending', 'provisioning', 'completed', 'failed'
    provider_status VARCHAR(64) NOT NULL DEFAULT 'none',
    provider_order_id VARCHAR(128),
    idempotency_key VARCHAR(128) NOT NULL UNIQUE,
    failure_reason TEXT,
    retry_count INT NOT NULL DEFAULT 0,
    invoice_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_domain_orders_user_id ON domain_orders(user_id);
CREATE INDEX IF NOT EXISTS idx_domain_orders_domain_name ON domain_orders(domain_name);
CREATE INDEX IF NOT EXISTS idx_domain_orders_status ON domain_orders(payment_status, provisioning_status);

-- 5. Domain Contact Details (Registrant, Admin, Tech, Billing)
CREATE TABLE IF NOT EXISTS domain_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    contact_type VARCHAR(32) NOT NULL, -- 'registrant', 'admin', 'tech', 'billing'
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    organization VARCHAR(150),
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(50) NOT NULL,
    address1 VARCHAR(255) NOT NULL,
    address2 VARCHAR(255),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100) NOT NULL,
    postal_code VARCHAR(32) NOT NULL,
    country VARCHAR(4) NOT NULL, -- ISO 2-letter country code
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_domain_contact_type UNIQUE (domain_id, contact_type)
);
CREATE INDEX IF NOT EXISTS idx_domain_contacts_domain_id ON domain_contacts(domain_id);

-- 6. Domain Nameservers
CREATE TABLE IF NOT EXISTS domain_nameservers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    nameserver VARCHAR(255) NOT NULL,
    position INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_domain_nameserver_pos UNIQUE (domain_id, position)
);
CREATE INDEX IF NOT EXISTS idx_domain_nameservers_domain_id ON domain_nameservers(domain_id);

-- 7. Domain DNS Records (A, AAAA, CNAME, MX, TXT, NS, SRV, CAA)
CREATE TABLE IF NOT EXISTS domain_dns_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    record_type VARCHAR(16) NOT NULL, -- 'A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'SRV', 'CAA'
    name VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    ttl INT NOT NULL DEFAULT 3600,
    priority INT,
    provider_record_id VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_domain_dns_records_domain_id ON domain_dns_records(domain_id);

-- 8. Domain Inbound Transfers
CREATE TABLE IF NOT EXISTS domain_transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID REFERENCES domains(id) ON DELETE SET NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain_name VARCHAR(255) NOT NULL,
    auth_code_encrypted TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'payment_pending', 'submitted', 'processing', 'completed', 'failed', 'cancelled'
    provider_order_id VARCHAR(128),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    failure_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_domain_transfers_user ON domain_transfers(user_id);
CREATE INDEX IF NOT EXISTS idx_domain_transfers_domain ON domain_transfers(domain_name);

-- 9. Domain Renewals
CREATE TABLE IF NOT EXISTS domain_renewals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    years INT NOT NULL DEFAULT 1,
    amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    payment_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    provider_order_id VARCHAR(128),
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'
    old_expiry_date TIMESTAMPTZ,
    new_expiry_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_domain_renewals_domain ON domain_renewals(domain_id);

-- 10. Domain Webhooks (Strict Idempotency for Payment & Registrar Events)
CREATE TABLE IF NOT EXISTS domain_webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(64) NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    external_event_id VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'processed',
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_domain_webhook_provider_event UNIQUE (provider, external_event_id)
);
CREATE INDEX IF NOT EXISTS idx_domain_webhooks_external_id ON domain_webhooks(external_event_id);

-- 11. Domain Provider Transactions (Accounting Audit Trail)
CREATE TABLE IF NOT EXISTS domain_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID REFERENCES domains(id) ON DELETE SET NULL,
    order_id UUID REFERENCES domain_orders(id) ON DELETE SET NULL,
    provider VARCHAR(64) NOT NULL DEFAULT 'resellerclub',
    operation VARCHAR(64) NOT NULL, -- 'availability', 'registration', 'renewal', 'transfer', 'dns_update', 'ns_update', 'lock'
    request_id VARCHAR(128),
    provider_order_id VARCHAR(128),
    amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00, -- Customer charged amount
    cost NUMERIC(12, 2) NOT NULL DEFAULT 0.00,   -- Registrar wholesale cost
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    status VARCHAR(32) NOT NULL DEFAULT 'success', -- 'success', 'failed', 'pending'
    error_code VARCHAR(64),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_domain_transactions_domain ON domain_transactions(domain_id);
CREATE INDEX IF NOT EXISTS idx_domain_transactions_order ON domain_transactions(order_id);

-- 12. Domain Audit Logs
CREATE TABLE IF NOT EXISTS domain_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID REFERENCES domains(id) ON DELETE SET NULL,
    domain_name VARCHAR(255) NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(64) NOT NULL, -- e.g. 'DOMAIN_REGISTERED', 'DOMAIN_RENEWED', 'DOMAIN_NAMESERVERS_UPDATED', etc.
    details TEXT NOT NULL,
    ip_address VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_domain_audit_logs_domain ON domain_audit_logs(domain_name);
CREATE INDEX IF NOT EXISTS idx_domain_audit_logs_user ON domain_audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_domain_audit_logs_created_at ON domain_audit_logs(created_at DESC);

-- ============================================================================
-- SEED INITIAL SUPPORTED TLDS & PRICINGS
-- ============================================================================
INSERT INTO domain_tlds (tld, enabled, registration_enabled, transfer_enabled, renewal_enabled, min_years, max_years, provider, is_popular, category)
VALUES
    ('com', TRUE, TRUE, TRUE, TRUE, 1, 10, 'resellerclub', TRUE, 'popular'),
    ('net', TRUE, TRUE, TRUE, TRUE, 1, 10, 'resellerclub', TRUE, 'popular'),
    ('org', TRUE, TRUE, TRUE, TRUE, 1, 10, 'resellerclub', TRUE, 'popular'),
    ('xyz', TRUE, TRUE, TRUE, TRUE, 1, 10, 'resellerclub', TRUE, 'tech'),
    ('io',  TRUE, TRUE, TRUE, TRUE, 1, 5,  'resellerclub', TRUE, 'tech'),
    ('co',  TRUE, TRUE, TRUE, TRUE, 1, 5,  'resellerclub', FALSE, 'popular'),
    ('tech', TRUE, TRUE, TRUE, TRUE, 1, 10, 'resellerclub', FALSE, 'tech'),
    ('store', TRUE, TRUE, TRUE, TRUE, 1, 10, 'resellerclub', FALSE, 'business'),
    ('online', TRUE, TRUE, TRUE, TRUE, 1, 10, 'resellerclub', FALSE, 'popular'),
    ('info', TRUE, TRUE, TRUE, TRUE, 1, 10, 'resellerclub', FALSE, 'popular'),
    ('biz',  TRUE, TRUE, TRUE, TRUE, 1, 10, 'resellerclub', FALSE, 'business')
ON CONFLICT (tld) DO UPDATE SET
    registration_enabled = EXCLUDED.registration_enabled,
    renewal_enabled = EXCLUDED.renewal_enabled,
    transfer_enabled = EXCLUDED.transfer_enabled,
    updated_at = NOW();

INSERT INTO domain_prices (tld, registration_cost, registration_price, renewal_cost, renewal_price, transfer_cost, transfer_price, currency, enabled)
VALUES
    ('com',    10.29, 14.99, 10.99, 16.99, 10.29, 14.99, 'USD', TRUE),
    ('net',    12.49, 16.99, 13.19, 18.99, 12.49, 16.99, 'USD', TRUE),
    ('org',    11.89, 15.99, 12.49, 17.99, 11.89, 15.99, 'USD', TRUE),
    ('xyz',     1.99,  2.99, 10.49, 13.99,  9.99, 12.99, 'USD', TRUE),
    ('io',     32.50, 44.99, 36.50, 49.99, 32.50, 44.99, 'USD', TRUE),
    ('co',      9.99, 14.99, 23.50, 29.99, 21.00, 27.99, 'USD', TRUE),
    ('tech',    3.89,  5.99, 17.50, 23.99, 16.00, 21.99, 'USD', TRUE),
    ('store',   2.99,  4.99, 26.50, 34.99, 24.00, 31.99, 'USD', TRUE),
    ('online',  1.89,  2.99, 22.50, 28.99, 20.00, 26.99, 'USD', TRUE),
    ('info',    4.29,  6.99, 16.50, 21.99, 15.00, 19.99, 'USD', TRUE),
    ('biz',     7.50, 11.99, 16.50, 21.99, 15.00, 19.99, 'USD', TRUE)
ON CONFLICT (tld) DO UPDATE SET
    registration_cost = EXCLUDED.registration_cost,
    registration_price = EXCLUDED.registration_price,
    renewal_cost = EXCLUDED.renewal_cost,
    renewal_price = EXCLUDED.renewal_price,
    transfer_cost = EXCLUDED.transfer_cost,
    transfer_price = EXCLUDED.transfer_price,
    updated_at = NOW();
