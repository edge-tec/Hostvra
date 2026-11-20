# HOSTVRA — P12 END-TO-END TEST REPORT

**Execution Date & Time:** 2026-11-20T17:19:30+06:00  
**Certification Status:** **PASS**

---

## 1. Automated Verification Summary

| Test Suite / Verification Gate | Total Tests / Routes | Passed | Failed | Status |
| :--- | :--- | :--- | :--- | :--- |
| `bash -n update.sh update-server.sh install.sh` | 3 scripts | 3 | 0 | **PASS** |
| `go vet ./...` | All Go packages | All | 0 | **PASS** |
| Commercial, Webhook, P10 & P12 Handler Race Suite (`-race`) | 13 test groups (18 subtests) | 18 | 0 | **PASS** |
| Migration Parity (`TestMigrations_ParityAndDeterminism`) | 38 SQL migration pairs | 38 | 0 | **PASS** |
| Core Backend Packages (`store`, `billing`, `crypto`, `auth`, `rbac`, `payment`, `domains`, `update`, `email`) | 9 package suites | 9 | 0 | **PASS** |
| Next.js Production Build (`cd apps/web && npm run build`) | 59 application routes | 59 | 0 | **PASS** |
| Static Linux `amd64` API Binary (`bin/hostvra-api-linux-amd64`) | 1 ELF binary (`218109ac...`) | 1 | 0 | **PASS** |
| Live Contabo VPS Post-Cutover Smoke Test | Pending operator SSH run | — | — | **NOT VERIFIED** |
