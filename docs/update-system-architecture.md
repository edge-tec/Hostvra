# Hostvra Update System Architecture & System Audit

**Document Status**: Production Architecture Specification  
**Version**: 1.0.0  
**Target Release**: Hostvra Live Update System (v1.0.0 → Future Updates)

---

## 1. Executive Summary & Existing System Audit

This document establishes the architectural baseline for the **Hostvra Live Update System**. Before modifying any production behavior or introducing automated updates, this audit details the existing repository layout, component boundaries, version detection hooks, persistent data paths, and potential migration risks.

### Current System Identifiers
* **Current Hostvra API Version**: `1.0.0` (defined in `apps/api/cmd/server/main.go:AppVersion`)
* **Current Agent Version**: `1.0.0` (defined in `apps/agent/cmd/agent/main.go:AgentVersion`)
* **Current Web Dashboard Version**: `1.0.0` (defined in `apps/web/package.json`)
* **Current Database Schema Version**: `4` (`migrations/0001_initial_schema.sql` through `migrations/0004_webserver_management_schema.sql`)
* **Current Binary Installation Paths**:
  * Control Plane API: `/usr/local/bin/hostvra-api`
  * Node Daemon Agent: `/usr/local/bin/hostvra-agent`
* **Current Configuration Paths**:
  * API Environment File: `/etc/hostvra/api.env`
  * Agent Configuration: `/etc/hostvra/agent.json`
* **Current Persistent Data Paths**:
  * PostgreSQL DB / In-memory data store: `/var/lib/hostvra/`
  * Updates & Backups staging: `/var/lib/hostvra/updates_backup/`
* **Current Logging Paths**:
  * System logs: `/var/log/hostvra/`
* **Current Active Systemd Units**:
  * `hostvra-api.service`
  * `hostvra-agent.service`

---

## 2. Component Inventory & Reusability Matrix

| Component | Current State | Reusability in Live Update | Required Additions |
| :--- | :--- | :--- | :--- |
| **Installer** (`deployment/installer/install.sh`) | Shell script for fresh installation | Reusable for binary placement and dependency checks | Upgrade-awareness: must detect existing installs and prevent directory wiping |
| **Existing Updater** (`deployment/installer/update.sh`) | Primitive binary copy and `systemctl restart` | Serves as emergency fallback script | Complete replacement by daemon-level stateful update engine with Ed25519 signature checks |
| **Backup System** (`apps/api/internal/handlers/backups.go`) | Database and filesystem snapshot manager | Fully reusable as pre-update safety checkpoint | Hook into update lifecycle to mandate snapshot verification before stage 1 |
| **Database Migrations** (`migrations/*.sql`) | Raw numbered SQL files executed on startup | Reusable SQL assets | `database_migrations` tracking table with checksum validation and execution locking |
| **Agent Telemetry & RPC** (`apps/agent/internal/client/`) | Heartbeat & metrics ingestion | Reusable for rolling fleet updates | Self-update staging routine in `/opt/hostvra-agent/releases/` |
| **RBAC Matrix** (`apps/api/internal/rbac/`) | Hierarchical permission engine with Super Admin / Admin / Operator | Reusable permission enforcement | New permissions: `system.update.view`, `system.update.check`, `system.update.start`, `system.update.rollback` |
| **Licensing** (`apps/api/internal/license/`) | Community, Pro, Enterprise tiered features | Entitlement checks | Enforce channel eligibility (e.g. Early Beta / Nightly access for registered licenses) |

---

## 3. Risks & Safety Guarantees

### 3.1. Compatibility Risks
* **OS & Toolchain Matrix**:
  * Supported: Ubuntu 22.04 LTS, Ubuntu 24.04 LTS, Debian 12 (amd64 and arm64).
  * Incompatibility Risk: Node agents on outdated glibc or kernel versions.
  * *Mitigation*: Update manifest contains strict `os_compatibility` and `arch` constraints tested before extraction.

### 3.2. Migration Risks
* **Database Deadlocks & Schema Drift**:
  * Risk: Long-running database updates blocking control plane transactions.
  * *Mitigation*: Migrations run inside explicit transactions with statement timeouts and pre-migration database snapshot.

### 3.3. Rollback Risks
* **Incomplete Rollback / Orphaned Binaries**:
  * Risk: Target binary fails to execute, but previous binary was overwritten.
  * *Mitigation*: Atomic directory structure using `/opt/hostvra/releases/<version>/` and atomic symlink `/opt/hostvra/current`. The active binary is never directly overwritten.

---

## 4. Architectural Target Model

```
+-------------------------------------------------------------+
|                      Hostvra Web Panel                      |
|            Settings > System Updates (Realtime UI)          |
+------------------------------+------------------------------+
                               | HTTPS / WebSocket
+------------------------------v------------------------------+
|                 Hostvra Core Control Plane API              |
|   +-----------------------------------------------------+   |
|   | Update Job Engine (Persistent State Machine)        |   |
|   | - Manifest Retriever & Ed25519 Verification        |   |
|   | - Semver Comparator (Stable / Beta / Nightly)       |   |
|   | - Pre-Update Backup Orchestrator                    |   |
|   | - DB Migration Framework with Execution Locks       |   |
|   | - Health Check & Smoke Test Engine                  |   |
|   | - Automatic Rollback Coordinator                    |   |
|   +-----------------------------------------------------+   |
+------------------------------+------------------------------+
                               | gRPC / HTTPS Agent Heartbeat
+------------------------------v------------------------------+
|                    Hostvra Linux Agent                      |
|   /opt/hostvra-agent/releases/<version>/                    |
|   - Staged binary validation                                |
|   - Zero-disruption to customer sites (Nginx, PHP, MySQL)   |
|   - Automatic watchdog recovery                             |
+-------------------------------------------------------------+
```

---

## 5. Phase 0 Exit Criteria
- Complete codebase and directory audit performed.
- All version points, configuration paths, and systemd units documented.
- Zero production disruption occurred during the audit phase.
- Ready to proceed to **Phase 1: Version & Release Management**.
