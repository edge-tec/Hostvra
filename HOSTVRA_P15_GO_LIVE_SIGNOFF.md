# HOSTVRA — PHASE P15: PRODUCTION GO-LIVE SIGNOFF & RUNBOOK

**Audit Date:** 2026-11-20  
**Project:** HOSTVRA Enterprise Hosting & Crypto Billing Platform  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Linux AMD64 Binary (`bin/hostvra-api-linux-amd64`) SHA-256:**  
`ad3a72982b21580a39ec11aeb45e3e58b03ee61bb548c38722f263c85fc7ce98`

---

## 1. Engineering & Security Signoff Matrix

| Signoff Discipline | Status | Evidence Reference |
| :--- | :---: | :--- |
| **Go Backend & Concurrency Engineering** | **SIGNED OFF** | `HOSTVRA_P15_BACKEND_API_AUDIT.md` |
| **Financial Systems & Double-Entry Ledger** | **SIGNED OFF** | `HOSTVRA_P15_FINANCIAL_INTEGRITY_CERTIFICATION.md` |
| **Blockchain, Crypto & HD Treasury Security** | **SIGNED OFF** | `HOSTVRA_P15_CRYPTO_HD_WALLET_SECURITY_AUDIT.md` |
| **Database Reliability & Disaster Recovery (`RPO=0`)** | **SIGNED OFF** | `HOSTVRA_P15_DATABASE_DR_CERTIFICATION.md` |
| **Application Security & Penetration Audit** | **SIGNED OFF** | `HOSTVRA_P15_ENTERPRISE_SECURITY_AUDIT.md` |
| **Next.js Responsive UI & Commercial Frontend** | **SIGNED OFF** | `HOSTVRA_P15_FRONTEND_QA_REPORT.md` |
| **Live Production VPS Cutover (`https://hostvra.com`)** | **CONDITIONAL ON OPERATOR SSH COMMAND** | `HOSTVRA_P15_LIVE_VPS_DEPLOYMENT_REPORT.md` |

---

## 2. Final Operator Go-Live Cutover Runbook (`root@hostvra`)

Execute the following 2 steps in your Contabo VPS root terminal (`root@hostvra`):

### Step 1 — Pull & Deploy P15 Certified Release
```bash
cd /root/Hostvra && git fetch origin && git pull --ff-only origin main && bash update.sh
```

### Step 2 — Verify Live Routes & Binary Checksum
```bash
shasum -a 256 /usr/local/bin/hostvra-api
for r in / /login /register /order /status /terms /privacy /refund-policy /sla /acceptable-use-policy /admin/commercial /api/v1/public/status; do
  printf "%-28s -> " "$r"
  curl -s -o /dev/null -w "HTTP %{http_code}\n" "https://hostvra.com$r"
done
```
Expected post-cutover output:
- `/usr/local/bin/hostvra-api` SHA-256: `ad3a72982b21580a39ec11aeb45e3e58b03ee61bb548c38722f263c85fc7ce98`
- `/`, `/login`, `/register`, `/order`, `/status`, `/terms`, `/privacy`, `/refund-policy`, `/sla`, `/acceptable-use-policy`, `/api/v1/public/status` -> `HTTP 200`
- `/admin/commercial` -> `HTTP 307` (protected admin redirect to `/login`)
