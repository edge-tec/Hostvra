# HOSTVRA — PHASE P15: FRONTEND & COMMERCIAL UI QA REPORT

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P15.9 — Frontend & Commercial UI Verification  
**Status:** **CODEBASE PASS (59/59 Next.js Routes Verified; Live Public Route Middleware Pending VPS `update.sh`)**

---

## 1. Next.js Edge Middleware & Public Route Accessibility

Inspected `apps/web/src/middleware.ts`:
- `PUBLIC_PATHS` explicitly includes:
  - `/login`, `/register`, `/forgot-password`, `/reset-password`, `/verify-email`
  - `/order`, `/status`
  - `/terms`, `/privacy`, `/refund-policy`, `/sla`, `/acceptable-use-policy`
  - `/webmail/login`, `/webmail/sso`, `/api`, `/_next`, `/favicon.ico`
- And `pathname === '/'` is permitted directly.
- All protected customer routes (`/dashboard`, `/billing`, `/affiliate`, `/domains`, `/websites`, `/databases`, `/email`, `/security`, `/backups`, `/support`, `/settings`) and admin routes (`/admin`, `/admin/commercial`, `/admin/crypto`, `/admin/billing`, `/admin/users`, `/admin/servers`, `/admin/nodes`, `/admin/security`, `/admin/settings`) require `hostvra_token` cookie and redirect unauthenticated users to `/login?redirect=<path>`.

---

## 2. Commercial & Responsive UI Verification

1. **Order Checkout (`apps/web/src/app/order/page.tsx`)**:
   - Plan selector (`monthly` / `yearly` toggle), optional domain input, promo code input with real-time validation (`POST /api/v1/billing/coupons/validate` passing `plan_id`), payment method selector (`Crypto`, `Credit Wallet`, `Card`), and order submission (`POST /api/v1/billing/orders/checkout`).
   - Responsive 1-column mobile layout stacking cleanly into a 3-column (`lg:grid-cols-3`) layout with sticky order summary on desktop.
2. **Customer Affiliate Portal (`apps/web/src/app/affiliate/page.tsx`)**:
   - Displays unique referral link (`https://hostvra.com/register?ref=HV-...`) with one-click copy, commission metrics (`15%` default rate, total referrals, approved balance, total paid), and withdrawal request modal (`Credit Wallet`, `USDT BEP20`, `USDT TRC20`, `PayPal`, `Bank Transfer`).
3. **Admin Commercial Hub (`apps/web/src/app/admin/commercial/page.tsx`)**:
   - 3 responsive tabs:
     - **Promo Coupons**: Create coupon, edit coupon modal (`PUT /api/v1/admin/commercial/coupons/{id}`), toggle active/inactive, delete coupon.
     - **Affiliate Partners & Payouts**: View partner balances and approve/reject pending withdrawal requests (`POST /api/v1/admin/commercial/withdrawals/{id}/process`).
     - **Wallet Credit Adjustment & Dunning**: Credit/debit any customer wallet (`POST /api/v1/admin/commercial/wallets/adjust`) and trigger manual dunning pass (`POST /api/v1/admin/commercial/dunning/run`).
4. **Admin Crypto & HD Treasury (`apps/web/src/app/admin/crypto/page.tsx`)**:
   - Wrapped inside `DashboardShell` with responsive scrollable tabs (`.tabs-scroll`), responsive modals, and full Trust Wallet treasury management.
