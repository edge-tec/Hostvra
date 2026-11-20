# HOSTVRA — P14.4 FINANCIAL INTEGRITY & BILLING CERTIFICATION

**Certification Date & Time:** 2026-11-20T18:16:30+06:00  
**Certification Verdict:** **PASS**

---

## 1. Twelve Critical Financial Scenarios Tested (`-race`)

| # | Financial Scenario | Test Function | Result |
| :--- | :--- | :--- | :--- |
| 1 | Duplicate payment callback / webhook replay | `TestBillingWebhook_SecurityAndReplay/ValidWebhookAndReplayDeduplication` | **PASS** |
| 2 | Concurrent wallet invoice payment requests (10-way race on same invoice) | `TestP14_ConcurrentPayInvoiceWithWalletAndPlanScopedCouponsAndSweepPersistence` | **PASS** (Wallet debited strictly once `$25.00`; `onInvoicePaid` provisioning fired strictly once) |
| 3 | Paying cancelled or refunded invoice via wallet | `TestP14_ConcurrentPayInvoiceWithWalletAndPlanScopedCouponsAndSweepPersistence` | **PASS** (`400 INVALID_INVOICE_STATUS`) |
| 4 | Concurrent wallet double-spend attempt (2x `$15` debits on `$20` balance) | `TestCommercial_WalletDoubleSpendPreventionAndPublicStatus` | **PASS** (Strictly 1 debit succeeds) |
| 5 | Negative balance / invalid adjustment direction attempts | `TestP10_WalletAdjustAndWithdrawalInputValidation` | **PASS** (`400 Bad Request`) |
| 6 | Rejected withdrawal re-approval exploit & 5-way concurrent approval race | `TestP10_WithdrawalRejectThenApproveBlockedAndConcurrentApproval` | **PASS** (Re-approval blocked; 5-way race credits wallet strictly once) |
| 7 | Failed wallet checkout preserving single-use coupon | `TestP10_CouponInputValidationAndWalletPrecheckPreservesCoupon` | **PASS** (Coupon not burned on insufficient funds) |
| 8 | Verified refund webhook reversing affiliate commission | `TestP10_WebhookRefundReversesAffiliateCommission` | **PASS** (`$15.00 -> $0.00`) |
| 9 | Concurrent coupon redemption limit (10 buyers on `MaxUses=3` coupon) | `TestCommercial_CouponConcurrentRedemptionLimit` | **PASS** (Strictly 3 redemptions) |
| 10 | Plan-restricted coupon (`ApplicablePlanIDs`) enforcement | `TestP14_ConcurrentPayInvoiceWithWalletAndPlanScopedCouponsAndSweepPersistence` | **PASS** (`400` on Starter plan, `201` on Enterprise plan) |
| 11 | IEEE-754 cent precision (`$15.30 - $5.10 = $10.20` exact withdrawal) | `TestP12_FinancialInvariantsLedgerReconciliationAndPrecision` | **PASS** |
| 12 | 35-entry concurrent double-entry wallet ledger reconciliation | `TestP12_FinancialInvariantsLedgerReconciliationAndPrecision` | **PASS** (`$203.00 - $152.25 = $50.75`) |
