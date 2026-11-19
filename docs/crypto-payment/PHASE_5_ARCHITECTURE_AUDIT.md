# Phase 5 — Architecture Audit (Hostvra Enterprise HD Wallet & Automated Treasury Engine)

**Repository:** `/Volumes/EDGE/Hostvra-main`  
**Stack:** Go 1.26 Backend (`apps/api`) + Next.js 15 / React / TypeScript Frontend (`apps/web`) + PostgreSQL / Disk-Persisted Store  
**Audit Date:** 2026-11-19

---

## 1. Deployment & Trust Model Classification

Hostvra operates under **Trust Model #3 (Trusted Central Server Backend + Thin Web Frontend)**:

- **Trusted Boundary (Server-Side Go Binary — `/usr/local/bin/hostvra-api`)**:
  - Hosts all REST API endpoints, JWT + RBAC middleware (`PermAdmin`, `PermOwner`), cryptographic key derivation, transaction verification, ledger mutations, package provisioning, and background blockchain scanners.
  - Runs 24/7 as a `systemd` service (`hostvra-api`), independent of whether any user browser or admin tab is open.
- **Untrusted Boundary (Next.js Web Frontend — `apps/web`)**:
  - Communicates exclusively via authenticated HTTP API calls (`/api/v1/crypto/*` and `/api/v1/admin/crypto/*`).
  - Never receives master seeds, extended private keys (`xprv`), raw private keys, or direct database write permissions.
  - Cannot mark invoices paid, credit balances, or alter treasury sweep destinations.

---

## 2. Existing System Inventory (Pre-Phase 5)

| Component | File Path | Status Before Phase 5 |
|---|---|---|
| Network & Currency Registry | `apps/api/internal/store/crypto_store.go`, `0035_inhouse_crypto_payment_treasury.sql` | 8 assets in MemoryStore (missing `SOL`, `POL`, `USDT_ERC20` in MemoryStore seed). |
| Multi-Chain RPC Client | `apps/api/internal/crypto/rpc_client.go` | Supports EVM JSON-RPC (`eth_getBalance`, `eth_getLogs`, `eth_getTransactionReceipt`, block scan), TRON (`TronGrid`), BTC/LTC/DOGE. Missing Solana RPC (`getBalance`, `getSignaturesForAddress`, `getTransaction`) and raw transaction broadcast (`eth_sendRawTransaction`, etc.). |
| Invoice & Verification Service | `apps/api/internal/crypto/service.go` | Shared-wallet fallback via `AcquireDepositAddress`. Relies on shared merchant Trust Wallet address when pool is empty. |
| Package Activation / Settlement | `apps/api/internal/handlers/billing.go` (`SettleCryptoPayment`) | Atomic invoice settlement, subscription renewal/creation, hosting quota provisioning (`provisionUserForPlan`), email notification. |
| Crypto HTTP Handlers | `apps/api/internal/handlers/crypto.go` | Public checkout endpoints & Admin endpoints for invoices, treasuries, ledger, settings. |
| Background Scanner | `apps/api/internal/crypto/scanner.go` | Single periodic scanner (20s interval) for pending invoices. No sweep worker, no reconciliation worker, no RPC health worker. |
| Exchange Rate Pricing | `apps/api/internal/crypto/pricing.go` | Live USD conversion with caching. |
| Frontend Checkout UI | `apps/web/src/app/billing/crypto/[id]/page.tsx` | Real-time 3.5s polling, QR code, copy buttons, Trust Wallet deep link, TxHash fast-track & Check On-Chain. |
| Frontend Admin UI | `apps/web/src/app/admin/crypto/page.tsx` | Invoices, Treasuries (Trust Wallet addresses), Ledger, Settings tabs. Missing HD Wallet overview, Sweep Jobs queue, Security & Signer status, Emergency Pause. |

---

## 3. Critical Architectural Gaps Identified

1. **Shared Deposit Address Collision Risk**:
   - Currently, every customer paying with BNB/USDT/TRX receives the same merchant Trust Wallet address. Under concurrent payments or existing wallet balances, attribution by balance alone can cause false-positive matches or cross-user collisions.
2. **Missing Standards-Compliant HD Derivation**:
   - No BIP-39 / BIP-32 / BIP-44 / SLIP-0010 derivation engine exists.
3. **Missing Watch-Only & Isolated Signer Architecture**:
   - Need strict separation between watch-only public derivation (`xpub` / derived public keys) used for invoice address generation, and an isolated server-side signing service (`IsolatedCryptoSigner`) protected by environment-injected / AES-256-GCM encrypted vault keys with zero exposure to SQL tables or API responses.
4. **Missing Automated Treasury Sweep Engine**:
   - Once funds arrive in a user's unique HD deposit address, an independent state machine (`TreasurySweepEngine`) is required to sweep eligible balances to the admin-configured **Trust Wallet** treasury address with threshold checks, gas fee reserves, daily limits, idempotency keys, emergency pause, and retry recovery.
5. **Missing All-9 Asset Coverage**:
   - Must ensure full parity across all 9 required assets:
     1. `ETH` (Ethereum)
     2. `USDT_ERC20` (Ethereum ERC20)
     3. `BNB` (BNB Smart Chain)
     4. `USDT_BEP20` (BNB Smart Chain BEP20)
     5. `POL` (Polygon PoS)
     6. `TRX` (TRON)
     7. `USDT_TRC20` (TRON TRC20)
     8. `BTC` (Bitcoin)
     9. `SOL` (Solana)

---

## 4. Target Phase 5 Architecture

```
Customer Checkout (/billing/crypto/[id])
        │
        ▼
[CryptoService.CreateInvoice]
        │
        ▼
[HDWalletService.AllocateUserAddress] ──(Watch-Only / Atomic Index)──▶ Unique User Deposit Address
        │                                                               (BIP-44 / BIP-84 / SLIP-0010)
        ▼
[DepositMonitorService (24/7 Worker)] ──(Live Blockchain RPC)──▶ Detects & Verifies Exact Deposit
        │                                                         (Network, Asset, Contract, Amount, Confs)
        ▼
[PaymentVerification & Atomic Settlement] ──▶ Immutable Ledger + Package Activation (Exactly Once)
        │
        ▼ (Decoupled State Machine — Failure Never Affects User Package)
[TreasurySweepEngine (24/7 Worker)]
        │
        ├─ Checks: Auto-Sweep Enabled, Not Paused, Min Threshold, Daily Limit, Gas Reserve
        ├─ Calls: IsolatedSignerService (Policy-Checked Signing, Keys Never in DB/API)
        └─ Broadcasts Sweep TX ──▶ Merchant Trust Wallet Treasury Address
```
