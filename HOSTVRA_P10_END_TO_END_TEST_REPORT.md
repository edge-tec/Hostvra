# HOSTVRA — P10 END-TO-END TEST REPORT

**Test Date:** 2026-11-20  
**Environment:** Local macOS ARM64 workspace (`/Volumes/EDGE/Hostvra-main`), Go 1.27.1 (`-race`), Node.js / Next.js 15.5.25

---

## 1. Automated Test Execution Matrix

| Test Suite / Command | Result | Evidence / Notes |
| :--- | :--- | :--- |
| `go vet ./...` | **PASS** | 0 errors or warnings across all packages. |
| `go test -v -race ./internal/handlers -run "TestCommercial_\|TestP10_\|TestBillingWebhook_"` | **PASS** | 10/10 test groups passed in 2.947s with race detector enabled (concurrent coupon limit, self-referral & reversal, wallet double-spend, coupon update/toggle, E2E referral+checkout+commission, P10 rejected-withdrawal exploit block & concurrent approval, P10 webhook refund commission reversal, P10 coupon validation & wallet precheck, P10 wallet/withdrawal input validation, webhook security & replay). |
| `go test -race ./internal/store` | **PASS** | Includes `TestMigrations_ParityAndDeterminism` (38/38 migrations parity verified). |
| `go test -race ./internal/billing/...` | **PASS** | Automated Dunning Worker (renewal generation, wallet Auto-Pay, overdue suspension). |
| `go test -race ./internal/crypto/...` | **PASS** | Multi-chain BIP-44/84/SLIP-0010 derivation, isolated signer, TRON TRC20 ABI encoding, sweep engine. |
| `go test -race ./internal/auth/... ./internal/rbac/... ./internal/payment/... ./internal/domains/... ./internal/quota/... ./internal/update/... ./internal/email/...` | **PASS** | All core platform packages pass with `-race`. |
| `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../../bin/hostvra-api-linux-amd64 ./cmd/server` | **PASS** | Clean static Linux production binary built. |
| `cd apps/web && npm run build` | **PASS** | Next.js 15.5.25 production build passed; 59/59 routes compiled with 0 TypeScript/build errors. |
| Live Browser E2E (Playwright/Cypress against live VPS) | **NOT VERIFIED** | No live staging/production server running in this local workspace. |
| Linux-only Postfix CLI test (`TestEmailHandler_AdvancedEndpoints`) | **NOT APPLICABLE** | Requires Linux `postqueue`/`postsuper` binaries absent on macOS dev host; runs on production Linux VPS. |
