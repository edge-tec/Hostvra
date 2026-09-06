# Hostvra Live Update Troubleshooting & Recovery Guide

This guide covers resolution steps for potential failure modes during automated or manual upgrades.

---

## 1. Quick Diagnostics

To diagnose an update issue on the command line:

```bash
# 1. Check current system status
hostvra update status

# 2. View recent update execution logs
hostvra update history

# 3. Inspect systemd journal logs
journalctl -u hostvra-api -n 100 --no-pager
journalctl -u hostvra-agent -n 100 --no-pager
```

---

## 2. Common Scenarios and Fixes

### Scenario A: "Update locked: another update job is currently in progress"

**Cause**: An update process crashed or was killed while holding the state lock.

**Fix**:
1. Check if the previous job is truly dead:
   ```bash
   ps aux | grep -E "hostvra-api|hostvra-agent"
   ```
2. Trigger rollback or clear the stale lock via CLI:
   ```bash
   hostvra update rollback -y
   ```

---

### Scenario B: "Signature verification failed" / "Ed25519 signature invalid"

**Cause**: Release package corrupted during download or tampered with in transit.

**Fix**:
1. The update engine automatically halts at the `VERIFYING` step and makes NO filesystem changes.
2. Verify system clock:
   ```bash
   timedatectl
   ```
3. Re-run `hostvra update install` to trigger a clean re-download.

---

### Scenario C: Post-Upgrade Health Probe Failed

**Cause**: New binary failed smoke tests (e.g. port collision, missing library, or bad config syntax).

**Automated Action**:
The orchestrator automatically reverts the symlink `/opt/hostvra/current` back to the prior known-good version without administrator intervention.

**Manual Verification**:
```bash
ls -la /opt/hostvra/current
curl -I http://127.0.0.1:8080/health
```

---

### Scenario D: Restoring from Pre-Upgrade Safety Snapshot

If manual restoration of `/etc/hostvra` and database state is required:

```bash
# Locate most recent pre-upgrade snapshot
ls -lt /var/lib/hostvra/backups/pre-upgrade-*.tar.gz | head -n 1

# Extract snapshot to root filesystem
sudo tar -xzf /var/lib/hostvra/backups/pre-upgrade-<timestamp>.tar.gz -C /

# Restart services
sudo systemctl restart hostvra-api hostvra-agent
```
