# HOSTVRA — PHASE P14: ADMIN & COMMERCIAL MANAGEMENT AUDIT

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P14.7 — Admin & Commercial Management Audit  
**Status:** **PASS**

---

## 1. Admin Commercial Console (`/admin/commercial`)

Inspected `apps/web/src/app/admin/commercial/page.tsx` and corresponding backend handlers in `apps/api/internal/handlers/commercial.go`:

### 1.1 Coupon Management
- **Create Coupon (`POST /api/v1/admin/commercial/coupons`)**: Validates discount type (`percentage` vs `fixed`), bounds (`0 < percentage <= 100`, `0 < fixed <= 100000`), max redemptions, single-use per user, minimum order amount, expiry timestamp, and applicable plan codes.
- **Edit Coupon (`PUT /api/v1/admin/commercial/coupons/{id}`)**: Full update modal allowing modification of discount value, active status, expiration date, max redemptions, and plan restrictions.
- **Delete Coupon (`DELETE /api/v1/admin/commercial/coupons/{id}`)**: Removes coupon cleanly.
- **P14-03 Enhancement**: Plan-scoped coupons (`ApplicablePlanIDs`) are now strictly enforced in both `POST /api/v1/billing/coupons/validate` and `POST /api/v1/billing/orders/checkout` using normalized plan-code matching (`isCouponApplicableToPlan`).

### 1.2 Affiliate Program Administration
- **List Affiliates (`GET /api/v1/admin/commercial/affiliates`)**: Displays referral code, commission rate, total referrals, total earned, available balance, and status (`active`, `pending`, `suspended`).
- **Update Affiliate (`PUT /api/v1/admin/commercial/affiliates/{id}`)**: Allows admin to adjust individual commission percentage (`0 < rate <= 100`) and toggle affiliate status (`active` / `suspended`).

### 1.3 Withdrawal Review & Settlement
- **Approve / Mark Paid / Reject (`PUT /api/v1/admin/commercial/withdrawals/{id}`)**:
  - **Credit Wallet Payout**: Credits the customer's wallet atomically in a single transaction when marked `paid`, and records a double-entry ledger transaction.
  - **Crypto Payout**: Records external TX hash/notes on `paid`.
  - **Rejection Safety (P10-01)**: Rejects invalid transitions from terminal states (`paid` or `rejected`) so a rejected withdrawal whose balance was already refunded to the affiliate cannot be re-approved or double-paid.

---

## 2. Admin Crypto & HD Treasury Console (`/admin/crypto`)

- Wrapped inside `DashboardShell` so admin sidebar navigation remains accessible on all devices.
- Provides full control over:
  1. Payment Networks & RPC Health
  2. Crypto Invoices & Confirmations
  3. HD Wallet Seeds (`xpub` / encrypted seed import, index tracking, address derivation)
  4. Treasury Sweep Policies & Manual Sweep Execution
  5. Cold/Hot Treasury Address Book
