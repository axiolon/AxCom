---
title: "Thresholds & Reports"
description: "Global SLO thresholds, JSON report output, stock-seeding notes, and CI pipeline integration for k6 load tests."
sidebar_position: 4
---

# Thresholds & Reports

<DocBadge status="under-review" version="v0.1.0-alpha" />

---

## SLO Thresholds

Defined in `load-tests/config/profiles.js` and applied uniformly to every profile:

| Metric | Threshold | Meaning |
|---|---|---|
| `http_req_failed` | `rate < 0.01` | HTTP error rate must stay below 1% |
| `http_req_duration` | `p(95) < 350ms` | 95th-percentile response time under 350 ms |

When either threshold is crossed, k6 exits with code **99**. This makes it directly usable as a CI gate — the pipeline step fails if SLOs are violated.

```js
// config/profiles.js
export const THRESHOLDS = {
  http_req_failed:   ['rate<0.01'],
  http_req_duration: ['p(95)<350'],
};
```

---

## Stock Error Note

`400 Insufficient Stock` responses count as HTTP failures and increment `http_req_failed`. The `setup()` function seeds stock per profile to prevent this:

| Profile | Stock per variant |
|---|---|
| `smoke` | 100 |
| `load` | 5 000 |
| `stress` / `spike` | 50 000 |
| `soak` | 10 000 |

If you see unexpected stock errors, either re-run `setup()` by resetting the database, or manually increase quantities for the affected variants.

---

## Reports

After every run, `handleSummary` automatically writes a timestamped JSON file:

```
load-tests/reports/<profile>_<YYYY-MM-DD>.json
```

The file contains the full k6 metrics object — counters, rates, trends, and histogram buckets — and can be ingested by Grafana, DataDog, or any JSON-aware observability tool.

```js
// main.js — handleSummary
export function handleSummary(data) {
  const timestamp = new Date().toISOString().split('T')[0];
  return {
    'stdout': textSummary(data, { indent: '  ', enableColors: true }),
    [`load-tests/reports/${profile}_${timestamp}.json`]: JSON.stringify(data, null, 2),
  };
}
```

The `reports/` directory is git-ignored. Commit only reports you explicitly want to version.

---

## CI Integration

k6 exits with code `99` when thresholds are crossed. Wire it into your pipeline:

```yaml
# GitHub Actions example
- name: Run load test
  run: k6 run -e PROFILE=load load-tests/main.js
  env:
    BASE_URL: ${{ secrets.STAGING_URL }}
```

The step will fail if either SLO threshold is violated, blocking the merge or deploy.

### Metrics Grouping in Dashboards

Every HTTP call in the test suite is tagged with a `name:` param:

```js
// helpers/api.js example
http.get(`${baseUrl}/api/products`, { tags: { name: 'GET /api/products' } });
```

In k6 Cloud, Grafana, or any Prometheus-compatible backend, filter by this tag to get per-route latency breakdowns instead of aggregate totals.
