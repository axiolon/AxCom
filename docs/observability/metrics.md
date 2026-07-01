---
title: "Metrics"
description: "Complete catalog of every Prometheus metric exported by AxCom, including labels, types, recording rules, and PromQL examples."
sidebar_position: 2
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

# Metrics

All metrics share the namespace `ecom_engine`. The full metric name follows the pattern `ecom_engine_<subsystem>_<name>`.

Metrics are registered at application startup by the `pkg/metrics` package. Prometheus scrapes the `/metrics` endpoint every **15 seconds**.

> For Go API usage (registering collectors in code), see [pkg/metrics](../pkg/metrics.md).

---

## HTTP Metrics

Updated by the metrics middleware on every completed HTTP request.

### `ecom_engine_http_requests_total`

**Type:** Counter | **Labels:** `method`, `route`, `status`

Counts completed HTTP requests partitioned by HTTP method (GET, POST, etc.), route template (e.g. `/api/v1/products/:id`), and HTTP status code string (e.g. `200`, `404`, `500`).

```promql
# Request rate across all routes
rate(ecom_engine_http_requests_total[1m])

# 5xx error rate per route
rate(ecom_engine_http_requests_total{status=~"5.."}[1m])

# Top routes by request volume
topk(10, sum by (route) (rate(ecom_engine_http_requests_total[5m])))
```

### `ecom_engine_http_request_duration_seconds`

**Type:** Histogram | **Labels:** `method`, `route`
**Buckets:** 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s

Tracks request latency in seconds. Use the `_bucket`, `_sum`, and `_count` suffixes to compute percentile latency.

```promql
# p99 latency across all routes
histogram_quantile(0.99,
  sum by (le) (rate(ecom_engine_http_request_duration_seconds_bucket[5m]))
)

# p95 latency per route
histogram_quantile(0.95,
  sum by (le, route) (rate(ecom_engine_http_request_duration_seconds_bucket[5m]))
)
```

### `ecom_engine_http_requests_in_flight`

**Type:** Gauge | **Labels:** none

Current number of HTTP requests actively being processed. Useful for detecting traffic spikes and connection saturation.

```promql
ecom_engine_http_requests_in_flight
```

---

## Database Pool Metrics

Exposed by `DBPoolCollector`, a custom `prometheus.Collector` that reads live pool statistics from the PostgreSQL connection pool (`pgxpool`).

| Metric                                               | Type    | Description                                           |
| ---------------------------------------------------- | ------- | ----------------------------------------------------- |
| `ecom_engine_db_pool_max_conns`                      | Gauge   | Configured maximum connections                        |
| `ecom_engine_db_pool_total_conns`                    | Gauge   | Total connections (acquired + idle)                   |
| `ecom_engine_db_pool_acquired_conns`                 | Gauge   | Connections currently in use                          |
| `ecom_engine_db_pool_idle_conns`                     | Gauge   | Connections available for acquisition                 |
| `ecom_engine_db_pool_acquire_count_total`            | Counter | Cumulative successful acquisitions                    |
| `ecom_engine_db_pool_empty_acquire_count_total`      | Counter | Acquisitions that waited because pool was empty       |
| `ecom_engine_db_pool_acquire_duration_seconds_total` | Counter | Cumulative time spent waiting to acquire a connection |

```promql
# Pool utilization % (use recording rule instead)
100 * ecom_engine_db_pool_acquired_conns
  / clamp_min(ecom_engine_db_pool_max_conns, 1)

# Rate of empty-pool waits
rate(ecom_engine_db_pool_empty_acquire_count_total[1m])
```

---

## Cache Metrics

Used by the two-layer cache (`L1` = in-memory, `L2` = Redis).

### `ecom_engine_cache_requests_total`

**Type:** Counter | **Labels:** `layer` (`L1`/`L2`), `operation` (`get`/`set`/`delete`/`increment`), `result` (`hit`/`miss`/`error`)

```promql
# L2 (Redis) hit rate
sum(rate(ecom_engine_cache_requests_total{layer="L2", result="hit"}[5m]))
/ clamp_min(sum(rate(ecom_engine_cache_requests_total{layer="L2"}[5m])), 0.001)
```

### `ecom_engine_cache_operation_duration_seconds`

**Type:** Histogram | **Labels:** `layer`, `operation`
**Buckets:** 0.1ms, 0.5ms, 1ms, 5ms, 10ms, 25ms, 50ms, 100ms, 500ms

Cache operation latency. Buckets are skewed toward sub-millisecond ranges expected of L1/L2 caches.

```promql
# p99 Redis GET latency
histogram_quantile(0.99,
  sum by (le) (rate(ecom_engine_cache_operation_duration_seconds_bucket{layer="L2", operation="get"}[5m]))
)
```

### `ecom_engine_cache_stampede_dedup_total`

**Type:** Counter

Counts requests collapsed by singleflight — concurrent callers for the same missing key that shared a single fetch. A high value means stampede protection is actively working under load.

### `ecom_engine_cache_negative_cache_total`

**Type:** Counter

Counts null sentinels written when the fetch function confirmed a record does not exist. A sudden spike may indicate a cache penetration attack.

### `ecom_engine_cache_memory_items`

**Type:** Gauge

Current number of items in the L1 in-memory cache.

### `ecom_engine_cache_memory_max_items`

**Type:** Gauge

Configured maximum capacity of the L1 cache.

```promql
# L1 fill ratio
ecom_engine_cache_memory_items / ecom_engine_cache_memory_max_items
```

### `ecom_engine_cache_memory_evictions_total`

**Type:** Counter | **Labels:** `reason` (`expired`, `lfu`)

Evictions from the L1 cache. `expired` = TTL elapsed; `lfu` = evicted by capacity (Least Frequently Used policy).

---

## Redis Pool Metrics

Exposed by `CacheRedisPoolCollector`, a custom collector reading go-redis pool stats.

| Metric                                           | Type    | Description                                     |
| ------------------------------------------------ | ------- | ----------------------------------------------- |
| `ecom_engine_cache_redis_pool_total_conns`       | Gauge   | Total connections in Redis pool                 |
| `ecom_engine_cache_redis_pool_idle_conns`        | Gauge   | Idle connections available                      |
| `ecom_engine_cache_redis_pool_stale_conns_total` | Counter | Stale connections removed                       |
| `ecom_engine_cache_redis_pool_hits_total`        | Counter | Pool hits (free connection found immediately)   |
| `ecom_engine_cache_redis_pool_misses_total`      | Counter | Pool misses (new connection had to be dialled)  |
| `ecom_engine_cache_redis_pool_timeouts_total`    | Counter | Callers that timed out waiting for a connection |

```promql
# Redis pool hit rate
rate(ecom_engine_cache_redis_pool_hits_total[1m])
/ clamp_min(
    rate(ecom_engine_cache_redis_pool_hits_total[1m])
    + rate(ecom_engine_cache_redis_pool_misses_total[1m]),
    0.001
  )
```

---

## Rate Limiting Metrics

### `ecom_engine_ratelimit_requests_total`

**Type:** Counter | **Labels:** `bucket`, `decision`

**`bucket` values:**

| Value         | Description                                    |
| ------------- | ---------------------------------------------- |
| `global`      | Shared global bucket across all IPs            |
| `tier:public` | Per-IP bucket for unauthenticated requests     |
| `tier:auth`   | Per-IP bucket for authenticated users          |
| `tier:admin`  | Per-IP bucket for admin users                  |
| `ep:auth`     | Endpoint-specific bucket for `/api/auth/*`     |
| `ep:checkout` | Endpoint-specific bucket for `/api/checkout/*` |
| `ep:payments` | Endpoint-specific bucket for `/api/payments/*` |

**`decision` values:** `allowed`, `denied`

```promql
# Total denial rate across all buckets
rate(ecom_engine_ratelimit_requests_total{decision="denied"}[1m])

# Denials by bucket (which tier/endpoint is being limited)
sum by (bucket) (rate(ecom_engine_ratelimit_requests_total{decision="denied"}[5m]))

# % of requests denied
100 * rate(ecom_engine_ratelimit_requests_total{decision="denied"}[5m])
  / clamp_min(rate(ecom_engine_ratelimit_requests_total[5m]), 0.001)
```

### `ecom_engine_ratelimit_backend_active`

**Type:** Gauge | **Labels:** `backend` (`redis`, `memory`)

Set to `1` for the currently active rate-limit backend, `0` for the inactive one.

```promql
# Is Redis currently the active backend?
ecom_engine_ratelimit_backend_active{backend="redis"}
```

### `ecom_engine_ratelimit_backend_fallbacks_total`

**Type:** Counter

Incremented whenever the distributed Redis-backed rate limiter falls back to per-instance in-memory limiting. Any non-zero value means distributed rate limiting is not being enforced.

```promql
# Fallbacks in the last hour
increase(ecom_engine_ratelimit_backend_fallbacks_total[1h])
```

### `ecom_engine_ratelimit_backend_recoveries_total`

**Type:** Counter

Incremented when the rate limiter successfully recovers from in-memory fallback back to Redis after 3 consecutive healthy probes (~90 seconds of stability).

### `ecom_engine_ratelimit_redis_errors_total`

**Type:** Counter

Counts all Redis errors encountered by the rate limiter (per-request and health probe failures). Rising values indicate Redis instability even before a full fallback occurs.

---

## Runtime & Process Metrics

Exposed by `RuntimeCollector`, which reads Go runtime stats (`runtime.ReadMemStats`) and OS process stats (`gopsutil`).

### Go Heap

| Metric                                 | Type  | Description                                   |
| -------------------------------------- | ----- | --------------------------------------------- |
| `ecom_engine_runtime_heap_alloc_bytes` | Gauge | Bytes currently allocated on the heap         |
| `ecom_engine_runtime_heap_sys_bytes`   | Gauge | Total bytes obtained from the OS for the heap |
| `ecom_engine_runtime_heap_objects`     | Gauge | Number of live heap objects                   |
| `ecom_engine_runtime_stack_sys_bytes`  | Gauge | Bytes used by goroutine stacks                |
| `ecom_engine_runtime_next_gc_bytes`    | Gauge | Heap size at which the next GC will trigger   |

### Go GC

| Metric                                       | Type    | Description                                        |
| -------------------------------------------- | ------- | -------------------------------------------------- |
| `ecom_engine_runtime_gc_cycles_total`        | Counter | Total GC cycles completed since startup            |
| `ecom_engine_runtime_gc_pause_seconds_total` | Counter | Cumulative stop-the-world GC pause time in seconds |

```promql
# GC pause time consumed per second (alerting threshold)
rate(ecom_engine_runtime_gc_pause_seconds_total[1m])

# GC cycles per minute
rate(ecom_engine_runtime_gc_cycles_total[1m]) * 60
```

### Goroutines

| Metric                           | Type  | Description                       |
| -------------------------------- | ----- | --------------------------------- |
| `ecom_engine_runtime_goroutines` | Gauge | Current number of live goroutines |

A monotonically growing goroutine count over 15+ minutes is a strong indicator of a goroutine leak.

### OS Process

| Metric                                 | Type  | Description                                        |
| -------------------------------------- | ----- | -------------------------------------------------- |
| `ecom_engine_process_cpu_percent`      | Gauge | Process CPU usage as a percentage (0–100 per core) |
| `ecom_engine_process_memory_rss_bytes` | Gauge | Resident Set Size — physical memory in use         |
| `ecom_engine_process_memory_vms_bytes` | Gauge | Virtual Memory Size — total virtual address space  |

> **Note:** OS metrics are collected via `gopsutil`. If the process runs in a restricted container that denies `/proc` access, these metrics are silently omitted and the other runtime metrics continue to be exported normally.

---

## Recording Rules

Pre-computed expressions in `prometheus/rules/recording-rules.yml`. Use these in dashboards and alerts instead of recomputing the same `rate()` on every query.

### HTTP

| Rule name                                      | Expression                                       | Description            |
| ---------------------------------------------- | ------------------------------------------------ | ---------------------- |
| `job:ecom_engine_http_requests_total:rate1m`   | `sum(rate(ecom_engine_http_requests_total[1m]))` | Total request rate     |
| `route:ecom_engine_http_requests_total:rate1m` | `sum by (route, method, status) (rate(...[1m]))` | Request rate per route |
| `job:ecom_engine_http_5xx_rate_pct:rate1m`     | `100 * 5xx_rate / total_rate`                    | 5xx error %            |
| `job:ecom_engine_http_4xx_rate_pct:rate1m`     | `100 * 4xx_rate / total_rate`                    | 4xx error %            |

### Database

| Rule name                                            | Expression                                        | Description            |
| ---------------------------------------------------- | ------------------------------------------------- | ---------------------- |
| `job:ecom_engine_db_pool_utilization_pct`            | `100 * acquired / max`                            | Pool utilization %     |
| `job:ecom_engine_db_pool_empty_acquires:rate1m`      | `rate(empty_acquire_count_total[1m])`             | Empty-acquire rate     |
| `job:ecom_engine_db_pool_acquire_duration_ms:rate1m` | `rate(acquire_duration_seconds_total[1m]) * 1000` | Acquire duration in ms |

### Cache

| Rule name                                   | Expression               | Description                |
| ------------------------------------------- | ------------------------ | -------------------------- |
| `layer:ecom_engine_cache_hit_rate:rate1m`   | `hits / total by layer`  | Hit rate per layer (L1/L2) |
| `job:ecom_engine_cache_memory_fill_ratio`   | `items / max_items`      | L1 fill ratio              |
| `job:ecom_engine_cache_redis_pool_hit_rate` | `hits / (hits + misses)` | Redis pool hit ratio       |

### Rate Limiting

| Rule name                                  | Expression                                                    | Description |
| ------------------------------------------ | ------------------------------------------------------------- | ----------- |
| `job:ecom_engine_ratelimit_denied:rate1m`  | `sum(rate(ratelimit_requests_total{decision="denied"}[1m]))`  | Denial rate |
| `job:ecom_engine_ratelimit_allowed:rate1m` | `sum(rate(ratelimit_requests_total{decision="allowed"}[1m]))` | Allow rate  |

### Runtime

| Rule name                                        | Expression                         | Description                               |
| ------------------------------------------------ | ---------------------------------- | ----------------------------------------- |
| `job:ecom_engine_runtime_gc_pause_rate:rate1m`   | `rate(gc_pause_seconds_total[1m])` | GC pause s/s (used by alerting threshold) |
| `job:ecom_engine_runtime_gc_cycles:rate1m`       | `rate(gc_cycles_total[1m]) * 60`   | GC cycles/min                             |
| `job:ecom_engine_runtime_heap_utilization_ratio` | `heap_alloc / heap_sys`            | Fraction of OS-granted heap in use        |
