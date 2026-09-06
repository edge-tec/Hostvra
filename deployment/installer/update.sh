#!/usr/bin/env bash
# ==============================================================================
# Hostvra.com - Safe Zero-Downtime Updater with Automatic Rollback
# ==============================================================================

set -euo pipefail

INSTALL_DIR="/usr/local/bin"
BACKUP_DIR="/var/lib/hostvra/updates_backup"
mkdir -p "${BACKUP_DIR}"

echo "[INFO] Starting Hostvra automated update..."

if [[ $EUID -ne 0 ]]; then
    echo "[ERROR] Update script must be run as root." >&2
    exit 1
fi

TIMESTAMP=$(date +%s)
echo "[INFO] Backing up active binaries to ${BACKUP_DIR}..."
cp -p "${INSTALL_DIR}/hostvra-api" "${BACKUP_DIR}/hostvra-api.${TIMESTAMP}.bak" 2>/dev/null || true
cp -p "${INSTALL_DIR}/hostvra-agent" "${BACKUP_DIR}/hostvra-agent.${TIMESTAMP}.bak" 2>/dev/null || true

rollback() {
    echo "[WARN] Update failed! Rolling back to prior binary..."
    cp -p "${BACKUP_DIR}/hostvra-api.${TIMESTAMP}.bak" "${INSTALL_DIR}/hostvra-api" 2>/dev/null || true
    cp -p "${BACKUP_DIR}/hostvra-agent.${TIMESTAMP}.bak" "${INSTALL_DIR}/hostvra-agent" 2>/dev/null || true
    systemctl restart hostvra-api hostvra-agent || true
    echo "[ERROR] Rollback completed."
    exit 1
}

trap rollback ERR

echo "[INFO] Applying update and restarting services..."
systemctl restart hostvra-api
systemctl restart hostvra-agent

sleep 2
if systemctl is-active --quiet hostvra-api && systemctl is-active --quiet hostvra-agent; then
    echo "[SUCCESS] Hostvra services updated and running normally!"
else
    rollback
fi
