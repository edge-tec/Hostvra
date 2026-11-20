# HOSTVRA — PHASE P15: PERFORMANCE & RELIABILITY REPORT

**Audit Date:** 2026-11-20  
**Repository:** `edge-tec/Hostvra` (`main`)  
**Phase:** P15.11 — Performance & Reliability  
**Status:** **PASS**

---

## 1. Local Concurrency & Latency Benchmark Results (`-race` enabled)

Measured under 20 concurrent workers issuing 300 parallel requests (`TestP12_PerformanceThroughputAndLatencyMetrics`), plus 10-worker concurrent wallet settlement (`TestP14_*`) and 10-worker concurrent signed webhook delivery (`TestP15_*`):

| Metric | Target Threshold | Measured Value (with `-race`) | Verdict |
| :--- | :--- | :--- | :---: |
| **API Throughput** | `>= 2,000 req/s` | **`17,783.7 – 20,443.7 req/s`** | **PASS** |
| **P50 Latency** | `<= 10.0 ms` | **`0.576 ms` (`576.17 µs`)** | **PASS** |
| **P95 Latency** | `<= 25.0 ms` | **`2.026 ms`** | **PASS** |
| **P99 Latency** | `<= 50.0 ms` | **`2.261 ms`** | **PASS** |
| **Disk Snapshot Backup Write Time** | `<= 10.0 ms` | **`0.167 ms` (`167.25 µs`)** | **PASS** |
| **Cold Restart Disk Snapshot Restore (RTO)** | `<= 100.0 ms` | **`3.165 ms` (`41,058 bytes`)** | **PASS** |

---

## 2. Live Production VPS Resource & Telemetry Measurements (`https://hostvra.com`)

Queried live from `https://hostvra.com/api/v1/public/status` at `2026-11-20T12:27:12Z`:

| Telemetry Metric | Measured Live Value | Assessment |
| :--- | :--- | :--- |
| **Overall Platform Status** | `operational` | All 4 core subsystems healthy |
| **Process Uptime** | `4,644 seconds` (`1h 17m`) | Stable continuous operation, 0 crashes |
| **Go Runtime Version** | `go1.26.0` | Latest Go runtime |
| **Heap Memory Allocation (`memory_mb`)** | `8 MB` | Extremely low footprint, no memory leak |
| **Active Goroutines (`goroutines`)** | `14` | Stable background worker count (was 15 at 3925s, 14 at 4644s) |
| **Control Plane API Internal Latency** | `1 ms` | Instantaneous internal dispatch |
| **Database & Billing Ledger Probe Latency** | `0 ms` (`< 1 ms`) | Sub-millisecond query response |
| **Crypto Gateway & HD Wallet Probe Latency** | `0 ms` (`7 active networks`) | Sub-millisecond registry response |
