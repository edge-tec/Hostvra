# HOSTVRA — PHASE P18.5: BACKEND REGRESSION REPORT

**UTC Timestamp:** `2026-11-20T14:39:18Z`  
**Go Version:** `go1.27.1 darwin/arm64`  
**Exit Code:** `0`  
**Status:** **PASS (LOCAL — 0 Data Races, 0 `go vet` Warnings)**

---

## 1. Command & Actual Output
```text
$ cd apps/api && go version && go vet ./... && go test -count=1 -race ./internal/auth ./internal/rbac ./internal/billing/... ./internal/crypto/... ./internal/store ./internal/handlers -run "TestCommercial_|TestP10_|TestP12_|TestP13_|TestP14_|TestP15_|TestP16_|TestBillingWebhook_|TestPassword|TestJWT|TestRBAC|TestCrypto|TestHD|TestDunning|TestMemoryStore"
go version go1.27.1 darwin/arm64
ok  	hostvra/api/internal/auth	2.823s
ok  	hostvra/api/internal/rbac	2.607s
ok  	hostvra/api/internal/billing	1.553s
ok  	hostvra/api/internal/crypto	2.251s
ok  	hostvra/api/internal/store	1.639s
ok  	hostvra/api/internal/handlers	3.831s
```

## 2. Platform-Specific Test Disclosure
- Local macOS does not provide Linux `/home` quota accounting (`internal/quota`) or Postfix `/usr/sbin/postdrop` & `/usr/sbin/postqueue` binaries; those Linux-host tests require execution on the Linux Contabo VPS.
