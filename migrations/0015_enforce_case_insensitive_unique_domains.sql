-- Migration: 0015_enforce_case_insensitive_unique_domains.sql
-- Description: Enforce case-insensitive unique primary domain across all active websites

CREATE UNIQUE INDEX IF NOT EXISTS idx_websites_lower_primary_domain_unique
ON websites (LOWER(primary_domain))
WHERE deleted_at IS NULL;
