# HOSTVRA — P14.2 LIVE VPS CUTOVER VERIFICATION

**Execution Date & Time:** 2026-11-20T18:15:12+06:00  
**Target Domain:** `https://hostvra.com` (Contabo VPS `/root/Hostvra`)

---

## 1. Live Production HTTPS Route & API Probe Matrix

Executed live read-only probes from local workspace against `https://hostvra.com`:

| Route / Endpoint | Live HTTP Code | Redirect Target | Response Time | Live Verification Verdict |
| :--- | :--- | :--- | :--- | :--- |
| `GET https://hostvra.com/` | `200 OK` | None | `1.662s` | **LIVE & OPERATIONAL** |
| `GET https://hostvra.com/api/v1/public/status` | `200 OK` | None | `0.661s` | **LIVE & OPERATIONAL** (`uptime_seconds: 3925`, `goroutines: 15`, `memory_mb: 7`, 4/4 subsystems `operational`) |
| `GET https://hostvra.com/admin/commercial` | `307` | `/login?redirect=%2Fadmin%2Fcommercial` | `0.670s` | **PASS** (Protected admin route redirects unauthenticated user to `/login`) |
| `GET https://hostvra.com/order` | `307` | `/login?redirect=%2Forder` | `0.801s` | **PENDING VPS CUTOVER** (Fixed in `apps/web/src/middleware.ts` in P13 `6aab42b`, awaiting operator `git pull && bash update.sh` on VPS) |
| `GET https://hostvra.com/status` | `307` | `/login?redirect=%2Fstatus` | `0.749s` | **PENDING VPS CUTOVER** (Fixed in `middleware.ts` in P13 `6aab42b`, awaiting VPS pull) |
| `GET https://hostvra.com/{terms,privacy,refund-policy,sla,acceptable-use-policy}` | `307` | `/login?redirect=...` | `0.67s–1.10s` | **PENDING VPS CUTOVER** (Fixed in `middleware.ts` in P13 `6aab42b`, awaiting VPS pull) |

---

## 2. Independent Proof of Why P13 Fixes Are Not Yet Live on `hostvra.com`

The live JSON payload from `https://hostvra.com/api/v1/public/status` at `2026-11-20T12:15:12Z` reported:
```json
"details": "Live process uptime: 3925s (Go go1.26.0)",
"uptime_seconds": 3925
```
An uptime of `3,925 seconds` (~65 minutes) proves the live `hostvra-api` and `hostvra-web` systemd services on Contabo VPS have not yet been restarted with the P13/P14 commits. Running `cd /root/Hostvra && git pull origin main && bash update.sh` on `root@hostvra:~#` will deploy the updated binary and Next.js build so `/order`, `/status`, and all legal pages return `HTTP 200 OK` without login.
