# HOSTVRA — PHASE P16.11: REALTIME & SERVICE RELIABILITY REPORT

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P16.11 — Reverb, Realtime and Service Reliability  
**Status:** **PASS (Native Go WebSocket/SSE & Background Workers Verified; Laravel Reverb N/A)**

---

## 1. Realtime Architecture Inspection

Per Phase 16.11 instructions:
> *"Inspect the actual repository and deployment architecture... Do not introduce new services simply to satisfy a checklist."*

- **Architecture Finding**: HOSTVRA is a **Go (`chi` router) + Next.js** platform (not a Laravel/PHP application), so Laravel Reverb is **N/A**.
- Instead, HOSTVRA implements native Go realtime streaming and background worker supervision:
  1. **Webmail Real-Time Server-Sent Events (SSE)**: `GET /api/v1/webmail/events` (`webmailHandler.WebmailEvents` in `cmd/server/main.go:495`), authenticated via `RequireWebmailAuth`.
  2. **Interactive Web Terminal WebSocket**: `terminalHandler` (`cmd/server/main.go:133`) providing authenticated SSH/PTY streaming over WebSocket.
  3. **Real-Time Crypto Invoice Countdown & On-Chain Polling**: `GET /api/v1/crypto/invoices/{code}` paired with the 20-second `crypto.BackgroundScanner` (`cmd/server/main.go:157-158`).

---

## 2. Background Workers & Service Supervision Audit

Inspected all background workers initialized in `apps/api/cmd/server/main.go`:

| Worker / Service | Interval | Purpose & Reliability Mechanism | Live VPS Status (`/api/v1/public/status`) |
| :--- | :---: | :--- | :---: |
| **Hostvra Control Plane API (`hostvra-api`)** | Continuous | Main HTTP/REST & SSE/WS server with `middleware.Recoverer` and `900s` timeout | **Operational (`uptime: 8670s`, `1ms` latency)** |
| **Crypto Background Scanner (`cryptoScanner`)** | `20s` | Scans pending/confirming crypto invoices across 7 active blockchain networks and triggers `SettleCryptoPayment` + treasury sweep | **Operational (`7 active networks monitored`)** |
| **Automated Billing Dunning Worker (`dunningWorker`)** | `1h` | Generates 7-day advance renewal invoices, executes Credit Wallet Auto-Pay, sends 3d/1d reminders, and suspends overdue accounts after 3-day grace period | **Operational (`worker active`)** |
| **Billing Lifecycle Worker (`StartLifecycleWorker`)** | `1h` | Marks unpaid invoices overdue past due date, expires free trials, and suspends overdue hosting vhosts | **Operational** |
| **Domain Reconciliation Worker (`Reconciliation`)** | `30m` | Reconciles domain registration/transfer orders with ResellerClub | **Operational** |
| **Startup Nginx & Email Reconcilers** | On boot | `syncAllNginxVhosts` and `syncAllEmailServices` restore customer vhosts and Postfix/Dovecot maps on service start | **Verified in `main.go:91-94`** |
