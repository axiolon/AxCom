/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 *
 * THROUGHPUT (RPS) CEILING TEST
 * =============================
 * Drives a fixed request rate using ramping-arrival-rate, independent of
 * response time. This finds the exact req/s at which the server can no
 * longer keep up.
 *
 * Unlike ramping-vus (which adds concurrency), ramping-arrival-rate
 * guarantees a specific RPS — when the server slows down, k6 allocates
 * more VUs to maintain the rate, until maxVUs is hit and iterations
 * start dropping.
 *
 * Usage:
 *   k6 run load-tests/scenarios/throughput.js
 *   k6 run -e BASE_URL=http://localhost:8080 load-tests/scenarios/throughput.js
 *
 * Watch for:
 *   - dropped_iterations counter > 0           → k6 can't keep up (VU pool exhausted)
 *   - http_req_duration{p(95)} crossing 350ms  → latency SLO breach
 *   - http_req_failed rate crossing 1%          → error SLO breach
 */

import { check, group, sleep } from 'k6';
import http from 'k6/http';
import { Counter, Trend } from 'k6/metrics';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.3/index.js';

// Custom metrics
const droppedRequests = new Counter('custom_dropped_requests');
const serverErrors = new Counter('server_5xx_errors');

export const options = {
  scenarios: {
    throughput_ramp: {
      executor: 'ramping-arrival-rate',

      startRate: 10,       // Begin at 10 req/s
      timeUnit: '1s',

      // VU pool — k6 allocates from this pool to maintain the target rate
      preAllocatedVUs: 100,
      maxVUs: 1000,

      stages: [
        { duration: '1m',  target: 50  },   //  10 →  50 req/s
        { duration: '2m',  target: 100 },   //  50 → 100 req/s
        { duration: '2m',  target: 200 },   // 100 → 200 req/s
        { duration: '2m',  target: 300 },   // 200 → 300 req/s
        { duration: '2m',  target: 400 },   // 300 → 400 req/s
        { duration: '2m',  target: 500 },   // 400 → 500 req/s
        { duration: '1m',  target: 0   },   // ramp down
      ],
    },
  },

  thresholds: {
    // Soft limits — observe where they cross
    http_req_failed:   [{ threshold: 'rate<0.05',   abortOnFail: false }],
    http_req_duration: [{ threshold: 'p(95)<1000',  abortOnFail: false }],

    // Hard abort: 25% errors means system is overwhelmed
    http_req_failed:   [{ threshold: 'rate<0.25',   abortOnFail: true, delayAbortEval: '15s' }],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  // Alternate between health check (ultra-light) and product listing (heavier)
  const endpoints = [
    { url: `${BASE_URL}/healthz`,       name: 'GET /healthz' },
    { url: `${BASE_URL}/api/products`,  name: 'GET /api/products' },
    { url: `${BASE_URL}/api/categories`, name: 'GET /api/categories' },
  ];

  // Pick a weighted endpoint: 20% healthz, 60% products, 20% categories
  const rand = Math.random();
  let endpoint;
  if (rand < 0.20) {
    endpoint = endpoints[0];
  } else if (rand < 0.80) {
    endpoint = endpoints[1];
  } else {
    endpoint = endpoints[2];
  }

  const res = http.get(endpoint.url, {
    headers: { 'Accept': 'application/json' },
    timeout: '10s',
    tags: { name: endpoint.name },
  });

  // Track 5xx errors separately
  if (res.status >= 500) {
    serverErrors.add(1);
  }

  check(res, {
    'status is 2xx':    (r) => r.status >= 200 && r.status < 300,
    'latency < 350ms':  (r) => r.timings.duration < 350,
    'latency < 1s':     (r) => r.timings.duration < 1000,
    'no server error':  (r) => r.status < 500,
  });
}

export function handleSummary(data) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').split('T')[0];

  const p95 = data.metrics.http_req_duration ? data.metrics.http_req_duration.values['p(95)'] : 'N/A';
  const p99 = data.metrics.http_req_duration ? data.metrics.http_req_duration.values['p(99)'] : 'N/A';
  const failRate = data.metrics.http_req_failed ? data.metrics.http_req_failed.values.rate : 'N/A';
  const totalReqs = data.metrics.http_reqs ? data.metrics.http_reqs.values.count : 'N/A';
  const avgRPS = data.metrics.http_reqs ? data.metrics.http_reqs.values.rate : 'N/A';
  const dropped = data.metrics.dropped_iterations ? data.metrics.dropped_iterations.values.count : 0;
  const fivexx = data.metrics.server_5xx_errors ? data.metrics.server_5xx_errors.values.count : 0;

  console.log('\n╔══════════════════════════════════════════╗');
  console.log('║      THROUGHPUT (RPS) TEST RESULTS       ║');
  console.log('╠══════════════════════════════════════════╣');
  console.log(`║  Total requests:    ${totalReqs}`);
  console.log(`║  Avg RPS:           ${typeof avgRPS === 'number' ? avgRPS.toFixed(1) : avgRPS}`);
  console.log(`║  p95 latency:       ${typeof p95 === 'number' ? p95.toFixed(0) + ' ms' : p95}`);
  console.log(`║  p99 latency:       ${typeof p99 === 'number' ? p99.toFixed(0) + ' ms' : p99}`);
  console.log(`║  Error rate:        ${typeof failRate === 'number' ? (failRate * 100).toFixed(2) + '%' : failRate}`);
  console.log(`║  5xx errors:        ${fivexx}`);
  console.log(`║  Dropped iters:     ${dropped}`);
  if (dropped > 0) {
    console.log('║  ⚠️  Dropped iterations = k6 VU pool exhausted');
    console.log('║     Server cannot handle this request rate');
  }
  console.log('╚══════════════════════════════════════════╝\n');

  return {
    'stdout': textSummary(data, { indent: '  ', enableColors: true }),
    [`load-tests/reports/throughput_${timestamp}.json`]: JSON.stringify(data, null, 2),
  };
}
