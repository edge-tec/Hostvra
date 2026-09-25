-- Hostvra Database Migration 0009: Production Email Hosting & Webmail Subsystem Schema
-- Enterprise PostgreSQL DDL for Signatures, Suppressions, Rate Limits, and Webmail Messages & Attachments

-- ============================================================================
-- 1. EMAIL SIGNATURES
-- ============================================================================
CREATE TABLE IF NOT EXISTS email_signatures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mailbox_id UUID UNIQUE NOT NULL REFERENCES email_mailboxes(id) ON DELETE CASCADE,
    plain_text TEXT NOT NULL DEFAULT '',
    html_text TEXT NOT NULL DEFAULT '',
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_email_signatures_mailbox ON email_signatures(mailbox_id);

-- ============================================================================
-- 2. EMAIL SUPPRESSION LIST (Hard Bounces, Complaints, Unsubscribes)
-- ============================================================================
CREATE TABLE IF NOT EXISTS email_suppressions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    domain_id UUID REFERENCES email_domains(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    reason VARCHAR(50) NOT NULL, -- hard_bounce, complaint, unsubscribe, manual
    bounce_code VARCHAR(50) DEFAULT '',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(server_id, email)
);
CREATE INDEX IF NOT EXISTS idx_email_suppressions_email ON email_suppressions(email);
CREATE INDEX IF NOT EXISTS idx_email_suppressions_server ON email_suppressions(server_id);

-- ============================================================================
-- 3. EMAIL RATE LIMITS & QUOTA TRACKING
-- ============================================================================
CREATE TABLE IF NOT EXISTS email_rate_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    mailbox_id UUID REFERENCES email_mailboxes(id) ON DELETE CASCADE,
    domain_id UUID REFERENCES email_domains(id) ON DELETE CASCADE,
    window_hour TIMESTAMPTZ NOT NULL,
    sent_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(mailbox_id, window_hour)
);
CREATE INDEX IF NOT EXISTS idx_email_rate_limits_window ON email_rate_limits(mailbox_id, window_hour);

-- ============================================================================
-- 4. WEBMAIL MESSAGES (Maildir Metadata & Fast Query Cache)
-- ============================================================================
CREATE TABLE IF NOT EXISTS webmail_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mailbox_id UUID NOT NULL REFERENCES email_mailboxes(id) ON DELETE CASCADE,
    account_email VARCHAR(255) NOT NULL,
    folder VARCHAR(50) NOT NULL DEFAULT 'inbox', -- inbox, sent, drafts, trash, spam, archive
    message_id VARCHAR(255),
    from_name VARCHAR(255) NOT NULL DEFAULT '',
    from_email VARCHAR(255) NOT NULL,
    to_name VARCHAR(255) NOT NULL DEFAULT '',
    to_email VARCHAR(255) NOT NULL,
    cc TEXT DEFAULT '',
    bcc TEXT DEFAULT '',
    subject TEXT NOT NULL DEFAULT '',
    snippet TEXT NOT NULL DEFAULT '',
    body_text TEXT NOT NULL DEFAULT '',
    body_html TEXT NOT NULL DEFAULT '',
    is_unread BOOLEAN NOT NULL DEFAULT TRUE,
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,
    is_important BOOLEAN NOT NULL DEFAULT FALSE,
    has_attachment BOOLEAN NOT NULL DEFAULT FALSE,
    priority VARCHAR(20) NOT NULL DEFAULT 'normal', -- normal, high, low
    size_bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_webmail_messages_mailbox_folder ON webmail_messages(mailbox_id, folder, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_webmail_messages_account ON webmail_messages(account_email);
CREATE INDEX IF NOT EXISTS idx_webmail_messages_unread ON webmail_messages(mailbox_id, is_unread);

-- ============================================================================
-- 5. WEBMAIL ATTACHMENTS
-- ============================================================================
CREATE TABLE IF NOT EXISTS webmail_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES webmail_messages(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL DEFAULT 'application/octet-stream',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    storage_path TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_webmail_attachments_msg ON webmail_attachments(message_id);
