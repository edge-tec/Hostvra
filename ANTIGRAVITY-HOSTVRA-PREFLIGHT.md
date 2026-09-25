# ANTIGRAVITY-HOSTVRA PREFLIGHT AUDIT
**Date:** 2026-09-24
**Phase:** 1 — Read-Only Repository Audit
**Status:** COMPLETE — No code modifications made

---

## 1. CURRENT ARCHITECTURE

### Monorepo Structure

```
/Hostvra/
├── apps/
│   ├── api/           Go 1.26 — Core REST API server (Chi router)
│   ├── agent/         Go 1.22 — Node agent (telemetry + infrastructure ops)
│   └── web/           Next.js 15 + React 19 + TypeScript — Dashboard UI
├── migrations/        9 sequential PostgreSQL DDL migrations
├── packages/          (workspace root for shared TS packages — currently empty)
├── tests/             (empty — no integration test suite here)
├── deployment/        (deployment assets directory)
├── bin/               (compiled binaries output)
├── docker-compose.yml PostgreSQL 16 + Redis 7 local dev stack
├── Makefile           Build/test/run targets
├── install.sh         Production server installer
├── update.sh          In-place update script
├── ecosystem.config.js PM2 process management
└── .github/workflows/ci.yml  GitHub Actions CI
```

### Runtime Versions (CONFIRMED)

| Runtime | Version | Source |
|---------|---------|--------|
| Go | 1.27.1 darwin/arm64 | `go version` |
| Node.js | v22.22.0 | `node --version` |
| Python (system) | 3.9.6 | `python3 --version` |
| Python (venv) | 3.12.x | `.venv/site-packages` |
| Google Antigravity SDK | 0.1.18 (in .venv) | `.venv/google_antigravity*.dist-info` |
| google-genai | 2.25.0 | `.venv/site-packages` |

> NOTE: System Python is 3.9.6 which does NOT satisfy SDK requirement (>=3.10).
> The .venv uses Python 3.12 and DOES satisfy the requirement.
> All AI layer scripts MUST use the .venv Python interpreter.

### apps/api — Go REST API (Port 8080)

**Module:** hostvra/api (go 1.26)
**Key Deps:** chi/v5, chi/cors, golang-jwt/jwt/v5, google/uuid, lib/pq, golang.org/x/crypto, hostvra/agent (local replace)
**Architecture Pattern:** Handler → Store → PostgreSQL/MemoryStore
**Entrypoint:** apps/api/cmd/server/main.go (966 lines)

**62 handler files confirmed in internal/handlers/**

Handlers include: auth, agent, websites, databases, dns, email, firewall, ssl,
php, webserver, files, backups, updates, terminal, billing, docker, cron, ftp,
waf, accounts, domain_registrar, domain_admin, audit, alerts, settings, team,
apikeys, webmail, dashboard, health, installer, appstore, migration, license, servers

**Middleware Chain (confirmed):**
1. middleware.RequestID
2. middleware.RealIP
3. middleware.Logger
4. middleware.Recoverer
5. middleware.Timeout(300s)
6. CORS (allow-all origins — SECURITY FINDING)
7. Security headers (X-Content-Type-Options, X-Frame-Options, X-XSS-Protection)
8. MaxBytesReader (10MB body limit)

### apps/agent — Go Node Agent

**Module:** hostvra/agent (go 1.22)
**Entrypoint:** apps/agent/cmd/agent/main.go

Architecture:
- Enrollment via token -> saves agent.json to disk
- Telemetry loop: collect metrics -> POST /api/v1/agent/heartbeat (every 10s)
- OS detection via internal/osadapter

**17 pkg/ packages confirmed:** backup, cron, database, docker, email (NEW-untracked),
files, firewall, ftp, installer, isolation, osadapter, php, security, ssl,
updater, waf, webserver

### apps/web — Next.js 15 Dashboard

**Framework:** Next.js 15.1.7 + React 19 + TypeScript 5.7 + TailwindCSS 3.4
**34 page routes confirmed** covering all hosting primitives

---

## 2. EXISTING CAPABILITIES

### Authentication & Authorization
- JWT-based (github.com/golang-jwt/jwt/v5)
- Argon2id password hashing
- RBAC: 5 roles (owner, admin, manager, developer, viewer)
- 60+ fine-grained permissions
- Super-admin bypass flag (is_superadmin on users table)
- Optional JWT middleware for public endpoints
- API key authentication (separate table)

### Database Architecture — PostgreSQL 16, 9 Migrations

| Migration | Schema Coverage |
|-----------|----------------|
| 0001 | organizations, users, RBAC, servers, websites, databases, SSL, audit_logs |
| 0002 | Email hosting (domains, mailboxes, aliases, forwarders, DKIM) |
| 0003 | PHP management (versions, extensions, FPM pools) |
| 0004 | Web server management (server fleet, vhosts) |
| 0005 | Update system (update_jobs, snapshots, channels) |
| 0006 | Domain reseller system (registrars, TLDs, domain orders) |
| 0007 | Billing (plans, subscriptions, invoices, payment gateways, tickets) |
| 0008 | File manager enterprise (favorites, labels, recent) |
| 0009 | Email enterprise production schema (untracked — NEW) |

In-memory fallback store exists for development.

### Existing Audit Logging
- AuditLog model: org_id, user_id, action, resource_type, resource_id, ip, user_agent, status, error_message, metadata (JSONB)
- audit.Logger.Log() called from every sensitive handler
- Stored in PostgreSQL audit_logs table
- Listed via GET /api/v1/audit-logs (RBAC: audit.view)

### Existing Update System
internal/update/ — 22 files:
- orchestrator.go: coordinates download -> verify -> snapshot -> deploy -> health-check
- snapshot.go: filesystem backup to /var/lib/hostvra/updates_backup
- deployer.go: deploys to /opt/hostvra
- verifier.go: SHA256 package verification
- rolling.go, scheduler.go, job.go

### Existing CI/CD (GitHub Actions)
- test-api: go test -v -race ./...
- test-agent: go test -v -race ./...
- build-web: npm run build (typecheck + build)
- cross-compile: Linux amd64/arm64 for API + Agent + CLI

---

## 3. EXISTING SECURITY BOUNDARIES

### Enforced
- JWT authentication on all protected routes
- RBAC permission middleware on every route
- Argon2id password hashing
- 10MB request body limit
- Standard security headers
- Production config validation at startup
- Audit logging of sensitive operations
- Enrollment token one-use consumption

### NOT Enforced (Findings for AI Layer)

| Finding | Severity | Location |
|---------|----------|----------|
| CORS allows all origins | Medium | main.go:157 |
| Terminal executes arbitrary shell — no allowlist | HIGH | handlers/terminal.go |
| No rate limiting on auth endpoints | Medium | handlers/auth.go |
| No CSRF token validation | Medium | CORS config |
| Redis declared in config but not used in Go | Low | config.go |

---

## 4. EXISTING EXECUTION BOUNDARIES

```
User Request (HTTP)
      |
  JWT Auth + RBAC
      |
  Handler layer (apps/api/internal/handlers/)
      |
  Store interface (PostgreSQL / MemoryStore)
      | (for infrastructure ops)
  hostvra/agent/pkg/* (direct OS syscalls)
      |
  OS / systemctl / ufw / certbot / php-fpm / nginx / etc.
```

IMPORTANT: The Node Agent communicates INBOUND to the API (heartbeat only).
The API calls hostvra/agent/pkg libraries DIRECTLY as a Go package import.
There is NO separate agent RPC protocol — agent is a library dependency of the API.

CRITICAL: The terminal handler (/api/v1/terminal/execute) executes ANY shell
command passed by an authenticated user with terminal.access permission.
There are NO command allowlists or denylists in the existing code.
The AI layer MUST NOT call this endpoint without command validation.

---

## 5. EXISTING TEST COVERAGE

### Go API Tests: 26 test files in handlers/
accounts, agent, auth, backups, billing, cron, dashboard, databases, docker,
domain_registrar, email, filemanager, files, firewall, ftp, installer, perf_bench,
php, settings, ssl, tickets, updates, waf, webmail, webserver, websites_isolation

### Go Agent Tests: 23 test files in pkg/
backup, database, ftp, php, ssl, webserver, firewall, and others

### Go Update Tests: 11 test files
deployer, e2e_lifecycle, job, measurable_thresholds, migrator, orchestrator,
rolling, scheduler, snapshot, verifier, version

### Store Tests: email_store_test.go

### CI Command: go test -v -race ./...

### Frontend Tests: NONE (build-only validation)
### Integration Tests: tests/ directory is EMPTY

---

## 6. MISSING COMPONENTS

| Component | Status | Notes |
|-----------|--------|-------|
| Integration test suite | MISSING | tests/ is empty |
| Frontend unit tests | MISSING | No Jest/Vitest config |
| E2E tests | MISSING | No Playwright/Cypress |
| Rate limiting middleware | MISSING | No implementation |
| Redis usage in Go | MISSING | In config but not imported |
| MCP server | MISSING | No existing MCP integration |
| AI/LLM layer | MISSING | No AI components |
| Command execution policy | MISSING | Terminal has no allowlist |
| .antigravity/ directory | MISSING | To be created |

---

## 7. INTEGRATION RISKS

| Risk | Severity | Mitigation |
|------|----------|-----------|
| System Python 3.9.6 does not satisfy SDK requirement | HIGH | Use .venv (Python 3.12) |
| AI calling terminal.execute with dangerous commands | CRITICAL | Strict command allowlist in AI layer |
| AI writing to production database directly | HIGH | Read-only default; approval gate for writes |
| AI modifying Go source during active development | HIGH | Approval gate + git checkpoint before writes |
| Active uncommitted changes in 10 files | Medium | Document and preserve; checkpoint before changes |
| .venv is untracked in git | Low | AI scripts must explicitly activate .venv |

### Uncommitted Changes (CONFIRMED by git status):

Modified files:
- apps/api/cmd/server/main.go
- apps/api/internal/handlers/email.go
- apps/api/internal/handlers/files.go
- apps/api/internal/handlers/websites.go
- apps/api/internal/store/email_models.go
- apps/api/internal/store/email_store.go
- apps/api/internal/store/store.go
- apps/web/src/app/email/page.tsx
- apps/web/src/app/websites/page.tsx
- apps/web/src/components/WebmailClient.tsx

Untracked files:
- .venv/
- apps/agent/pkg/email/
- apps/api/internal/handlers/webmail.go
- apps/api/internal/handlers/webmail_test.go
- apps/api/internal/store/migrations/0009_email_enterprise_production_schema.sql
- migrations/0009_email_enterprise_production_schema.sql

These represent in-progress email/webmail development. The AI layer MUST NOT disrupt this.

---

## 8. RECOMMENDED IMPLEMENTATION PLAN

### Phase 1: COMPLETE — Repository Audit (no code changes)

### Phase 2: Antigravity SDK Integration Design
- Create .antigravity/ directory structure
- Use existing .venv (Python 3.12 with SDK 0.1.18 already installed)
- Document GEMINI_API_KEY requirement

### Phase 3: Skills Architecture
- SKILL.md describing real Hostvra architecture
- Skills: hostvra-architecture, go-backend, linux-hosting, database, frontend, security, zero-demo-audit

### Phase 4: Agent Architecture
- Orchestrator -> Architect, Backend, Frontend, Linux, Database, Security, QA agents
- Each agent with tight SDK permissions via SubagentConfig

### Phase 5: Tool/MCP Gateway
- Python tools wrapping Hostvra API (read-only by default)
- Git tools (status, log, diff — read-only)
- Test runner tools (non-destructive)
- Command execution with allowlist/denylist enforcement

### Phase 6: Permission & Command Safety
- YAML policy files: read-only.yaml, development.yaml, production.yaml
- Python CommandSafetyEngine with hardcoded blocked patterns
- Allowlists enforced programmatically — NOT by AI instruction alone

### Phase 7: Human Approval System
- approval.py — interactive gate for write/execute/production ops
- Shows: operation, risk, affected files/services/commands, rollback plan

### Phase 8: Evidence & Audit System
- evidence.py — timestamped JSON evidence records per task
- Stored in .antigravity/reports/ — never deleted on failure

### Phase 9: Testing
- QA agent runs go test -count=1 ./... in apps/api and apps/agent
- Reports test results in evidence record

### Phase 10: Production-Readiness Validation
- All acceptance gates verified
- ANTIGRAVITY-HOSTVRA-IMPLEMENTATION-REPORT.md generated

---

## PREFLIGHT CONCLUSION

Hostvra is a production-grade enterprise hosting control panel with:
- ~200,000+ lines of Go across API + Agent
- 9 database migrations covering 40+ tables
- 34 frontend pages covering all hosting primitives
- 60+ test files
- Existing RBAC, audit logging, JWT auth, production config guards

Primary risks for AI integration:
1. CRITICAL: Terminal handler executes arbitrary shell — AI must not use without validation
2. HIGH: Active uncommitted development must be preserved
3. HIGH: Python venv must be used (not system Python 3.9.6)

Current branch: main (up to date with origin/main)
Last commit: 24af9a4 feat(filemanager): HostVra Enterprise File Manager v3.0

PHASE 1 COMPLETE. READY FOR PHASE 2-10 IMPLEMENTATION.
