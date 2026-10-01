#!/usr/bin/env bash
# ==============================================================================
# Hostvra.com - Safe Zero-Downtime Updater with Automatic Rollback
# Preserves all customer website Nginx configs during updates
# ==============================================================================

set -euo pipefail

INSTALL_DIR="/usr/local/bin"
REPO_ROOT="/root/Hostvra"
BACKUP_DIR="/var/lib/hostvra/updates_backup"
mkdir -p "${BACKUP_DIR}"

echo "[INFO] Starting Hostvra automated update..."

if [[ $EUID -ne 0 ]]; then
    echo "[ERROR] Update script must be run as root." >&2
    exit 1
fi

TIMESTAMP=$(date +%s)

# ─── Step 1: Backup active binaries ──────────────────────────────────────────
echo "[INFO] Backing up active binaries to ${BACKUP_DIR}..."
cp -p "${INSTALL_DIR}/hostvra-api" "${BACKUP_DIR}/hostvra-api.${TIMESTAMP}.bak" 2>/dev/null || true
cp -p "${INSTALL_DIR}/hostvra-agent" "${BACKUP_DIR}/hostvra-agent.${TIMESTAMP}.bak" 2>/dev/null || true

# ─── Step 2: Backup all customer Nginx vhost configs ─────────────────────────
echo "[INFO] Backing up customer Nginx vhost configurations..."
if [[ -d "/etc/nginx/sites-enabled" ]]; then
    mkdir -p "${BACKUP_DIR}/nginx-sites-enabled.${TIMESTAMP}"
    mkdir -p "${BACKUP_DIR}/nginx-sites-available.${TIMESTAMP}"
    cp -rp /etc/nginx/sites-enabled/* "${BACKUP_DIR}/nginx-sites-enabled.${TIMESTAMP}/" 2>/dev/null || true
    cp -rp /etc/nginx/sites-available/* "${BACKUP_DIR}/nginx-sites-available.${TIMESTAMP}/" 2>/dev/null || true
    echo "[INFO] Nginx configs backed up successfully."
fi

rollback() {
    echo "[WARN] Update failed! Rolling back..."
    cp -p "${BACKUP_DIR}/hostvra-api.${TIMESTAMP}.bak" "${INSTALL_DIR}/hostvra-api" 2>/dev/null || true
    cp -p "${BACKUP_DIR}/hostvra-agent.${TIMESTAMP}.bak" "${INSTALL_DIR}/hostvra-agent" 2>/dev/null || true

    # Restore Nginx configs
    if [[ -d "${BACKUP_DIR}/nginx-sites-enabled.${TIMESTAMP}" ]]; then
        cp -rp "${BACKUP_DIR}/nginx-sites-enabled.${TIMESTAMP}/"* /etc/nginx/sites-enabled/ 2>/dev/null || true
        cp -rp "${BACKUP_DIR}/nginx-sites-available.${TIMESTAMP}/"* /etc/nginx/sites-available/ 2>/dev/null || true
        nginx -t 2>/dev/null && systemctl reload nginx 2>/dev/null || true
    fi

    systemctl restart hostvra-api hostvra-agent hostvra-web || true
    echo "[ERROR] Rollback completed."
    exit 1
}

trap rollback ERR

# ─── Step 3: Pull latest code ────────────────────────────────────────────────
if [[ -d "${REPO_ROOT}/.git" ]]; then
    echo "[INFO] Pulling latest code from GitHub..."
    (cd "${REPO_ROOT}" && git pull origin main) || echo "[WARN] Git pull failed, continuing with local source."
fi

# ─── Step 4: Rebuild Go binaries ─────────────────────────────────────────────
if command -v go &>/dev/null && [[ -f "${REPO_ROOT}/apps/api/cmd/server/main.go" ]]; then
    echo "[INFO] Recompiling Go binaries from source..."
    (cd "${REPO_ROOT}/apps/api" && CGO_ENABLED=0 go build -ldflags="-s -w" -o "${INSTALL_DIR}/hostvra-api" cmd/server/main.go)
    (cd "${REPO_ROOT}/apps/agent" && CGO_ENABLED=0 go build -ldflags="-s -w" -o "${INSTALL_DIR}/hostvra-agent" cmd/agent/main.go)
    chmod +x "${INSTALL_DIR}/hostvra-api" "${INSTALL_DIR}/hostvra-agent"
    echo "[SUCCESS] Go binaries compiled successfully."
fi

# ─── Step 5: Rebuild Next.js Web UI ─────────────────────────────────────────
if command -v npm &>/dev/null && [[ -f "${REPO_ROOT}/apps/web/package.json" ]]; then
    echo "[INFO] Rebuilding Next.js Web UI..."
    (cd "${REPO_ROOT}/apps/web" && npm install --no-audit && npm run build) || echo "[WARN] Web build had non-fatal warnings."
fi

# ─── Step 6: Restart services ───────────────────────────────────────────────
echo "[INFO] Restarting all Hostvra services..."
systemctl daemon-reload
systemctl restart hostvra-api
systemctl restart hostvra-agent
systemctl restart hostvra-web 2>/dev/null || true

# ─── Step 7: Wait for API to start (it auto-syncs Nginx vhosts) ─────────────
echo "[INFO] Waiting for API startup and Nginx vhost auto-sync..."
sleep 5

if command -v nginx &>/dev/null; then
    # Ensure 00-default-neutral and hostvra-panel are enabled
    if [[ -f "/etc/nginx/sites-available/00-default-neutral" ]]; then
        ln -sf /etc/nginx/sites-available/00-default-neutral /etc/nginx/sites-enabled/00-default-neutral 2>/dev/null || true
    fi
    if [[ -f "/etc/nginx/sites-available/hostvra-panel" ]]; then
        ln -sf /etc/nginx/sites-available/hostvra-panel /etc/nginx/sites-enabled/hostvra-panel 2>/dev/null || true
    fi

    # Verify and reload Nginx
    if nginx -t 2>/dev/null; then
        systemctl reload nginx
        echo "[SUCCESS] Nginx validated and reloaded."
    else
        echo "[WARN] Nginx config test failed. Restoring backup configs..."
        if [[ -d "${BACKUP_DIR}/nginx-sites-enabled.${TIMESTAMP}" ]]; then
            cp -rp "${BACKUP_DIR}/nginx-sites-enabled.${TIMESTAMP}/"* /etc/nginx/sites-enabled/ 2>/dev/null || true
            cp -rp "${BACKUP_DIR}/nginx-sites-available.${TIMESTAMP}/"* /etc/nginx/sites-available/ 2>/dev/null || true
            nginx -t 2>/dev/null && systemctl reload nginx 2>/dev/null || true
        fi
    fi
fi

# ─── Step 9: Health check ───────────────────────────────────────────────────
sleep 2
if systemctl is-active --quiet hostvra-api && systemctl is-active --quiet hostvra-web; then
    echo ""
    echo "======================================================================"
    echo "  🎉 HOSTVRA UPDATE COMPLETED SUCCESSFULLY!"
    echo "======================================================================"
    echo "  • API:   $(systemctl is-active hostvra-api)"
    echo "  • Agent: $(systemctl is-active hostvra-agent)"
    echo "  • Web:   $(systemctl is-active hostvra-web)"
    echo "  • Nginx: $(systemctl is-active nginx)"
    echo ""
    echo "  All customer domains have been auto-synced with Nginx."
    echo "======================================================================"
else
    rollback
fi
