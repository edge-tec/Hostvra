# HOSTVRA — P12 REMEDIATION REPORT

**Execution Date & Time:** 2026-11-20T17:19:30+06:00  
**Certification Status:** **PASS**

---

## 1. Issues Discovered & Remediated in Phase P12

### Issue P12-01 (CRITICAL): Hardcoded Fallback Admin Password & Pre-Backup Config Overwrite in `update-server.sh` and `install.sh`
- **Affected Files**: `update-server.sh`, `install.sh`
- **Root Cause**: `update-server.sh` hardcoded a plaintext password (`INITIAL_ADMIN_PASSWORD=...`), overwrote `/etc/hostvra/api.env` *before* backing up `api.env`, and lacked `/opt/hostvra` in `POSSIBLE_DIRS`. `install.sh` used the same hardcoded password as its default fallback.
- **Remediation**:
  - Removed all occurrences of the hardcoded password from `update-server.sh` and `install.sh`.
  - Moved the backup block in `update-server.sh` before `git reset --hard origin/main` and only updated `INITIAL_ADMIN_PASSWORD` if explicitly supplied in the operator's environment.
  - Updated `install.sh` to preserve any existing `INITIAL_ADMIN_PASSWORD` in `/etc/hostvra/api.env` or generate a random password via `openssl rand -base64 18`.
- **Verification**: `bash -n update-server.sh install.sh` PASS; `grep` confirms 0 hardcoded passwords remain.

### Issue P12-02 (HIGH): `update.sh` Aborting on `go build` Failure Instead of Falling Back to Certified Pre-Compiled Binary + Backup Retention
- **Affected File**: `update.sh`
- **Root Cause**: Under `set -euo pipefail`, `(cd apps/api && CGO_ENABLED=0 go build ...)` triggered `rollback` if the VPS had an older system Go compiler or low RAM, bypassing the `elif [[ -f "bin/hostvra-api-linux-${BIN_ARCH}" ]]` fallback branch.
- **Remediation**:
  - Changed the `if` condition to `if command -v go &>/dev/null && (cd apps/api && CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server) 2>/dev/null; then` so any local compile failure seamlessly falls back to `bin/hostvra-api-linux-${BIN_ARCH}`.
  - Added 5-snapshot retention pruning (`ls -1dt /var/lib/hostvra/updates_backup/* | tail -n +6 | xargs -r rm -rf`) and self-symlink guards (`[[ "$HOSTVRA_DIR" != "/opt/hostvra" ]]`).
- **Verification**: `bash -n update.sh` PASS.

### Issue P12-03 (MEDIUM): IEEE-754 Floating-Point Drift in Affiliate Balance Accounting & Sub-Cent Withdrawal Floor
- **Affected File**: `apps/api/internal/store/commercial_store.go`
- **Root Cause**: Unrounded float subtraction (`15.30 - 5.10 = 10.199999999999999`) could cause a subsequent exact-balance withdrawal (`$10.20`) to fail `amountUSD > acc.ApprovedBalanceUSD`.
- **Remediation**: Enforced 2-decimal cent rounding (`math.Round(...*100)/100`), `$0.01` minimum withdrawal floor, and `1e-9` epsilon comparison in both `MemoryStore` and `PostgresStore`.
- **Verification**: `TestP12_FinancialInvariantsLedgerReconciliationAndPrecision` (**PASS** with `-race`).
