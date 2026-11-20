# HOSTVRA — P20.6 & P20.7: GO BACKEND & AGENT TESTS

**Date / UTC Timestamp:** `2026-11-20T15:00:58Z`  
**Environment:** Local macOS (`go1.27.1 darwin/arm64`)  
**Status:** **PASS (CORE & AGENT BUILD) / PARTIAL (LINUX-ONLY HOST TESTS REQUIRE LINUX)**

## Commands & Actual Output
```text
$ cd apps/agent && go vet ./... && go build ./...
AGENT_BUILD_OK

$ cd apps/api && go vet ./... && go test -race -count=1 ./internal/auth ./internal/rbac ./internal/billing/... ./internal/crypto/... ./internal/store ./internal/handlers -run "TestCommercial_|TestP10_|TestP12_|TestP13_|TestP14_|TestP15_|TestP16_|TestBillingWebhook_|TestPassword|TestJWT|TestRBAC|TestCrypto|TestHD|TestDunning|TestMemoryStore"
ok  	hostvra/api/internal/auth	2.416s
ok  	hostvra/api/internal/rbac	2.367s
ok  	hostvra/api/internal/billing	1.475s
ok  	hostvra/api/internal/crypto	2.132s
ok  	hostvra/api/internal/store	1.588s
ok  	hostvra/api/internal/handlers	3.285s
```

## Linux Static Binaries Verified
- `bin/hostvra-api-linux-amd64`: `ELF 64-bit LSB executable, x86-64, statically linked` (`c2c6b01c492c4937de22199540b78672c3510fc679c43497b22f80ab06f490b4`)
- `bin/hostvra-agent-linux-amd64`: `ELF 64-bit LSB executable, x86-64, statically linked, stripped` (`dd7f92743548a3461142a87da2e7fb5a073986d64e11c679d39f927aebda2c17`)
- **Known Limitations:** Linux `/home` quota (`internal/quota`) and Postfix queue (`/usr/sbin/postdrop`, `/usr/sbin/postqueue`) tests require execution on Linux.
