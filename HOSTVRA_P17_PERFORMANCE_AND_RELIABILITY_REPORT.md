# HOSTVRA — PHASE P17.12: PERFORMANCE & RELIABILITY REPORT

**UTC Timestamp:** `2026-11-20T14:26:05Z`  
**Verification Method:** Live HTTPS `/api/v1/public/status` sampling + Local Go `-race` benchmarks  
**Status:** **PASS**

---

## 1. Live Production VPS Telemetry (`https://hostvra.com/api/v1/public/status`)
Sampled across 3 independent timestamps over 2 hours (`12:27:12Z`, `13:34:17Z`, `14:26:05Z`):
- **Uptime:** `4,644s` -> `8,670s` -> **`11,777s` (`3h 16m 17s`, 0 crashes/restarts)**
- **Memory Allocation (`memory_mb`):** `8 MB` -> `8 MB` -> **`6 MB`** (healthy GC, zero memory leak)
- **Active Goroutines (`goroutines`):** `14` -> `14` -> **`14`** (flat, zero goroutine leak)
- **API Internal Latency:** `1 ms`
- **Database Probe Latency:** `0 ms` (`< 1 ms`)
- **Crypto Gateway:** `7 active blockchain networks monitored` (`0 ms` probe latency)

## 2. Local Concurrency Benchmarks (`TestP12_PerformanceThroughputAndLatencyMetrics`, `-race`)
- **Throughput:** `16,600.0 – 20,443.7 req/s` (300 requests, 20 concurrent workers)
- **P50 Latency:** `0.517 ms` | **P95 Latency:** `2.160 ms` | **P99 Latency:** `2.286 ms`
- **Distinction Notice:** Throughput figures are local `-race` benchmarks; live VPS metrics above are real production telemetry from `https://hostvra.com`.
