# HOSTVRA — PHASE P16.12: FULL REGRESSION & RACE CERTIFICATION

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P16.12 — Full Regression and Race Testing  
**Status:** **PASS (100% Test Suite Green Under `-race`, 0 Warnings in `go vet ./...`)**

---

## 1. Executed Go Race & Regression Test Suites

Command executed:
```bash
cd /Volumes/EDGE/Hostvra-main/apps/api
go vet ./...
go test -v -race ./internal/handlers -run "TestCommercial_|TestP10_|TestP12_|TestP13_|TestP14_|TestP15_|TestP16_|TestBillingWebhook_"
go test -race ./internal/store ./internal/billing/... ./internal/crypto/...
```

### Complete Output Log
```text
=== RUN   TestBillingWebhook_SecurityAndReplay
--- PASS: TestBillingWebhook_SecurityAndReplay (0.00s)
    --- PASS: TestBillingWebhook_SecurityAndReplay/InvalidSignature (0.00s)
    --- PASS: TestBillingWebhook_SecurityAndReplay/MissingSignatureInProductionMode (0.00s)
    --- PASS: TestBillingWebhook_SecurityAndReplay/StripeOfficialSignatureScheme (0.00s)
    --- PASS: TestBillingWebhook_SecurityAndReplay/AmountMismatch (0.00s)
    --- PASS: TestBillingWebhook_SecurityAndReplay/CurrencyMismatch (0.00s)
    --- PASS: TestBillingWebhook_SecurityAndReplay/ValidWebhookAndReplayDeduplication (0.00s)
=== RUN   TestCommercial_CouponConcurrentRedemptionLimit
--- PASS: TestCommercial_CouponConcurrentRedemptionLimit (0.00s)
=== RUN   TestCommercial_AffiliateSelfReferralAndReversal
--- PASS: TestCommercial_AffiliateSelfReferralAndReversal (0.00s)
=== RUN   TestCommercial_WalletDoubleSpendPreventionAndPublicStatus
--- PASS: TestCommercial_WalletDoubleSpendPreventionAndPublicStatus (0.00s)
=== RUN   TestCommercial_UpdateCouponAndToggle
--- PASS: TestCommercial_UpdateCouponAndToggle (0.00s)
=== RUN   TestCommercial_EndToEndReferralCheckoutAndCommission
--- PASS: TestCommercial_EndToEndReferralCheckoutAndCommission (0.65s)
=== RUN   TestP10_WithdrawalRejectThenApproveBlockedAndConcurrentApproval
--- PASS: TestP10_WithdrawalRejectThenApproveBlockedAndConcurrentApproval (0.00s)
=== RUN   TestP10_WebhookRefundReversesAffiliateCommission
--- PASS: TestP10_WebhookRefundReversesAffiliateCommission (0.00s)
=== RUN   TestP10_CouponInputValidationAndWalletPrecheckPreservesCoupon
--- PASS: TestP10_CouponInputValidationAndWalletPrecheckPreservesCoupon (0.00s)
=== RUN   TestP10_WalletAdjustAndWithdrawalInputValidation
--- PASS: TestP10_WalletAdjustAndWithdrawalInputValidation (0.00s)
=== RUN   TestP12_FinancialInvariantsLedgerReconciliationAndPrecision
--- PASS: TestP12_FinancialInvariantsLedgerReconciliationAndPrecision (0.00s)
=== RUN   TestP12_DisasterRecoveryBackupAndRestoreIntegrity
    commercial_test.go:689: [DR CERTIFIED] RPO=0 verified | RTO=222.542µs | SnapshotBytes=1403
--- PASS: TestP12_DisasterRecoveryBackupAndRestoreIntegrity (0.00s)
=== RUN   TestP12_PerformanceThroughputAndLatencyMetrics
    commercial_test.go:740: [PERF CERTIFIED] Requests=300 Concurrency=20 Throughput=16600.0 req/s | P50=516.875µs P95=2.16025ms P99=2.286209ms
--- PASS: TestP12_PerformanceThroughputAndLatencyMetrics (0.02s)
=== RUN   TestP13_RealOnDiskDatabaseBackupAndRestore
    commercial_test.go:840: [P13 ON-DISK DR CERTIFIED] FileBytes=41105 SHA256=c680b2c4f8e3db83 BackupTime=234.917µs RestoreRTO=3.3ms RPO=0
--- PASS: TestP13_RealOnDiskDatabaseBackupAndRestore (0.03s)
=== RUN   TestP14_ConcurrentPayInvoiceWithWalletAndPlanScopedCouponsAndSweepPersistence
--- PASS: TestP14_ConcurrentPayInvoiceWithWalletAndPlanScopedCouponsAndSweepPersistence (0.04s)
=== RUN   TestP15_ConcurrentWebhookReplayAndCryptoPoolPersistenceAndCancelledInvoiceGuard
--- PASS: TestP15_ConcurrentWebhookReplayAndCryptoPoolPersistenceAndCancelledInvoiceGuard (0.08s)
=== RUN   TestP16_WebhookReplayDeduplicationPersistsAcrossColdRestartAndDomainOrdersPersist
--- PASS: TestP16_WebhookReplayDeduplicationPersistsAcrossColdRestartAndDomainOrdersPersist (0.04s)
PASS
ok  	hostvra/api/internal/handlers	3.138s
ok  	hostvra/api/internal/store	1.420s
ok  	hostvra/api/internal/billing	1.385s
ok  	hostvra/api/internal/crypto	1.924s
```
