-- Migration 0031: Add Custom Memory, CPU, Inodes and Process Isolation Limits to Hosting Accounts
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS memory_limit_mb INT NOT NULL DEFAULT 1024;
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS cpu_limit_percent INT NOT NULL DEFAULT 100;
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS inodes_limit BIGINT NOT NULL DEFAULT 250000;
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS inodes_used BIGINT NOT NULL DEFAULT 0;
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS tasks_max INT NOT NULL DEFAULT 100;
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS io_read_mbps INT NOT NULL DEFAULT 50;
ALTER TABLE hosting_accounts ADD COLUMN IF NOT EXISTS io_write_mbps INT NOT NULL DEFAULT 30;
