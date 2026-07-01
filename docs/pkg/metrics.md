---
title: metrics
sidebar_label: metrics
sidebar_position: 11
---

# metrics

<DocBadge status="under-review" version="v0.1.0-alpha" />

The `metrics` package registers all **Prometheus metrics** for the application in a single place. Metrics are auto-registered with the default Prometheus registry on package import (via `promauto`).

**Import path:** `ecom-engine/pkg/metrics`

> **Note:** This package imports `internal/infra/db` to access the `PoolStatsProvider` interface for database pool metrics. This is the one exception to the `pkg/` rule of no `internal/` imports — it exists to keep all metric registrations in one canonical location.

> For the full metric catalog, PromQL examples, recording rules, Grafana dashboards, and alert playbooks, see the **[Observability](../observability/overview.md)** section.

---

## Metric namespace

All metrics share the namespace `ecom_engine`. The full metric name follows the pattern:

```
ecom_engine_<subsystem>_<name>
```

---

## HTTP metrics

These are updated by the metrics middleware in the gateway layer. They cover every completed HTTP request.

### HTTPRequestsTotal

```
ecom_engine_http_requests_total
```

**Type:** Counter
**Labels:** `method`, `route`, `status`

Counts completed HTTP requests partitioned by HTTP method (GET, POST, etc.), route template (e.g. `/api/v1/products/:id`), and HTTP status code.

Use this to compute request rates and error rates per endpoint.

### HTTPRequestDuration

```
ecom_engine_http_request_duration_seconds
```

**Type:** Histogram
**Labels:** `method`, `route`
**Buckets:** 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s

Tracks request latency in seconds. Use the `_bucket`, `_sum`, and `_count` suffixes to compute percentile latency (p50, p95, p99) in Prometheus.

### HTTPRequestsInFlight

```
ecom_engine_http_requests_in_flight
```

**Type:** Gauge
**Labels:** none

Current number of HTTP requests actively being processed. Useful for detecting traffic spikes and connection saturation.

---

## Database pool metrics

### DBPoolCollector

A `prometheus.Collector` implementation that reads live PostgreSQL connection pool statistics from any `infradb.PoolStatsProvider`.

Construct and register it at startup:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "ecom-engine/pkg/metrics"
)

collector := metrics.NewDBPoolCollector(pgPool) // pgPool implements PoolStatsProvider
prometheus.MustRegister(collector)
```

Metrics exposed:

| Metric | Type | Description |
|---|---|---|
| `ecom_engine_db_pool_max_conns` | Gauge | Configured maximum connections in pool |
| `ecom_engine_db_pool_total_conns` | Gauge | Current total connections (acquired + idle) |
| `ecom_engine_db_pool_acquired_conns` | Gauge | Connections currently in use |
| `ecom_engine_db_pool_idle_conns` | Gauge | Connections available for acquisition |
| `ecom_engine_db_pool_acquire_count_total` | Counter | Cumulative successful acquisitions |
| `ecom_engine_db_pool_empty_acquire_count_total` | Counter | Acquisitions that waited because pool was empty |
| `ecom_engine_db_pool_acquire_duration_seconds_total` | Counter | Cumulative time waiting to acquire connections |

---

## Cache metrics

Used by the cache infrastructure layer to track L1 (in-memory) and L2 (Redis) cache operations.

### CacheRequestsTotal

```
ecom_engine_cache_requests_total
```

**Type:** Counter
**Labels:** `layer` (`L1`/`L2`), `operation` (`get`/`set`/`delete`/`increment`), `result` (`hit`/`miss`/`error`)

Use this to calculate the cache hit rate per layer:

```promql
rate(ecom_engine_cache_requests_total{result="hit"}[5m])
/
rate(ecom_engine_cache_requests_total[5m])
```

### CacheOperationDuration

```
ecom_engine_cache_operation_duration_seconds
```

**Type:** Histogram
**Labels:** `layer`, `operation`
**Buckets:** 0.1ms, 0.5ms, 1ms, 5ms, 10ms, 25ms, 50ms, 100ms, 500ms

Tracks cache operation latency. Buckets are skewed toward sub-millisecond ranges expected of L1/L2 caches.

### CacheStampedeDedupTotal

```
ecom_engine_cache_stampede_dedup_total
```

**Type:** Counter

Counts requests collapsed by singleflight — concurrent callers for the same missing key that shared a single fetch call. A high value means stampede protection is actively firing under load.

### CacheNegativeCacheTotal

```
ecom_engine_cache_negative_cache_total
```

**Type:** Counter

Counts null sentinels written to cache when the fetch function confirmed a record does not exist. A sudden spike may indicate a cache penetration attack (repeated lookups for non-existent keys).

### CacheMemoryItems

```
ecom_engine_cache_memory_items
```

**Type:** Gauge

Current number of items in the L1 in-memory cache. Alert if this approaches `CacheMemoryMaxItems`.

### CacheMemoryMaxItems

```
ecom_engine_cache_memory_max_items
```

**Type:** Gauge

Configured maximum item capacity of the L1 cache. Exposed as a gauge for dashboard ratio calculations:

```promql
ecom_engine_cache_memory_items / ecom_engine_cache_memory_max_items
```

### CacheMemoryEvictionsTotal

```
ecom_engine_cache_memory_evictions_total
```

**Type:** Counter
**Labels:** `reason` (`expired`, `lfu`)

Evictions from the L1 cache. `expired` means TTL elapsed; `lfu` means the item was evicted due to capacity (Least Frequently Used policy).

---

## Redis pool metrics

### CacheRedisPoolCollector

A `prometheus.Collector` that exposes connection pool statistics from the go-redis client.

```go
collector := metrics.NewCacheRedisPoolCollector(redisClient)
prometheus.MustRegister(collector)
```

Metrics exposed:

| Metric | Type | Description |
|---|---|---|
| `ecom_engine_cache_redis_pool_total_conns` | Gauge | Total connections in Redis pool |
| `ecom_engine_cache_redis_pool_idle_conns` | Gauge | Idle connections available |
| `ecom_engine_cache_redis_pool_stale_conns_total` | Counter | Stale connections removed |
| `ecom_engine_cache_redis_pool_hits_total` | Counter | Pool hits (free connection found) |
| `ecom_engine_cache_redis_pool_misses_total` | Counter | Pool misses (new connection dialed) |
| `ecom_engine_cache_redis_pool_timeouts_total` | Counter | Callers that timed out waiting for a connection |
