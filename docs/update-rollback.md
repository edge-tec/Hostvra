# Hostvra Emergency Rollback Runbook

This runbook documents the procedures for rolling back the Hostvra Control Plane and Agent fleet to a previous stable state.

---

## 1. Rollback Architecture

Hostvra uses a zero-copy atomic rollback strategy:
1. **Release Retention**: Previous release directories in `/opt/hostvra/releases/<version>` are never deleted during an update.
2. **Atomic Pointer**: The active release is a POSIX symlink at `/opt/hostvra/current`.
3. **Database Safeguards**: Pre-update snapshots preserve database tables and configuration files in compressed SHA-256-verified archives under `/var/lib/hostvra/backups/`.

---

## 2. Triggering a Rollback

### Method 1: Web UI
1. Navigate to **Settings > System Updates** (`/settings/updates`).
2. Click the red **Rollback** button in the top-right toolbar.
3. Review the confirmation dialog and click **Confirm Rollback**.
4. The system will switch back to the previous release and reload services.

### Method 2: Command Line (Fastest)
```bash
# Non-interactive CLI rollback
sudo hostvra update rollback -y
```

### Method 3: Manual Low-Level Recovery (Emergency / Disaster Recovery)
If the API daemon cannot be contacted:

```bash
# 1. Identify previous release directory
ls -la /opt/hostvra/releases/

# 2. Atomically point symlink to previous release
sudo ln -sfn /opt/hostvra/releases/1.0.0 /opt/hostvra/current

# 3. Restore config snapshot if modified
LATEST_BACKUP=$(ls -t /var/lib/hostvra/backups/pre-upgrade-*.tar.gz | head -n 1)
if [[ -n "$LATEST_BACKUP" ]]; then
    sudo tar -xzf "$LATEST_BACKUP" -C /
fi

# 4. Restart services
sudo systemctl daemon-reload
sudo systemctl restart hostvra-api hostvra-agent

# 5. Verify health
curl -I http://127.0.0.1:8080/health
```

---

## 3. Post-Rollback Audit

After rolling back:
1. Check the audit logs in the Web UI under **Audit Logs** (`/audit-logs`) or via API:
   ```bash
   hostvra update history
   ```
2. Verify that customer websites, DNS zones, and email services remain uninterrupted.
