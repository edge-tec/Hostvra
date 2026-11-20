# HOSTVRA — PHASE P19.4: BACKEND REGRESSION REPORT

**Date / UTC Timestamp:** `2026-11-20T14:48:01Z`  
**Environment:** Local macOS (`go1.27.1 darwin/arm64`)  
**Exit Code:** `0` | **Status:** **PASS (LOCAL)**

## Evidence
```text
$ cd apps/api && go vet ./... && go test -count=1 -race ./internal/auth ./internal/rbac ./internal/billing/... ./internal/crypto/... ./internal/store ./internal/handlers -run "TestCommercial_|TestP10_|TestP12_|TestP13_|TestP14_|TestP15_|TestP16_|TestBillingWebhook_|TestPassword|TestJWT|TestRBAC|TestCrypto|TestHD|TestDunning|TestMemoryStore"
ok  	hostvra/api/internal/auth	2.342s
ok  	hostvra/api/internal/rbac	2.382s
ok  	hostvra/api/internal/billing	1.489s
ok  	hostvra/api/internal/crypto	2.024s
ok  	hostvra/api/internal/store	1.499s
ok  	hostvra/api/internal/handlers	3.349s
```
- **Platform Disclosure:** Linux-only `/home` quota (`internal/quota`) and Postfix queue (`/usr/sbin/postdrop`, `/usr/sbin/postqueue`) tests require execution on Linux.
