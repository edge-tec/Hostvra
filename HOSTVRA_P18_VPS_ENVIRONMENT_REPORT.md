# HOSTVRA — PHASE P18.2: VPS ENVIRONMENT REPORT

**UTC Timestamp:** `2026-11-20T14:39:23Z`  
**Status:** **PARTIAL (Live HTTPS Telemetry Verified; OS/Systemd Inspection Blocked on SSH)**

---

## 1. Live Production Environment Telemetry (`https://hostvra.com/api/v1/public/status`)
- **HTTP Reverse Proxy:** `nginx` serving `HTTP/2` with `TLSv1.3 / AEAD-CHACHA20-POLY1305-SHA256`
- **TLS Certificate:** `CN=hostvra.com`, issued by `Let's Encrypt (CN=YE1)`, valid `Oct 1 2026` to `Dec 30 2026 GMT`
- **API Process Runtime:** `Go go1.26.0`, `service: hostvra-api`, `version: 1.0.0`
- **Live Uptime:** `12,575 seconds` (`3h 29m 35s`)
- **Memory Footprint:** `9 MB` RSS | **Goroutines:** `14`
- **OS-Level Commands (`uname -a`, `df -h`, `free -h`, `ss -tulpn`, `systemctl --failed`):** **BLOCKED** pending operator SSH session on `root@hostvra.com`.
