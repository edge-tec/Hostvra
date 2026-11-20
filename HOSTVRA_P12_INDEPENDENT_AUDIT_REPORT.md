# HOSTVRA — P12 INDEPENDENT PRODUCTION AUDIT REPORT

**Audit Date & Time:** 2026-11-20T17:19:00+06:00  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Audited P11 Baseline Commit:** `2c2613687e8d48d65e4c1d00faa4997b63016fe1`  
**P10 Parent Commit:** `59bf61f700524b57797537b6e501508673870bf5`  
**Audit Verdict:** **PASS (Local Codebase, Financial, DR, Performance & Security) / NOT VERIFIED (Live Contabo VPS Cutover Pending Operator Command)**

---

## 1. Phase 12.1 — Git & Production Baseline Verification

Independently verified via Git and filesystem inspection:
- `git rev-parse HEAD`: `2c2613687e8d48d65e4c1d00faa4997b63016fe1` (Matches expected P11 baseline).
- `git rev-parse HEAD~1`: `59bf61f700524b57797537b6e501508673870bf5` (Matches expected P10 parent).
- Pre-P12 `bin/hostvra-api-linux-amd64` SHA-256: `59ffa98d127382b56d1e55c75b9be7d975feec196fd019a9c00e9d201b875a37` (`ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked`).
- Database migration parity (`TestMigrations_ParityAndDeterminism`): **PASS** (38/38 SQL files identical between `migrations/` and `apps/api/internal/store/migrations/`).

---

## 2. Independent Findings Discovered & Remediated in Phase P12

Rather than trusting prior P10/P11 reports, a fresh line-by-line audit of deployment scripts (`update.sh`, `update-server.sh`, `install.sh`) and financial store operations (`apps/api/internal/store/commercial_store.go`) identified **3 real defects**, all fixed and regression-tested in P12:

| ID | Severity | Component | Root Cause Discovered | P12 Fix Implemented | Status |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **P12-01** | **CRITICAL** | `update-server.sh` & `install.sh` | Hardcoded plaintext fallback admin password in Git; `update-server.sh` overwrote `INITIAL_ADMIN_PASSWORD` in `/etc/hostvra/api.env` *before* taking a backup of `api.env` and missed `/opt/hostvra` in `POSSIBLE_DIRS`. | Removed hardcoded password from `update-server.sh` & `install.sh`; `install.sh` now preserves existing `INITIAL_ADMIN_PASSWORD` or generates a CSPRNG password (`openssl rand -base64 18`); `update-server.sh` backs up `api.env` first, never overwrites credentials unless `INITIAL_ADMIN_PASSWORD` env var is explicitly passed, includes `/opt/hostvra`, and verifies `/health` + `/api/v1/public/status`. | **PASS** |
| **P12-02** | **HIGH** | `update.sh` | Under `set -euo pipefail` + `trap rollback ERR`, if `go build` failed on a low-RAM VPS or older system Go toolchain, `update.sh` aborted instead of falling back to the pre-compiled `bin/hostvra-api-linux-${BIN_ARCH}` binary; also lacked backup retention pruning and self-symlink guard. | Made `go build` conditional so any build error falls back cleanly to `bin/hostvra-api-linux-${BIN_ARCH}`; added 5-snapshot retention pruning in `/var/lib/hostvra/updates_backup`; guarded symlink creation with `[[ "$HOSTVRA_DIR" != "/opt/hostvra" ]]`. | **PASS** |
| **P12-03** | **MEDIUM** | `apps/api/internal/store/commercial_store.go` | IEEE-754 floating-point subtraction drift (`15.30 - 5.10 = 10.199999999999999`) in affiliate balance accounting could reject an exact remaining-balance withdrawal (`$10.20`), and sub-cent dust withdrawals (`< $0.01`) were not blocked. | Enforced 2-decimal cent rounding (`math.Round(...*100)/100`) and `$0.01` minimum across `RecordAffiliateCommission`, `ReverseAffiliateCommissionByInvoice`, `RequestAffiliateWithdrawal`, and `ApproveAffiliateWithdrawal` (MemoryStore & PostgresStore). Verified by `TestP12_FinancialInvariantsLedgerReconciliationAndPrecision`. | **PASS** |
