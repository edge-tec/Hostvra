# HOSTVRA — PHASE P15: CRYPTO & HD WALLET SECURITY AUDIT

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P15.6 — Crypto & HD Wallet Security Audit  
**Status:** **PASS**

---

## 1. Multi-Chain & HD Wallet Architecture Audit

Audited `apps/api/internal/crypto/` (`service.go`, `hd_wallet.go`, `treasury_sweep.go`, `signer.go`, `rpc.go`) and `apps/api/internal/store/` (`crypto_store.go`, `hd_wallet_store.go`):

| Security Control | Implementation Details | Verification Evidence | Verdict |
| :--- | :--- | :--- | :---: |
| **HD Derivation Index Reservation** | `ReserveNextDerivationIndex` locks `m.mu`, starts at index `1` (reserving index `0` for merchant primary address), increments monotonically, and calls `m.saveToDiskLocked()` | `hd_wallet_store.go:76-93` | **PASS** |
| **Duplicate Address & Index Prevention** | `CreateHDWalletAddress` rejects duplicate `(NetworkID, DerivationIndex)` and duplicate `(NetworkID, DepositAddress)` | `hd_wallet_store.go:95-119` | **PASS** |
| **Duplicate On-Chain `TxHash` Prevention** | `CreateCryptoInvoice`, `UpdateCryptoInvoice`, and `VerifyInvoiceOnChain` reject any `TxHash` already bound to another `paid` or `confirming` invoice (`ErrCryptoDuplicateTxHash`) | `crypto_store.go:406-412, 536-542`, `service.go:414-418` | **PASS** |
| **Token Contract & Network Validation** | `VerifyEVMTransaction` and `VerifyTronTransaction` validate the token contract address (`USDT_TRC20`, `USDT_BEP20`, `USDT_ERC20`), recipient address, decimals, and minimum confirmation thresholds | `crypto/rpc.go` & `crypto/service.go` | **PASS** |
| **Private Key & Seed Isolation** | `IsolatedSigner` encrypts master mnemonic seeds with AES-256-GCM; struct fields carry `json:"-"` on API responses so seeds/private keys are never serialized over HTTP | `crypto/signer.go` & `store/hd_wallet_models.go` | **PASS** |
| **Treasury Sweep Idempotency & Caps** | `CreateTreasurySweepJob` enforces unique `IdempotencyKey` (`ErrDuplicateSweepJob`), checks `emergency_sweep_pause`, `min_sweep_usd`, `max_single_sweep_usd`, and `max_daily_sweep_usd` (`GetDailySweptUSD`) | `crypto/treasury_sweep.go` & `hd_wallet_store.go` | **PASS** |
| **Complete On-Disk Persistence (P13 + P14 + P15)** | All HD wallet accounts, addresses, deposit allocations, sweep jobs, sweep logs, derivation audits (P14-01), legacy deposit address pools, and internal crypto ledger entries (P15-03) flush atomically via `m.saveToDiskLocked()` | `TestP13_*`, `TestP14_*`, `TestP15_*` | **PASS** |
