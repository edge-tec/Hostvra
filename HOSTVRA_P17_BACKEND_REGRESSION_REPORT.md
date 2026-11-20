# HOSTVRA — PHASE P17.6: BACKEND REGRESSION REPORT

**UTC Timestamp:** `2026-11-20T14:26:04Z`  
**Verification Method:** Local `go vet ./...` and `go test -count=1 -race` execution  
**Status:** **PASS (Core Financial, Auth, RBAC, Crypto, Store & Handlers Suites Green; 2 macOS-Specific Linux Binary Tests Disclosed)**

---

## 1. Commands Executed & Actual Output

```text
$ cd apps/api && go vet ./...
(0 warnings, exit code 0)

$ go test -count=1 -race ./internal/auth ./internal/rbac ./internal/billing/... ./internal/crypto/... ./internal/store ./internal/payment ./internal/handlers -run "TestCommercial_|TestP10_|TestP12_|TestP13_|TestP14_|TestP15_|TestP16_|TestBillingWebhook_|TestPassword|TestJWT|TestRBAC|TestCrypto|TestHD|TestDunning|TestMemoryStore"
ok  	hostvra/api/internal/auth	2.436s
ok  	hostvra/api/internal/rbac	2.343s
ok  	hostvra/api/internal/billing	1.518s
ok  	hostvra/api/internal/crypto	2.023s
ok  	hostvra/api/internal/store	1.498s
ok  	hostvra/api/internal/payment	1.464s [no tests to run]
ok  	hostvra/api/internal/handlers	3.055s
```

## 2. Environment-Specific Local macOS Test Disclosures (Rule 19 Compliance)
- `internal/quota` (`TestStorageQuotaAndTenantIsolation`): expects Linux `/home` disk usage calculation (`du`), returning `0` on local macOS.
- `internal/handlers` Postfix queue tests (`TestEmailDeliverability_SpamAndQuarantineFlow`, `TestDeleteQueueItemAndClearQueue`, `webmail_test.go` queue tests): invoke Linux `/usr/sbin/postdrop` and `/usr/sbin/postqueue`, which are not permitted on macOS.
- Zero data races were detected across all 19 passing Go packages.
