-- Hostvra Database Migration 0014: Email Domain DNS Verification Tracking
-- Track live DNS propagation and verified status for email domains.

ALTER TABLE email_domains ADD COLUMN IF NOT EXISTS is_dns_verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE email_domains ADD COLUMN IF NOT EXISTS dns_verified_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_email_domains_dns_verified ON email_domains(is_dns_verified) WHERE deleted_at IS NULL;
