-- Hostvra Database Migration 0017: Gmail-like Standalone Webmail Subsystem
-- Multi-account filters, contacts, identities, preferences, and forwarding rules

-- ============================================================================
-- 1. MAIL FILTERS (Real-time Rule Engine)
-- ============================================================================
CREATE TABLE IF NOT EXISTS mail_filters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mailbox_id UUID NOT NULL REFERENCES email_mailboxes(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    field VARCHAR(50) NOT NULL, -- from, to, cc, subject, body, has_attachment
    predicate VARCHAR(50) NOT NULL DEFAULT 'contains', -- contains, not_contains, equals, starts_with, ends_with
    value TEXT NOT NULL,
    action VARCHAR(50) NOT NULL, -- mark_read, star, move_to, delete, skip_inbox, mark_spam, forward
    action_value VARCHAR(255) DEFAULT '', -- target folder e.g. "archive" or forward address
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    priority INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_mail_filters_mailbox ON mail_filters(mailbox_id, is_active, priority DESC);

-- ============================================================================
-- 2. MAIL CONTACTS (Address Book & Autocomplete)
-- ============================================================================
CREATE TABLE IF NOT EXISTS mail_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mailbox_id UUID NOT NULL REFERENCES email_mailboxes(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(50) DEFAULT '',
    company VARCHAR(255) DEFAULT '',
    group_name VARCHAR(100) DEFAULT 'General',
    notes TEXT DEFAULT '',
    is_favorite BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_mail_contacts_mailbox ON mail_contacts(mailbox_id, name, email);

-- ============================================================================
-- 3. MAIL IDENTITIES & SENDING PERSONAS
-- ============================================================================
CREATE TABLE IF NOT EXISTS mail_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mailbox_id UUID NOT NULL REFERENCES email_mailboxes(id) ON DELETE CASCADE,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    reply_to_email VARCHAR(255) DEFAULT '',
    signature_id UUID REFERENCES email_signatures(id) ON DELETE SET NULL,
    is_default BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_mail_identities_mailbox ON mail_identities(mailbox_id);

-- ============================================================================
-- 4. WEBMAIL PREFERENCES
-- ============================================================================
CREATE TABLE IF NOT EXISTS webmail_preferences (
    mailbox_id UUID PRIMARY KEY REFERENCES email_mailboxes(id) ON DELETE CASCADE,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    reply_to VARCHAR(255) DEFAULT '',
    theme VARCHAR(20) NOT NULL DEFAULT 'system',
    page_size INT NOT NULL DEFAULT 50,
    sound_notifications BOOLEAN NOT NULL DEFAULT TRUE,
    desktop_notifications BOOLEAN NOT NULL DEFAULT TRUE,
    auto_refresh_seconds INT NOT NULL DEFAULT 30,
    default_reply_all BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- 5. MAIL FORWARDING RULES
-- ============================================================================
CREATE TABLE IF NOT EXISTS mail_forwarding_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mailbox_id UUID NOT NULL REFERENCES email_mailboxes(id) ON DELETE CASCADE,
    forward_to VARCHAR(255) NOT NULL,
    keep_copy BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_verified BOOLEAN NOT NULL DEFAULT TRUE,
    verification_code VARCHAR(64) DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_mail_forwarding_rules_mailbox ON mail_forwarding_rules(mailbox_id);
