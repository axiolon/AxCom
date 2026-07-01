# Axiolon ecom-engine – Load Test Suite

> **k6-based** performance & durability testing for the ecom-backend API.
> Covers browsing, cart, and full checkout flows against a live local (or remote) server.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Directory Structure](#directory-structure)
3. [Quick Start](#quick-start)
4. [Profiles](#profiles)
5. [Scenarios](#scenarios)
6. [Environment Variables](#environment-variables)
7. [Thresholds & SLOs](#thresholds--slos)
8. [Reports](#reports)
9. [Breaking-Point Testing Strategy](#breaking-point-testing-strategy)
10. [Tips & Known Gotchas](#tips--known-gotchas)

---

## Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| [k6](https://k6.io/docs/getting-started/installation/) | ≥ 0.50 | `winget install k6` or `choco install k6` |
| ecom-backend | running | `go run ./cmd/...` or Docker |
| MongoDB | replica-set | required for transactions when `db.type: mongodb` |
| PostgreSQL | 15+ | alternative to MongoDB; run `./migrate up` before starting |

> Only one database backend is needed. Set `db.type` to `mongodb` or `postgres` in your `config.yaml`.

Start the backend before running any test:

```powershell
# Terminal 1 – backend
cd ecom-backend
go run ./cmd/api/...

# Terminal 2 – run tests
k6 run -e PROFILE=smoke load-tests/main.js
```

---

## Directory Structure

```
load-tests/
├── main.js                   # Entrypoint: setup / default / handleSummary / teardown
├── config/
│   └── profiles.js           # Named load profiles + SLO thresholds
├── helpers/
│   ├── api.js                # Typed HTTP wrappers with URL-name tags
│   └── data.js               # Test-data generators (users, products)
├── scenarios/
│   ├── browsing.js           # Product listing & search
│   ├── cart.js               # Add-to-cart & cart management
│   └── checkout.js           # Full order lifecycle + admin fulfilment
└── reports/                  # Auto-generated JSON reports (gitignored)
```

---

## Quick Start

```powershell
# Smoke test (1 VU, 10 s) – sanity check
k6 run -e PROFILE=smoke load-tests/main.js

# Run a single scenario in smoke mode
k6 run -e PROFILE=smoke -e SCENARIO=browsing  load-tests/main.js
k6 run -e PROFILE=smoke -e SCENARIO=cart      load-tests/main.js
k6 run -e PROFILE=smoke -e SCENARIO=checkout  load-tests/main.js

# Normal load test against a remote server
k6 run -e PROFILE=load -e BASE_URL=https://api.example.com load-tests/main.js

# Stress test (local, skip rate limiting)
k6 run -e PROFILE=stress -e SKIP_RATE_LIMIT=true load-tests/main.js

# Spike test
k6 run -e PROFILE=spike load-tests/main.js

# Soak test (30 min sustained)
k6 run -e PROFILE=soak load-tests/main.js
```

---

## Profiles

Profiles are defined in [`config/profiles.js`](./config/profiles.js).
Pass a profile name via `-e PROFILE=<name>`. An unknown name throws immediately with the list of valid options.

| Profile | Executor | Peak VUs | Total Duration | Purpose |
|---------|----------|----------|----------------|---------|
| `smoke` | constant-vus | **1** | 10 s | Sanity – does the suite wiring work? |
| `load` | ramping-vus | **50** | ~3 m 30 s | Typical production traffic |
| `stress` | ramping-vus | **150** | ~4 m 30 s | Above-normal load – finds bottlenecks |
| `spike` | ramping-vus | **200** | ~1 m 10 s | Sudden traffic burst – tests elasticity |
| `soak` | ramping-vus | **30** | ~31 m | Long-duration – detects memory leaks & drift |

### Stages Detail

<details>
<summary><strong>load</strong></summary>

| Stage | Duration | VUs |
|-------|----------|-----|
| Ramp up | 30 s | 0 → 20 |
| Steady | 1 m | 20 |
| Ramp up | 30 s | 20 → 50 |
| Steady | 1 m | 50 |
| Ramp down | 30 s | 50 → 0 |

</details>

<details>
<summary><strong>stress</strong></summary>

| Stage | Duration | VUs |
|-------|----------|-----|
| Ramp | 30 s | 0 → 50 |
| Steady | 1 m | 50 |
| Ramp | 30 s | 50 → 100 |
| Steady | 1 m | 100 |
| Ramp | 30 s | 100 → 150 |
| Steady | 1 m | 150 |
| Ramp down | 30 s | 150 → 0 |

</details>

<details>
<summary><strong>spike</strong></summary>

| Stage | Duration | VUs |
|-------|----------|-----|
| Baseline | 10 s | 0 → 20 |
| **Spike** | 10 s | 20 → 200 |
| Hold spike | 30 s | 200 |
| Scale back | 10 s | 200 → 20 |
| Ramp down | 10 s | 20 → 0 |

</details>

<details>
<summary><strong>soak</strong></summary>

| Stage | Duration | VUs |
|-------|----------|-----|
| Ramp up | 30 s | 0 → 30 |
| Sustained | **30 m** | 30 |
| Ramp down | 30 s | 30 → 0 |

</details>

---

## Scenarios

All scenarios are wrapped in k6 `group()` blocks for structured metrics.
Each HTTP call carries a `name` tag so metrics are grouped by route in k6 Cloud / dashboards.

### `browsing` – Product Discovery
Simulates a user browsing the catalogue.
- `GET /api/products` — product listing
- `GET /api/products/:id` — product detail
- `GET /api/products/search?q=...` — search
- Think time: **1–3 s** randomised between steps

### `cart` – Shopping
Simulates a user adding items to their cart and reviewing it.
- Registers & logs in a unique user per VU iteration
- `POST /api/cart/items` — add variant (randomly selected)
- `GET /api/cart` — view cart
- Checks: HTTP 200, item count ≥ 1, response body is valid JSON
- Think time: **1–2 s** randomised

### `checkout` – Full Order Lifecycle
The most complex scenario; tests the complete purchase + fulfilment flow.
- Registers & logs in a unique user per VU iteration
- Adds a random variant to cart
- `POST /api/orders` — place order
- A **fresh admin token** is minted inline (prevents expiry during long soak tests)
- Admin transitions the order through `pay → ship → complete`
- Checks: correct status at each transition step
- Think time: **1–3 s** randomised between steps

### Traffic Distribution (Mixed Mode)
When no `SCENARIO` is set, each VU iteration is routed randomly:

| Scenario | Weight |
|----------|--------|
| browsing | 60% |
| cart | 30% |
| checkout | 10% |

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PROFILE` | `smoke` | Load profile name (see [Profiles](#profiles)) |
| `SCENARIO` | *(all)* | Force a single scenario: `browsing`, `cart`, `checkout` |
| `BASE_URL` | `http://localhost:8080` | Target API base URL |
| `SKIP_RATE_LIMIT` | `false` | Set `true` to disable backend rate limiting (local testing) |

---

## Thresholds & SLOs

Defined in `config/profiles.js` and applied to every profile:

| Metric | Threshold | Meaning |
|--------|-----------|---------|
| `http_req_failed` | `rate < 0.01` | Error rate must stay below 1% |
| `http_req_duration` | `p(95) < 350ms` | 95th-percentile latency under 350 ms |

A threshold failure causes k6 to exit with a non-zero code — useful for CI gates.

> **Note on stock errors**: `400 insufficient stock` responses count as failures.
> The `setup()` function seeds adequate inventory per profile (`smoke: 100`, `load: 5 000`,
> `stress / spike: 50 000`, `soak: 10 000`). If you see stock errors, either re-seed or
> increase quantities in `main.js`.

---

## Reports

After every run, `handleSummary` automatically writes a timestamped JSON report:

```
load-tests/reports/<profile>_<YYYY-MM-DD>.json
```

The report contains the full k6 metrics object (counters, rates, trends) and can be ingested by Grafana, DataDog, or any JSON-aware tool.

> Reports directory is git-ignored. Commit only reports you explicitly want to track.

---

## Breaking-Point Testing Strategy

> **Goal**: find the exact VU / RPS count at which the system violates its SLOs
> (`p95 latency > 350 ms` or `error rate > 1%`).

The current profiles top out at **200 VUs** (spike). To systematically find the breaking point, use the following step-load approach.

### Strategy: Step-Load Test with `ramping-arrival-rate`

Unlike `ramping-vus`, the `ramping-arrival-rate` executor drives a **fixed request rate** regardless of response time, making it much better for finding throughput limits.

Create a new file `load-tests/scenarios/breaking-point.js`:

```javascript
// load-tests/scenarios/breaking-point.js
// Run with: k6 run load-tests/scenarios/breaking-point.js
import { check } from 'k6';
import { ApiClient } from '../helpers/api.js';

export const options = {
  scenarios: {
    break_point: {
      executor: 'ramping-arrival-rate',
      startRate: 10,           // Start at 10 req/s
      timeUnit: '1s',
      preAllocatedVUs: 50,
      maxVUs: 300,             // Hard ceiling (your current VU budget)
      stages: [
        { duration: '2m', target: 50  },  // 10 → 50  req/s
        { duration: '2m', target: 100 },  // 50 → 100 req/s
        { duration: '2m', target: 150 },  // 100 → 150 req/s
        { duration: '2m', target: 200 },  // 150 → 200 req/s
        { duration: '2m', target: 250 },  // 200 → 250 req/s
        { duration: '2m', target: 300 },  // 250 → 300 req/s
        { duration: '1m', target: 0   },  // ramp down
      ],
    },
  },
  thresholds: {
    // These are SOFT limits here – we want to SEE them cross, not abort
    http_req_failed:   [{ threshold: 'rate<0.05',   abortOnFail: false }],
    http_req_duration: [{ threshold: 'p(95)<1000',  abortOnFail: false }],
    // Hard abort: if error rate hits 20%, stop immediately
    'http_req_failed{scenario:break_point}': [{ threshold: 'rate<0.20', abortOnFail: true }],
  },
};

export default function () {
  const api = new ApiClient(__ENV.BASE_URL || 'http://localhost:8080');
  const res = api.getProducts();
  check(res, {
    'status 200': (r) => r.status === 200,
    'p95 < 350ms': (r) => r.timings.duration < 350,
  });
}
```

```powershell
# Run the breaking-point test
k6 run -e BASE_URL=http://localhost:8080 load-tests/scenarios/breaking-point.js
```

### What to Look For

| Signal | Interpretation |
|--------|----------------|
| `http_req_duration{p95}` starts climbing past 350 ms | Latency SLO breached — note the RPS at this point |
| `http_req_failed rate` exceeds 1% | Error SLO breached |
| `dropped_iterations` counter appears | k6 cannot allocate VUs fast enough — real system saturation |
| Server CPU/memory spikes in OS monitor | Identify whether bottleneck is CPU, memory, or I/O |

### Recommended Test Sequence

Run tests in this order to avoid burning time:

1. **`smoke`** — confirm suite is wired correctly (30 seconds)
2. **`load`** — baseline: system performs comfortably? (3.5 minutes)
3. **`stress`** — where does latency start climbing? (4.5 minutes)
4. **`breaking-point.js`** — at what exact RPS does SLO break? (13 minutes)
5. **`spike`** — can the system recover from a sudden burst? (1 minute)
6. **`soak`** — does the system degrade over time at 30 VUs? (31 minutes)

### Interpreting the Results

```
                  │ latency
     350ms ───────┼──────────────────────────── SLO line
                  │                 ╱ breaking point
                  │              ╱
                  │          ╱
                  │      ╱
                  │  ╱
                  └─────────────────────────── RPS / VUs
                     ↑ safe zone   ↑ degraded   ↑ failed
```

Once you identify the breaking point RPS, set **80% of that value** as your autoscaling trigger threshold for production.

---

## Resource-Constrained Testing (Limits Testing)

To find the physical boundaries and breaking points of the backend, we run k6 scenarios against a resource-constrained Docker environment. 

### Constrained Architecture

Two compose files are provided — choose the one matching your database backend:

| File | Database | Notes |
|------|----------|-------|
| [`deployments/loadtest/docker-compose.yml`](../deployments/loadtest/docker-compose.yml) | MongoDB | Replica-set initialised automatically by `db-init` container |
| [`deployments/loadtest/docker-compose.postgres.yml`](../deployments/loadtest/docker-compose.postgres.yml) | PostgreSQL | Schema applied automatically by `migrate` container |

Both stacks enforce the same hard resource limits:
- **App Service**: **2.0 CPUs**, **1 GB RAM** (`GOMEMLIMIT=900MiB`, `GOMAXPROCS=2`).
- **DB Service**: **1.0 CPUs**, **512 MB RAM**.
  - MongoDB: WiredTiger cache capped at **256 MB**.
  - PostgreSQL: `shared_buffers=128MB`, `effective_cache_size=256MB`, `max_connections=100`.
- **Config**: Pool size and timeouts are tightened (`max_pool_size: 50`, `pool_acquire_timeout: 3s`) to surface contention and starvation rapidly under load.

### Starting the Constrained Environment

Stop any conflicting containers first, then bring up the stack for your target database:

**MongoDB (default):**
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

### Running Limit-Testing Scenarios
All scenarios are executed from the **host machine** (your PC) so the k6 runner does not steal CPU/RAM resources from the Docker containers under test.

#### 1. Maximum Concurrent Users (`max-concurrency.js`)
Finds the maximum number of simultaneous users the backend can handle before connection refusal, high response times, or container OOM-kills occur.
```powershell
k6 run load-tests/scenarios/max-concurrency.js
```
- **Pattern**: Ramps VUs from `0 → 100 → 300 → 500 → 700 → 1000` (read-only categories and products endpoints).

#### 2. High Request Throughput (`throughput.js`)
Drives a fixed request rate using `ramping-arrival-rate` to find the exact RPS ceiling of the server.
```powershell
k6 run load-tests/scenarios/throughput.js
```
- **Pattern**: Ramps target arrival rate from `10 → 50 → 100 → 200 → 300 → 400 → 500 req/s`.

#### 3. Database Saturation (`db-saturation.js`)
Pushes the database connection pool to exhaustion by hammering DB-heavy endpoints (reads and writes) with constant VUs and zero think time. Works against both MongoDB and PostgreSQL backends.
```powershell
k6 run load-tests/scenarios/db-saturation.js
```
- **Pattern**: 200 constant VUs, zero think-time. Monitors for `pool_timeout_errors`.

#### 4. Memory Leak & GC Pressure (`memory-leak.js`)
Runs moderate load for a sustained 30-minute period to detect Go heap growth, GC thrashing, or memory leaks under pressure.
```powershell
k6 run load-tests/scenarios/memory-leak.js
```
- **Pattern**: 50 constant VUs for 30 minutes. Run `docker stats` concurrently to monitor memory behavior.

#### 5. Race Conditions under Load (`race-conditions.js`)
Verifies atomic stock adjustments and transaction isolation by having 100 VUs buy items from a shared pool of 3 variants with restricted stock (50 units each).
```powershell
k6 run load-tests/scenarios/race-conditions.js
```
- **Pattern**: 100 constant VUs checkout 2 items each simultaneously. Checks that inventory never goes negative.

---

## Reading the Results

When running the limit tests, look for these signals to identify the exact constraint:

| Symptom / Metric | Root Cause | Recommended Action |
|:---|:---|:---|
| `pool_timeout_errors > 0` | DB pool size (50) exhausted. Connection acquisition timed out (>3s). | Increase `max_pool_size` or optimize database queries. |
| `dropped_iterations > 0` | k6 VU pool exhausted. The server cannot process requests fast enough to maintain target RPS. | Scale backend horizontal count or optimize CPU efficiency. |
| `connection_errors > 0` | Server socket queue is full, refusing new TCP connections. | Implement OS-level socket tuning or add reverse proxy queues. |
| Memory usage increases linearly in `docker stats` | Go heap or goroutine leak in backend code. | Analyze heap profiles using `go tool pprof` and fix unclosed resources. |
| Product stock goes negative in `teardown()` of race test | Missing transactional isolation / atomic locks. | MongoDB: use atomic `$inc` with a stock-floor guard. PostgreSQL: use `SELECT ... FOR UPDATE` or a `CHECK` constraint. |

---

## Tips & Known Gotchas

### Local Testing
- Use `SKIP_RATE_LIMIT=true` to bypass backend rate limiting when running high-VU tests locally.
- **MongoDB**: must be running in **replica-set mode** (transactions required for checkout).
- **PostgreSQL**: run `./migrate up` before starting the backend to apply the schema.
- Keep `BASE_URL` pointing to `http://localhost:8080` (no trailing slash).

### Data Isolation
- Every VU registers a unique user (`loadtest_<uuid>@axiolon.test`) — no shared-state conflicts.
- The `setup()` function seeds 3 products with sufficient stock for the selected profile.
- **Leftover data is NOT cleaned up** after a run. Reset the database between full regression runs.

### Admin Token Expiry
- The checkout scenario **mints a fresh admin token per iteration** to survive long soak tests.
- The global `merchantToken` from `setup()` is kept only for reference/setup steps.

### CI Integration
k6 exits with code `99` when thresholds are crossed. Wire it into your pipeline:

```yaml
# GitHub Actions example
- name: Run load test
  run: k6 run -e PROFILE=load load-tests/main.js
  env:
    BASE_URL: ${{ secrets.STAGING_URL }}
```

### Metrics Grouping
All HTTP calls are tagged with a `name:` param (e.g., `name: GET /api/products`).
In k6 Cloud or Grafana, filter by this tag to see per-route latency breakdowns.

### Adding a New Scenario
1. Create `load-tests/scenarios/<name>.js` – export a default function `(api, setupData) => void`
2. Import and call it in `main.js` inside the traffic distribution block
3. Add the scenario name to the `SCENARIO` env-var docs above
