-- 0029_email_deliverability_spam_imap_migration.sql
-- Phase Eight: External SMTP Relay, Spam Filtering & Quarantine, and Live IMAP Mailbox Migration

-- 1. External SMTP Relays (Amazon SES, SendGrid, Mailgun, Brevo, Postmark, Custom)
CREATE TABLE IF NOT EXISTS email_smtp_relays (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    domain_id UUID REFERENCES email_domains(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL DEFAULT 'Default Relay',
    provider VARCHAR(64) NOT NULL DEFAULT 'custom',
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL DEFAULT 587,
    username VARCHAR(255) NOT NULL DEFAULT '',
    password_encrypted TEXT NOT NULL DEFAULT '',
    auth_type VARCHAR(32) NOT NULL DEFAULT 'plain',
    encryption VARCHAR(32) NOT NULL DEFAULT 'starttls',
    sender_domain VARCHAR(255) NOT NULL DEFAULT '',
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    last_test_at TIMESTAMPTZ,
    last_test_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    last_test_error TEXT NOT NULL DEFAULT '',
    rate_limit_per_second INTEGER NOT NULL DEFAULT 0,
    rate_limit_per_day INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_email_smtp_relays_org ON email_smtp_relays(organization_id);
CREATE INDEX IF NOT EXISTS idx_email_smtp_relays_domain ON email_smtp_relays(domain_id);
CREATE INDEX IF NOT EXISTS idx_email_smtp_relays_sender ON email_smtp_relays(sender_domain);

-- 2. Spam Filtering & Action Thresholds (Rspamd / SpamAssassin)
CREATE TABLE IF NOT EXISTS email_spam_filter_settings (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    domain_id UUID REFERENCES email_domains(id) ON DELETE CASCADE,
    engine VARCHAR(32) NOT NULL DEFAULT 'rspamd',
    reject_score NUMERIC(5,2) NOT NULL DEFAULT 15.0,
    add_header_score NUMERIC(5,2) NOT NULL DEFAULT 6.0,
    greylist_score NUMERIC(5,2) NOT NULL DEFAULT 4.0,
    enable_greylisting BOOLEAN NOT NULL DEFAULT TRUE,
    enable_bayes BOOLEAN NOT NULL DEFAULT TRUE,
    enable_clamav BOOLEAN NOT NULL DEFAULT FALSE,
    quarantine_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    quarantine_retention_days INTEGER NOT NULL DEFAULT 14,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_email_spam_settings_org ON email_spam_filter_settings(organization_id);
CREATE INDEX IF NOT EXISTS idx_email_spam_settings_domain ON email_spam_filter_settings(domain_id);

-- 3. Spam Quarantine Items
CREATE TABLE IF NOT EXISTS email_spam_quarantine_items (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    domain_id UUID NOT NULL REFERENCES email_domains(id) ON DELETE CASCADE,
    recipient VARCHAR(255) NOT NULL,
    sender VARCHAR(255) NOT NULL,
    subject TEXT NOT NULL DEFAULT '',
    score NUMERIC(5,2) NOT NULL DEFAULT 0.0,
    symbols TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'quarantined',
    message_id VARCHAR(255) NOT NULL DEFAULT '',
    mail_path TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_email_spam_quarantine_org ON email_spam_quarantine_items(organization_id);
CREATE INDEX IF NOT EXISTS idx_email_spam_quarantine_domain ON email_spam_quarantine_items(domain_id);
CREATE INDEX IF NOT EXISTS idx_email_spam_quarantine_recipient ON email_spam_quarantine_items(recipient);
CREATE INDEX IF NOT EXISTS idx_email_spam_quarantine_status ON email_spam_quarantine_items(status);

-- 4. Live IMAP Mailbox Migrations
CREATE TABLE IF NOT EXISTS email_imap_migrations (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    destination_mailbox_id UUID NOT NULL REFERENCES email_mailboxes(id) ON DELETE CASCADE,
    destination_email VARCHAR(255) NOT NULL,
    source_host VARCHAR(255) NOT NULL,
    source_port INTEGER NOT NULL DEFAULT 993,
    source_username VARCHAR(255) NOT NULL,
    source_password_encrypted TEXT NOT NULL,
    source_use_ssl BOOLEAN NOT NULL DEFAULT TRUE,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    total_folders INTEGER NOT NULL DEFAULT 0,
    current_folder VARCHAR(255) NOT NULL DEFAULT '',
    total_messages INTEGER NOT NULL DEFAULT 0,
    synced_messages INTEGER NOT NULL DEFAULT 0,
    skipped_duplicates INTEGER NOT NULL DEFAULT 0,
    failed_messages INTEGER NOT NULL DEFAULT 0,
    bytes_transferred BIGINT NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_email_imap_migrations_org ON email_imap_migrations(organization_id);
CREATE INDEX IF NOT EXISTS idx_email_imap_migrations_mailbox ON email_imap_migrations(destination_mailbox_id);
CREATE INDEX IF NOT EXISTS idx_email_imap_migrations_status ON email_imap_migrations(status);
