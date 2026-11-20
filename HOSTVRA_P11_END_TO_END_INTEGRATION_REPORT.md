# HOSTVRA — P11 END-TO-END INTEGRATION REPORT

**Execution Date & Time:** 2026-11-20T17:08:00+06:00  
**Verified Commit:** `59bf61f700524b57797537b6e501508673870bf5`

---

## 1. End-to-End Integration Workflow Verification

| End-to-End Flow | Frontend Route(s) | Backend Endpoint(s) | Verification Status |
| :--- | :--- | :--- | :--- |
| **1. Partner Referral Capture & Registration** | `/register?ref=HV-XXXX` (`apps/web/src/app/register/page.tsx`) | `POST /api/v1/auth/register` (`AuthHandler.Register` -> `RecordAffiliateReferral`) | **PASS** (`TestCommercial_EndToEndReferralCheckoutAndCommission`) |
| **2. Direct Order Cart & Coupon Validation** | `/order` (`apps/web/src/app/order/page.tsx`) | `GET /api/v1/billing/plans`, `POST /api/v1/billing/coupons/validate` | **PASS** (Next.js build + handler race tests) |
| **3. Authoritative Order Checkout & Atomic Coupon Redemption** | `/order` -> Crypto / Card / Credit Wallet | `POST /api/v1/billing/orders/checkout` (`CommercialHandler.CheckoutOrder` -> `RedeemCouponAtomic`) | **PASS** (`TestCommercial_EndToEndReferralCheckoutAndCommission` + `TestP10_CouponInputValidationAndWalletPrecheckPreservesCoupon`) |
| **4. Payment Settlement -> Hosting Provisioning -> Affiliate Commission** | `/billing/crypto/[id]`, `/billing`, `/affiliate` | `SettleCryptoPayment`, `ActivatePaidInvoice`, `VerifyInvoicePayment`, `HandleWebhook`, `PayInvoiceWithWallet` -> `provisionUserForPlan` + `RecordAffiliateCommission` | **PASS** |
| **5. Refund Webhook -> Commission Reversal** | Payment Gateway Webhook | `POST /api/v1/billing/webhook/{gateway}` -> `ReverseAffiliateCommissionByInvoice` | **PASS** (`TestP10_WebhookRefundReversesAffiliateCommission`) |
| **6. Affiliate Withdrawal Request & Admin Approval/Rejection** | `/affiliate` & `/admin/commercial` | `POST /api/v1/affiliate/withdraw`, `POST /api/v1/admin/commercial/withdrawals/{id}/process` | **PASS** (`TestP10_WithdrawalRejectThenApproveBlockedAndConcurrentApproval`) |
| **7. Admin Coupon CRUD & Dunning Trigger** | `/admin/commercial` (`apps/web/src/app/admin/commercial/page.tsx`) | `GET/POST/PUT/DELETE /api/v1/admin/commercial/coupons`, `POST /api/v1/admin/commercial/dunning/run` | **PASS** (`TestCommercial_UpdateCouponAndToggle`) |
| **8. Public System Status & Legal/SEO Pages** | `/status`, `/terms`, `/privacy`, `/refund-policy`, `/sla`, `/acceptable-use-policy`, `/robots.txt`, `/sitemap.xml` | `GET /api/v1/public/status` | **PASS** (59/59 Next.js static/dynamic pages compiled) |
