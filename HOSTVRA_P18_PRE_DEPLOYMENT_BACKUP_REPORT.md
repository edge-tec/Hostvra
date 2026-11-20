# HOSTVRA — PHASE P18.3: PRE-DEPLOYMENT BACKUP REPORT

**UTC Timestamp:** `2026-11-20T14:39:23Z`  
**Status:** **LOCAL SCRIPT VERIFIED / LIVE VPS BACKUP PENDING OPERATOR SSH EXECUTION**

---

## 1. Automated Pre-Deployment Backup Procedure (`update.sh`)
- `update.sh` automatically creates `/var/backups/hostvra/pre-update-YYYYMMDD-HHMMSS/` before modifying any Git commit or binary.
- Preserves: `git-commit.txt`, `service-states.txt`, `/usr/local/bin/hostvra-api`, `/usr/local/bin/hostvra-agent`, `/etc/hostvra/hostvra.env` (`0600`), `/var/lib/hostvra/store.json`, `web-next.tar.gz`, and active PostgreSQL/MariaDB dumps.
- Preserves `/var/lib/hostvra/vault/hd_master.enc` (`0600`) untouched.
- Enforces latest-5 backup retention policy.
- **Live Execution Status:** Blocked until operator runs `bash update.sh` inside `root@hostvra`.
