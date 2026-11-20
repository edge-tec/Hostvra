# HOSTVRA — P14.3 COMPLETE BACKEND & API AUDIT

**Audit Date & Time:** 2026-11-20T18:16:30+06:00

---

## 1. Backend API Endpoint & Security Verification

| Endpoint Group | HTTP Methods & Routes | Auth & RBAC Policy | P14 Audit Finding & Status |
| :--- | :--- | :--- | :--- |
| **Public Status & Coupon Validate** | `GET /api/v1/public/status`, `POST /api/v1/billing/coupons/validate` | Public (rate-limited) | **FIXED IN P14 (`P14-03`)**: Added `ApplicablePlanIDs` eligibility enforcement when `plan_id` is provided. (**PASS**) |
| **Customer Order Checkout** | `POST /api/v1/billing/orders/checkout` | JWT Required (`auth.GetClaims`) | **FIXED IN P14 (`P14-03`)**: Enforces `isCouponApplicableToPlan(c.ApplicablePlanIDs, plan)` before `RedeemCouponAtomic`, pre-checks wallet balance, and computes authoritative server-side totals. (**PASS**) |
| **Customer Credit Wallet** | `GET /api/v1/wallet`, `PUT /api/v1/wallet/autopay`, `POST /api/v1/wallet/pay-invoice` | JWT Required + Owner check (`inv.UserID == claims.UserID`) | **FIXED IN P14 (`P14-02`)**: Serialized `PayInvoiceWithWallet` with `h.payMu` so concurrent requests cannot trigger duplicate `onInvoicePaid` provisioning, and rejected `cancelled`/`refunded` invoices with `400`. (**PASS**) |
| **Customer Affiliate Hub** | `GET /api/v1/affiliate/dashboard`, `POST /api/v1/affiliate/withdraw` | JWT Required | Enforces `$0.01` floor, 2-decimal cent rounding, method allowlist, and destination address check. (**PASS**) |
| **Admin Commercial Suite** | `GET/POST/PUT/DELETE /api/v1/admin/commercial/coupons`, `GET /affiliates`, `POST /withdrawals/{id}/process`, `POST /wallet/adjust`, `POST /dunning/run` | `rbac.RequireAdmin()` + `rbac.RequirePermission(rbac.PermBillingManage)` | Enforces coupon bounds (`0 < pct <= 100`), blocks re-approving rejected withdrawals (`ErrWithdrawalAlreadyProcessed`), and logs all admin actions to `audit_logs`. (**PASS**) |
| **Billing Webhooks** | `POST /api/v1/billing/webhook/{gateway}` | HMAC-SHA256 signature (`X-Signature`) + replay check | Verifies amount & currency; handles `paid` and `refunded`/`chargeback`/`reversed` (reversing affiliate commissions). (**PASS**) |
