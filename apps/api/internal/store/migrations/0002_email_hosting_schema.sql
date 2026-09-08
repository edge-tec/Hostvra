-- Hostvra Database Migration 0002: Email Hosting Subsystem Schema
-- Production PostgreSQL DDL for Domain Email, Postfix/Dovecot Accounts, Aliases, Forwarders, and DKIM Keys.

-- ============================================================================
-- 1. EMAIL DOMAINS
-- ============================================================================
CREATE TABLE IF NOT EXISTS email_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    domain VARCHAR(255) NOT NULL,
    mail_hostname VARCHAR(255) NOT NULL, -- e.g. "mail.example.com"
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, suspended, disabled
    storage_limit_bytes BIGINT NOT NULL DEFAULT 53687091200, -- 50 GB default domain quota
    storage_used_bytes BIGINT NOT NULL DEFAULT 0,
    spam_threshold NUMERIC(4, 2) NOT NULL DEFAULT 6.0,
    dkim_selector VARCHAR(50) NOT NULL DEFAULT 'default',
    is_catchall_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    catchall_mailbox_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(server_id, domain)
);
CREATE INDEX IF NOT EXISTS idx_email_domains_org ON email_domains(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_email_domains_server ON email_domains(server_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_email_domains_domain ON email_domains(domain) WHERE deleted_at IS NULL;

-- ============================================================================
-- 2. EMAIL MAILBOXES (Virtual Mailboxes)
-- ============================================================================
CREATE TABLE IF NOT EXISTS email_mailboxes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES email_domains(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    local_part VARCHAR(100) NOT NULL, -- e.g. "info", "admin"
    email VARCHAR(255) NOT NULL,      -- e.g. "info@example.com"
    password_hash VARCHAR(255) NOT NULL, -- SHA512-CRYPT / Argon2id hash compatible with Dovecot
    name VARCHAR(255) NOT NULL DEFAULT '',
    quota_bytes BIGINT NOT NULL DEFAULT 5368709120, -- 5 GB default
    used_bytes BIGINT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_suspended BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(domain_id, local_part)
);
CREATE INDEX IF NOT EXISTS idx_email_mailboxes_domain ON email_mailboxes(domain_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_email_mailboxes_email ON email_mailboxes(email) WHERE deleted_at IS NULL;

-- Set foreign key for catchall mailbox after email_mailboxes is defined
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_email_domains_catchall') THEN 
        ALTER TABLE email_domains 
            ADD CONSTRAINT fk_email_domains_catchall 
            FOREIGN KEY (catchall_mailbox_id) 
            REFERENCES email_mailboxes(id) 
            ON DELETE SET NULL;
    END IF;
END $$;

-- ============================================================================
-- 3. EMAIL ALIASES (Virtual Aliases)
-- ============================================================================
CREATE TABLE IF NOT EXISTS email_aliases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES email_domains(id) ON DELETE CASCADE,
    source_address VARCHAR(255) NOT NULL,      -- e.g. "support@example.com"
    destination_address VARCHAR(255) NOT NULL, -- e.g. "info@example.com"
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(domain_id, source_address, destination_address)
);
CREATE INDEX IF NOT EXISTS idx_email_aliases_source ON email_aliases(source_address);

-- ============================================================================
-- 4. EMAIL FORWARDERS
-- ============================================================================
CREATE TABLE IF NOT EXISTS email_forwarders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES email_domains(id) ON DELETE CASCADE,
    mailbox_id UUID REFERENCES email_mailboxes(id) ON DELETE CASCADE,
    source_address VARCHAR(255) NOT NULL,
    forward_address VARCHAR(255) NOT NULL, -- External e.g. "user@gmail.com"
    keep_copy BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- 5. EMAIL AUTORESPONDERS (Vacation Notices)
-- ============================================================================
CREATE TABLE IF NOT EXISTS email_autoresponders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mailbox_id UUID UNIQUE NOT NULL REFERENCES email_mailboxes(id) ON DELETE CASCADE,
    subject VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    start_at TIMESTAMPTZ,
    end_at TIMESTAMPTZ,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- 6. DKIM KEYS
-- ============================================================================
CREATE TABLE IF NOT EXISTS email_dkim_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID UNIQUE NOT NULL REFERENCES email_domains(id) ON DELETE CASCADE,
    selector VARCHAR(50) NOT NULL DEFAULT 'default',
    private_key_pem TEXT NOT NULL, -- Encrypted at rest
    public_key_dns TEXT NOT NULL,  -- e.g. "v=DKIM1; k=rsa; p=MIIBIj..."
    key_size INT NOT NULL DEFAULT 2048,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- 7. EMAIL DELIVERY & AUDIT LOGS (Metadata Only, No Bodies)
-- ============================================================================
CREATE TABLE IF NOT EXISTS email_delivery_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    domain_id UUID REFERENCES email_domains(id) ON DELETE SET NULL,
    message_id VARCHAR(255),
    queue_id VARCHAR(100),
    sender VARCHAR(255) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL, -- delivered, queued, deferred, bounced, rejected
    spam_score NUMERIC(4, 2) DEFAULT 0.0,
    failure_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_email_logs_server_created ON email_delivery_logs(server_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_email_logs_sender ON email_delivery_logs(sender);
CREATE INDEX IF NOT EXISTS idx_email_logs_recipient ON email_delivery_logs(recipient);
