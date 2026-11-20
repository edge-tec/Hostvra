# HOSTVRA — P13 FINANCIAL LIVE & STAGING VERIFICATION REPORT

**Execution Date & Time:** 2026-11-20T17:42:00+06:00  
**Test Suite:** `go test -v -race ./internal/handlers -run "TestCommercial_|TestP10_|TestP12_|TestP13_|TestBillingWebhook_"`

---

## 1. Financial Workflow Verification Summary

| Financial Workflow | Verification Evidence | Result |
| :--- | :--- | :--- |
| **Live Billing Ledger & Crypto Gateway Health** | `GET https://hostvra.com/api/v1/public/status` -> `Primary Database & Billing Ledger: operational (0ms)`, `In-House Crypto Gateway & HD Wallet: operational (7 active blockchain networks monitored)`. | **PASS (Live Verified)** |
| **Deposit & Double-Entry Wallet Credit/Debit** | 20 concurrent `$10.15` credits (`$203.00`) + 15 concurrent `$10.15` debits (`$152.25`) = exact `$50.75` balance across 35 contiguous ledger records; double-spend blocked (`TestCommercial_WalletDoubleSpendPreventionAndPublicStatus`). | **PASS (`-race`)** |
| **Withdrawal Processing & Anti-Exploit Guard** | Rejected withdrawal cannot be re-approved (`ErrWithdrawalAlreadyProcessed`); 5 concurrent approvals credit wallet strictly once (`TestP10_WithdrawalRejectThenApproveBlockedAndConcurrentApproval`). | **PASS (`-race`)** |
| **Affiliate Balance Cent Precision** | IEEE-754 drift test (`$15.30 - $5.10 = $10.20` exact remaining withdrawal) passes with 2-decimal cent rounding (`TestP12_FinancialInvariantsLedgerReconciliationAndPrecision`). | **PASS (`-race`)** |
| **Webhook Idempotency & Refund Reversal** | Paid webhook credits 15% commission (`$15.00`); replayed webhook is deduplicated; refund webhook transitions invoice to `refunded` and reverses commission to `$0.00` (`TestP10_WebhookRefundReversesAffiliateCommission`). | **PASS (`-race`)** |
| **On-Disk Financial Persistence (`store.json`)** | Fixed in P13: all wallet balances, immutable wallet ledger entries, coupon redemptions, affiliate commissions, and withdrawals are atomically persisted to `store.json` via temporary-file + rename (`saveToDiskLocked`) on every mutation (`TestP13_RealOnDiskDatabaseBackupAndRestore`). | **PASS (`-race`)** |
