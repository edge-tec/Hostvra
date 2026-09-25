---
name: hostvra-architecture
description: |
  Complete architecture reference for the Hostvra hosting control panel monorepo.
  Describes the REAL confirmed structure discovered during preflight audit on 2026-09-24.
  Use this skill to understand any component before modifying, analyzing, or testing it.
---

# Hostvra Architecture Reference

## Source of Truth
This document reflects the CONFIRMED repository structure.
Do NOT assume capabilities not listed here.
Do NOT create implementations that contradict this architecture.

---

## Monorepo Layout

```
/Hostvra/
├── apps/api/        Go 1.26 REST API  (module: hostvra/api)
├── apps/agent/      Go 1.22 Node Agent (module: hostvra/agent)
├── apps/web/        Next.js 15 / React 19 / TypeScript / TailwindCSS 3
├── migrations/      9 PostgreSQL SQL migration files (applied via docker-compose init)
├── packages/        npm workspace root (currently empty)
├── tests/           EMPTY — no integration tests exist
├── bin/             compiled binary outputs
├── docker-compose.yml  PostgreSQL 16 + Redis 7 local dev
├── Makefile         build / test / run targets
├── install.sh       production server installer (bash)
├── update.sh        in-place update script (bash)
└── ecosystem.config.js  PM2 process management
```

---

## apps/api — Core REST API

**Entry point:** `apps/api/cmd/server/main.go`
**Router:** github.com/go-chi/chi/v5
**Auth:** github.com/golang-jwt/jwt/v5 (JWT HS256)
**Password:** golang.org/x/crypto (Argon2id)
**DB client:** github.com/lib/pq (PostgreSQL native)
**Agent lib:** hostvra/agent (local replace directive)

### Internal Package Map

| Package | Purpose |
|---------|---------|
| `internal/handlers/` | HTTP handlers (62 files) |
| `internal/store/` | Data access layer — Store interface + PostgresStore + MemoryStore |
| `internal/auth/` | JWT middleware, hasher, claims |
| `internal/rbac/` | Role-permission matrix, RequirePermission middleware |
| `internal/audit/` | AuditLog struct and Logger.Log() |
| `internal/config/` | Config struct loaded from environment variables |
| `internal/migration/` | Migration manager |
| `internal/update/` | Live update orchestrator (22 files) |
| `internal/dns/` | DNS zone/record service |
| `internal/domains/` | Domain management + ResellerClub client |
| `internal/alerts/` | Fleet alert engine |
| `internal/license/` | License manager |
| `internal/response/` | JSON response helpers |
| `internal/appstore/` | App store |

### API Endpoint Groups (all under /api/v1)

```
Public:
  POST /auth/register
  POST /auth/login
  GET  /auth/me              (JWT required)
  POST /agent/enroll         (enrollment token)
  POST /agent/heartbeat      (agent key)
  GET  /billing/plans
  GET  /domains/search, /domains/whois, /domains/tlds
  GET  /support/articles
  POST /support/ai-assistant
  GET  /dashboard/overview   (optional JWT)

Protected (JWT + RBAC):
  /servers/*          fleet management, metrics
  /servers/{id}/php/* PHP versions, extensions, ini, FPM pools
  /servers/{id}/webservers/*  Nginx/Apache/OLS/LiteSpeed
  /websites/*         website CRUD, SSL, WAF, PHP, isolation, backup
  /databases/*        DB CRUD, query, export, import, tools
  /audit-logs/        read only
  /dns/*              zones + records
  /alerts/*           rules + channels
  /backups/*          on-demand + scheduled + cloud destinations
  /license/*          activation
  /system/updates/*   live update orchestration
  /system/fix, /system/restart
  /terminal/*         web terminal (EXECUTE ARBITRARY SHELL — HIGH RISK)
  /files/*, /filemanager/*  Enterprise File Manager v3.0
  /firewall/*         UFW rules + Fail2ban
  /waf/*              ModSecurity v3 + OWASP CRS
  /ssl/*              Let's Encrypt + custom certs
  /cron/*             Linux crontab
  /docker/*           container/image management
  /ftp/*              Pure-FTPd users
  /apps/*             App Store
  /team/*             members + invites
  /api-keys/*         API key management
  /email/*            domains, mailboxes, aliases, forwarders, queue, services
  /webmail/*          IMAP webmail (messages, send, move, delete)
  /billing/*          plans, subscriptions, invoices, gateways
  /accounts/*         multi-tenant hosting accounts (WHM-style)
  /domains/*          domain registrar + management
  /installer/*        1-click app templates
  /settings/*         system settings
```

### Store Interface
The `Store` interface in `internal/store/store.go` defines ALL database operations.
Two implementations: `PostgresStore` (production) and `MemoryStore` (dev fallback).

### RBAC System
5 roles: `owner` (all), `admin`, `manager`, `developer`, `viewer`
60+ permissions following pattern: `resource.action`
Enforced via `rbac.RequirePermission(perm)` middleware on every route.

### Audit Logging
`audit.Logger.Log(ctx, r, action, resourceType, resourceID, status, errorMsg, metadata)`
Called from EVERY sensitive handler. Persisted to `audit_logs` PostgreSQL table.

### Production Safety
- PostgreSQL required in production (startup exits if unavailable)
- JWT_SECRET must be strong (≥32 chars, non-default)
- DATABASE_URL must not contain dev credentials
- ResellerClub must be in production mode

---

## apps/agent — Node Agent

**Entry point:** `apps/agent/cmd/agent/main.go`
**Communication:** Inbound only — POSTs heartbeat to API every 10 seconds

### Architecture
The agent does NOT receive commands from the API.
The API imports `hostvra/agent/pkg/*` as a Go library (local replace directive).
Infrastructure operations happen in-process within the API server.

### pkg/ Packages (confirmed capabilities)

| Package | What it controls |
|---------|-----------------|
| `backup` | rsync/tar backup + S3 upload/download |
| `cron` | Linux crontab read/write |
| `database` | MySQL + PostgreSQL: create, drop, dump, restore |
| `docker` | Docker daemon: containers, images, prune |
| `email` | Postfix + Dovecot: domains, mailboxes, DKIM (NEW — untracked) |
| `files` | File system: read, write, chmod, archive, extract |
| `firewall` | UFW rules + Fail2ban jails/bans |
| `ftp` | Pure-FTPd: user create/delete/password |
| `installer` | 1-click: WordPress, Laravel, Next.js, Drupal, phpMyAdmin |
| `isolation` | cgroups v2 user resource isolation |
| `osadapter` | OS/distro detection (Ubuntu, Debian, RHEL, etc.) |
| `php` | PHP versions, extensions, php.ini, FPM pools, health |
| `security` | Security scanning |
| `ssl` | Let's Encrypt ACME (HTTP-01 + DNS-01) + custom cert import |
| `updater` | Live update deployment |
| `waf` | ModSecurity v3 + OWASP CRS config |
| `webserver` | Nginx, Apache, OpenLiteSpeed, LiteSpeed Enterprise |

---

## apps/web — Next.js Dashboard

**Framework:** Next.js 15.1.7 (App Router)
**UI:** React 19, TailwindCSS 3.4, lucide-react, clsx, tailwind-merge
**TypeScript:** 5.7.3

### Page Routes (34 pages confirmed)
`/accounts`, `/admin`, `/alerts`, `/api`, `/api-keys`, `/app-store`,
`/audit-logs`, `/backups`, `/billing`, `/cron`, `/dashboard`, `/databases`,
`/dns`, `/docker`, `/domains`, `/email`, `/files`, `/firewall`, `/ftp`,
`/license`, `/login`, `/php`, `/phpmyadmin`, `/register`, `/servers`,
`/settings`, `/ssl`, `/support`, `/team`, `/terminal`, `/waf`,
`/webmail`, `/webservers`, `/websites`

---

## Database — PostgreSQL 16

### Migration Files (applied in order)
```
migrations/0001_initial_schema.sql      — organizations, users, RBAC, servers, websites, DBs, SSL, audit_logs
migrations/0002_email_hosting_schema.sql — email domains, mailboxes, aliases, forwarders
migrations/0003_php_management_schema.sql — PHP versions, extensions, FPM pools
migrations/0004_webserver_management_schema.sql — web server fleet, vhosts
migrations/0005_update_system_schema.sql  — update_jobs, snapshots, channels
migrations/0006_domain_reseller_system.sql — registrars, TLDs, domain orders
migrations/0007_billing_accounts_tickets_schema.sql — billing, subscriptions, invoices
migrations/0008_file_manager_enterprise_schema.sql — favorites, labels, recent files
migrations/0009_email_enterprise_production_schema.sql — (NEW, untracked)
```

### Critical Tables
- `organizations` — multi-tenancy root
- `users` — identity (email, argon2id hash, 2FA)
- `organization_members` — user ↔ org ↔ role
- `roles` / `permissions` / `role_permissions` — RBAC
- `servers` — enrolled nodes
- `server_enrollment_tokens` — one-use enrollment
- `server_metrics` — time-series telemetry
- `audit_logs` — all sensitive actions (JSONB metadata)
- `websites` — hosted domains with SSL/WAF flags
- `databases` / `database_users` — managed databases
- `ssl_certificates` — cert store
- `email_domains` / `mailboxes` / `email_aliases` — email hosting
- `php_versions` / `php_extensions` / `fpm_pools` — PHP management
- `update_jobs` / `update_snapshots` — update tracking
- `billing_plans` / `subscriptions` / `invoices` — billing
- `hosting_accounts` — WHM-style multi-tenancy
- `cron_jobs` — Linux crontab records
- `file_favorites` / `file_labels` / `file_recent` — file manager

---

## Infrastructure Execution Model

```
AI Orchestrator
    |
    | (Python tools — controlled, allowlisted)
    v
Hostvra API  (/api/v1/*)
    |
    | (JWT auth + RBAC enforcement)
    v
handlers/  (Go — validates all inputs)
    |
    v
agent/pkg/*  (Go library — direct OS calls)
    |
    v
Linux OS  (systemctl / ufw / certbot / nginx / php-fpm / etc.)
```

NEVER bypass the API layer to call OS commands directly.
ALWAYS use the existing RBAC system for authorization.
NEVER call /api/v1/terminal/execute without command safety validation.

---

## Build System

```bash
make test-api     # cd apps/api && go test -v -race ./...
make test-agent   # cd apps/agent && go test -v -race ./...
make test-web     # cd apps/web && npm run build
make build-api    # CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/hostvra-api
make build-agent  # CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/hostvra-agent
make run-api      # cd apps/api && go run cmd/server/main.go
make run-web      # cd apps/web && npm run dev
```

---

## Environment Variables (from .env.example)

```
ENVIRONMENT=development|production
PORT=8080
DATABASE_URL=postgres://...
JWT_SECRET=<min 32 chars in production>
RESELLERCLUB_RESELLER_ID=...
RESELLERCLUB_API_KEY=...
RESELLERCLUB_MODE=sandbox|production
DOMAIN_ENCRYPTION_SECRET=<32 chars>
HOSTVRA_SYSTEM_EDITION=community|pro|enterprise
```

---

## Known Active Development (as of 2026-09-24)

An email/webmail feature is IN PROGRESS with 10 modified files and 6 untracked files.
The AI layer MUST NOT modify or disrupt these files:
- apps/api/internal/handlers/email.go (modified)
- apps/api/internal/handlers/webmail.go (new)
- apps/api/internal/store/email_models.go (modified)
- apps/api/internal/store/email_store.go (modified)
- apps/web/src/app/email/page.tsx (modified)
- apps/web/src/components/WebmailClient.tsx (modified)
- apps/agent/pkg/email/ (new package)
