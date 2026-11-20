# HOSTVRA — PHASE P17.7: FINANCIAL INTEGRITY CERTIFICATION

**UTC Timestamp:** `2026-11-20T14:26:04Z`  
**Verification Method:** Non-cached Go race-enabled concurrency tests (`go test -count=1 -race`)  
**Status:** **PASS (LOCAL — 100% Invariants Verified)**

---

## 1. Financial Invariant Verification Matrix

| Test ID | Expected Behavior | Actual Result | Evidence | Verdict |
| :--- | :--- | :--- | :--- | :---: |
| **FIN-01** | 10 concurrent wallet payments for same invoice settle & provision exactly once | 1 success, 9 idempotent, 1 callback | `TestP14_ConcurrentPayInvoiceWithWalletAndPlanScopedCouponsAndSweepPersistence` | **PASS** |
| **FIN-02** | 10 concurrent signed Stripe webhooks create exactly 1 subscription, 1 hosting account, 1 transaction | 1 sub, 1 account, 1 txn | `TestP15_ConcurrentWebhookReplayAndCryptoPoolPersistenceAndCancelledInvoiceGuard` | **PASS** |
| **FIN-03** | Webhook replay deduplication persists across cold restart from disk | Replayed webhook returns `idempotent_duplicate` | `TestP16_WebhookReplayDeduplicationPersistsAcrossColdRestartAndDomainOrdersPersist` | **PASS** |
| **FIN-04** | Cancelled & refunded invoices cannot be paid via wallet, gateway, or webhook | Rejected with `HTTP 400 INVALID_INVOICE_STATUS` | `TestP14_*` & `TestP15_*` | **PASS** |
| **FIN-05** | Refund webhook reverses affiliate commission and deducts exact cent-rounded balance | Commission marked `reversed`, balance reconciled | `TestP10_WebhookRefundReversesAffiliateCommission` | **PASS** |
| **FIN-06** | Rejected affiliate withdrawal cannot be re-approved or double-credited | Rejected with `ErrWithdrawalAlreadyProcessed` | `TestP10_WithdrawalRejectThenApproveBlockedAndConcurrentApproval` | **PASS** |
| **FIN-07** | Wallet ledger balance equals $\sum \text{credits} - \sum \text{debits}$ with 0 rounding drift | Exact cent reconciliation verified | `TestP12_FinancialInvariantsLedgerReconciliationAndPrecision` | **PASS** |
| **FIN-08** | Single-use coupon is not burned when wallet balance is insufficient | Coupon preserved on failed checkout | `TestP10_CouponInputValidationAndWalletPrecheckPreservesCoupon` | **PASS** |
| **FIN-09** | Plan-scoped coupons reject ineligible hosting plans | Rejected with `HTTP 400 COUPON_PLAN_INELIGIBLE` | `TestP14_*` | **PASS** |

**Remaining Risk:** Live production enforcement requires operator cutover on `root@hostvra`.
