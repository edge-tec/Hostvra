# HOSTVRA — PHASE P15: REMAINING RISKS & OPERATOR ACTIONS

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Current Certification Status:** **CONDITIONAL PASS**

---

## 1. Mandatory Requirement Blocking `UNCONDITIONAL PASS`

Per the Phase P15 Final Certification Requirements:
> *"`<UNCONDITIONAL PASS>` is permitted only when: Production deployment is independently verified... All required public routes work... Production binary matches the certified build. If any mandatory requirement cannot be verified, retain CONDITIONAL PASS or FAIL and explain exactly why."*

### Why `CONDITIONAL PASS` is Retained
1. **SSH Authentication Barrier from Local Audit Sandbox**:
   - Running `/usr/bin/ssh -o BatchMode=yes root@hostvra.com` from this local environment returned `root@hostvra.com: Permission denied (publickey,password)` because the operator's root SSH key/password is not stored on this local machine.
2. **Live Contabo VPS Has Not Yet Pulled P13/P14/P15 Commits**:
   - Live HTTP probing of `https://hostvra.com/api/v1/public/status` at `2026-11-20T12:27:12Z` shows `uptime_seconds: 4644` (`1h 17m`), and live probing of `/order`, `/status`, `/terms`, `/privacy`, `/refund-policy`, `/sla`, and `/acceptable-use-policy` still returns `HTTP 307 -> /login`.
   - This proves the Contabo VPS is still running the pre-P13 build started 77 minutes ago.

---

## 2. Single Operator Action to Achieve `UNCONDITIONAL PASS`

Log into the Contabo VPS (`root@hostvra`) and run:

```bash
cd /root/Hostvra && git fetch origin && git pull --ff-only origin main && bash update.sh
```

Once `bash update.sh` completes (approx. 45–60 seconds):
1. `/usr/local/bin/hostvra-api` will match SHA-256 `ad3a72982b21580a39ec11aeb45e3e58b03ee61bb548c38722f263c85fc7ce98`.
2. Next.js (`hostvra-web`) will run the updated `middleware.ts` and 100% responsive UI build, so `/order`, `/status`, `/terms`, `/privacy`, `/refund-policy`, `/sla`, and `/acceptable-use-policy` return `HTTP 200 OK` without redirecting to `/login`.
3. `/var/lib/hostvra/store.json` will persist all 22 entity maps across core, commercial, crypto, and HD wallet subsystems with `RPO=0`.
