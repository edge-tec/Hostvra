#!/usr/bin/env bash
# ==============================================================================
# Hostvra.com - Clean Uninstaller Script
# ==============================================================================

set -euo pipefail

if [[ $EUID -ne 0 ]]; then
    echo "[ERROR] Uninstaller must be run as root (sudo bash uninstall.sh)." >&2
    exit 1
fi

echo "======================================================================"
echo "          ⚠️ WARNING: HOSTVRA UNINSTALLATION PROCEDURE ⚠️"
echo "======================================================================"
echo "This will stop Hostvra API and Hostvra Agent daemon services,"
echo "remove systemd service units, and delete installed binaries."
echo "======================================================================"

read -p "Are you sure you want to proceed? (y/N): " -r CONFIRM
if [[ ! $CONFIRM =~ ^[Yy]$ ]]; then
    echo "Uninstallation cancelled."
    exit 0
fi

echo "[INFO] Stopping and disabling Hostvra systemd services..."
systemctl stop hostvra-api.service 2>/dev/null || true
systemctl stop hostvra-agent.service 2>/dev/null || true
systemctl disable hostvra-api.service 2>/dev/null || true
systemctl disable hostvra-agent.service 2>/dev/null || true

echo "[INFO] Removing systemd service units..."
rm -f /etc/systemd/system/hostvra-api.service
rm -f /etc/systemd/system/hostvra-agent.service
systemctl daemon-reload

echo "[INFO] Removing binaries..."
rm -f /usr/local/bin/hostvra-api
rm -f /usr/local/bin/hostvra-agent

read -p "Do you also wish to permanently delete database and configurations in /var/lib/hostvra? (y/N): " -r DEL_DATA
if [[ $DEL_DATA =~ ^[Yy]$ ]]; then
    rm -rf /var/lib/hostvra
    rm -rf /etc/hostvra
    rm -rf /var/log/hostvra
    echo "[INFO] Hostvra data directories removed."
else
    echo "[INFO] Hostvra data preserved at /var/lib/hostvra and /etc/hostvra."
fi

echo "[SUCCESS] Hostvra has been successfully uninstalled from this server."
