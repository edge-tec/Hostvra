-- Migration 0021: Mailbox Rate Limiting per hour and per day
ALTER TABLE email_mailboxes
    ADD COLUMN IF NOT EXISTS rate_limit_per_hour INT NOT NULL DEFAULT 100,
    ADD COLUMN IF NOT EXISTS rate_limit_per_day INT NOT NULL DEFAULT 1000;
