# HOSTVRA — PHASE P15: PRODUCTION DEPLOYMENT & ROLLBACK REPORT

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P15.2 & P15.12 — Production Deployment & Rollback Verification  
**Status:** **AUTOMATED SCRIPT & BINARY VERIFIED — READY FOR OPERATOR EXECUTION ON `root@hostvra`**

---

## 1. Independent Inspection of `update.sh`

Inspected `/Volumes/EDGE/Hostvra-main/update.sh` line-by-line before recommending execution:

1. **Non-Destructive Path Normalization**:
   - Detects the repository directory (`INSTALL_DIR`) and ensures `/opt/hostvra` and `/root/Hostvra` resolve to the same physical directory via symlink (`ln -sfn`).
2. **Pre-Update Backup Before Any Git Mutation**:
   - Creates `/var/backups/hostvra/pre-update-YYYYMMDD-HHMMSS`.
   - Saves `git-commit.txt`, `service-states.txt`, `/usr/local/bin/hostvra-api`, `/usr/local/bin/hostvra-agent`, `/etc/hostvra/hostvra.env`, `/var/lib/hostvra/store.json`, `web-next.tar.gz`, and database dumps (`pg_dump` / `mysqldump`) **before** `git fetch` / `git reset --hard origin/main`.
3. **Preservation of `.env`, TLS, Wallet Data & Persistent Store**:
   - Never overwrites `/etc/hostvra/hostvra.env`, `/var/lib/hostvra/store.json`, `/var/lib/hostvra/crypto_treasuries.json`, or `/etc/letsencrypt/*`.
4. **Pre-Compiled Static Binary Fallback**:
   - If `go build` fails or Go is not in `PATH`, automatically copies `bin/hostvra-api-linux-amd64` (SHA-256: `ad3a72982b21580a39ec11aeb45e3e58b03ee61bb548c38722f263c85fc7ce98`) to `/usr/local/bin/hostvra-api`.
5. **Automatic Rollback Trap (`trap 'rollback' ERR`)**:
   - If any step or post-restart `/health` check fails, `rollback()` restores the previous Git commit, binaries, Next.js build archive, and `/var/lib/hostvra/store.json`, then restarts `hostvra-api` and `hostvra-web`.
6. **5-Backup Retention Policy**:
   - Retains the latest 5 valid backups in `/var/backups/hostvra/` and removes older ones.

---

## 2. Exact Operator Deployment Command (`root@hostvra`)

Run in your authenticated Contabo VPS root shell:

```bash
cd /root/Hostvra && git fetch origin && git pull --ff-only origin main && bash update.sh
```

### Post-Deployment Verification Command
```bash
shasum -a 256 /usr/local/bin/hostvra-api bin/hostvra-api-linux-amd64
for r in / /login /register /order /status /terms /privacy /refund-policy /sla /acceptable-use-policy /admin/commercial /api/v1/public/status; do
  printf "%-28s -> " "$r"
  curl -s -o /dev/null -w "HTTP %{http_code}\n" "https://hostvra.com$r"
done
```
