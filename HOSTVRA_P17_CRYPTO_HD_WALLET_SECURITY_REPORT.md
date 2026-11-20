# HOSTVRA — PHASE P17.8: CRYPTO & HD WALLET SECURITY REPORT

**UTC Timestamp:** `2026-11-20T14:26:04Z`  
**Verification Method:** Source code audit (`internal/crypto/`, `internal/store/`) + `go test -count=1 -race ./internal/crypto/...`  
**Status:** **PASS (LOCAL)**

---

## 1. Cryptographic & HD Wallet Controls Verified
- **Vault Encryption:** AES-256-GCM encrypted master seed stored at `/var/lib/hostvra/vault/hd_master.enc` with `0600` file permissions (`crypto/signer.go:115-243`).
- **Private Key Isolation & Zeroization:** Ephemeral child private keys are derived in memory inside `SignEVMLegacySweepTx` and immediately wiped via `defer privKey.Zero()` (`crypto/signer.go:341-345`). No private key or mnemonic is ever logged or serialized to HTTP responses.
- **Derivation Index Protection:** `ReserveNextDerivationIndex` (`hd_wallet_store.go:76-93`) starts at index `1` (protecting merchant Trust Wallet index `0`), increments atomically under `m.mu.Lock()`, and flushes immediately via `m.saveToDiskLocked()`.
- **Treasury Allowlist:** `SignEVMLegacySweepTx` rejects any destination not present in the admin-approved Trust Wallet allowlist (`crypto/signer.go:331-338`).
- **Concurrent On-Chain Verification & Duplicate Deposit Protection:** `CryptoService.VerifyInvoiceOnChain` is serialized with `s.mu.Lock()` (`crypto/service.go:222-225`), and `CreateCryptoInvoice`/`UpdateCryptoInvoice` reject duplicate `TxHash` (`ErrCryptoDuplicateTxHash`).
- **Automated Test Evidence:** `ok hostvra/api/internal/crypto 2.023s` (`-count=1 -race`).
- **Remaining Risk:** Live `/var/lib/hostvra/vault/hd_master.enc` file permissions (`0600`) must be confirmed on the Contabo VPS during operator cutover.
