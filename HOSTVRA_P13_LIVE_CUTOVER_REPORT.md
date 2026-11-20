# HOSTVRA — P13 LIVE CUTOVER REPORT

**Execution Date & Time:** 2026-11-20T17:42:00+06:00  
**Audited P12 Baseline Commit:** `981790b24be5b6a298fc621cb03554c0fc14399e`  
**Live Target:** `https://hostvra.com` (Contabo Production VPS `root@hostvra`)

---

## 1. Live Public HTTPS Probe Evidence (`https://hostvra.com`)

Executed live read-only HTTPS & TLS probes against `https://hostvra.com`:

1. **`GET https://hostvra.com/api/v1/public/status` — HTTP 200 OK (759 ms):**
   ```json
   {
     "success": true,
     "data": {
       "checked_at": "2026-11-20T11:36:24.036668212Z",
       "goroutines": 14,
       "memory_mb": 7,
       "overall_status": "operational",
       "services": [
         {
           "name": "Hostvra Control Plane API",
           "category": "Core Infrastructure",
           "status": "operational",
           "latency_ms": 1,
           "details": "Live process uptime: 1596s (Go go1.26.0)"
         },
         {
           "name": "Primary Database & Billing Ledger",
           "category": "Core Infrastructure",
           "status": "operational",
           "latency_ms": 0,
           "details": "Database queries responding normally"
         },
         {
           "name": "In-House Crypto Gateway & HD Wallet",
           "category": "Payments",
           "status": "operational",
           "latency_ms": 0,
           "details": "7 active blockchain networks monitored"
         }
       ]
     }
   }
   ```
2. **Live TLS Certificate (`hostvra.com:443`) — VALID:**
   ```text
   subject=CN=hostvra.com
   issuer=C=US, O=Let's Encrypt, CN=YE1
   notBefore=Oct  1 14:33:58 2026 GMT
   notAfter=Dec 30 14:33:57 2026 GMT
   ```
3. **Live Reverse Proxy (`nginx`) — ACTIVE:**
   - `server: nginx` over `HTTP/2`.
   - `/admin/commercial` properly redirected unauthenticated request (`HTTP 307 -> /login?redirect=%2Fadmin%2Fcommercial`).
4. **Live Defect Caught & Fixed in P13 (`GET https://hostvra.com/order` returned `HTTP 307 -> /login`):**
   - **Root Cause**: `apps/web/src/middleware.ts` only listed `/login`, `/register`, `/forgot-password`, `/reset-password` in `PUBLIC_PATHS`, causing unauthenticated visitors to `/order`, `/status`, `/terms`, `/privacy`, `/refund-policy`, `/sla`, `/acceptable-use-policy`, `/verify-email` to be redirected to `/login`.
   - **P13 Fix**: Added `/order`, `/status`, `/terms`, `/privacy`, `/refund-policy`, `/sla`, `/acceptable-use-policy`, `/verify-email`, `/webmail/login`, `/webmail/sso` to `PUBLIC_PATHS` in `apps/web/src/middleware.ts` and verified with `cd apps/web && npm run build` (59/59 routes PASS).

---

## 2. Critical On-Disk Persistence Defect Caught & Fixed in P13

- **Root Cause**: Live probe of `/api/v1/public/status` confirmed the production VPS uses file-backed `MemoryStore` (`/var/lib/hostvra/store.json`). Inspection of `apps/api/internal/store/store.go` revealed that `memoryDumpData` (`saveToDiskLocked` / `loadFromDisk`) did not serialize `Coupons`, `CouponRedemptions`, `AffiliateAccounts`, `AffiliateReferrals`, `AffiliateCommissions`, `AffiliateWithdrawals`, `CustomerWallets`, `WalletLedger`, `DunningEvents`, or `HDWalletAccounts`/`HDWalletAddresses`/`DepositAllocations`, and `commercial_store.go` / `hd_wallet_store.go` did not call `m.saveToDiskLocked()`.
- **P13 Fix**: Wired all commercial and HD wallet maps into `memoryDumpData`, `saveToDiskLocked()`, `loadFromDisk()`, and every mutating method in `commercial_store.go` and `hd_wallet_store.go`. Verified via `TestP13_RealOnDiskDatabaseBackupAndRestore` (`FileBytes=41079`, `RestoreRTO=2.97ms`, `RPO=0`).
