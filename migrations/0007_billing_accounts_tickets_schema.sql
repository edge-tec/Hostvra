-- ============================================================================
-- Migration 0007: Enterprise Billing, Hosting Accounts, Support Tickets,
-- Knowledgebase & System Settings Schema
-- ============================================================================

-- 1. Hosting Plans Catalog
CREATE TABLE IF NOT EXISTS hosting_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    tier VARCHAR(50) NOT NULL DEFAULT 'starter',
    price_monthly NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    price_yearly NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    disk_space_mb BIGINT NOT NULL DEFAULT 10240,
    bandwidth_mb BIGINT NOT NULL DEFAULT 102400,
    max_websites INT NOT NULL DEFAULT 1,
    max_databases INT NOT NULL DEFAULT 5,
    max_mailboxes INT NOT NULL DEFAULT 10,
    max_ftp INT NOT NULL DEFAULT 5,
    dedicated_ip BOOLEAN NOT NULL DEFAULT FALSE,
    free_ssl BOOLEAN NOT NULL DEFAULT TRUE,
    features TEXT[] DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_hosting_plans_tier ON hosting_plans(tier);
CREATE INDEX IF NOT EXISTS idx_hosting_plans_active_sort ON hosting_plans(is_active, sort_order ASC);

-- 2. Subscriptions
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES hosting_plans(id) ON DELETE RESTRICT,
    plan_name VARCHAR(255) NOT NULL,
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, pending, suspended, cancelled, expired
    billing_cycle VARCHAR(50) NOT NULL DEFAULT 'monthly', -- monthly, yearly
    amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    disk_used_mb BIGINT NOT NULL DEFAULT 0,
    bandwidth_used_mb BIGINT NOT NULL DEFAULT 0,
    websites_count INT NOT NULL DEFAULT 0,
    next_billing_date TIMESTAMPTZ NOT NULL,
    auto_renew BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_org ON subscriptions(organization_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);

-- 3. Invoices
CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number VARCHAR(100) NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    plan_id UUID NOT NULL REFERENCES hosting_plans(id) ON DELETE RESTRICT,
    description TEXT NOT NULL,
    subtotal NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    tax NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    discount NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    total NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    status VARCHAR(50) NOT NULL DEFAULT 'unpaid', -- paid, unpaid, overdue, cancelled
    payment_method VARCHAR(50),
    transaction_id VARCHAR(255),
    due_date TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoices_user ON invoices(user_id);
CREATE INDEX IF NOT EXISTS idx_invoices_sub ON invoices(subscription_id);
CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(status);
CREATE INDEX IF NOT EXISTS idx_invoices_due ON invoices(due_date);

-- 4. Payment Gateway Configs
CREATE TABLE IF NOT EXISTS payment_gateway_configs (
    gateway VARCHAR(50) PRIMARY KEY, -- stripe, bkash, nagad, sslcommerz, paypal
    display_name VARCHAR(100) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    test_mode BOOLEAN NOT NULL DEFAULT TRUE,
    api_key TEXT,
    secret_key TEXT,
    merchant_id TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. Client Hosting Accounts (WHM Multi-Tenancy Accounts)
CREATE TABLE IF NOT EXISTS hosting_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    server_name VARCHAR(255),
    domain VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(64) NOT NULL UNIQUE,
    document_root VARCHAR(512) NOT NULL,
    plan_id UUID NOT NULL REFERENCES hosting_plans(id) ON DELETE RESTRICT,
    plan_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, suspended, pending, terminated
    suspend_reason TEXT,
    disk_limit_mb BIGINT NOT NULL DEFAULT 10240,
    disk_used_mb BIGINT NOT NULL DEFAULT 0,
    bandwidth_limit_mb BIGINT NOT NULL DEFAULT 102400,
    bandwidth_used_mb BIGINT NOT NULL DEFAULT 0,
    websites_limit INT NOT NULL DEFAULT 1,
    databases_limit INT NOT NULL DEFAULT 5,
    mailboxes_limit INT NOT NULL DEFAULT 10,
    ip_address VARCHAR(100),
    php_version VARCHAR(20) DEFAULT '8.3',
    ssl_active BOOLEAN NOT NULL DEFAULT TRUE,
    suspended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_hosting_accounts_org ON hosting_accounts(organization_id);
CREATE INDEX IF NOT EXISTS idx_hosting_accounts_user ON hosting_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_hosting_accounts_server ON hosting_accounts(server_id);
CREATE INDEX IF NOT EXISTS idx_hosting_accounts_domain ON hosting_accounts(domain);
CREATE INDEX IF NOT EXISTS idx_hosting_accounts_username ON hosting_accounts(username);

-- 6. Support Tickets
CREATE TABLE IF NOT EXISTS tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_number VARCHAR(100) NOT NULL UNIQUE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_email VARCHAR(255) NOT NULL,
    user_name VARCHAR(255) NOT NULL,
    department VARCHAR(50) NOT NULL DEFAULT 'technical', -- technical, billing, sales, abuse
    priority VARCHAR(50) NOT NULL DEFAULT 'medium', -- low, medium, high, urgent
    status VARCHAR(50) NOT NULL DEFAULT 'open', -- open, in_progress, answered, customer_reply, closed
    subject VARCHAR(500) NOT NULL,
    related_service VARCHAR(255),
    replies_count INT NOT NULL DEFAULT 0,
    last_reply_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tickets_org ON tickets(organization_id);
CREATE INDEX IF NOT EXISTS idx_tickets_user ON tickets(user_id);
CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
CREATE INDEX IF NOT EXISTS idx_tickets_dept ON tickets(department);

-- 7. Ticket Replies
CREATE TABLE IF NOT EXISTS ticket_replies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_email VARCHAR(255) NOT NULL,
    user_name VARCHAR(255) NOT NULL,
    is_staff BOOLEAN NOT NULL DEFAULT FALSE,
    is_private_note BOOLEAN NOT NULL DEFAULT FALSE,
    message TEXT NOT NULL,
    attachments TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ticket_replies_ticket ON ticket_replies(ticket_id);

-- 8. Canned Support Responses
CREATE TABLE IF NOT EXISTS canned_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    shortcut VARCHAR(100) NOT NULL UNIQUE,
    department VARCHAR(50) NOT NULL DEFAULT 'technical',
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 9. Knowledgebase Articles
CREATE TABLE IF NOT EXISTS knowledge_articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(500) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    category VARCHAR(100) NOT NULL DEFAULT 'hosting',
    content TEXT NOT NULL,
    summary TEXT,
    views INT NOT NULL DEFAULT 0,
    helpful_votes INT NOT NULL DEFAULT 0,
    unhelpful_votes INT NOT NULL DEFAULT 0,
    is_published BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_articles_slug ON knowledge_articles(slug);
CREATE INDEX IF NOT EXISTS idx_articles_category ON knowledge_articles(category);

-- 10. Global System Settings
CREATE TABLE IF NOT EXISTS system_settings (
    id INT PRIMARY KEY DEFAULT 1,
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed Initial Default Plans
INSERT INTO hosting_plans (
    name, slug, description, tier, price_monthly, price_yearly, currency,
    disk_space_mb, bandwidth_mb, max_websites, max_databases, max_mailboxes,
    max_ftp, dedicated_ip, free_ssl, features, is_active, sort_order
)
VALUES
    (
        'Starter Cloud', 'starter-cloud',
        'Ideal for personal websites, blogs, and small development projects.',
        'starter', 4.99, 49.99, 'USD',
        10240, 102400, 1, 2, 5, 2, FALSE, TRUE,
        ARRAY['1 Website', '10 GB NVMe Storage', '100 GB Bandwidth', 'Free SSL', 'Multi-PHP 7.4 - 8.3', 'Daily Automated Backups'],
        TRUE, 1
    ),
    (
        'Business Pro', 'business-pro',
        'High-performance cgroup-isolated hosting for high-traffic business applications.',
        'business', 9.99, 99.99, 'USD',
        51200, 512000, 5, 10, 25, 10, FALSE, TRUE,
        ARRAY['5 Websites', '50 GB NVMe Storage', '500 GB Bandwidth', 'Free Wildcard SSL', 'Isolated cgroups', 'ModSecurity WAF', 'Daily Backups'],
        TRUE, 2
    ),
    (
        'Enterprise Elite', 'enterprise-elite',
        'Dedicated resources, priority support, and enterprise uptime SLA.',
        'enterprise', 24.99, 249.99, 'USD',
        204800, 2048000, 25, 50, 100, 25, TRUE, TRUE,
        ARRAY['25 Websites', '200 GB NVMe Storage', '2 TB Bandwidth', 'Dedicated IPv4', 'Pure-FTPd Multi-User', 'Priority SLA', 'Custom PHP Pools'],
        TRUE, 3
    )
ON CONFLICT (slug) DO NOTHING;

-- Seed Default Payment Gateways
INSERT INTO payment_gateway_configs (gateway, display_name, enabled, test_mode)
VALUES
    ('stripe', 'Stripe Card Payments', TRUE, TRUE),
    ('paypal', 'PayPal Express', FALSE, TRUE),
    ('bkash', 'bKash Mobile Payment', TRUE, TRUE),
    ('nagad', 'Nagad Direct Pay', TRUE, TRUE),
    ('sslcommerz', 'SSLCommerz Gateway', FALSE, TRUE)
ON CONFLICT (gateway) DO NOTHING;

-- Seed Initial Canned Responses
INSERT INTO canned_responses (title, shortcut, department, content)
VALUES
    ('DNS Propagation Notice', 'dns_prop', 'technical', 'Hello,\n\nWe have verified your DNS records. They are fully provisioned across our authoritative nameservers. Due to ISP DNS caching, global propagation may take 15 to 60 minutes.\n\nBest regards,\nHostvra Support Team'),
    ('PHP Memory & Upload Limits', 'php_limit', 'technical', 'Hello,\n\nWe have updated your PHP runtime limits. The configuration memory_limit has been updated and PHP-FPM pools have been reloaded. Please test your application now.\n\nBest regards,\nHostvra Support Team'),
    ('SSL Certificate Verification', 'ssl_verify', 'technical', 'Hello,\n\nTo complete automated Let''s Encrypt SSL issuance, port 80 must be reachable and your A record must point directly to your server IP. We verified your DNS record and re-issued the 90-day SSL certificate.\n\nBest regards,\nHostvra Support Team'),
    ('Invoice Payment Confirmation', 'invoice_paid', 'billing', 'Hello,\n\nThank you for your payment. Your invoice has been marked PAID and automated renewal for your hosting subscription is confirmed.\n\nBest regards,\nHostvra Billing Department')
ON CONFLICT (shortcut) DO NOTHING;
