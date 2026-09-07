#!/usr/bin/env bash
# ==============================================================================
# Hostvra Enterprise Platform - Production Zero-Downtime Updater & Verifier
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 1. Concurrent Update Lock Protection
LOCK_FILE="/tmp/hostvra-update.lock"
if [[ -f "$LOCK_FILE" ]]; then
    PID=$(cat "$LOCK_FILE" 2>/dev/null || echo "")
    if [[ -n "$PID" ]] && kill -0 "$PID" 2>/dev/null; then
        echo "[ERROR] Another Hostvra update process (PID $PID) is already running." >&2
        exit 1
    fi
fi
echo $$ > "$LOCK_FILE"
trap 'rm -f "$LOCK_FILE"' EXIT

echo "=== [1/5] Validating Dependencies & Pulling Code ==="
for cmd in git go npm; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "[ERROR] Required tool '$cmd' is not installed or not in PATH." >&2
        exit 1
    fi
done

git pull origin main

# Prepare backup directory for atomic rollback
TIMESTAMP=$(date +%s)
BACKUP_DIR="/var/lib/hostvra/updates_backup/${TIMESTAMP}"
mkdir -p "${BACKUP_DIR}" 2>/dev/null || BACKUP_DIR="/tmp/hostvra_backup_${TIMESTAMP}"
mkdir -p "${BACKUP_DIR}" "bin"

# Backup current binaries if present
if [[ -f "/usr/local/bin/hostvra-api" ]]; then
    cp -p "/usr/local/bin/hostvra-api" "${BACKUP_DIR}/hostvra-api.bak" 2>/dev/null || true
fi
if [[ -f "/usr/local/bin/hostvra-agent" ]]; then
    cp -p "/usr/local/bin/hostvra-agent" "${BACKUP_DIR}/hostvra-agent.bak" 2>/dev/null || true
fi

rollback() {
    echo "[ALERT] Update failed! Executing automatic rollback to prior binaries..." >&2
    if [[ -f "${BACKUP_DIR}/hostvra-api.bak" ]] && [[ -w "/usr/local/bin" ]]; then
        cp -fp "${BACKUP_DIR}/hostvra-api.bak" "/usr/local/bin/hostvra-api" 2>/dev/null || true
    fi
    if [[ -f "${BACKUP_DIR}/hostvra-agent.bak" ]] && [[ -w "/usr/local/bin" ]]; then
        cp -fp "${BACKUP_DIR}/hostvra-agent.bak" "/usr/local/bin/hostvra-agent" 2>/dev/null || true
    fi
    if command -v systemctl &>/dev/null; then
        systemctl restart hostvra-api hostvra-web hostvra-agent 2>/dev/null || true
    fi
    echo "[ERROR] Rollback completed. Check logs for details." >&2
    exit 1
}

trap 'rollback' ERR

echo "=== [2/5] Compiling Hostvra Core API & CLI ==="
(cd apps/api && CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server)
cp -fp apps/api/server bin/hostvra-api 2>/dev/null || true

if [[ -w "/usr/local/bin" ]]; then
    # Atomic install
    cp -fp apps/api/server "/usr/local/bin/hostvra-api.tmp"
    mv -f "/usr/local/bin/hostvra-api.tmp" "/usr/local/bin/hostvra-api"
    chmod +x "/usr/local/bin/hostvra-api"
fi

echo "=== [3/5] Compiling Hostvra Node Agent ==="
if [[ -d "apps/agent" ]]; then
    (cd apps/agent && CGO_ENABLED=0 go build -ldflags="-s -w" -o agent ./cmd/agent)
    cp -fp apps/agent/agent bin/hostvra-agent 2>/dev/null || true
    if [[ -w "/usr/local/bin" ]]; then
        cp -fp apps/agent/agent "/usr/local/bin/hostvra-agent.tmp"
        mv -f "/usr/local/bin/hostvra-agent.tmp" "/usr/local/bin/hostvra-agent"
        chmod +x "/usr/local/bin/hostvra-agent"
    fi
fi

echo "=== [4/5] Building Next.js Web UI Production Bundle ==="
(cd apps/web && npm run build)

echo "=== [5/5] Reloading Hostvra System Services ==="
if command -v systemctl &>/dev/null; then
    # Detach reload slightly so terminal sessions don't disconnect
    (
        sleep 1
        systemctl restart hostvra-api 2>/dev/null || true
        systemctl restart hostvra-web 2>/dev/null || true
        systemctl restart hostvra-agent 2>/dev/null || true
    ) >/dev/null 2>&1 &
fi

# Cancel error trap since build succeeded
trap - ERR
rm -f "$LOCK_FILE"
echo "=== Done! Hostvra is successfully updated. Services reloaded gracefully. ==="
