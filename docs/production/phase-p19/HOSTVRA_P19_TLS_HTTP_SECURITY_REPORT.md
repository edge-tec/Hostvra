# HOSTVRA — PHASE P19.15: TLS & HTTP SECURITY REPORT

**Date / UTC Timestamp:** `2026-11-20T14:47:43Z` – `2026-11-20T14:48:05Z`  
**Environment:** Live Production `https://hostvra.com`  
**Exit Code:** `0` | **Status:** **CONDITIONAL PASS (TLS & API PASS; Public `/order`, `/status`, Legal Pending VPS Cutover)**

## Evidence
- **TLS:** `TLSv1.3 / AEAD-CHACHA20-POLY1305-SHA256`, `HTTP/2`, `CN=hostvra.com`, Issuer `Let's Encrypt (CN=YE1)`, Valid `Oct 1 2026` – `Dec 30 2026 GMT`.
- **Security Headers Verified Live:** `Strict-Transport-Security`, `Content-Security-Policy`, `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy`, `Permissions-Policy`.
- **HTTP Statuses:**
  - `200 OK`: `/` (`1.57s`), `/login` (`2.17s`), `/register` (`0.80s`), `/api/v1/public/status` (`0.71s`), `/api/v1/health` (`2.24s`).
  - `307 -> /login` (Protected — PASS): `/admin`, `/admin/commercial`, `/admin/crypto`, `/dashboard`, `/billing`, `/affiliate`.
  - `307 -> /login` (Pending VPS Cutover): `/order`, `/status`, `/terms`, `/privacy`, `/refund-policy`, `/sla`, `/acceptable-use-policy`.
