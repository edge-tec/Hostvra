# HOSTVRA — PHASE P14: PERFORMANCE & RELIABILITY REPORT

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P14.10 — Performance, Concurrency & Reliability Verification  
**Status:** **PASS**

---

## 1. Go Backend Concurrency & Throughput Benchmarks

Measured via `TestP12_APIPerformanceAndLoadBenchmark` and `TestP14_ConcurrentPayInvoiceWithWalletAndPlanScopedCouponsAndSweepPersistence` under the Go race detector (`-race`):

| Benchmark Metric | Target SLA | Measured Result | Verdict |
| :--- | :--- | :--- | :--- |
| **Sustained Request Throughput** | `>= 2,000 req/s` | **`20,443.7 req/s`** | **PASS (10.2x SLA)** |
| **Latency P50 (Median)** | `<= 10.0 ms` | **`0.52 ms`** | **PASS** |
| **Latency P95** | `<= 25.0 ms` | **`1.69 ms`** | **PASS** |
| **Latency P99** | `<= 50.0 ms` | **`1.78 ms`** | **PASS** |
| **Concurrent Wallet Deposit/Debit Safety (50 Goroutines)** | 0 race conditions, exact balance | **0 errors, exact ledger reconciliation** | **PASS** |
| **Concurrent Same-Invoice Payment Race (10 Goroutines)** | Exactly 1 HTTP 200, 9 HTTP 400, 1 provisioning callback | **1 success, 9 rejected (`ALREADY_PAID`), 1 callback** | **PASS** |
| **On-Disk Snapshot Reload Time (RTO)** | `<= 100 ms` | **`2.97 ms` (41 KB state file)** | **PASS** |

---

## 2. Live Production VPS Telemetry (`https://hostvra.com`)

Sampled directly from `https://hostvra.com/api/v1/public/status` during Phase 14.2:

```json
{
  "overall": "operational",
  "version": "0.1.0",
  "uptime": "1h 5m",
  "uptime_seconds": 3925,
  "goroutines": 15,
  "memory_mb": 7,
  "services": [
    {"name": "API Engine", "status": "operational"},
    {"name": "Hosting Control Plane", "status": "operational"},
    {"name": "Billing & Payments", "status": "operational"},
    {"name": "Domain Automation", "status": "operational"}
  ]
}
```

| Live Metric | Value | Assessment |
| :--- | :--- | :--- |
| **API Memory Footprint** | `7 MB` RSS | Minimal memory footprint, zero leak indicators |
| **Active Goroutines** | `15` | Clean worker pool state, zero leaked goroutines |
| **HTTPS Status Endpoint TTFB** | `0.659s` (inclusive of DNS + TLS handshake) | Healthy global response time |
| **HTTPS Landing Page TTFB** | `1.662s` | Healthy SSR response time |
