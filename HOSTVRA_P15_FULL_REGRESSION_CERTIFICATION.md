# HOSTVRA — PHASE P15: FULL REGRESSION & RACE CERTIFICATION

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P15.10 — Full Regression & Race Testing  
**Status:** **PASS (100% Tests Passing Under `-race`, Zero Warnings in `go vet ./...`)**

---

## 1. Regression Test Execution Log (`go test -v -race`)

Executed across all backend packages with the Go race detector enabled:

```text
=== RUN   TestBillingWebhook_SecurityAndReplay
    --- PASS: TestBillingWebhook_SecurityAndReplay/InvalidSignature (0.00s)
    --- PASS: TestBillingWebhook_SecurityAndReplay/MissingSignatureInProductionMode (0.00s)
    --- PASS: TestBillingWebhook_SecurityAndReplay/StripeOfficialSignatureScheme (0.00s)
    --- PASS: TestBillingWebhook_SecurityAndReplay/AmountMismatch (0.00s)
    --- PASS: TestBillingWebhook_SecurityAndReplay/CurrencyMismatch (0.00s)
    --- PASS: TestBillingWebhook_SecurityAndReplay/ValidWebhookAndReplayDeduplication (0.00s)
--- PASS: TestBillingWebhook_SecurityAndReplay (0.00s)
=== RUN   TestCommercial_CouponConcurrentRedemptionLimit
--- PASS: TestCommercial_CouponConcurrentRedemptionLimit (0.00s)
=== RUN   TestCommercial_AffiliateSelfReferralAndReversal
--- PASS: TestCommercial_AffiliateSelfReferralAndReversal (0.00s)
=== RUN   TestCommercial_WalletDoubleSpendPreventionAndPublicStatus
--- PASS: TestCommercial_WalletDoubleSpendPreventionAndPublicStatus (0.00s)
=== RUN   TestCommercial_UpdateCouponAndToggle
--- PASS: TestCommercial_UpdateCouponAndToggle (0.00s)
=== RUN   TestCommercial_EndToEndReferralCheckoutAndCommission
--- PASS: TestCommercial_EndToEndReferralCheckoutAndCommission (0.54s)
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
    commercial_test.go:689: [DR CERTIFIED] RPO=0 verified | RTO=210.25µs | SnapshotBytes=1403
--- PASS: TestP12_DisasterRecoveryBackupAndRestoreIntegrity (0.00s)
=== RUN   TestP12_PerformanceThroughputAndLatencyMetrics
    commercial_test.go:740: [PERF CERTIFIED] Requests=300 Concurrency=20 Throughput=17783.7 req/s | P50=576.167µs P95=2.025667ms P99=2.260834ms
--- PASS: TestP12_PerformanceThroughputAndLatencyMetrics (0.02s)
=== RUN   TestP13_RealOnDiskDatabaseBackupAndRestore
    commercial_test.go:840: [P13 ON-DISK DR CERTIFIED] FileBytes=41058 SHA256=10029c2c243b9694 BackupTime=167.25µs RestoreRTO=3.164875ms RPO=0
--- PASS: TestP13_RealOnDiskDatabaseBackupAndRestore (0.03s)
=== RUN   TestP14_ConcurrentPayInvoiceWithWalletAndPlanScopedCouponsAndSweepPersistence
--- PASS: TestP14_ConcurrentPayInvoiceWithWalletAndPlanScopedCouponsAndSweepPersistence (0.03s)
=== RUN   TestP15_ConcurrentWebhookReplayAndCryptoPoolPersistenceAndCancelledInvoiceGuard
--- PASS: TestP15_ConcurrentWebhookReplayAndCryptoPoolPersistenceAndCancelledInvoiceGuard (0.04s)
PASS
ok  	hostvra/api/internal/handlers	2.897s
ok  	hostvra/api/internal/store	1.435s
ok  	hostvra/api/internal/billing	1.382s
ok  	hostvra/api/internal/crypto	1.942s
```

---

## 2. Production Linux AMD64 Binary Verification

- **Build Command**: `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/hostvra-api-linux-amd64 ./cmd/server`
- **File Type**: `ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked`
- **SHA-256**: `ad3a72982b21580a39ec11aeb45e3e58b03ee61bb548c38722f263c85fc7ce98`
