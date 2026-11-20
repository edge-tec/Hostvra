# HOSTVRA — PHASE P17: FINAL PRODUCTION SIGNOFF & OPERATOR CUTOVER COMMAND

**UTC Timestamp:** `2026-11-20T14:26:06Z`  
**Status:** **CONDITIONAL PASS -> Upgrades to UNCONDITIONAL PASS Upon Operator Cutover**

---

## 1. Exact Operator Cutover Command (`root@hostvra`)

Run inside your authorized Contabo VPS root terminal:

```bash
cd /root/Hostvra && git fetch origin && git pull --ff-only origin main && git rev-parse HEAD && bash update.sh
```

## 2. Post-Cutover Verification Command (`root@hostvra`)

```bash
shasum -a 256 /usr/local/bin/hostvra-api /root/Hostvra/bin/hostvra-api-linux-amd64
systemctl is-active hostvra-api hostvra-web
for r in / /login /register /order /status /terms /privacy /refund-policy /sla /acceptable-use-policy /api/v1/public/status /api/v1/health /admin /admin/commercial /admin/crypto /dashboard /billing /affiliate; do
  printf "%-28s -> " "$r"
  curl -s -o /dev/null -w "HTTP %{http_code}\n" "https://hostvra.com$r"
done
```

### Expected Post-Cutover Result
- `/usr/local/bin/hostvra-api` SHA-256: `c2c6b01c492c4937de22199540b78672c3510fc679c43497b22f80ab06f490b4`
- `hostvra-api` & `hostvra-web`: `active`
- `/`, `/login`, `/register`, `/order`, `/status`, `/terms`, `/privacy`, `/refund-policy`, `/sla`, `/acceptable-use-policy`, `/api/v1/public/status`, `/api/v1/health` -> **`HTTP 200`**
- `/admin`, `/admin/commercial`, `/admin/crypto`, `/dashboard`, `/billing`, `/affiliate` -> **`HTTP 307`** (redirects to `/login`)
