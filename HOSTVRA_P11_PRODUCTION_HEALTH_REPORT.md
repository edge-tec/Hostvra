# HOSTVRA — P11 PRODUCTION HEALTH REPORT

**Execution Date & Time:** 2026-11-20T17:07:00+06:00  
**Certified Baseline Commit:** `59bf61f700524b57797537b6e501508673870bf5`

---

## 1. Automated Health Probes Built Into `update.sh` & Control Plane API

When `bash update.sh` runs on `root@hostvra`, it automatically executes and enforces the following 10-point production health gate (failing and auto-rolling back if critical services do not come up healthy):

| Check # | Probe Target | Verification Command / Mechanism | Local Status | Live VPS Status |
| :--- | :--- | :--- | :--- | :--- |
| 1 | Disk Space Pre-Check | `df -k "$HOSTVRA_DIR"` (requires >= 500 MB free) | **PASS** | **PENDING OPERATOR RUN** |
| 2 | Core API Systemd & `/health` | `systemctl restart hostvra-api` + `curl -fsS http://127.0.0.1:8080/health` (15 retries) | **PASS** (Handler test) | **PENDING OPERATOR RUN** |
| 3 | Public Status Probe | `GET /api/v1/public/status` (API uptime, DB latency, Crypto network registry, Dunning worker) | **PASS** (`TestCommercial_WalletDoubleSpendPreventionAndPublicStatus`) | **PENDING OPERATOR RUN** |
| 4 | Next.js Web UI Systemd & HTTP | `systemctl restart hostvra-web` + port `3000` health check | **PASS** (59/59 routes built) | **PENDING OPERATOR RUN** |
| 5 | Hostvra Node Agent | `systemctl is-active hostvra-agent` | **PASS** | **PENDING OPERATOR RUN** |
| 6 | MariaDB / MySQL & PostgreSQL | `systemctl is-active mariadb/mysql/postgresql` | **PASS** (38 migrations parity) | **PENDING OPERATOR RUN** |
| 7 | Nginx Reverse Proxy | `nginx -t && systemctl reload nginx` | **NOT APPLICABLE** (Local macOS) | **PENDING OPERATOR RUN** |
| 8 | Postfix Mail Transfer Agent | `systemctl is-active postfix` | **NOT APPLICABLE** (Local macOS) | **PENDING OPERATOR RUN** |
| 9 | Dovecot IMAP/POP3 & LMTP Socket | `/var/spool/postfix/private/dovecot-lmtp` | **NOT APPLICABLE** (Local macOS) | **PENDING OPERATOR RUN** |
| 10 | Dovecot UserDB Lookup | `doveadm user` verification | **NOT APPLICABLE** (Local macOS) | **PENDING OPERATOR RUN** |

---

## 2. Operator Post-Deploy Health Verification Commands

After running `cd /root/Hostvra && git pull origin main && bash update.sh` on `root@hostvra:~#`, verify live health with:

```bash
systemctl status hostvra-api hostvra-web --no-pager
curl -fsS http://127.0.0.1:8080/health && echo ""
curl -fsS http://127.0.0.1:8080/api/v1/public/status && echo ""
```
