---
title: "Grafana Dashboards"
description: "Complete guide to all eight provisioned Grafana dashboards — what each panel shows, when to use each dashboard, and how to navigate between them."
sidebar_position: 5
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

# Grafana Dashboards

Eight dashboards are provisioned automatically in the `ecom-engine` folder when Grafana starts. All dashboards share a **30-second auto-refresh** and default to a **3-hour time window**.

**Starting point:** Always open **Service Health** first during an incident. It shows the critical SLO indicators in one view and links naturally to the focused dashboards.

---

## Navigation Guide

```
Incident starts
    ↓
Service Health — is something broken?
    ├─ 5xx spike → HTTP Traffic → drill by route
    ├─ High latency → HTTP Traffic → check DB / Cache
    ├─ DB pool > 90% → Database
    ├─ Cache hit rate < 50% → Cache
    ├─ Memory / CPU spike → Runtime & Process
    ├─ Auth failures → Security
    └─ Unexplained → Logs (filter by error)
                 → Traces (via trace.id in logs)
```

---

## Service Health

**UID:** `ecom-engine-health` | **Purpose:** On-call first look, SLO monitoring

The highest-level view. Covers the most critical indicators across all subsystems in a single screen.

### Panels

**Row: Service Overview**

| Panel | Type | Metric / Query | Description |
|-------|------|---------------|-------------|
| Request Rate | Stat | `job:ecom_engine_http_requests_total:rate1m` | Requests per second — baseline for "is traffic normal?" |
| 5xx Error Rate | Stat | `job:ecom_engine_http_5xx_rate_pct:rate1m` | Red threshold at 5%. Any sustained value here needs investigation. |
| P50 Latency | Stat | `histogram_quantile(0.50, ...)` | Median response time. Healthy target: < 100ms. |
| In-Flight Requests | Stat | `ecom_engine_http_requests_in_flight` | High in-flight with low request rate = stuck handlers. |
| Error Log Events | Stat | Loki count of `log.level=error` in 5m | Log-level error count — catches issues not reflected in HTTP status codes. |
| Warn Log Events | Stat | Loki count of `log.level=warn` in 5m | Leading indicator — warns often precede errors. |

**Row: Trends**

| Panel | Type | Description |
|-------|------|-------------|
| Error & Warn Rate | Timeseries | Error and warn log volume over time. Spot when an issue started. |
| HTTP Request Rate | Timeseries | Request volume trend. Distinguish traffic-driven spikes from internal failures. |

### When to use

Open this dashboard at the start of every on-call incident. If anything is red, navigate to the focused dashboard matching the symptom.

---

## HTTP Traffic

**UID:** `ecom-engine-http` | **Purpose:** Request rate, latency, and error investigation by route

### Panels

**Row: Summary**

| Panel | Type | Description |
|-------|------|-------------|
| Request Rate | Stat | Total requests/s |
| 5xx Error Rate | Stat | Percentage of requests returning 5xx |
| 4xx Error Rate | Stat | Percentage of requests returning 4xx |
| In-Flight Requests | Stat | Concurrent requests being processed |
| P50 Latency | Stat | Median response time |
| P99 Latency | Stat | 99th percentile — your slowest 1% of requests |

**Row: Request Traffic**

| Panel | Type | Description |
|-------|------|-------------|
| Request Rate by Route | Timeseries | Per-route traffic volume. Identify which endpoint is receiving the spike. |
| Response Time Percentiles (p50/p95/p99) | Timeseries | Latency trends over time across all routes. Watch for p99 diverging from p50. |

**Row: Status Codes**

| Panel | Type | Description |
|-------|------|-------------|
| Requests by Status Code Class | Timeseries | 2xx / 3xx / 4xx / 5xx stacked — see how the error class distribution changes. |

**Row: Error Analysis**

| Panel | Type | Description |
|-------|------|-------------|
| Top Error Messages | Table | Most frequent error log messages in the time range. Pivot from metric to root cause. |

### When to use

- 5xx spike in Service Health → open this dashboard to identify the route
- Latency complaint → compare p50 vs p99 to distinguish outliers from systemic slowness
- After a deploy → confirm request rate and error rate return to baseline

---

## Database

**UID:** `ecom-engine-db` | **Purpose:** PostgreSQL connection pool monitoring

### Panels

**Row: Pool Status**

| Panel | Type | Metric | Description |
|-------|------|--------|-------------|
| DB Pool Utilization | Gauge | `job:ecom_engine_db_pool_utilization_pct` | Current pool utilisation %. Yellow at 70%, red at 90%. Above 90% acquire timeouts become likely. |
| DB Pool Connections | Stat | acquired / idle / max | Three stats in one: how many connections are in use, available, and the configured max. |

**Row: Pool Trends**

| Panel | Type | Description |
|-------|------|-------------|
| Pool Connections & Acquire Rate | Timeseries | Acquired, idle, and total connections over time alongside the rate of empty-pool waits. |

### What to look for

- **Utilization gauge red (>90%):** Pool is near exhaustion. Acquire timeouts will start causing request failures. Increase `DB_POOL_MAX_CONNS` or reduce query duration.
- **Rising empty-acquire rate:** Requests are waiting for a free connection. Correlates with high latency in the HTTP Traffic dashboard.
- **Total conns < max_conns but utilization high:** Pool size is the right size but queries are slow — investigate slow queries.

---

## Cache

**UID:** `ecom-engine-cache` | **Purpose:** L1/L2 cache efficiency and Redis pool health

### Panels

**Row: Cache Summary**

| Panel | Type | Description |
|-------|------|-------------|
| L1 (Memory) Hit Rate | Stat | Fraction of L1 lookups returning a hit. Healthy: > 80% for hot data. |
| L2 (Redis) Hit Rate | Stat | Fraction of L2 (Redis) lookups returning a hit. Healthy: > 50%. |
| L1 Memory Fill Ratio | Gauge | Current items / max items in the in-memory cache. Yellow at 80%, red at 95%. |
| Stampede Dedup (5m) | Stat | Count of singleflight collapses. High value = stampede protection firing under load. |

**Row: Cache Operations**

| Panel | Type | Description |
|-------|------|-------------|
| Cache Requests by Layer & Result | Timeseries | Hit/miss/error counts per layer over time. Spot when hit rate dropped. |
| Cache Operation Latency (p50/p99) | Timeseries | L1 and L2 get latency percentiles. L1 should be sub-millisecond; L2 < 5ms. |

**Row: L1 Memory Cache**

| Panel | Type | Description |
|-------|------|-------------|
| L1 Memory Items | Timeseries | Item count over time. Flat line at max = capacity eviction is happening. |
| L1 Evictions by Reason | Timeseries | `expired` (TTL) vs `lfu` (capacity) evictions. Rising LFU evictions = L1 is too small. |

**Row: Redis (L2) Pool**

| Panel | Type | Description |
|-------|------|-------------|
| Redis Pool Connections | Stat | Total and idle Redis connections at a glance. |
| Redis Pool Hits / Misses / Timeouts | Timeseries | Pool hit rate and timeout rate over time. Timeout spikes indicate Redis is overloaded or unreachable. |

### What to look for

- **L2 hit rate drops suddenly:** Cold start after deploy, cache invalidation storm, or TTL too short.
- **Redis pool timeouts increasing:** Redis CPU saturation, network partition, or pool size too small.
- **Fallback alert fires:** Distributed rate limiting is no longer enforced — check Redis connectivity.

---

## Business Events

**UID:** `ecom-engine-business` | **Purpose:** Order, payment, cart, and catalog event monitoring

### Panels

**Row: Business Stats**

| Panel | Type | Description |
|-------|------|-------------|
| Order Events | Stat | Order-related event count in the time range |
| Payment Events | Stat | Payment-related event count |
| Cart Events | Stat | Cart add/remove/clear event count |
| Checkout Events | Stat | Checkout initiated / completed counts |

**Row: Event Rates**

| Panel | Type | Description |
|-------|------|-------------|
| Orders & Payments | Timeseries | Order and payment event rates over time. Correlated drops indicate a funnel problem. |
| Cart & Checkout | Timeseries | Cart activity and checkout conversion over time. |
| Catalog, Inventory & Shipping | Timeseries | Product view, inventory check, and shipping event rates. |

### When to use

- "Revenue is down" — check if order/payment rates dropped or if cart → checkout conversion fell.
- After a deploy — confirm business event rates return to expected levels.
- During a sale or campaign — monitor event volume in real time.

---

## Security

**UID:** `ecom-engine-security` | **Purpose:** Authentication failures and rate limiting health

### Panels

**Row: Authentication**

| Panel | Type | Description |
|-------|------|-------------|
| Auth Events | Timeseries | Login success / failure rates over time. A spike in failures may indicate a credential-stuffing attack. |

**Row: Rate Limiting**

| Panel | Type | Description |
|-------|------|-------------|
| Rate Limit Decisions (Prometheus) | Timeseries | Allow vs deny request counts over time. Normal traffic produces very low deny rates. |
| Rate Limit Backend | Stat | Current backend: `redis` (distributed) or `memory` (per-instance fallback). Red if fallback is active. |
| Backend Fallbacks (1h) | Stat | Count of Redis → memory fallback events in the last hour. Any non-zero value warrants investigation. |

### What to look for

- **Auth failure spike:** Possible brute-force or credential-stuffing attack. Check source IPs in the Logs dashboard.
- **Rate limit backend = memory:** Distributed limiting is not enforced. Investigate Redis connectivity.
- **Deny rate rising with no traffic increase:** Rate limit thresholds may be too tight, or a bot is probing.

---

## Logs

**UID:** `ecom-engine-logs` | **Purpose:** Ad-hoc log search with level and trace ID filtering

### Variables

| Variable | Type | Description |
|----------|------|-------------|
| Log Level | Multi-select | Filter the log stream to one or more levels (`debug`, `info`, `warn`, `error`) |

### Panels

**Row: Log Overview**

| Panel | Type | Description |
|-------|------|-------------|
| Log Events by Level | Timeseries | Error, warn, info, debug event volume over time. Spot when an issue started even before the alert fires. |
| Log Level Distribution | Pie chart | Proportional breakdown of log levels in the selected time range. |

**Row: Live Log Stream**

| Panel | Type | Description |
|-------|------|-------------|
| ecom-engine Live Logs | Logs | Full raw log stream, filterable by level. Click any log line to expand structured fields. Click `trace.id` to jump to Tempo. |

### Workflow

1. Set **Log Level** variable to `error` to filter out noise.
2. Find a relevant log line and expand it.
3. Copy the `trace.id` value.
4. Open **Explore → Tempo** and search by trace ID to open the full trace.

---

## Runtime & Process

**UID:** `ecom-engine-runtime` | **Purpose:** Go heap, GC, goroutines, CPU, and memory

### Panels

**Row: Resource Overview**

| Panel | Type | Thresholds | Description |
|-------|------|-----------|-------------|
| CPU Usage | Stat | Yellow > 60%, Red > 80% | Process CPU %. Sustained high CPU needs profiling. |
| Memory RSS | Stat | Yellow > 512 MiB, Red > 1 GiB | Physical memory in use. Growing RSS over time = memory leak. |
| Heap Alloc | Stat | Yellow > 256 MiB, Red > 512 MiB | Live Go heap allocations. |
| Goroutines | Stat | Yellow > 200, Red > 500 | Current goroutine count. Monotonic growth = goroutine leak. |
| GC Cycles / min | Stat | Yellow > 30, Red > 60 | GC frequency. Very high rates inflate p99 latency. |

**Row: CPU & Memory Trends**

| Panel | Type | Description |
|-------|------|-------------|
| CPU Usage Over Time | Timeseries | CPU % trend — identify when it started rising and correlate with deploy or traffic changes. |
| Memory Over Time | Timeseries | RSS, VMS, and heap allocated on a single graph. RSS growing while heap is stable = memory held outside the Go allocator. |

**Row: Go Runtime Details**

| Panel | Type | Description |
|-------|------|-------------|
| Goroutines Over Time | Timeseries | Goroutine count trend. Use the 15-minute window to distinguish normal bursts from a true leak. |
| GC Activity | Timeseries | Dual-axis: GC pause time (s/s) on the left, GC cycles/min on the right. High pause time with normal cycle count = large heap objects. |
| Heap Breakdown | Timeseries | Four lines: heap allocated, heap sys (OS granted), nextGC target, and heap objects count. Useful for diagnosing fragmentation or unexpectedly large heap reservation. |

### What to look for

- **Goroutines monotonically increasing:** Goroutine leak — a goroutine is started but never exits. Check for unclosed channels, missing `cancel()` calls, or stuck HTTP/DB calls.
- **GC pause rate > 50ms/s:** GC is consuming significant CPU time. Reduce heap allocation rate — look for hot paths creating many small objects.
- **RSS growing while Heap Alloc is stable:** Memory is being held outside Go's allocator (e.g. CGo, large `mmap` regions, or a growing `sync.Pool`).
- **CPU spike with no traffic increase:** A background goroutine is doing expensive work — check GC cycle rate and goroutine count for correlation.
