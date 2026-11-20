# HOSTVRA — PHASE P16: PRODUCTION GO-LIVE SIGNOFF & OPERATOR CUTOVER RUNBOOK

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Certified Linux AMD64 Binary (`bin/hostvra-api-linux-amd64`) SHA-256:**  
`c2c6b01c492c4937de22199540b78672c3510fc679c43497b22f80ab06f490b4`  
**Current Certification Verdict:** **CONDITIONAL PASS** -> **Upgrades to UNCONDITIONAL PASS upon Operator Cutover**

---

## 1. Exact Operator Cutover Commands (`root@hostvra`)

Log into your Contabo VPS as `root` and execute:

```bash
cd /root/Hostvra
git fetch origin
git pull --ff-only origin main
git rev-parse HEAD
bash update.sh
```

---

## 2. Post-Cutover Live Verification Commands (`root@hostvra`)

Immediately after `bash update.sh` finishes, run:

```bash
# 1. Verify deployed binary SHA-256 matches c2c6b01c492c4937de22199540b78672c3510fc679c43497b22f80ab06f490b4
shasum -a 256 /usr/local/bin/hostvra-api /root/Hostvra/bin/hostvra-api-linux-amd64

# 2. Verify systemd services are active
systemctl is-active hostvra-api hostvra-web

# 3. Verify all public routes return HTTP 200 and protected routes return HTTP 307
for r in / /login /register /order /status /terms /privacy /refund-policy /sla /acceptable-use-policy /admin /admin/commercial /dashboard /api/v1/public/status; do
  printf "%-28s -> " "$r"
  curl -s -o /dev/null -w "HTTP %{http_code}\n" "https://hostvra.com$r"
done
```

### Expected Post-Cutover HTTP Status Codes
- `/`, `/login`, `/register`, `/order`, `/status`, `/terms`, `/privacy`, `/refund-policy`, `/sla`, `/acceptable-use-policy`, `/api/v1/public/status` -> **`HTTP 200`**
- `/admin`, `/admin/commercial`, `/dashboard` (unauthenticated) -> **`HTTP 307`** (redirects to `/login?redirect=...`)
