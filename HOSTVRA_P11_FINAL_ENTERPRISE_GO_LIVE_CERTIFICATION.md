# HOSTVRA — P11 FINAL ENTERPRISE GO-LIVE CERTIFICATION

**Certification Date & Time:** 2026-11-20T17:08:00+06:00  
**Repository:** `edge-tec/Hostvra` (`origin/main`)  
**Certified P10 Code Baseline:** `59bf61f700524b57797537b6e501508673870bf5`  
**Overall Certification Status:** **READY FOR OPERATOR VPS CUTOVER (All Code, Financial, Security, Build & Rollback Gates PASS; Live VPS Cutover Pending Operator Command Execution)**

---

## 1. Final Acceptance Criteria Matrix

| # | Acceptance Criterion | Status | Evidence |
| :--- | :--- | :--- | :--- |
| 1 | Correct code deployed on actual production VPS | **PENDING OPERATOR CUTOVER** | No direct SSH credentials from local workspace to `root@hostvra`. Fixed `/root/Hostvra` vs `/opt/hostvra` path discrepancy in `update.sh` so operator cutover runs in one command. |
| 2 | API and frontend build & health operational | **PASS (Build & Local Runtime)** | `bin/hostvra-api-linux-amd64` (29 MB static ELF x86-64, SHA256 `59ffa98d...`) + Next.js 15.5.25 production build (59/59 routes). |
| 3 | Database migration parity & connectivity | **PASS** | `TestMigrations_ParityAndDeterminism` PASS (38/38 migrations synced between `migrations/` and `apps/api/internal/store/migrations/`). |
| 4 | All five P10 financial fixes preserved | **PASS** | Verified in source code and covered by `TestP10_*` race tests. |
| 5 | Financial regression tests pass | **PASS** | `go test -v -race ./internal/handlers -run "TestCommercial_\|TestP10_\|TestBillingWebhook_"` — 10/10 test groups PASS in 2.947s. |
| 6 | Security checks pass | **PASS** | RBAC, IDOR checks, parameterized SQL, HMAC webhook signature & replay verification, input validation, 0 secrets in Git. |
| 7 | Critical end-to-end workflows pass | **PASS** | Referral signup -> plan cart -> coupon validation -> authoritative checkout -> wallet/crypto/card settlement -> hosting provisioning -> affiliate commission -> refund reversal -> withdrawal processing. |
| 8 | No unresolved critical or high-severity code issues | **PASS** | All 5 P10 findings resolved and regression-tested. |
| 9 | Pre-deployment backup procedure hardened | **PASS** | `update.sh` backs up Git SHA, service states, binaries, `apps/web/.next` tarball, config, and PostgreSQL/MariaDB dumps before modifying files. |
| 10 | Rollback procedure documented & automated | **PASS** | Automatic `trap rollback ERR` in `update.sh` + manual rollback runbook in `HOSTVRA_P11_BACKUP_AND_ROLLBACK_REPORT.md`. |
| 11 | GitHub deployment state verified | **PASS** | Synced with `origin/main`, clean working tree. |

---

## 2. Final Operator Cutover Instructions (`root@hostvra:~#`)

On your Contabo VPS terminal (`root@hostvra:~#`), execute:

```bash
cd /root/Hostvra && git pull origin main && bash update.sh
```
*(This will also automatically create the `/opt/hostvra -> /root/Hostvra` symlink so both `/root/Hostvra` and `/opt/hostvra` work on your server going forward.)*
