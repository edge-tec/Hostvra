# HOSTVRA — PHASE P14: REMAINING RISKS & OPERATOR ACTIONS

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Current Status:** **CONDITIONAL PASS**

---

## 1. Required Action to Upgrade from `CONDITIONAL PASS` to `UNCONDITIONAL PASS`

### Action 1 — Execute Production Cutover on Contabo VPS (`CRITICAL — OPERATOR ACTION`)
The repository (`origin/main`) and pre-compiled Linux AMD64 binary (`bin/hostvra-api-linux-amd64`) contain all P10–P14 fixes, but the live Contabo VPS (`https://hostvra.com`) has not yet pulled the latest commits (`uptime_seconds: 3925`).

**Run on `root@hostvra`:**
```bash
cd /root/Hostvra && git pull origin main && bash update.sh
```

**Verify immediately after cutover:**
```bash
for r in / /order /status /terms /privacy /refund-policy /sla /acceptable-use-policy /admin/commercial /api/v1/public/status; do
  printf "%-28s -> " "$r"
  curl -s -o /dev/null -w "HTTP %{http_code}\n" "https://hostvra.com$r"
done
```
Expected result:
- `/`, `/order`, `/status`, `/terms`, `/privacy`, `/refund-policy`, `/sla`, `/acceptable-use-policy`, `/api/v1/public/status` -> `HTTP 200`
- `/admin/commercial` (unauthenticated) -> `HTTP 307` (redirects to `/login?redirect=%2Fadmin%2Fcommercial`)

---

## 2. Operational & Residual Risk Register

| Risk ID | Severity | Description | Mitigation / Recommendation |
| :--- | :--- | :--- | :--- |
| **RISK-01** | **HIGH (Until Cutover)** | Live VPS is running pre-P13 Next.js middleware where `/order` and `/status` redirect unauthenticated visitors to `/login`. | Fixed in commit `6aab42b` and verified in repo. Eliminated as soon as operator runs `cd /root/Hostvra && git pull origin main && bash update.sh`. |
| **RISK-02** | **LOW** | Single-node file-backed store (`/var/lib/hostvra/store.json`) resides on the same Contabo disk volume unless synced off-host. | `update.sh` creates timestamped backups in `/var/backups/hostvra/` (retaining the latest 5). Recommend adding a daily off-site `rclone` / S3 sync cron job for `/var/backups/hostvra/`. |
| **RISK-03** | **LOW** | Master encryption key (`HOSTVRA_MASTER_KEY` / `JWT_SECRET`) stored in `/etc/hostvra/hostvra.env`. | Ensure `/etc/hostvra/hostvra.env` keeps `chmod 600 root:root` permissions (enforced by `install.sh` and `update.sh`) and store a backup of `hostvra.env` in an off-server password manager. |
