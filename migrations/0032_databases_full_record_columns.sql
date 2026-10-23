-- Migration 0032: Add missing columns to databases table for full record persistence
-- The databases table was missing username, password, quota, note, host_allow, backup_status,
-- backup_count, location, and in_recycle_bin columns. Without these, PostgresStore INSERT/SELECT
-- operations silently dropped fields that the MemoryStore correctly persisted.

ALTER TABLE databases ADD COLUMN IF NOT EXISTS username VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE databases ADD COLUMN IF NOT EXISTS password VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE databases ADD COLUMN IF NOT EXISTS quota VARCHAR(50) NOT NULL DEFAULT 'Not set';
ALTER TABLE databases ADD COLUMN IF NOT EXISTS note TEXT NOT NULL DEFAULT '';
ALTER TABLE databases ADD COLUMN IF NOT EXISTS host_allow VARCHAR(100) NOT NULL DEFAULT 'localhost';
ALTER TABLE databases ADD COLUMN IF NOT EXISTS backup_status VARCHAR(50) NOT NULL DEFAULT 'Not exist';
ALTER TABLE databases ADD COLUMN IF NOT EXISTS backup_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE databases ADD COLUMN IF NOT EXISTS location VARCHAR(100) NOT NULL DEFAULT 'Localhost';
ALTER TABLE databases ADD COLUMN IF NOT EXISTS in_recycle_bin BOOLEAN NOT NULL DEFAULT FALSE;
