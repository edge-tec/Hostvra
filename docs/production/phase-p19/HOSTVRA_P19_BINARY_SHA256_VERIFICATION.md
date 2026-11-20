# HOSTVRA — PHASE P19.4: BINARY SHA-256 VERIFICATION

**Date / UTC Timestamp:** `2026-11-20T14:46:23Z`  
**Environment:** Local Release Artifact (`bin/hostvra-api-linux-amd64`)  
**Exit Code:** `0` | **Status:** **PASS (RELEASE ARTIFACT) / BLOCKED (LIVE VPS MATCH)**

## Evidence
```text
c2c6b01c492c4937de22199540b78672c3510fc679c43497b22f80ab06f490b4  bin/hostvra-api-linux-amd64
bin/hostvra-api-linux-amd64: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked
```
- **Remediation:** Compare `shasum -a 256 /usr/local/bin/hostvra-api` on VPS after operator runs `bash update.sh`.
