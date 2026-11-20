# HOSTVRA — PHASE P17.14: FULL REGRESSION REPORT

**UTC Timestamp:** `2026-11-20T14:26:04Z`  
**Verification Method:** Non-cached Go race test suite (`go test -count=1 -race`) + Next.js production build (`npm run build`)  
**Status:** **PASS (LOCAL)**

---

## 1. Regression Summary
- **Go Static Analysis (`go vet ./...`):** **PASS** (0 warnings)
- **Go Core & Commercial Race Suites (`-count=1 -race`):** **PASS** across `internal/auth`, `internal/rbac`, `internal/billing`, `internal/crypto`, `internal/store`, `internal/handlers` (`TestCommercial_*`, `TestP10_*`, `TestP12_*`, `TestP13_*`, `TestP14_*`, `TestP15_*`, `TestP16_*`, `TestBillingWebhook_*`), `internal/domains/resellerclub`, `internal/migration`, `internal/agenttask`, `internal/alerts`, `internal/appstore`, `internal/dns`, `internal/domains`, `internal/domains/namecheap`, `internal/email`, `internal/iputil`, `internal/license`, `internal/payment`, `internal/update`.
- **Next.js Frontend Production Build:** **PASS** (`59/59` routes compiled cleanly in `2.9s`).
- **Disclosed Local macOS Exceptions:** `internal/quota` (`TestStorageQuotaAndTenantIsolation`) and Postfix queue tests (`postdrop`/`postqueue`) require Linux host binaries/paths.
