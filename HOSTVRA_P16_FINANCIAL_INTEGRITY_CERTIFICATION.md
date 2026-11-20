# HOSTVRA — PHASE P16.6: FINANCIAL INTEGRITY & PAYMENT CERTIFICATION

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P16.6 — Financial Integrity and Payment Certification  
**Status:** **PASS (15/15 Financial Invariants Certified Under Go Race Detector)**

---

## 1. Verification of 15 Mandatory Financial Requirements

| # | Financial Area | Verification Mechanism & Invariant Guard | Automated Test Evidence | Result |
| :-: | :--- | :--- | :--- | :---: |
| **1** | **Wallet Payment Settlement** | `CommercialHandler.PayInvoiceWithWallet` serialized via `payMu` + `DebitWalletAtomic` (`wallet_pay:<invID>`) | `TestP14_ConcurrentPayInvoiceWithWalletAndPlanScopedCouponsAndSweepPersistence` | **PASS** |
| **2** | **Signed Webhook Validation** | HMAC-SHA256 signature verification + official Stripe `t=...,v1=...` verification in `HandleWebhook` | `TestBillingWebhook_SecurityAndReplay` | **PASS** |
| **3** | **Webhook Replay Protection** | `GetDomainWebhook` / `RecordDomainWebhook` persisted to disk (`P16-01`) under `BillingHandler.settleMu` (`P15-01`) | `TestP16_WebhookReplayDeduplicationPersistsAcrossColdRestartAndDomainOrdersPersist` | **PASS** |
| **4** | **Invoice State Transitions** | Strict state transitions (`unpaid`/`overdue` -> `paid` -> `refunded`); `cancelled` & `refunded` cannot transition to `paid` (`P15-02`) | `TestP14_*` & `TestP15_*` | **PASS** |
| **5** | **Duplicate Payment Prevention** | 10 concurrent wallet requests and 10 concurrent signed webhooks each settle the invoice **exactly once** | `TestP14_*` & `TestP15_*` | **PASS** |
| **6** | **Refund State Transitions** | Refund webhooks transition paid invoice to `refunded` idempotently and record replay protection | `TestP10_WebhookRefundReversesAffiliateCommission` | **PASS** |
| **7** | **Affiliate Commission Reversal** | `ReverseAffiliateCommissionByInvoice` deducts commission from affiliate balance on refund/chargeback | `TestP10_WebhookRefundReversesAffiliateCommission` | **PASS** |
| **8** | **Subscription Provisioning** | `ActivatePaidInvoice` / `HandleWebhook` / `SettleCryptoPayment` creates or renews exactly 1 subscription per invoice | `TestP15_*` | **PASS** |
| **9** | **Hosting Account Provisioning** | `provisionUserForPlan` creates 1 hosting account on initial purchase and unsuspends/updates quotas on renewal | `TestP14_*` & `TestP15_*` | **PASS** |
| **10** | **Coupon Eligibility & Limits** | Atomic global `MaxUses`, per-user `MaxUsesPerUser`, `FirstOrderOnly`, `MinOrderUSD`, and `isCouponApplicableToPlan` | `TestCommercial_CouponConcurrentRedemptionLimit` & `TestP14_*` | **PASS** |
| **11** | **Currency Precision** | 2-decimal cent rounding (`math.Round(...*100)/100`) on USD invoices, discounts, and affiliate balances; 4-decimal ledger rounding | `TestP12_FinancialInvariantsLedgerReconciliationAndPrecision` | **PASS** |
| **12** | **Ledger Debit/Credit Balance** | Every wallet balance equals $\sum \text{credits} - \sum \text{debits}$ across immutable `WalletLedgerEntry` records | `TestP12_FinancialInvariantsLedgerReconciliationAndPrecision` | **PASS** |
| **13** | **Payment Reconciliation** | Every settled invoice creates a matching `PaymentTransaction` or `WalletLedgerEntry` and updates linked `DomainOrder` (`P16-02`) | `TestP15_*` & `TestP16_*` | **PASS** |
| **14** | **Crypto Transaction Verification** | `CryptoService.VerifyInvoiceOnChain` serialized via `s.mu` (`P16-03`) + `ErrCryptoDuplicateTxHash` anti-replay | `go test -race ./internal/crypto/...` | **PASS** |
| **15** | **Treasury Sweep Idempotency** | `CreateTreasurySweepJob` enforces unique `IdempotencyKey` (`ErrDuplicateSweepJob`) and persists to disk (`P14-01`) | `TestP14_*` | **PASS** |
