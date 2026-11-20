# HOSTVRA — PHASE P15: FINANCIAL INTEGRITY & BILLING CERTIFICATION

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P15.5 — Financial Integrity & Billing Certification  
**Status:** **PASS (13/13 Financial Invariants Independently Certified Under `-race`)**

---

## 1. Financial Invariant Verification Matrix

| # | Financial Requirement | Implementation & Guard Location | Automated Regression Test | Verdict |
| :-: | :--- | :--- | :--- | :---: |
| **1** | **Invoice Amount Calculations** | Authoritative server-side plan price lookup in `CheckoutOrder`, `CreateSubscription`, `CreateCheckoutSession` | `TestCommercial_EndToEndReferralCheckoutAndCommission` | **PASS** |
| **2** | **Decimal Precision & Cent Rounding** | `math.Round(...*100)/100` for USD discounts, commissions, and affiliate balances; `math.Round(...*10000)/10000` for wallet ledger | `TestP12_FinancialInvariantsLedgerReconciliationAndPrecision` | **PASS** |
| **3** | **Zero-Value Invoice Handling** | 100%-discounted (`$0.00`) orders settle immediately with `promo_free` without calling `DebitWalletAtomic` (`amountUSD <= 0`) | `TestP14_ConcurrentPayInvoiceWithWalletAndPlanScopedCouponsAndSweepPersistence` | **PASS** |
| **4** | **Cancelled / Refunded Invoice Protection** | Blocked across `PayInvoiceWithWallet` (P14-02) AND `PayInvoice`, `CheckoutInvoice`, `VerifyInvoicePayment`, `HandleWebhook`, `SettleCryptoPayment` (P15-02) | `TestP14_*` & `TestP15_*` | **PASS** |
| **5** | **Concurrent Payment Prevention** | `CommercialHandler.payMu` + `BillingHandler.settleMu` + `DebitWalletAtomic` idempotency key `wallet_pay:<invoice_id>` | `TestP14_*` & `TestP15_*` | **PASS** |
| **6** | **Duplicate Webhook Protection** | `BillingHandler.settleMu` + `GetDomainWebhook` external event ID deduplication + HMAC signature verification | `TestBillingWebhook_SecurityAndReplay`, `TestP15_*` | **PASS** |
| **7** | **Refund Idempotency & Commission Reversal** | `HandleWebhook` (`refunded`/`chargeback`/`reversed`) marks invoice refunded once and calls `ReverseAffiliateCommissionByInvoice` | `TestP10_WebhookRefundReversesAffiliateCommission` | **PASS** |
| **8** | **Coupon Plan Restrictions** | `isCouponApplicableToPlan(c.ApplicablePlanIDs, plan)` enforced in both `ValidateCoupon` and `CheckoutOrder` | `TestP14_*` | **PASS** |
| **9** | **Single-Use Coupon Preservation on Failed Wallet Checkout** | Wallet balance pre-checked in `CheckoutOrder` before `RedeemCouponAtomic` is called | `TestP10_CouponInputValidationAndWalletPrecheckPreservesCoupon` | **PASS** |
| **10** | **Wallet Balance Consistency & Double-Entry Ledger** | Every `CreditWalletAtomic` / `DebitWalletAtomic` records `BalanceBeforeUSD`, `AmountUSD`, `BalanceAfterUSD`, and `IdempotencyKey` | `TestP12_FinancialInvariantsLedgerReconciliationAndPrecision` | **PASS** |
| **11** | **Affiliate Withdrawal Terminal State Lock** | `ApproveAffiliateWithdrawal` rejects any non-`pending` withdrawal (`ErrWithdrawalAlreadyProcessed`) | `TestP10_WithdrawalRejectThenApproveBlockedAndConcurrentApproval` | **PASS** |
| **12** | **Provisioning Exactly Once** | Protected by `payMu` (wallet) and `settleMu` (webhook/crypto/admin) so `onInvoicePaid` / `provisionUserForPlan` executes exactly once per paid invoice | `TestP14_*` & `TestP15_*` | **PASS** |
| **13** | **Audit Log Completeness** | All coupon, wallet, withdrawal, checkout, and webhook actions write structured audit entries via `h.audit.Log` | Verified in test output logs | **PASS** |
