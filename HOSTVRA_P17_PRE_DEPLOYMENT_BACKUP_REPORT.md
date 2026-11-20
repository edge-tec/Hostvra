# HOSTVRA — PHASE P17.3: PRE-DEPLOYMENT BACKUP REPORT

**UTC Timestamp:** `2026-11-20T14:25:47Z`  
**Verification Method:** Static & automated inspection of `update.sh` + local on-disk backup/restore drill  
**Status:** **LOCAL SCRIPT & DR VERIFIED / LIVE VPS BACKUP EXECUTION BLOCKED ON SSH**

---

## 1. Automated Pre-Deployment Backup Coverage (`update.sh`)
Verified in `/Volumes/EDGE/Hostvra-main/update.sh`:
- Creates `/var/backups/hostvra/pre-update-YYYYMMDD-HHMMSS` **before** `git reset --hard origin/main`.
- Backs up `git-commit.txt`, `service-states.txt`, `/usr/local/bin/hostvra-api`, `/usr/local/bin/hostvra-agent`, `/etc/hostvra/hostvra.env`, `/var/lib/hostvra/store.json`, `web-next.tar.gz`, and active PostgreSQL/MariaDB dumps.
- Preserves `/var/lib/hostvra/vault/hd_master.enc` (`0600`) and `/var/lib/hostvra/crypto_treasuries.json` untouched.
- Enforces 5-backup retention policy.

## 2. Local Isolated Backup Checksum & Recovery Verification
- `TestP13_RealOnDiskDatabaseBackupAndRestore`: `FileBytes = 41,105 bytes`, `SHA-256 = c680b2c4f8e3db83`, `BackupTime = 234.92 µs`, `RestoreRTO = 3.30 ms`, `RPO = 0`.
- **Remaining Risk:** Live pre-deployment backup on Contabo VPS will execute automatically as Step 1 when the operator runs `bash update.sh` in `root@hostvra`.
