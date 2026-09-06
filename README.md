# Hostvra — Modern Self-Hosted Cloud & Server Management Platform

<p align="center">
  <strong>Production-grade self-hosted VPS and dedicated server management platform.</strong><br>
  Built with Go 1.22+, Next.js 15, native Linux daemon agents, granular RBAC, and commercial cryptographic licensing.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/Next.js-15-black?style=flat&logo=next.js" alt="Next.js" />
  <img src="https://img.shields.io/badge/Architecture-Monorepo-blue?style=flat" alt="Monorepo" />
  <img src="https://img.shields.io/badge/Licensing-Ed25519%20Signed-emerald?style=flat" alt="Licensing" />
  <img src="https://img.shields.io/badge/Security-Zero%20Raw%20Exec-purple?style=flat" alt="Security" />
</p>

---

## 1. Overview

**Hostvra** (`hostvra.com`) is an enterprise-ready, open-core server and cloud hosting management platform inspired by tools such as aaPanel, cPanel, and CyberPanel, designed from scratch with modern systems engineering and security best practices:

- **Strict Separation of Concerns**: The central control plane (`apps/api` + `apps/web`) communicates with server agents (`apps/agent`) via authenticated, strictly-typed REST contracts.
- **Zero Insecure Shell Execution**: No arbitrary `POST /execute` or remote bash execution endpoints. Every server operation is a strongly typed, deterministic function with parameters validated before execution.
- **Atomic Configuration Staging with Safe Rollback**: Nginx vhost updates, PHP-FPM configurations, and restore operations are staged and validated (e.g. `nginx -t`) prior to promotion, with automatic rollback on failure.
- **Zero-Shell Telemetry Collection**: Server hardware telemetry (CPU delta, memory, load averages, disk usage, network I/O) is collected directly via `/proc` and `syscall.Statfs` without shelling out.
- **SSH Lockout Guard**: Firewall configurations (UFW / nftables) strictly forbid deleting or denying SSH (port 22) without explicit override.
- **Cryptographic Offline Licensing**: License keys are digitally signed using Ed25519, allowing verifiable air-gapped activation without continuous phone-home requirements.

---

## 2. Monorepo Architecture

```
hostvra/
├── apps/
│   ├── web/               # Next.js 15 App Router Frontend (23 Production Views, Tailwind CSS)
│   ├── api/               # Go Central Control Plane API (Chi Router, RBAC, JWT, Audit Logger)
│   └── agent/             # Go Native Linux Node Daemon (OS Abstraction, Hardware Telemetry)
├── bin/                   # Statically linked compiled binaries (linux/amd64 & linux/arm64)
├── deployment/
│   ├── installer/         # install.sh, update.sh (with rollback), uninstall.sh
│   └── systemd/           # Hardened systemd service units (hostvra-api, hostvra-agent)
├── migrations/            # Versioned SQL migrations for PostgreSQL
├── packages/
│   └── types/             # Shared contracts and type definitions
├── .github/workflows/     # CI/CD Pipeline (Go tests, Next.js build, Cross-compilation)
├── docker-compose.yml     # Local database and cache stack
├── Makefile               # Consolidated build, test, and run orchestration
└── README.md
```

---

## 3. Core Feature Matrix Across 7 Completed Phases

| Phase | Feature Domain | Key Implementation Details |
| :--- | :--- | :--- |
| **Phase 1** | **Foundation & Security** | PostgreSQL initial schema migration, Argon2id hashing, short-lived JWTs, dual-mode store (PostgreSQL + In-memory fallback), centralized audit logging. |
| **Phase 2** | **Native Agent & Telemetry** | OS Abstraction Layer (`DebianAdapter`, `RHELAdapter`), zero-shell `/proc` telemetry collector, agent handshake & permanent key exchange, systemd sandbox. |
| **Phase 3** | **Hosting & Server Ops** | Nginx virtual host generator with safe rollback transactions, PHP-FPM pool manager (PHP 8.4-8.1), ACME Let's Encrypt automated SSL, MySQL / Postgres DB manager. |
| **Phase 4** | **Runtime Features** | Sandboxed file manager with path traversal protection, 5-field cron manager, Docker controller, firewall manager with SSH lockout guard. |
| **Phase 5** | **Backups, DNS & Alerts** | Deterministic `.tar.gz` backups with smart exclusions, pre-restore verification, AWS SigV4 S3/R2 offsite sync, RFC authoritative DNS with BIND export, fleet alerting engine. |
| **Phase 6** | **Commercial & Teams** | Ed25519 cryptographic licensing (`Community`, `Pro`, `Enterprise`), multi-tenant team collaboration with RBAC, scoped API keys with one-time reveal. |
| **Phase 7** | **Production Deployment** | One-command curl installer (`curl -fsSL https://install.hostvra.com \| sudo bash`), safe zero-downtime updater with rollback, clean uninstaller, hardened systemd units. |

---

## 4. Production Installation (One-Command Deployment)

To install Hostvra on an Ubuntu, Debian, AlmaLinux, Rocky Linux, or RHEL server:

```bash
curl -fsSL https://install.hostvra.com | sudo bash
```

Or clone the repository and run:
```bash
sudo bash deployment/installer/install.sh
```

> **Detailed Installation Guidelines**:
> - [Complete Production Installation Guideline (English)](docs/installation-guide.md)
> - [সম্পূর্ণ প্রোডাকশন ইনস্টলেশন নির্দেশিকা (বাংলা)](docs/installation-guide-bn.md)

The installer automatically:
1. Detects OS and CPU architecture (`x86_64` or `aarch64/arm64`).
2. Creates an unprivileged system user `hostvra` with restricted home directory `/var/lib/hostvra`.
3. Generates cryptographic JWT secrets and random administrator passwords.
4. Registers and starts sandboxed systemd services (`hostvra-api` and `hostvra-agent`).
5. Configures the firewall (ports 22, 80, 443, 8080, and email stack 25, 465, 587, 993, 995).
6. Outputs a formatted completion banner with web panel URL and login credentials.

---

## 5. Local Development Quickstart

### Prerequisites
- **Go**: 1.22+ (`go version`)
- **Node.js**: 20+ (`node -v`)
- **Docker**: Optional for PostgreSQL (falls back to memory store automatically)

### Step 1: Clone and Test Monorepo
```bash
git clone https://github.com/hostvra/hostvra.git
cd hostvra

# Run full test suite with Go race detector and Next.js build
make test
```

### Step 2: Run Central API
```bash
# Starts on http://localhost:8080
make run-api
```

### Step 3: Run Web Control Panel
```bash
# Starts on http://localhost:3000
make run-web
```

### Step 4: Run Node Agent (Optional for local testing)
```bash
make run-agent
```

---

## 6. Build and Cross-Compilation

To compile native host binaries or statically linked standalone Linux binaries for deployment:

```bash
# Compile native binaries into bin/
make build

# Cross-compile for Linux (amd64 and arm64)
make compile-linux
```

Output binaries:
- `bin/hostvra-api-linux-amd64`
- `bin/hostvra-api-linux-arm64`
- `bin/hostvra-agent-linux-amd64`
- `bin/hostvra-agent-linux-arm64`

---

## 7. Web Control Panel Views (23 Routes)

- **Dashboard**: `/dashboard` — Fleet overview, real-time CPU/RAM/Disk metrics, active alerts.
- **Servers**: `/servers` — Fleet node inventory, enrollment tokens, node status.
- **Websites**: `/websites` — Virtual hosts, PHP version switcher, reverse proxy, SSL status.
- **Databases**: `/databases` — MySQL, MariaDB, and PostgreSQL databases and users.
- **SSL Certificates**: `/ssl` — Certificate inventory, Let's Encrypt issuance, expiration tracker.
- **File Manager**: `/files` — Sandboxed browser, code editor, permissions.
- **Cron Jobs**: `/cron` — Crontab manager, 5-field syntax validator, run-now action.
- **Docker**: `/docker` — Containers list, lifecycle controls, live logs.
- **Firewall**: `/firewall` — UFW/nftables rule configuration with SSH lockout guard.
- **Backups**: `/backups` — Deterministic tar.gz snapshots, S3 remote sync, atomic rollback.
- **DNS Zones**: `/dns` — A/AAAA/CNAME/MX records, Cloudflare proxy toggle, BIND zone export.
- **Alerts**: `/alerts` — Hardware threshold rules, active incidents, webhook notifications.
- **Team**: `/team` — Team members roster, RBAC invitations (`Admin`, `Manager`, `Developer`, `Viewer`).
- **API Keys**: `/api-keys` — Programmatic automation tokens, one-time secret reveal.
- **Licensing**: `/license` — Ed25519 cryptographic key activation, entitlements checklist.
- **Audit Logs**: `/audit-logs` — Immutable audit trail with IP, user, and action details.
- **Settings**: `/settings` — System parameters, branding, security configurations.
- **Auth**: `/login`, `/register`.

---

## 8. License

This repository is licensed under the Apache License 2.0. See [LICENSE](LICENSE) for details.
