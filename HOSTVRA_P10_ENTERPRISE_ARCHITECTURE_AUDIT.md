# HOSTVRA — P10 ENTERPRISE ARCHITECTURE AUDIT

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Audited Baseline:** `c8daa2011b0479b7f973a67078033ab421763e66` + Phase P10 Hardening Fixes

---

## 1. Module-by-Module Architecture Verification

| Subsystem | Implementation Location | Status | Notes |
| :--- | :--- | :--- | :--- |
| **Go Control Plane API** | `apps/api/cmd/server/main.go`, `apps/api/internal/*` | **PASS** | Clean Chi router, JWT + RBAC middleware, MemoryStore + PostgresStore dual implementation, 0 `go vet` warnings. |
| **Next.js 15 Frontend** | `apps/web/src/app/*`, `apps/web/src/components/*` | **PASS** | 59/59 routes compile cleanly in production build (`npm run build`), responsive `DashboardShell` + `Sidebar`. |
| **Authentication & Authorization** | `apps/api/internal/auth/*`, `apps/api/internal/rbac/*`, `apps/api/internal/handlers/auth.go` | **PASS** | Argon2id password hashing, JWT access/refresh tokens, optional TOTP 2FA, RBAC permissions (`PermBillingManage`, `RequireAdmin()`). |
| **Admin Commercial Management** | `apps/web/src/app/admin/commercial/page.tsx`, `apps/api/internal/handlers/commercial.go` | **PASS** | Full CRUD + active toggle for Coupons, Affiliate directory & withdrawal Approve/Reject, Dunning trigger, Wallet adjustment. |
| **Customer Order Cart & Checkout** | `apps/web/src/app/order/page.tsx`, `POST /api/v1/billing/orders/checkout` | **PASS** | Authoritative server-side plan lookup & coupon redemption; supports Zero-Fee Crypto, Card/Gateway, and Credit Wallet. |
| **Billing, Invoices & Webhooks** | `apps/api/internal/handlers/billing.go`, `apps/api/internal/store/billing_store.go` | **PASS** | HMAC-SHA256 webhook signature verification, replay protection, amount/currency verification, refund/chargeback handling (fixed in P10). |
| **In-House Crypto HD Wallet & Treasury** | `apps/api/internal/crypto/*` | **PASS** | BIP-44/BIP-84/SLIP-0010 derivation across 9+ assets, AES-256-GCM vault, TRON/BTC/SOL/EVM sweep signers, USD gas cap. |
| **Coupons, Affiliates & Credit Wallet** | `apps/api/internal/store/commercial_store.go`, `apps/api/internal/handlers/commercial.go` | **PASS** | Double-entry wallet ledger with idempotency keys, atomic coupon redemption, referral attribution & commission settlement. |
| **Hosting Provisioning** | `apps/api/internal/handlers/billing.go` (`provisionUserForPlan`, `ActivatePaidInvoice`) | **PASS** | Zero-touch provisioning on Crypto, Gateway, Webhook, Admin, and Credit Wallet payments; unsuspends suspended accounts & syncs quotas. |
| **Automated Dunning Worker** | `apps/api/internal/billing/dunning.go` | **PASS** | Renewal invoice generation, wallet Auto-Pay execution, overdue grace period suspension, duplicate-notice prevention. |
| **Public Status, Legal & SEO** | `apps/web/src/app/{status,terms,privacy,refund-policy,sla,acceptable-use-policy}/page.tsx`, `robots.ts`, `sitemap.ts` | **PASS** | Live backend probes (`/api/v1/public/status`), linked in homepage navbar/footer and dashboard sidebar. |
| **Email Notifications** | `apps/api/internal/email/service.go` | **PARTIAL** | Transactional email hooks wired; live SMTP delivery requires production credentials on VPS (`NOT VERIFIED` live). |
| **Live Production VPS Deployment** | `update.sh`, systemd services | **NOT VERIFIED** | Deployment script inspected (`update.sh`); live VPS execution and health probes require operator execution on production server. |

---

## 2. Database Schema & Migration Parity

- **Root migrations (`migrations/`) vs Embedded migrations (`apps/api/internal/store/migrations/`)**: **PASS** (`TestMigrations_ParityAndDeterminism` verifies exact count and SHA-256 checksum parity across all 38 SQL migration files including `0036` and `0037`).
- **P10 Bug Fixed**: `PostgresStore.RecordAffiliateReferral` previously incremented `affiliate_accounts.total_referrals` unconditionally even when `ON CONFLICT (referred_user_id) DO NOTHING` inserted 0 rows. Fixed to check `res.RowsAffected() > 0`.
