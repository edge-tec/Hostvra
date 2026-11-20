# HOSTVRA — PHASE P17.9: LIVE HTTP VERIFICATION REPORT

**UTC Timestamp:** `2026-11-20T14:25:47Z` – `2026-11-20T14:26:06Z`  
**Verification Method:** Live HTTPS `curl` probes against `https://hostvra.com`  
**Status:** **CONDITIONAL PASS (API & Protected Routes Verified; Public `/order`, `/status`, Legal Pages Pending VPS `update.sh`)**

---

## 1. Live URL Probe Results (`https://hostvra.com`)

| Endpoint | HTTP Status | Redirect Target | Latency | Verdict |
| :--- | :---: | :--- | :---: | :---: |
| `/` | `200 OK` | — | `1.785s` | **PASS** |
| `/login` | `200 OK` | — | `0.986s` | **PASS** |
| `/register` | `200 OK` | — | `0.751s` | **PASS** |
| `/api/v1/public/status` | `200 OK` | — | `0.855s` | **PASS** |
| `/api/v1/health` | `200 OK` | — | `0.852s` | **PASS** |
| `/admin` | `307` | `/login?redirect=%2Fadmin` | `0.717s` | **PASS (Auth Enforced)** |
| `/admin/commercial` | `307` | `/login?redirect=%2Fadmin%2Fcommercial` | `0.761s` | **PASS (Auth Enforced)** |
| `/admin/crypto` | `307` | `/login?redirect=%2Fadmin%2Fcrypto` | `0.577s` | **PASS (Auth Enforced)** |
| `/dashboard` | `307` | `/login?redirect=%2Fdashboard` | `0.689s` | **PASS (Auth Enforced)** |
| `/billing` | `307` | `/login?redirect=%2Fbilling` | `0.795s` | **PASS (Auth Enforced)** |
| `/affiliate` | `307` | `/login?redirect=%2Faffiliate` | `0.759s` | **PASS (Auth Enforced)** |
| `/order` | `307` | `/login?redirect=%2Forder` | `0.594s` | **PENDING VPS CUTOVER** |
| `/status` | `307` | `/login?redirect=%2Fstatus` | `0.987s` | **PENDING VPS CUTOVER** |
| `/terms`, `/privacy`, `/refund-policy`, `/sla`, `/acceptable-use-policy` | `307` | `/login?redirect=...` | `0.71s–0.98s` | **PENDING VPS CUTOVER** |

## 2. Live Security Headers Observed (`curl -sI https://hostvra.com/api/v1/health`)
```text
HTTP/2 405
server: nginx
date: Fri, 20 Nov 2026 14:26:06 GMT
allow: GET
content-security-policy: default-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'self';
permissions-policy: accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()
referrer-policy: strict-origin-when-cross-origin
strict-transport-security: max-age=31536000; includeSubDomains; preload
x-content-type-options: nosniff
x-frame-options: DENY
x-xss-protection: 1; mode=block
```
