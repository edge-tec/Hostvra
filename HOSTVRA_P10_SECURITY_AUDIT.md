# HOSTVRA — P10 APPLICATION SECURITY AUDIT (OWASP)

**Audit Date:** 2026-11-20  
**Methodology:** Independent static source inspection + negative/adversarial automated tests (`-race`)

---

## 1. OWASP & Commercial Security Control Matrix

| Security Area | Findings & Controls Verified | Status |
| :--- | :--- | :--- |
| **Broken Access Control / Admin Protection** | All `/api/v1/admin/commercial/*` routes wrapped in `rbac.RequireAdmin()` + `rbac.RequirePermission(rbac.PermBillingManage)`. Customer wallet/affiliate/checkout routes require valid JWT claims (`auth.GetClaims`). | **PASS** |
| **IDOR Prevention** | `PayInvoiceWithWallet` verifies `inv.UserID == claims.UserID`; `PayInvoice` enforces owner/org/admin checks; affiliate/wallet endpoints scope strictly to `claims.UserID`. | **PASS** |
| **SQL Injection** | All PostgreSQL queries in `commercial_store.go` and `billing_store.go` use strictly parameterized `$1..$N` placeholders (`ExecContext`/`QueryRowContext`/`QueryContext`). | **PASS** |
| **Webhook Signature Verification & Replay Protection** | `HandleWebhook` validates HMAC-SHA256 (`X-Signature`) against configured gateway `SecretKey`, verifies amount & currency, and records event IDs in `DomainWebhook` to block replays. | **PASS** |
| **SSRF Protection** | Outbound alert webhooks validated via `iputil.ValidateSafeWebhookURL`. | **PASS** |
| **Input Validation Hardening (Fixed in P10)** | Fixed missing bounds checks on coupon creation/update (`discount_value > 0`, `percentage <= 100`, `min_order_usd >= 0`), coupon validation (`order_amount_usd >= 0`), wallet adjustments (`direction in {credit,debit}`, finite positive amount), and affiliate withdrawals (method allowlist + required destination details). | **PASS** |
| **Self-Referral & Abuse Prevention** | `RecordAffiliateReferral` enforces `ErrSelfReferralForbidden`; `AuthHandler.Register` safely ignores invalid/self-referral codes without leaking internal state. | **PASS** |
| **Secret Management** | HD wallet mnemonic encrypted at rest with AES-256-GCM; write-only Admin UI; no production secrets or `.env` credentials tracked in Git. | **PASS** |
| **Live Network Penetration / WAF / Rate-Limit Load Test** | Requires live staging/production deployment environment. | **NOT VERIFIED** |

---

## 2. Residual Risks & Operational Recommendations

- Ensure `JWT_SECRET`, `HD_VAULT_MASTER_KEY`, and payment gateway `SecretKey` values are provisioned via systemd environment files (`0600` root-only permissions) on the VPS.
- Enforce HTTPS/TLS termination (Nginx/Certbot) in front of `hostvra-api` and `hostvra-web` so referral cookies/tokens and webhook payloads are encrypted in transit.
- Monitor `audit_logs` for repeated `commercial.wallet.admin_adjust` or `billing.webhook.refunded` events.
