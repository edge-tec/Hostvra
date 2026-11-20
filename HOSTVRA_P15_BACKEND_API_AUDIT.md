# HOSTVRA — PHASE P15: COMPLETE BACKEND & API AUDIT

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P15.4 — Complete Backend & API Audit  
**Status:** **PASS (3 Additional Concurrency, State-Guard & Persistence Bugs Discovered & Fixed in P15)**

---

## 1. Go Backend Packages Audited

Inspected all backend packages in `apps/api/internal/`:
- `handlers/billing.go` (2,983 lines) — Plans, subscriptions, free trials, invoices, gateway checkout, payment verification, webhook receiver, automated billing lifecycle worker.
- `handlers/commercial.go` (797 lines) — Public status, coupon validation/CRUD, authoritative order checkout, credit wallet & auto-pay, affiliate dashboard/withdrawals, dunning trigger.
- `handlers/crypto.go` (651 lines) — Crypto currencies, crypto invoice creation & on-chain verification, treasury quick setup, HD wallet & sweep management.
- `crypto/service.go`, `crypto/hd_wallet.go`, `crypto/treasury_sweep.go`, `crypto/signer.go` — Multi-chain RPC verification, BIP44/SLIP44 address derivation, AES-256-GCM seed encryption, automated treasury sweep engine.
- `store/store.go`, `store/commercial_store.go`, `store/crypto_store.go`, `store/hd_wallet_store.go` — MemoryStore (with atomic disk snapshot `/var/lib/hostvra/store.json`) and PostgresStore implementations.

---

## 2. Bugs Discovered & Fixed in Phase P15

### Bug P15-01 — Concurrent Webhook & Gateway Settlement Race (`apps/api/internal/handlers/billing.go`)
- **Discovery**: While `CommercialHandler.PayInvoiceWithWallet` was serialized with `payMu sync.Mutex` in P14, `BillingHandler` (`HandleWebhook`, `PayInvoice`, `SettleCryptoPayment`, `VerifyInvoicePayment`) had no settlement mutex. If a payment gateway delivered two concurrent webhook requests for the same invoice at the exact same millisecond, both goroutines could read `inv.Status == store.InvoiceStatusUnpaid` before `RecordDomainWebhook` or `UpdateInvoice` completed, leading to duplicate `CreateSubscription`, duplicate `provisionUserForPlan`, and duplicate `RecordPaymentTransaction`.
- **Fix Applied**: Added `settleMu sync.Mutex` to `BillingHandler` and acquired `h.settleMu.Lock()` / `defer h.settleMu.Unlock()` in `SettleCryptoPayment`, `PayInvoice`, and `HandleWebhook`.
- **Regression Test**: Verified by 10-goroutine race test in `TestP15_ConcurrentWebhookReplayAndCryptoPoolPersistenceAndCancelledInvoiceGuard`.

### Bug P15-02 — Missing `cancelled` / `refunded` Invoice Status Guards in Gateway/Webhook Settlement (`apps/api/internal/handlers/billing.go`)
- **Discovery**: `PayInvoice`, `CheckoutInvoice`, `VerifyInvoicePayment`, `HandleWebhook` (for `"paid"` events), and `SettleCryptoPayment` checked `if inv.Status == store.InvoiceStatusPaid` for idempotency, but did not block `InvoiceStatusCancelled` or `InvoiceStatusRefunded`. A late webhook or manual pay call could transition a cancelled or refunded invoice back to `paid`.
- **Fix Applied**: Added explicit `if inv.Status == store.InvoiceStatusCancelled || inv.Status == store.InvoiceStatusRefunded` guards returning `HTTP 400 INVALID_INVOICE_STATUS` (or error in `SettleCryptoPayment`) across all 5 settlement entrypoints.
- **Regression Test**: Verified in `TestP15_ConcurrentWebhookReplayAndCryptoPoolPersistenceAndCancelledInvoiceGuard`.

### Bug P15-03 — Legacy Crypto Deposit Pool & Internal Crypto Ledger Missing `m.saveToDiskLocked()` (`apps/api/internal/store/crypto_store.go`)
- **Discovery**: In `apps/api/internal/store/crypto_store.go`, six `MemoryStore` mutators modified in-memory maps/slices under `m.mu.Lock()` without calling `m.saveToDiskLocked()`:
  1. `UpdateCryptoNetwork`
  2. `UpdateCryptoCurrency`
  3. `AcquireDepositAddress` (all 3 allocation branches)
  4. `ReleaseDepositAddress`
  5. `CreateDepositAddress`
  6. `RecordCryptoLedger`
- **Fix Applied**: Added `m.saveToDiskLocked()` to all 6 methods in `apps/api/internal/store/crypto_store.go`.
- **Regression Test**: Verified cold disk reload persistence in `TestP15_ConcurrentWebhookReplayAndCryptoPoolPersistenceAndCancelledInvoiceGuard`.
