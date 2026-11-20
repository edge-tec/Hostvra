# HOSTVRA — PHASE P16.14: ROLLBACK & DISASTER RECOVERY CERTIFICATION

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P16.14 — Rollback and Disaster Recovery Certification  
**Status:** **PASS**

---

## 1. Rollback Mechanism Verification (`update.sh`)

Inspected the automated `rollback()` function in `/Volumes/EDGE/Hostvra-main/update.sh`:

1. **Trigger Condition**: Automatically invoked via `trap 'rollback' ERR` if any build, installation, systemd restart, or post-deployment HTTP health check fails.
2. **Step-by-Step Restoration Sequence**:
   - Restores previous Git commit from `$BACKUP_DIR/git-commit.txt`.
   - Restores previous `/usr/local/bin/hostvra-api` and `/usr/local/bin/hostvra-agent` from `$BACKUP_DIR/*.bak`.
   - Restores previous Next.js build (`apps/web/.next`) from `$BACKUP_DIR/web-next.tar.gz`.
   - Restores `/var/lib/hostvra/store.json` from `$BACKUP_DIR/store.json.bak` only if the store file was corrupted or missing.
   - Restarts `hostvra-api`, `hostvra-web`, and `hostvra-agent`.

---

## 2. Post-Rollback & Post-Restart Financial Invariant Verification

Verified via `TestP13_RealOnDiskDatabaseBackupAndRestore`, `TestP14_*`, `TestP15_*`, and `TestP16_WebhookReplayDeduplicationPersistsAcrossColdRestartAndDomainOrdersPersist`:

| Invariant Check After Cold Restart / Restore | Mechanism | Verified |
| :--- | :--- | :---: |
| **No Duplicate Payments** | `WalletLedgerEntry.IdempotencyKey` (`wallet_pay:<invID>`) and `Invoice.Status == Paid` persist on disk | **PASS** |
| **No Duplicate Webhook Processing** | `DomainWebhooks` (`<gateway>:<eventID>`) now persists in `store.json` (`P16-01`) | **PASS** |
| **No Duplicate Subscriptions or Hosting Accounts** | `Subscriptions` and `HostingAccounts` maps persist on disk and link to `Invoice.SubscriptionID` | **PASS** |
| **No Lost Financial Ledger Entries** | `CustomerWallets`, `WalletLedger`, `CryptoLedger`, `AffiliateAccounts`, `AffiliateCommissions`, and `AffiliateWithdrawals` flush atomically on every mutation | **PASS** |
| **No Reused HD Wallet Derivation Indexes** | `HDWalletAccounts` (`NextDerivationIndex`) and `HDWalletAddresses` flush atomically inside `ReserveNextDerivationIndex` | **PASS** |
| **No Reused Crypto Pool Deposit Addresses** | `CryptoDepositAddrs` (`status = "assigned"`) flushes atomically inside `AcquireDepositAddress` (`P15-03`) | **PASS** |
| **No Lost Domain Order Payment States** | `DomainOrders` flushes atomically inside `CreateDomainOrder` and `UpdateDomainOrder` (`P16-02`) | **PASS** |
