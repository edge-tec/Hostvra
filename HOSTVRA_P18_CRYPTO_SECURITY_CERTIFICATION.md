# HOSTVRA — PHASE P18.8: CRYPTO SECURITY CERTIFICATION

**UTC Timestamp:** `2026-11-20T14:39:18Z`  
**Status:** **PASS (LOCAL `-count=1 -race`)**

---

## 1. Crypto & Treasury Security Controls
- **Vault Encryption:** AES-256-GCM encrypted master seed at `/var/lib/hostvra/vault/hd_master.enc` (`0600` permissions).
- **Private Key Handling:** Ephemeral child keys are wiped via `defer privKey.Zero()` immediately after signing; zero private keys or seeds appear in logs, API responses, or Git history.
- **Treasury Allowlist:** `SignEVMLegacySweepTx` rejects any sweep destination not in the admin-approved Trust Wallet allowlist.
- **Duplicate Transaction Rejection:** `ErrCryptoDuplicateTxHash` blocks reusing any `TxHash` across multiple invoices.
- **Test Output:** `ok hostvra/api/internal/crypto 2.251s` (`-count=1 -race`).
