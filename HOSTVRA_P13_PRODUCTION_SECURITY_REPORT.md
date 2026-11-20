# HOSTVRA — P13 PRODUCTION SECURITY REPORT

**Execution Date & Time:** 2026-11-20T17:42:00+06:00  
**Target:** `https://hostvra.com` & Repository `edge-tec/Hostvra`

---

## 1. Live & Repository Security Verification Matrix

| Security Check | Method / Evidence | Status |
| :--- | :--- | :--- |
| **Live TLS / HTTPS Certificate** | `openssl s_client -servername hostvra.com -connect hostvra.com:443`: Valid Let's Encrypt certificate (`CN=hostvra.com`, valid through `Dec 30 14:33:57 2026 GMT`). | **PASS (Live Verified)** |
| **Live Admin Route Gate (`/admin/commercial`)** | Unauthenticated `GET https://hostvra.com/admin/commercial` returns `HTTP 307` redirect to `/login?redirect=%2Fadmin%2Fcommercial`; backend `/api/v1/admin/commercial/*` enforces `rbac.RequireAdmin()` + `PermBillingManage`. | **PASS (Live + Code Verified)** |
| **Public Route Accessibility vs Protected Dashboard** | Fixed `apps/web/src/middleware.ts` so public commercial/status/legal pages (`/order`, `/status`, `/terms`, `/privacy`, `/refund-policy`, `/sla`, `/acceptable-use-policy`) are accessible without auth while `/dashboard`, `/billing`, `/affiliate`, `/admin/*` remain protected. | **PASS** |
| **Zero Hardcoded Passwords in Deployment Scripts** | Verified in P12/P13: hardcoded fallback password removed from `update-server.sh` and `install.sh`; existing `/etc/hostvra/api.env` credentials preserved with `0600` permissions. | **PASS** |
| **Webhook Signature & Replay Protection** | `TestBillingWebhook_SecurityAndReplay` + `TestP10_WebhookRefundReversesAffiliateCommission` (`-race`): HMAC-SHA256 verification, amount/currency match, replay deduplication. | **PASS** |
| **Host-Level Firewall (`ufw` / `iptables`) & Local `/etc/hostvra/api.env` Permissions** | Configured by `install.sh` (`chmod 600 /etc/hostvra/api.env`); direct local root shell inspection requires operator execution on `root@hostvra`. | **CONDITIONAL (Pending Operator Shell Check)** |
