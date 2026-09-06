# Hostvra — Modern Self-Hosted Cloud & Server Management Platform

<p align="center">
  <strong>Production-grade self-hosted VPS and dedicated server management platform.</strong><br>
  Independent architecture, native Go agent, modern Next.js control plane, granular RBAC, and commercial licensing.
</p>

---

## 1. Overview

**Hostvra** (`hostvra.com`) is an open-core, enterprise-ready server control panel designed for developers, systems engineers, and cloud hosting providers. Inspired by platforms like aaPanel and cPanel, Hostvra is re-imagined from the ground up with:

- **Strict Separation of Concerns**: Central control plane (`apps/api` + `apps/web`) communicates via authenticated REST/WebSocket with the lightweight native Linux agent (`apps/agent`).
- **Zero Insecure Shell Execution**: No raw `POST /execute` endpoints. All operations are strictly typed contracts (`CreateWebsite`, `ConfigureFirewall`, `RestartNginx`).
- **Safe State Transitions**: Atomic configuration staging, syntax validation, automated backup, and automatic rollback on health check failure.
- **Local Node Autonomy**: Server agents continue serving traffic, managing databases, and executing cron jobs even during temporary control plane outages.
- **Granular RBAC**: Multi-tenant organizations with strict role isolation (`Owner`, `Admin`, `Manager`, `Developer`, `Viewer`).

---

## 2. Monorepo Structure

```
hostvra/
├── apps/
│   ├── web/               # Next.js 15 App Router Frontend (React, TypeScript, Tailwind)
│   ├── api/               # Go Central Control Plane API (Chi, JWT, Argon2id, RBAC)
│   └── agent/             # Go Node Daemon (Runs on managed Linux servers)
├── packages/
│   ├── types/             # Shared data types and contracts
│   ├── api-client/        # Generated API client
│   └── ui/                # Shared design system components
├── deployment/
│   ├── docker/            # Docker Compose production and dev stacks
│   ├── installer/         # install.sh, uninstall.sh, update.sh
│   └── systemd/           # hostvra-agent.service
├── migrations/            # Versioned SQL migrations (PostgreSQL)
├── docs/                  # In-depth architectural and operational guides
├── Makefile               # Streamlined build and test orchestration
├── docker-compose.yml     # Local dev database and cache stack
└── README.md
```

---

## 3. Quick Start (Development)

### Prerequisites
- **Node.js**: v20+ (`node -v`)
- **Go**: v1.22+ (`go version`)
- **Docker & Docker Compose** (for PostgreSQL and Redis)

### Step 1: Start Database Stack
```bash
make docker-up
```

### Step 2: Run Go API Server
```bash
# Starts on port 8080 by default
make run-api
```

### Step 3: Run Next.js Web Control Panel
```bash
# Starts on port 3000
make run-web
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

---

## 4. Testing & Verification

Run the entire automated test suite:
```bash
make test
```

Or individual suites:
```bash
# Test Go API (Argon2id, JWT, RBAC permission matrix)
make test-api

# Build Next.js production bundle
make build-web
```

---

## 5. Security & Authentication Model

1. **Password Hashing**: Argon2id with 64MB memory, 3 iterations, 2 threads, 16-byte random salt.
2. **Access Tokens**: Short-lived HMAC-SHA256 JWT tokens.
3. **Refresh Tokens**: Opaque 32-byte cryptographic random tokens stored exclusively as SHA256 hashes.
4. **Server Enrollment**: Single-use, time-limited cryptographic enrollment tokens exchanged for permanent mutual server tokens.
5. **Audit Trail**: Every administrative mutation records user ID, IP address, user agent, target resource, and status.

---

## 6. License

Licensed under the Apache License, Version 2.0. See [LICENSE](file:///Users/mizanurrahman/claude/Hostvra/LICENSE) for details.
