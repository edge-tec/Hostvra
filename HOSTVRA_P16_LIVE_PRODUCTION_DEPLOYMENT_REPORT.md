# HOSTVRA — PHASE P16.4: LIVE PRODUCTION DEPLOYMENT REPORT

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P16.4 — Deploy P15/P16 to the Live VPS  
**Live Deployment Status:** **BLOCKED PENDING OPERATOR SSH EXECUTION (`Permission denied (publickey,password)`)**

---

## 1. Pre-Deployment Verification of `update.sh`

Inspected `/Volumes/EDGE/Hostvra-main/update.sh` against all 9 Phase 16.4 safety criteria:

| Safety Requirement | Implementation in `update.sh` | Verified |
| :--- | :--- | :---: |
| **1. Backup occurs before modifications** | Creates `/var/backups/hostvra/pre-update-YYYYMMDD-HHMMSS` before `git reset --hard origin/main` | **YES** |
| **2. Rollback trap works as intended** | `trap 'rollback' ERR` restores previous commit, binaries, `web-next.tar.gz`, and `store.json.bak` | **YES** |
| **3. No production secrets are overwritten** | `/etc/hostvra/hostvra.env` is backed up and never overwritten during update | **YES** |
| **4. Database migrations are safe** | Uses `CREATE TABLE IF NOT EXISTS` and `ADD COLUMN IF NOT EXISTS` in `EnsureAllSchemas` | **YES** |
| **5. API binary installation is atomic** | Compiles to `/tmp/hostvra-api-new` (or copies `bin/hostvra-api-linux-amd64`) and installs atomically to `/usr/local/bin/hostvra-api` | **YES** |
| **6. Frontend deployment is atomic** | Builds `apps/web` (`npm run build`) and only restarts `hostvra-web` if the build succeeds | **YES** |
| **7. Service restart failures are detected** | Checks `systemctl is-active --quiet hostvra-api` and `hostvra-web` | **YES** |
| **8. Health checks are executed** | Polls `http://127.0.0.1:8080/health` and `http://127.0.0.1:3000/login` after restart | **YES** |
| **9. Existing customer data is preserved** | `/var/lib/hostvra/store.json`, `/var/lib/hostvra/vault/hd_master.enc`, and PostgreSQL tables are preserved | **YES** |

---

## 2. Operator Cutover Procedure (`root@hostvra`)

Because non-interactive SSH from the local audit sandbox returned `root@hostvra.com: Permission denied (publickey,password)`, run the following verified cutover commands in your Contabo root terminal:

```bash
cd /root/Hostvra
git fetch origin
git pull --ff-only origin main
git rev-parse HEAD
bash update.sh
```

### Post-Cutover Verification on VPS
```bash
shasum -a 256 /usr/local/bin/hostvra-api bin/hostvra-api-linux-amd64
systemctl status hostvra-api hostvra-web --no-pager
```
Expected SHA-256 for both `/usr/local/bin/hostvra-api` and `bin/hostvra-api-linux-amd64`:
`c2c6b01c492c4937de22199540b78672c3510fc679c43497b22f80ab06f490b4`
