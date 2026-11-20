# HOSTVRA — P13 FINAL GO-LIVE CERTIFICATION

**Certification Date & Time:** 2026-11-20T17:43:30+06:00  
**Repository:** `edge-tec/Hostvra` (`origin/main`)  
**Audited P12 Baseline Commit:** `981790b24be5b6a298fc621cb03554c0fc14399e`  
**Updated P13 Linux `amd64` Binary SHA-256:** `881718b1b7f00852d6d2d83756607ae2c266be30477967a728cfe18ab3f7758f`  
**Final Certification Decision:** **CONDITIONAL PASS**

---

## 1. Executive Decision & Rationale

In strict adherence to the Phase P13 rule (*"P12 is CONDITIONAL PASS. Do not upgrade its certification status without actual live evidence"*):

1. **Live Production HTTPS & API Probe Verified (`https://hostvra.com`)**:
   - `GET https://hostvra.com/api/v1/public/status` is **LIVE and returning HTTP 200 OK** (`overall_status: "operational"`, process uptime `1596s`, database latency `0ms`, `7 active blockchain networks monitored`).
   - TLS X.509 certificate on `hostvra.com:443` is **LIVE and valid** (`Let's Encrypt`, valid until `Dec 30 14:33:57 2026 GMT`).
   - Nginx HTTP/2 reverse proxy and `/admin/commercial` authentication redirect (`HTTP 307 -> /login`) are **LIVE and operational**.
2. **Two Real Production Defects Caught via Live Probe & Fixed in P13**:
   - **P13-01 (`apps/web/src/middleware.ts`)**: Live probe of `https://hostvra.com/order` revealed Next.js middleware was redirecting unauthenticated visitors to `/login` (`HTTP 307`). Fixed by adding `/order`, `/status`, `/terms`, `/privacy`, `/refund-policy`, `/sla`, `/acceptable-use-policy`, `/verify-email`, `/webmail/login`, and `/webmail/sso` to `PUBLIC_PATHS` in `apps/web/src/middleware.ts` and rebuilding the 59-route Next.js bundle.
   - **P13-02 (`apps/api/internal/store/store.go`, `commercial_store.go`, `hd_wallet_store.go`)**: Live probe confirmed `/var/lib/hostvra/store.json` file-backed storage is active on the VPS, and code inspection revealed `memoryDumpData` was not persisting commercial tables (coupons, wallets, ledger, affiliates) or HD wallet derivation indices to `store.json`. Fixed by adding all commercial & HD wallet fields to `memoryDumpData`, `saveToDiskLocked()`, `loadFromDisk()`, and all mutating store methods, and verified with a physical on-disk database backup & restore test (`TestP13_RealOnDiskDatabaseBackupAndRestore`, `FileBytes=41079`, `RestoreRTO=2.97ms`, `RPO=0`).
3. **Why Final Decision Remains `CONDITIONAL PASS`**:
   - While `https://hostvra.com/api/v1/public/status` is live and operational, the P13 fixes (`middleware.ts` public paths and `store.json` commercial disk persistence in `bin/hostvra-api-linux-amd64`) must be pulled and restarted on `root@hostvra` by the server operator (`cd /root/Hostvra && git pull origin main && bash update.sh`), and direct host-level SSH checks (`ufw status`, local `git rev-parse HEAD` inside `/root/Hostvra`) require operator terminal execution.

---

## 2. Summary Gate Scorecard

| Gate | Status | Evidence |
| :--- | :--- | :--- |
| **P13.1 — Live Deployment & TLS Verification** | **PASS (Public HTTPS) / CONDITIONAL (Pending P13 Pull)** | `https://hostvra.com/api/v1/public/status` -> `200 OK`; Let's Encrypt TLS valid; P13 `middleware.ts` & `store.json` fix ready in `origin/main`. |
| **P13.2 — Deployment Safety & Rollback** | **PASS** | No hardcoded passwords; pre-deploy backup before git/build changes; `.next` tarball rollback; binary fallback; 5-backup retention. |
| **P13.3 — Financial Integrity & Precision** | **PASS** | 14/14 handler test groups (19 subtests) PASS with `-race` (`2.796s`); IEEE-754 cent rounding & 35-entry double-entry ledger reconciliation verified. |
| **P13.4 — Security Verification** | **PASS** | Live TLS, RBAC, IDOR, SQLi, HMAC webhooks, public vs protected middleware routes verified. |
| **P13.5 — Real On-Disk Database Recovery** | **PASS** | `TestP13_RealOnDiskDatabaseBackupAndRestore`: physical `store.json` (`41,079 bytes`, SHA256 `2e80e2625af40069`), `BackupTime=195.92µs`, `RestoreRTO=2.97ms`, `RPO=0`. |
| **P13.6 — End-to-End Production & Build Tests** | **PASS** | `go vet ./...` PASS, `go test -v -race` PASS, Next.js `npm run build` (59/59 routes) PASS, Linux `amd64` static binary built (`881718b1...`). |

---

## 3. Final Operator Cutover & Verification Command (`root@hostvra:~#`)

Run this command on your Contabo VPS terminal (`root@hostvra:~#`) to apply the P13 fixes live and confirm `HTTP 200` on `/order` and `/api/v1/public/status`:

```bash
cd /root/Hostvra && git pull origin main && bash update.sh && \
curl -fsS -o /dev/null -w "LIVE_ORDER_HTTP:%{http_code}\n" https://hostvra.com/order && \
curl -fsS https://hostvra.com/api/v1/public/status && echo ""
```
