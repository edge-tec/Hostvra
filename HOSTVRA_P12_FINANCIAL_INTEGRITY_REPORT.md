# HOSTVRA — P12 FINANCIAL INTEGRITY REPORT

**Audit Date & Time:** 2026-11-20T17:19:30+06:00  
**Certification Status:** **PASS**  
**Test Command:** `go test -v -race ./internal/handlers -run "TestCommercial_|TestP10_|TestP12_|TestBillingWebhook_"`

---

## 1. Financial Invariants Verification

| # | Financial Invariant | Verification Evidence | Status |
| :--- | :--- | :--- | :--- |
| 1 | No unexplained balance changes | `TestP12_FinancialInvariantsLedgerReconciliationAndPrecision` verifies 20 concurrent `$10.15` credits (`$203.00`) and 15 concurrent `$10.15` debits (`$152.25`) reconcile to exact `$50.75` wallet balance. | **PASS** |
| 2 | No duplicate credits or debits | `CreditWalletAtomic` and `DebitWalletAtomic` enforce unique `idempotency_key`; 5 concurrent withdrawal approvals (`TestP10_WithdrawalRejectThenApproveBlockedAndConcurrentApproval`) credit wallet strictly once. | **PASS** |
| 3 | No negative balances | `DebitWalletAtomic` rejects debits exceeding `BalanceUSD`; `RequestAffiliateWithdrawal` rejects amounts exceeding `ApprovedBalanceUSD` or `< $0.01`. | **PASS** |
| 4 | Auditable double-entry ledger | All 35 concurrent operations in `TestP12_FinancialInvariantsLedgerReconciliationAndPrecision` produce contiguous `BalanceBeforeUSD` -> `BalanceAfterUSD` ledger records summing identically to `wallet.BalanceUSD`. | **PASS** |
| 5 | Failed transactions cannot partially modify balances | PostgreSQL `BeginTx` + `FOR UPDATE` rollback on error; `CheckoutOrder` pre-checks wallet balance before redeeming single-use coupons (`TestP10_CouponInputValidationAndWalletPrecheckPreservesCoupon`). | **PASS** |
| 6 | IEEE-754 decimal precision safety | Fixed in P12 (`P12-03`): `$15.30 - $5.10` withdrawal leaves exact `$10.20` (`math.Round(...*100)/100`), allowing exact remaining balance withdrawal without float drift rejection. | **PASS** |
| 7 | Refund commission reversal | `TestP10_WebhookRefundReversesAffiliateCommission` verifies HMAC-signed refund webhooks transition paid invoices to `refunded` and reverse affiliate commissions to `$0.00`. | **PASS** |
