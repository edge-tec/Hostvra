# HOSTVRA — P11 FINANCIAL REGRESSION REPORT

**Execution Date & Time:** 2026-11-20T17:07:00+06:00  
**Verified Commit:** `59bf61f700524b57797537b6e501508673870bf5`  
**Test Command:** `go test -v -race ./internal/handlers -run "TestCommercial_|TestP10_|TestBillingWebhook_"`  
**Result:** **PASS (10/10 test groups, 0 race conditions)**

---

## 1. P10 Financial Fix Preservation & Regression Verification

| Section | Financial Requirement | Verified Implementation & Test | Status |
| :--- | :--- | :--- | :--- |
| **A. Withdrawal Protection** | Rejected withdrawal cannot be approved again; already-paid withdrawal cannot be credited twice; 5 concurrent approval attempts credit wallet strictly once (`$20.00`). | `ApproveAffiliateWithdrawal` (`ErrWithdrawalAlreadyProcessed`), `AdminProcessWithdrawal` (`wd.Status == "paid"` check), `TestP10_WithdrawalRejectThenApproveBlockedAndConcurrentApproval` | **PASS** |
| **B. Refund & Commission Reversal** | Verified HMAC-signed `refunded`, `chargeback`, and `reversed` webhooks mark invoice `refunded`, reverse affiliate commission (`$15.00 -> $0.00`), and record replay protection. | `BillingHandler.HandleWebhook`, `ReverseAffiliateCommissionByInvoice`, `TestP10_WebhookRefundReversesAffiliateCommission` | **PASS** |
| **C. Coupon Integrity** | Insufficient wallet balance on `credit_wallet` checkout returns `400` without burning single-use coupon (`ONCE50`); 10 concurrent checkouts on `MaxUses=3` coupon allow strictly 3 redemptions. | `CommercialHandler.CheckoutOrder`, `RedeemCouponAtomic`, `TestP10_CouponInputValidationAndWalletPrecheckPreservesCoupon`, `TestCommercial_CouponConcurrentRedemptionLimit` | **PASS** |
| **D. Referral Integrity** | Duplicate referral submissions do not inflate `total_referrals`; self-referrals blocked (`ErrSelfReferralForbidden`). | `MemoryStore` & `PostgresStore.RecordAffiliateReferral` (`res.RowsAffected() > 0`), `TestCommercial_AffiliateSelfReferralAndReversal` | **PASS** |
| **E. Financial Input Validation** | Negative discounts, percentage `> 100%`, `NaN`/`Inf`, invalid wallet adjustment direction (`"mint"`), and empty crypto withdrawal addresses are rejected with `400 Bad Request`. | `validateCouponFields`, `AdminAdjustWallet`, `RequestAffiliateWithdrawal`, `TestP10_WalletAdjustAndWithdrawalInputValidation` | **PASS** |
