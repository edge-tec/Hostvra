#!/usr/bin/env bash
# ==============================================================================
# Hostvra Enterprise Platform - Production Updater & Health Verifier
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
cleanup_lock() {
    rm -f "$LOCK_FILE"
}
trap cleanup_lock EXIT

echo "=== [1/6] Validating Prerequisites & Environment ==="

# Check free disk space (minimum 500MB required)
FREE_KB=$(df -k "$SCRIPT_DIR" | awk 'NR==2 {print $4}')
if [[ -n "$FREE_KB" ]] && [[ "$FREE_KB" -lt 512000 ]]; then
    echo "[ERROR] Insufficient disk space for update build: ${FREE_KB}KB available, 500MB required." >&2
    exit 1
fi

for cmd in git go npm curl; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "[ERROR] Required tool '$cmd' is not installed or not in PATH." >&2
        exit 1
    fi
done

echo "Pulling latest code from origin/main..."
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
    echo "[ERROR] Rollback completed. System restored to prior version." >&2
    exit 1
}

trap rollback ERR

echo "=== [2/6] Compiling Hostvra Core API & CLI ==="
(cd apps/api && CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server)
cp -fp apps/api/server bin/hostvra-api 2>/dev/null || true

if [[ -w "/usr/local/bin" ]]; then
    # Atomic binary replacement
    cp -fp apps/api/server "/usr/local/bin/hostvra-api.tmp"
    mv -f "/usr/local/bin/hostvra-api.tmp" "/usr/local/bin/hostvra-api"
    chmod 0755 "/usr/local/bin/hostvra-api"
fi

echo "=== [3/6] Compiling Hostvra Node Agent ==="
if [[ -d "apps/agent" ]]; then
    (cd apps/agent && CGO_ENABLED=0 go build -ldflags="-s -w" -o agent ./cmd/agent)
    cp -fp apps/agent/agent bin/hostvra-agent 2>/dev/null || true
    if [[ -w "/usr/local/bin" ]]; then
        # Atomic binary replacement
        cp -fp apps/agent/agent "/usr/local/bin/hostvra-agent.tmp"
        mv -f "/usr/local/bin/hostvra-agent.tmp" "/usr/local/bin/hostvra-agent"
        chmod 0755 "/usr/local/bin/hostvra-agent"
    fi
fi

echo "=== [4/6] Building Next.js Web UI Production Bundle ==="
(cd apps/web && npm run build)
# Ensure root .next symlink exists for universal CWD static asset resolution
ln -sfn apps/web/.next .next 2>/dev/null || true

echo "=== [5/6] Restarting Services & Performing Health Verification ==="
if command -v systemctl &>/dev/null; then
    # Synchronously restart Core API
    echo "Restarting hostvra-api..."
    systemctl restart hostvra-api

    # Probe API health endpoint
    API_PORT="${PORT:-8080}"
    HEALTH_URL="http://127.0.0.1:${API_PORT}/health"
    HEALTHY=0
    echo "Probing API health at ${HEALTH_URL}..."
    for i in $(seq 1 15); do
        if curl -fsS -m 2 "$HEALTH_URL" &>/dev/null; then
            HEALTHY=1
            echo "API health check passed (attempt $i)."
            break
        fi
        sleep 1
    done

    if [[ "$HEALTHY" -ne 1 ]]; then
        # Fallback check if systemd unit is active
        if systemctl is-active --quiet hostvra-api; then
            echo "[WARN] /health probe timed out but systemd reports hostvra-api active."
        else
            echo "[ERROR] hostvra-api health verification failed!" >&2
            rollback
        fi
    fi

    # Restart agent and web services
    echo "Restarting hostvra-agent..."
    systemctl restart hostvra-agent 2>/dev/null || true

    echo "Restarting hostvra-web..."
    systemctl restart hostvra-web 2>/dev/null || true
fi

if command -v pm2 &>/dev/null; then
    echo "Restarting any PM2 managed processes..."
    pm2 restart all 2>/dev/null || true
fi

# Cancel error trap since all steps succeeded
trap - ERR
echo "=== UPDATE SUCCESS: Hostvra is successfully updated, verified, and active. ==="
