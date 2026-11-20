# HOSTVRA — PHASE P16.5: GO BACKEND ENTERPRISE AUDIT

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P16.5 — Go Backend Complete Production Audit  
**Status:** **PASS (3 New Edge-Case Bugs Found & Resolved in P16)**

---

## 1. Re-Audit of P15 Fixes (P15-01, P15-02, P15-03)

1. **P15-01 (Webhook & Payment Settlement Concurrency)**: Verified `BillingHandler.settleMu` serializes `SettleCryptoPayment`, `PayInvoice`, and `HandleWebhook`, and `CommercialHandler.payMu` serializes `PayInvoiceWithWallet`.
2. **P15-02 (Cancelled & Refunded Invoice Protection)**: Verified all 6 invoice settlement entrypoints (`PayInvoiceWithWallet`, `PayInvoice`, `CheckoutInvoice`, `VerifyInvoicePayment`, `HandleWebhook`, `SettleCryptoPayment`) reject `InvoiceStatusCancelled` and `InvoiceStatusRefunded`.
3. **P15-03 (Crypto Deposit Pool & Internal Ledger Persistence)**: Verified `UpdateCryptoNetwork`, `UpdateCryptoCurrency`, `AcquireDepositAddress`, `ReleaseDepositAddress`, `CreateDepositAddress`, and `RecordCryptoLedger` call `m.saveToDiskLocked()`.

---

## 2. Three New Edge-Case Bugs Discovered & Fixed in Phase P16

### Bug P16-01 — Webhook Replay Deduplication Table (`domainWebhooks`) Not Persisted Across Cold Restarts (`apps/api/internal/store/store.go` & `domain_store.go`)
- **Problem**: `BillingHandler.HandleWebhook` records processed webhook event IDs via `h.store.RecordDomainWebhook` and checks `h.store.GetDomainWebhook` to block replay attacks. In `MemoryStore`, `m.domainWebhooks` was omitted from `memoryDumpData` and `RecordDomainWebhook` did not call `m.saveToDiskLocked()`. If the API process restarted and a gateway re-sent an old refund or payment webhook with a new transaction ID or before invoice state check, the webhook replay cache was empty.
- **Fix**: Added `DomainWebhooks map[string]*DomainWebhook` to `memoryDumpData`, `saveToDiskLocked()`, and `loadFromDisk()` in `apps/api/internal/store/store.go`, and added `m.saveToDiskLocked()` to `RecordDomainWebhook` in `apps/api/internal/store/domain_store.go`.
- **Verification**: Proven by `TestP16_WebhookReplayDeduplicationPersistsAcrossColdRestartAndDomainOrdersPersist`.

### Bug P16-02 — Domain Order, Transfer & Renewal Mutators Missing `m.saveToDiskLocked()` (`apps/api/internal/store/domain_store.go`)
- **Problem**: `CreateDomainOrder`, `UpdateDomainOrder`, `CreateDomainTransfer`, `UpdateDomainTransfer`, and `CreateDomainRenewal` modified `MemoryStore` maps without calling `m.saveToDiskLocked()`. If a customer paid an invoice linked to a `DomainOrder`, `UpdateDomainOrder` marked `PaymentStatus = "paid"` in memory without flushing to disk unless another store write occurred afterward.
- **Fix**: Added `m.saveToDiskLocked()` to all 5 methods in `apps/api/internal/store/domain_store.go`.
- **Verification**: Proven by `TestP16_WebhookReplayDeduplicationPersistsAcrossColdRestartAndDomainOrdersPersist`.

### Bug P16-03 — Concurrent `VerifyInvoiceOnChain` Race Between Background Scanner and Manual API Call (`apps/api/internal/crypto/service.go`)
- **Problem**: `CryptoService.VerifyInvoiceOnChain` can be invoked simultaneously by the 20-second background scanner (`ScanAllPendingInvoices`), `GET /api/v1/crypto/invoices/{code}`, and `POST /api/v1/crypto/invoices/{code}/verify`. Without locking `s.mu` inside `VerifyInvoiceOnChain`, two concurrent calls for the same confirming crypto invoice could both observe `inv.Status != "paid"`, both transition it to `"paid"`, and both append a `CryptoInternalLedger` entry.
- **Fix**: Added `s.mu.Lock()` / `defer s.mu.Unlock()` at the entry of `VerifyInvoiceOnChain` in `apps/api/internal/crypto/service.go`.
