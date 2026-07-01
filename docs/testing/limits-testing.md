---
title: "Limits Testing"
description: "Resource-constrained Docker environment for finding physical boundaries — max concurrency, throughput ceiling, DB pool saturation, memory leak detection, and race-condition verification."
sidebar_position: 6
---

# Limits Testing

<DocBadge status="under-review" version="v0.1.0-alpha" />

Limits testing runs k6 scenarios against a **resource-constrained Docker stack** to find physical boundaries: the exact point at which connection limits, memory, or CPU become the bottleneck. The k6 runner executes on the host machine so it does not steal resources from the containers under test.

---

## Constrained Environment

Two Compose files are provided — choose the one matching your database backend:

| File | Database | Notes |
|---|---|---|
| `deployments/loadtest/docker-compose.yml` | MongoDB | Replica-set initialised automatically by `db-init` container |
| `deployments/loadtest/docker-compose.postgres.yml` | PostgreSQL | Schema applied automatically by `migrate` container |

### Resource Limits

Both stacks enforce hard limits to surface contention rapidly:

| Service | CPU limit | RAM limit | Notes |
|---|---|---|---|
| App (`ecom-backend`) | 2.0 CPUs | 1 GB | `GOMEMLIMIT=900MiB`, `GOMAXPROCS=2` |
| MongoDB | 1.0 CPUs | 512 MB | WiredTiger cache capped at 256 MB |
| PostgreSQL | 1.0 CPUs | 512 MB | `shared_buffers=128MB`, `max_connections=100` |

Connection pool and timeout settings are tightened in the loadtest config: `max_pool_size: 50`, `pool_acquire_timeout: 3s`.

---

## Starting the Constrained Stack

Stop any conflicting containers first, then bring up the stack:

**MongoDB:**

```powershell
cd deployments/loadtest
docker compose up -d --build
docker compose ps
```

**PostgreSQL:**

```powershell
cd deployments/loadtest
docker compose -f docker-compose.postgres.yml up -d --build
docker compose -f docker-compose.postgres.yml ps
```

---

## Limit Scenarios

### 1. Max Concurrency — `max-concurrency.js`

Finds the maximum number of simultaneous users the backend can handle before connection refusal, latency blowout, or OOM-kill.

```powershell
k6 run load-tests/scenarios/max-concurrency.js
```

**VU ramp:** `0 → 100 → 300 → 500 → 700 → 1000` against read-only endpoints (`/api/products`, `/api/categories`).

---

### 2. Throughput Ceiling — `throughput.js`

Drives a fixed arrival rate using `ramping-arrival-rate` to find the exact RPS ceiling of the server.

```powershell
k6 run load-tests/scenarios/throughput.js
```

**Rate ladder (req/s):** `10 → 50 → 100 → 200 → 300 → 400 → 500`

---

### 3. DB Pool Saturation — `db-saturation.js`

Hammers DB-heavy read and write endpoints with 200 constant VUs and zero think-time to exhaust the connection pool and observe `pool_timeout_errors`.

```powershell
k6 run load-tests/scenarios/db-saturation.js
```

Works against both MongoDB and PostgreSQL backends. Watch for `pool_timeout_errors` in the backend logs or k6 metrics.

---

### 4. Memory Leak Detection — `memory-leak.js`

Runs 50 VUs for 30 minutes at moderate load to detect Go heap growth, GC thrashing, or unclosed resource leaks.

```powershell
k6 run load-tests/scenarios/memory-leak.js
```

Run `docker stats` concurrently to monitor memory:

```powershell
# In a second terminal
docker stats --format "table {{.Name}}\t{{.MemUsage}}\t{{.CPUPerc}}"
```

A linearly growing RSS over the 30-minute window indicates a goroutine or heap leak. Use `go tool pprof` on a heap dump to locate the source.

---

### 5. Race Conditions — `race-conditions.js`

Verifies atomic stock adjustments and transaction isolation by having 100 VUs simultaneously purchase from shared variants with restricted stock (50 units each).

```powershell
k6 run load-tests/scenarios/race-conditions.js
```

**Pattern:** 100 constant VUs each checkout 2 items from the same pool. `teardown()` asserts that inventory never went negative.

Failure here means missing transactional isolation:
- **MongoDB**: use atomic `$inc` with a stock-floor guard.
- **PostgreSQL**: use `SELECT ... FOR UPDATE` or a `CHECK` constraint.

---

## Reading the Results

| Symptom | Root cause | Recommended action |
|---|---|---|
| `pool_timeout_errors > 0` | DB pool (50 connections) exhausted; acquisition timed out > 3 s | Increase `max_pool_size` or optimise slow queries |
| `dropped_iterations > 0` | k6 VU pool exhausted — system cannot process requests fast enough to maintain target RPS | Scale backend horizontally or optimise CPU efficiency |
| `connection_errors > 0` | Server socket queue is full, refusing new TCP connections | OS-level socket tuning or add reverse-proxy queue |
| Memory grows linearly in `docker stats` | Go heap or goroutine leak | Analyse heap profiles with `go tool pprof` and fix unclosed resources |
| Stock goes negative in race-conditions `teardown()` | Missing transactional isolation | MongoDB: atomic `$inc` + floor guard; PostgreSQL: `SELECT … FOR UPDATE` or `CHECK` constraint |

---

## Gotchas

- **Rate limiting**: use `-e SKIP_RATE_LIMIT=true` when running high-VU scenarios locally to avoid rate-limiter interference with the measurement.
- **MongoDB replica-set**: must be running in replica-set mode for the checkout scenario (requires transactions).
- **Data isolation**: every VU registers a unique user — no shared-state conflicts. However, leftover data (users, carts, orders) is never cleaned up. Reset the database between full regression runs.
- **Admin token expiry**: the checkout scenario mints a fresh admin token per iteration to survive long soak and race tests.
