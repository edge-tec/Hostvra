# HOSTVRA — PHASE P14: PRODUCTION DEPLOYMENT & ROLLBACK REPORT

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P14.11 — Deployment Automation & Rollback Safety Verification  
**Status:** **DEPLOYMENT SCRIPT & BINARY CERTIFIED — AWAITING OPERATOR CUTOVER ON CONTABO VPS**

---

## 1. Deployment Script (`update.sh`) Safety Audit

Inspected `/Volumes/EDGE/Hostvra-main/update.sh`:

1. **Path Normalization (`/opt/hostvra` <-> `/root/Hostvra`)**:
   - Automatically detects `/root/Hostvra` and creates the `/opt/hostvra -> /root/Hostvra` symlink so both paths work seamlessly.
2. **Pre-Cutover Atomic Backup (`/var/backups/hostvra/pre-update-YYYYMMDD-HHMMSS`)**:
   - Executed **before** `git reset --hard origin/main`.
   - Captures:
     - `git-commit.txt` (previous commit SHA)
     - `service-states.txt` (`hostvra-api`, `hostvra-web`, `hostvra-agent`)
     - Installed binaries (`/usr/local/bin/hostvra-api`, `/usr/local/bin/hostvra-agent`)
     - Environment config (`/etc/hostvra/hostvra.env`)
     - JSON store snapshot (`/var/lib/hostvra/store.json`)
     - PostgreSQL (`pg_dump`) and MySQL/MariaDB (`mysqldump`) dumps if present
     - Built Next.js bundle (`web-next.tar.gz`)
3. **5-Backup Retention Pruning**:
   - Automatically retains the 5 most recent valid backup directories and prunes older ones.
4. **Pre-Compiled Binary Fallback**:
   - If the VPS lacks a local Go compiler or `go build` fails, `update.sh` automatically falls back to installing the repository's pre-compiled Linux AMD64 static binary (`bin/hostvra-api-linux-amd64`).
5. **Automatic Rollback on Health Check Failure**:
   - Traps `ERR` and runs `rollback()`, restoring the previous commit, binaries, `web-next.tar.gz`, and `/var/lib/hostvra/store.json`, then restarting services.

---

## 2. Exact Operator Cutover Command for Contabo VPS

Because direct SSH access from this audit sandbox to `root@hostvra` requires the operator's Contabo SSH session, run the following single command on the Contabo VPS (`root@hostvra`) to deploy the P13 + P14 fixes to live production:

```bash
cd /root/Hostvra && git pull origin main && bash update.sh
```

### Post-Cutover Verification Command
Immediately after `bash update.sh` finishes, verify that all public pages return `HTTP 200` and `/admin/commercial` remains protected (`HTTP 307`):

```bash
for r in / /order /status /terms /privacy /refund-policy /sla /acceptable-use-policy /admin/commercial /api/v1/public/status; do
  printf "%-28s -> " "$r"
  curl -s -o /dev/null -w "HTTP %{http_code}\n" "https://hostvra.com$r"
done
```
