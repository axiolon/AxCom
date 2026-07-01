---
title: "Alerts"
description: "Complete catalog of all Prometheus and Grafana alerts for AxCom — thresholds, severity, and response playbooks."
sidebar_position: 6
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

# Alerts

Alerts are defined in two independent layers so they survive partial outages:

| Layer                         | Config file                           | Fires even if                             |
| ----------------------------- | ------------------------------------- | ----------------------------------------- |
| **Prometheus alerting rules** | `prometheus/rules/alerting-rules.yml` | Grafana is down                           |
| **Grafana unified alerting**  | `grafana/provisioning/alerting/*.yml` | — (Loki-based alerts only available here) |

Prometheus-native alerts cover all metric-based conditions. Grafana unified alerting adds log-derived alerts sourced from Loki.

---

## Severity Levels

| Severity   | Meaning                                                                           | Example action                   |
| ---------- | --------------------------------------------------------------------------------- | -------------------------------- |
| `critical` | Service is degraded or at risk of going down. Page on-call immediately.           | CPU >95%, 5xx rate >5%           |
| `warning`  | Degraded state that needs investigation but is not immediately service-impacting. | CPU >80%, cache hit rate &lt;50% |

---

## HTTP Alerts

### `HighHttpErrorRate`

| Property   | Value                                          |
| ---------- | ---------------------------------------------- |
| Severity   | `critical`                                     |
| Condition  | 5xx error rate > 5% for 5 minutes              |
| Expression | `job:ecom_engine_http_5xx_rate_pct:rate1m > 5` |
| File       | `alerting-rules.yml`                           |

**What it means:** More than 1 in 20 requests are returning server errors. This is visible to users.

**Response:**

1. Open **HTTP Traffic** dashboard — identify the route(s) with the spike.
2. Open **Logs** dashboard filtered to `error` — find the underlying error message.
3. Check **Database** dashboard — pool exhaustion causes 5xx responses when queries fail.
4. Check **Runtime & Process** — high CPU or memory pressure can cause handler timeouts.

---

### `NoHttpTraffic`

| Property   | Value                                                                                        |
| ---------- | -------------------------------------------------------------------------------------------- |
| Severity   | `critical`                                                                                   |
| Condition  | Request rate is zero for 5 minutes                                                           |
| Expression | `absent(ecom_engine_http_requests_total) or job:ecom_engine_http_requests_total:rate1m == 0` |
| File       | `alerting-rules.yml`                                                                         |

**What it means:** The service is receiving no traffic. Either the service is down, the load balancer is misconfigured, or a network partition has occurred.

**Response:**

1. Check if the container/process is running.
2. Check load balancer health checks.
3. If the process is running, verify it's listening on the expected port.

---

### `HighInFlightRequests`

| Property   | Value                                       |
| ---------- | ------------------------------------------- |
| Severity   | `warning`                                   |
| Condition  | In-flight request count > 100 for 2 minutes |
| Expression | `ecom_engine_http_requests_in_flight > 100` |
| File       | `alerting-rules.yml`                        |

**What it means:** Many requests are being held open simultaneously. Could indicate slow handlers, DB acquire timeouts holding connections, or an upstream service that is not responding.

**Response:**

1. Check **Database** dashboard — pool exhaustion causes handlers to block waiting for a connection.
2. Check **HTTP Traffic** p99 latency — slow routes inflate in-flight count.

---

### `HighP99Latency`

| Property   | Value                                                                                                    |
| ---------- | -------------------------------------------------------------------------------------------------------- |
| Severity   | `warning`                                                                                                |
| Condition  | p99 latency > 2 seconds for 5 minutes                                                                    |
| Expression | `histogram_quantile(0.99, sum by (le) (rate(ecom_engine_http_request_duration_seconds_bucket[5m]))) > 2` |
| File       | `alerting-rules.yml`                                                                                     |

**What it means:** The slowest 1% of requests are taking over 2 seconds. Users on slow connections or with large payloads will experience noticeable lag.

**Response:**

1. Open **HTTP Traffic** — narrow down to specific route(s) with high p99.
2. Check **Cache** — a low L2 hit rate forces more DB queries, inflating latency.
3. Check **Runtime & Process** — high GC pause rate inflates p99 independently of DB/cache.

---

## Database Alerts

### `DbPoolExhausted`

| Property   | Value                                          |
| ---------- | ---------------------------------------------- |
| Severity   | `critical`                                     |
| Condition  | DB pool utilization > 90% for 2 minutes        |
| Expression | `job:ecom_engine_db_pool_utilization_pct > 90` |
| File       | `alerting-rules.yml`                           |

**What it means:** Nearly all DB connections are in use. Acquire timeouts are imminent, which will cause request failures.

**Response:**

1. Increase `DB_POOL_MAX_CONNS` if the database can handle more connections.
2. Check for slow queries holding connections open longer than expected.
3. Check if a specific endpoint is responsible (HTTP Traffic dashboard → correlate with timing).

---

### `DbPoolEmptyAcquires`

| Property   | Value                                                 |
| ---------- | ----------------------------------------------------- |
| Severity   | `warning`                                             |
| Condition  | Empty-pool acquire rate > 0.5/s for 3 minutes         |
| Expression | `job:ecom_engine_db_pool_empty_acquires:rate1m > 0.5` |
| File       | `alerting-rules.yml`                                  |

**What it means:** Requests are waiting for a free DB connection. Pool is undersized relative to current traffic.

**Response:**

1. Consider increasing `DB_POOL_MAX_CONNS`.
2. Investigate whether the cache is working — cache hits reduce DB pressure.

---

## Cache Alerts

### `LowCacheHitRate` / `ecom-alert-redis-hit-rate-low`

| Property   | Value                                                          |
| ---------- | -------------------------------------------------------------- |
| Severity   | `warning`                                                      |
| Condition  | Redis (L2) cache hit rate &lt; 50% for 10 minutes              |
| Expression | `layer:ecom_engine_cache_hit_rate:rate1m{layer="redis"} < 0.5` |
| Files      | `alerting-rules.yml`, `cache-alerts.yml`                       |

**What it means:** More than half of L2 cache lookups are misses, forcing DB queries. Possible causes: cold start after deploy, cache invalidation storm, TTL too short, or the cache was flushed.

**Response:**

1. Check if this coincided with a deploy (expected cold start — will resolve in minutes).
2. Check L1 hit rate in **Cache** dashboard — if L1 is also low, the issue is upstream.
3. Check **Database** dashboard — rising DB acquire times confirm the cache miss is hitting the DB.

---

### `RedisPoolTimeouts` / `ecom-alert-redis-pool-timeouts`

| Property   | Value                                                         |
| ---------- | ------------------------------------------------------------- |
| Severity   | `warning`                                                     |
| Condition  | Redis pool timeout rate > 0.1/s for 5 minutes                 |
| Expression | `rate(ecom_engine_cache_redis_pool_timeouts_total[1m]) > 0.1` |
| Files      | `alerting-rules.yml`, `cache-alerts.yml`                      |

**What it means:** Requests are timing out while waiting for a Redis connection. Redis may be overloaded, unreachable, or the pool size is too small.

**Response:**

1. Check Redis CPU and memory usage.
2. Check network connectivity to Redis.
3. Consider increasing Redis pool max size if latency is normal.

---

### `RateLimitBackendFallback` / `ecom-alert-ratelimit-fallback`

| Property   | Value                                                             |
| ---------- | ----------------------------------------------------------------- |
| Severity   | `warning`                                                         |
| Condition  | Any backend fallback in the last 5 minutes                        |
| Expression | `increase(ecom_engine_ratelimit_backend_fallbacks_total[5m]) > 0` |
| `for`      | `0m` (fires immediately)                                          |
| Files      | `alerting-rules.yml`, `cache-alerts.yml`                          |

**What it means:** The distributed Redis-backed rate limiter fell back to per-instance in-memory limiting. Rate limits are no longer enforced across multiple instances — each instance enforces independently.

**Response:**

1. Check Redis connectivity (same as Redis pool timeout investigation).
2. The **Security** dashboard shows the current backend state.
3. Per-instance limiting continues to work — the service is not unprotected, but distributed enforcement is absent.

---

## Runtime & Process Alerts

### `HighCpuUsage` / `ecom-alert-high-cpu`

| Property   | Value                                      |
| ---------- | ------------------------------------------ |
| Severity   | `warning`                                  |
| Condition  | Process CPU > 80% for 5 minutes            |
| Expression | `ecom_engine_process_cpu_percent > 80`     |
| Files      | `alerting-rules.yml`, `runtime-alerts.yml` |

**What it means:** The Go process is consuming more than 80% of a CPU core for 5+ minutes. At this level performance remains acceptable but is approaching a ceiling.

**Response:**

1. Check **Runtime & Process** — see if GC cycle rate is elevated (GC CPU overhead).
2. Check **HTTP Traffic** — a specific route may be doing expensive computation.
3. Check goroutine count — a goroutine leak can cause CPU to rise as background work accumulates.

---

### `CriticalCpuUsage` / `ecom-alert-critical-cpu`

| Property   | Value                                      |
| ---------- | ------------------------------------------ |
| Severity   | `critical`                                 |
| Condition  | Process CPU > 95% for 2 minutes            |
| Expression | `ecom_engine_process_cpu_percent > 95`     |
| Files      | `alerting-rules.yml`, `runtime-alerts.yml` |

**What it means:** The process is CPU-saturated. Request handling is degraded. Service may become unresponsive.

**Response:**

1. Scale horizontally (add instances) immediately to shed load.
2. Enable CPU profiling (`PPROF_ENABLED=true` if available) and capture a profile.
3. Investigate after scaling — do not wait for the root cause before scaling.

---

### `HighMemoryRSS` / `ecom-alert-high-memory-rss`

| Property   | Value                                               |
| ---------- | --------------------------------------------------- |
| Severity   | `warning`                                           |
| Condition  | Process RSS > 1 GiB for 10 minutes                  |
| Expression | `ecom_engine_process_memory_rss_bytes > 1073741824` |
| Files      | `alerting-rules.yml`, `runtime-alerts.yml`          |

**What it means:** The process is using more than 1 GiB of physical memory. This can be caused by a memory leak, an oversized in-memory cache, or large request payloads being held in memory.

**Response:**

1. Check **Runtime & Process** — is Heap Alloc also growing, or is RSS growing independently?
2. If heap is stable but RSS is rising: memory is held outside Go's allocator (rare — check for large CGo allocations or `mmap` usage).
3. If heap is also growing: check the L1 cache fill ratio — reduce `CACHE_MEMORY_MAX_ITEMS` if it is near 100%.
4. Check goroutine count — a goroutine leak accumulates stack memory.

---

### `GoroutineLeak` / `ecom-alert-goroutine-leak`

| Property   | Value                                      |
| ---------- | ------------------------------------------ |
| Severity   | `warning`                                  |
| Condition  | Goroutine count > 500 for 15 minutes       |
| Expression | `ecom_engine_runtime_goroutines > 500`     |
| Files      | `alerting-rules.yml`, `runtime-alerts.yml` |

**What it means:** More than 500 goroutines are live. If the count is growing monotonically this is almost certainly a goroutine leak.

**Response:**

1. Open **Runtime & Process** — confirm the count is growing, not stable at 500+.
2. A goroutine profile (`/debug/pprof/goroutine`) shows exactly where goroutines are blocked.
3. Common causes: unclosed response bodies, missing `cancel()` calls on contexts, stuck channel receives, or DB/Redis connections that never time out.

---

### `HighGCPauseRate` / `ecom-alert-high-gc-pause`

| Property   | Value                                                 |
| ---------- | ----------------------------------------------------- |
| Severity   | `warning`                                             |
| Condition  | GC pause rate > 50ms/s for 5 minutes                  |
| Expression | `job:ecom_engine_runtime_gc_pause_rate:rate1m > 0.05` |
| Files      | `alerting-rules.yml`, `runtime-alerts.yml`            |

**What it means:** The Go garbage collector is consuming 50+ milliseconds of stop-the-world pause time every second. This directly inflates p99 latency — every request occasionally pauses while GC runs.

**Response:**

1. Correlate with `HighP99Latency` — if both fire together, GC is the likely cause.
2. Check **Runtime & Process** GC Activity panel — high cycle count with large heap means GC runs frequently.
3. Reduce heap allocation rate: profile with `GODEBUG=gccheckmark=1` or heap profiling to find high-allocation code paths.
4. Consider tuning `GOGC` to a higher value (less frequent GC, larger pauses but less total GC work) or using `runtime/debug.SetMemoryLimit`.

---

## Grafana Log Alerts

These alerts are sourced from Loki and are only available in Grafana unified alerting (not in Prometheus).

### `ErrorLogSpike`

| Property  | Value                             |
| --------- | --------------------------------- |
| Severity  | `warning`                         |
| Condition | > 50 error log lines in 5 minutes |
| Source    | Loki                              |

**What it means:** A burst of application errors is being logged. Catch issues that produce errors but don't necessarily return HTTP 5xx (e.g. background job failures, message processing errors).

---

### `PaymentErrorSpike`

| Property  | Value                                     |
| --------- | ----------------------------------------- |
| Severity  | `critical`                                |
| Condition | > 10 payment error log lines in 5 minutes |
| Source    | Loki                                      |

**What it means:** Payment processing is failing. Revenue impact is likely.

**Response:** Open **Logs** filtered to `error` + `payment`. Check external payment provider status page.

---

### `AuthFailureSpike`

| Property  | Value                                    |
| --------- | ---------------------------------------- |
| Severity  | `warning`                                |
| Condition | > 30 auth failure log lines in 5 minutes |
| Source    | Loki                                     |

**What it means:** A large number of authentication failures. Could indicate a credential-stuffing or brute-force attack.

**Response:** Open **Security** dashboard. Check for repeated failures from specific IPs or user accounts. Consider temporary IP blocking at the load balancer.

---

## Alert Summary Table

| Alert                      | Severity | Threshold                 | `for` |
| -------------------------- | -------- | ------------------------- | ----- |
| `HighHttpErrorRate`        | critical | 5xx > 5%                  | 5m    |
| `NoHttpTraffic`            | critical | rate = 0                  | 5m    |
| `HighInFlightRequests`     | warning  | in-flight > 100           | 2m    |
| `HighP99Latency`           | warning  | p99 > 2s                  | 5m    |
| `DbPoolExhausted`          | critical | utilization > 90%         | 2m    |
| `DbPoolEmptyAcquires`      | warning  | empty-acquires > 0.5/s    | 3m    |
| `LowCacheHitRate`          | warning  | Redis hit rate &lt; 50%   | 10m   |
| `RedisPoolTimeouts`        | warning  | timeout rate > 0.1/s      | 5m    |
| `RateLimitBackendFallback` | warning  | any fallback              | 0m    |
| `HighCpuUsage`             | warning  | CPU > 80%                 | 5m    |
| `CriticalCpuUsage`         | critical | CPU > 95%                 | 2m    |
| `HighMemoryRSS`            | warning  | RSS > 1 GiB               | 10m   |
| `GoroutineLeak`            | warning  | goroutines > 500          | 15m   |
| `HighGCPauseRate`          | warning  | GC pause > 50ms/s         | 5m    |
| `ErrorLogSpike`            | warning  | > 50 errors in 5m         | —     |
| `PaymentErrorSpike`        | critical | > 10 payment errors in 5m | —     |
| `AuthFailureSpike`         | warning  | > 30 auth failures in 5m  | —     |
