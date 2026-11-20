# HOSTVRA — FINAL COMMERCIAL INTEGRATION AUDIT

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra`  
**Branch:** `main`  
**Verified Baseline Commit:** `fc45a9e8a5f53e427a9ff28890629e356786861a` (`fc45a9e` confirmed ancestor of `HEAD`, clean working tree, synced with `origin/main`)

---

## 1. Executive Summary

Hostvra's core hosting infrastructure, in-house HD crypto payment gateway, automated billing dunning worker, real GitHub updater, and commercial backend tables/APIs (`0036` + `0037` migrations) are already implemented and tested. However, an independent code audit of `apps/api` and `apps/web` identified specific **end-to-end commercial integration and UI gaps** preventing administrators and customers from using coupons, affiliates, referral attribution, and direct cart checkout seamlessly.

---

## 2. Existing Working Features (Verified in Code)

| Module | Location | Current Status |
| :--- | :--- | :--- |
| Commercial DB Schema (Coupons, Affiliates, Wallet, Dunning) | `migrations/0037_ultra_enterprise_commercial_platform.sql` & embedded copy | Implemented & parity-tested |
| Coupon Store & Atomic Redemption | `apps/api/internal/store/commercial_store.go` (`CreateCoupon`, `GetCouponByCode`, `ListCoupons`, `DeleteCoupon`, `RedeemCouponAtomic`) | Implemented (Memory + Postgres) |
| Affiliate Store & Withdrawal Processing | `apps/api/internal/store/commercial_store.go` (`GetOrCreateAffiliateAccount`, `RecordAffiliateReferral`, `RecordAffiliateCommission`, `RequestAffiliateWithdrawal`, `ApproveAffiliateWithdrawal`) | Implemented (Memory + Postgres) |
| Customer Credit Wallet & Double-Entry Ledger | `apps/api/internal/store/commercial_store.go` (`CreditWalletAtomic`, `DebitWalletAtomic`, `ListWalletLedger`, `SetWalletAutoPay`) | Implemented (Memory + Postgres) |
| Commercial REST Endpoints | `apps/api/internal/handlers/commercial.go` & `apps/api/cmd/server/main.go` | Public status/coupon validate, Customer wallet/affiliate, Admin RBAC endpoints wired |
| Public Legal, Status, SEO Pages | `apps/web/src/app/{status,terms,privacy,refund-policy,sla,acceptable-use-policy}/page.tsx`, `robots.ts`, `sitemap.ts` | Implemented & building cleanly |
| Crypto HD Wallet & Settlement | `apps/api/internal/crypto/*`, `apps/api/internal/handlers/billing.go` (`SettleCryptoPayment`) | Implemented & tested |

---

## 3. Identified Integration & UI Gaps

### 3.1 Missing Frontend Functionality
1. **Admin Commercial Management UI (`/admin/commercial`) — MISSING**
   - `apps/web/src/app/admin/commercial/page.tsx` does not exist. Admins have no UI to create, edit, toggle, or delete coupons, review affiliates, approve/reject withdrawal requests, or manually trigger billing dunning.
2. **Navigation, Sidebar & Footer Links — INCOMPLETE**
   - `apps/web/src/components/Sidebar.tsx` does not link to `/admin/commercial` (Admin), `/order` (Customer), `/affiliate` (Customer), or `/status`.
   - `apps/web/src/app/page.tsx` homepage navbar and footer do not link to `/order`, `/affiliate`, `/status`, `/terms`, `/privacy`, `/refund-policy`, `/sla`, or `/acceptable-use-policy`.
3. **Referral Capture in `/register` — MISSING**
   - `apps/web/src/app/register/page.tsx` does not read `?ref=` from the URL, does not persist referral attribution in first-party storage, and does not transmit `referral_code` during registration.
4. **Affiliate Dashboard History Tables — INCOMPLETE**
   - `apps/web/src/app/affiliate/page.tsx` does not render the withdrawal request history table or wallet ledger history table (even though both are returned by the backend), and is not wrapped in `DashboardShell`.

### 3.2 Missing Backend Integration
1. **Coupon Edit / Active Toggle Endpoint — MISSING**
   - `UpdateCoupon` is missing from `store.Store`, `MemoryStore`, `PostgresStore`, `CommercialHandler`, and `main.go` (`PUT /api/v1/admin/commercial/coupons/{id}`).
2. **Referral Attribution in Registration — MISSING**
   - `RegisterRequest` and `AuthHandler.Register` in `apps/api/internal/handlers/auth.go` do not accept `referral_code` or persist attribution via `store.GetAffiliateByCode` + `store.RecordAffiliateReferral`.
3. **Affiliate Commission Trigger on Paid Invoices — INCOMPLETE**
   - `SettleCryptoPayment`, `VerifyInvoicePayment`, `HandlePaymentWebhook`, and `PayInvoice` in `apps/api/internal/handlers/billing.go` do not invoke `store.RecordAffiliateCommission` upon verified invoice payment.
4. **Authoritative Server-Side Order Checkout with Coupon Redemption — MISSING**
   - `/order` currently redirects to `/billing?plan=...&coupon=...` without creating an invoice or redeeming the coupon server-side.
   - Need `POST /api/v1/billing/orders/checkout` (authenticated) that computes authoritative plan pricing from the database, atomically redeems coupons via `store.RedeemCouponAtomic`, creates the `store.Invoice`, and supports immediate `credit_wallet` payment or handoff to Crypto (`/billing/crypto/{id}`) / Card (`/billing`).
5. **Hosting Provisioning on Credit Wallet Payment — MISSING**
   - `CommercialHandler.PayInvoiceWithWallet` marks the invoice paid and credits affiliate commission, but does not invoke `BillingHandler.ActivatePaidInvoice` to activate/renew the subscription and provision hosting quotas.

---

## 4. Security & Financial Risks Addressed by Implementation Plan

- **Client-side price tampering**: Prevented by computing all order totals and coupon discounts strictly server-side from authoritative database records in `POST /api/v1/billing/orders/checkout`.
- **Coupon race conditions / over-redemption**: Prevented by `store.RedeemCouponAtomic` (`SELECT ... FOR UPDATE` in PostgreSQL, mutex lock in MemoryStore).
- **Self-referral abuse**: Blocked in `RecordAffiliateReferral` (`ErrSelfReferralForbidden`) and ignored safely during registration.
- **Duplicate affiliate commissions**: Prevented by unique `invoice_id` idempotency in `RecordAffiliateCommission`.
- **RBAC bypass**: All `/api/v1/admin/commercial/*` endpoints protected by `rbac.RequirePermission(rbac.PermBillingManage)`.

---

## 5. Implementation Plan

1. Add `UpdateCoupon` across `store.Store`, `MemoryStore`, `PostgresStore`, `CommercialHandler`, and `main.go`.
2. Add `ActivatePaidInvoice` helper on `BillingHandler`, wire `RecordAffiliateCommission` into all verified invoice settlement paths (Crypto, Gateway Verify, Webhook, Admin Pay, Wallet Pay), and connect `CommercialHandler` to `BillingHandler.ActivatePaidInvoice`.
3. Add `POST /api/v1/billing/orders/checkout` for authoritative server-side order creation, atomic coupon redemption, and instant wallet/crypto/card checkout handoff.
4. Wire `referral_code` into `AuthHandler.Register` with self-referral blocking.
5. Build `/admin/commercial` UI (Coupons CRUD + Affiliates & Withdrawal Approval + Dunning Runner).
6. Update `Sidebar.tsx`, homepage `page.tsx` navbar/footer, `/register` referral capture, `/order` direct checkout flow, and `/affiliate` dashboard tables wrapped in `DashboardShell`.
7. Add automated Go tests with `-race`, verify migration parity, rebuild Linux API binary & Next.js production bundle, and push to `origin/main`.
