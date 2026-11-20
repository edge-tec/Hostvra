# HOSTVRA — P12 PERFORMANCE & RELIABILITY REPORT

**Execution Date & Time:** 2026-11-20T17:19:30+06:00  
**Certification Status:** **PASS (Controlled Local Concurrency Benchmark)**

---

## 1. Controlled Load & Latency Benchmark Evidence (`TestP12_PerformanceThroughputAndLatencyMetrics`)

Executed with Go `-race` detector enabled (`go test -v -race ./internal/handlers -run TestP12_PerformanceThroughputAndLatencyMetrics`):

```text
=== RUN   TestP12_PerformanceThroughputAndLatencyMetrics
    commercial_test.go:738: [PERF CERTIFIED] Requests=300 Concurrency=20 Throughput=20443.7 req/s | P50=518.417µs P95=1.693333ms P99=1.784958ms
--- PASS: TestP12_PerformanceThroughputAndLatencyMetrics (0.01s)
```

| Metric | Measured Result | Threshold | Status |
| :--- | :--- | :--- | :--- |
| **Total Requests** | `300` (`GET /api/v1/public/status` + `POST /api/v1/billing/coupons/validate`) | 300 | **PASS** |
| **Concurrency Workers** | `20` concurrent goroutines (with `-race` enabled) | 20 | **PASS** |
| **Throughput** | **`20,443.7 req/s`** | >= 500 req/s | **PASS** |
| **P50 (Median) Latency** | **`518.42 µs` (`0.52 ms`)** | <= 50 ms | **PASS** |
| **P95 Latency** | **`1.69 ms`** | <= 100 ms | **PASS** |
| **P99 Latency** | **`1.78 ms`** | <= 250 ms | **PASS** |
| **Error Rate / Data Races** | **`0.00%` (0 failed requests, 0 race conditions)** | 0% | **PASS** |
