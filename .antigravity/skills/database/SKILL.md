---
name: database
description: |
  Database analysis and safe operation guidelines for Hostvra PostgreSQL.
  Covers schema inspection, safe queries, and forbidden operations.
---

# Hostvra Database Reference

## Database Engine
- PostgreSQL 16 (production)
- Connection: DATABASE_URL environment variable
- Driver: github.com/lib/pq
- pgcrypto extension: enabled (for gen_random_uuid())

## Schema Overview

### Core Tables
| Table | Key Columns |
|-------|-------------|
| organizations | id (UUID), name, slug, plan_tier, max_servers, max_websites |
| users | id (UUID), email, password_hash (argon2id), is_superadmin, 2FA fields |
| organization_members | org_id, user_id, role_id — UNIQUE(org_id, user_id) |
| roles | id, organization_id, name (owner/admin/manager/developer/viewer) |
| permissions | id (e.g. 'servers.view'), module, description |
| role_permissions | role_id, permission_id |
| user_sessions | user_id, refresh_token_hash, expires_at, is_revoked |
| servers | id, org_id, hostname, ip_address, os, status, agent_key_hash |
| server_enrollment_tokens | token_hash, org_id, used_at, expires_at |
| server_metrics | server_id, cpu_percent, ram_used_mb, uptime_sec, timestamp |
| websites | id, org_id, server_id, domain, status, ssl_enabled, waf_enabled |
| databases (managed) | id, server_id, name, type (mysql/postgres), engine |
| database_users | id, database_id, username |
| ssl_certificates | id, org_id, website_id, domain, cert_pem, expires_at |
| audit_logs | id, org_id, user_id, action, resource_type, resource_id, status, metadata |

### Email Tables (migration 0002)
| Table | Purpose |
|-------|---------|
| email_domains | Hosted email domains |
| mailboxes | User mailboxes (Dovecot) |
| email_aliases | Address aliases |
| email_forwarders | Forwarder rules |

### Billing Tables (migration 0007)
| Table | Purpose |
|-------|---------|
| billing_plans | Service plans |
| subscriptions | Account subscriptions |
| invoices | Billing invoices |
| payment_gateways | Payment provider configs |
| support_tickets | Customer support tickets |

## Safe Read-Only Query Patterns

```sql
-- Organization listing
SELECT id, name, slug, plan_tier FROM organizations WHERE deleted_at IS NULL;

-- Users in org
SELECT u.email, u.full_name, r.name as role
FROM users u
JOIN organization_members om ON om.user_id = u.id
JOIN roles r ON r.id = om.role_id
WHERE om.organization_id = $1 AND u.deleted_at IS NULL;

-- Recent audit logs
SELECT action, resource_type, resource_id, status, created_at
FROM audit_logs
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT 100;

-- Server status
SELECT hostname, ip_address, status, last_heartbeat_at
FROM servers
WHERE organization_id = $1;
```

## BLOCKED Operations — NEVER EXECUTE WITHOUT EXPLICIT APPROVAL

```sql
DROP DATABASE hostvra;
DROP TABLE users;
DROP TABLE organizations;
TRUNCATE audit_logs;
DELETE FROM users;                          -- without WHERE clause
DELETE FROM organizations;                  -- without WHERE clause
ALTER TABLE users DROP COLUMN password_hash;
UPDATE users SET is_superadmin = true WHERE 1=1;
```

## Operations Requiring APPROVAL

```sql
-- Any DELETE with WHERE clause
DELETE FROM users WHERE id = $1;  -- requires approval

-- Any schema modification
ALTER TABLE ...;  -- requires approval

-- Any INSERT into critical tables
INSERT INTO users ...;  -- requires approval
INSERT INTO organizations ...;  -- requires approval
```

## Migration Rules

1. NEVER run migrations without checking current schema version first
2. New migrations must be additive (no DROP COLUMN in migration files)
3. Test migration on a copy of the schema before production
4. Migrations are in /migrations/ and applied in numeric order
5. The auto_migrator.go handles migration application

## Safe Diagnostic Queries

```sql
-- Check migration state (if migration tracking table exists)
SELECT * FROM schema_migrations ORDER BY applied_at DESC LIMIT 10;

-- Table sizes
SELECT relname, pg_size_pretty(pg_total_relation_size(relid))
FROM pg_stat_user_tables ORDER BY pg_total_relation_size(relid) DESC;

-- Check indexes
SELECT tablename, indexname, indexdef
FROM pg_indexes WHERE schemaname = 'public';

-- Active connections
SELECT count(*), state FROM pg_stat_activity GROUP BY state;

-- Slow queries (if pg_stat_statements enabled)
SELECT query, mean_exec_time, calls
FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;
```
