/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 *
 * MAX CONCURRENCY TEST
 * ====================
 * Finds the maximum number of simultaneous users the system can handle
 * before connections are refused or latency becomes unacceptable.
 *
 * Ramps from 0 → 1000 VUs in stages, holding each level for 2 minutes
 * so metrics stabilise. Uses read-only endpoints to isolate pure
 * concurrency from write contention.
 *
 * Usage:
 *   k6 run load-tests/scenarios/max-concurrency.js
 *   k6 run -e BASE_URL=http://localhost:8080 load-tests/scenarios/max-concurrency.js
 *
 * Watch for:
 *   - http_req_duration{p(95)} crossing 1s    → latency degradation
 *   - http_req_failed rate crossing 1%         → errors appearing
 *   - Container OOM-kill (check docker logs)   → memory ceiling hit
 */

import { check, group, sleep } from 'k6';
import http from 'k6/http';
import { Trend, Counter } from 'k6/metrics';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.3/index.js';

// Custom metrics
const connectionErrors = new Counter('connection_errors');
const timeoutErrors = new Counter('timeout_errors');

export const options = {
  scenarios: {
    max_concurrency: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s',  target: 50  },   // warm-up
        { duration: '2m',   target: 100 },   // baseline
        { duration: '2m',   target: 200 },   // moderate
        { duration: '2m',   target: 300 },   // current ceiling
        { duration: '2m',   target: 500 },   // beyond current limit
        { duration: '2m',   target: 700 },   // heavy
        { duration: '2m',   target: 1000 },  // extreme
        { duration: '1m',   target: 0 },     // ramp down
      ],
      gracefulRampDown: '30s',
    },
  },
  thresholds: {
    // Soft thresholds — let them cross so we see where the break happens
    http_req_failed:   [{ threshold: 'rate<0.10',  abortOnFail: false }],
    http_req_duration: [{ threshold: 'p(95)<2000', abortOnFail: false }],
    // Hard abort: if 40% of requests fail, system has collapsed
    http_req_failed:   [{ threshold: 'rate<0.40',  abortOnFail: true, delayAbortEval: '15s' }],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  group('concurrent read - products', function () {
    const res = http.get(`${BASE_URL}/api/products`, {
      headers: { 'Accept': 'application/json' },
      timeout: '15s',
      tags: { name: 'GET /api/products' },
    });

    // Track specific failure types
    if (res.error) {
      if (res.error.includes('connection refused') || res.error.includes('connection reset')) {
        connectionErrors.add(1);
      }
      if (res.error.includes('timeout') || res.error.includes('i/o timeout')) {
        timeoutErrors.add(1);
      }
    }

    check(res, {
      'status 200':     (r) => r.status === 200,
      'latency < 1s':   (r) => r.timings.duration < 1000,
      'latency < 2s':   (r) => r.timings.duration < 2000,
      'has body':       (r) => r.body && r.body.length > 0,
    });
  });

  group('concurrent read - categories', function () {
    const res = http.get(`${BASE_URL}/api/categories`, {
      headers: { 'Accept': 'application/json' },
      timeout: '15s',
      tags: { name: 'GET /api/categories' },
    });

    check(res, {
      'status 200': (r) => r.status === 200,
    });
  });

  // Minimal think time — we want max concurrency, not realistic pacing
  sleep(Math.random() * 0.3);
}

export function handleSummary(data) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').split('T')[0];

  // Extract key findings
  const p95 = data.metrics.http_req_duration ? data.metrics.http_req_duration.values['p(95)'] : 'N/A';
  const p99 = data.metrics.http_req_duration ? data.metrics.http_req_duration.values['p(99)'] : 'N/A';
  const failRate = data.metrics.http_req_failed ? data.metrics.http_req_failed.values.rate : 'N/A';
  const connErrs = data.metrics.connection_errors ? data.metrics.connection_errors.values.count : 0;
  const timeoutErrs = data.metrics.timeout_errors ? data.metrics.timeout_errors.values.count : 0;

  console.log('\n╔══════════════════════════════════════════╗');
  console.log('║     MAX CONCURRENCY TEST RESULTS         ║');
  console.log('╠══════════════════════════════════════════╣');
  console.log(`║  p95 latency:       ${typeof p95 === 'number' ? p95.toFixed(0) + ' ms' : p95}`);
  console.log(`║  p99 latency:       ${typeof p99 === 'number' ? p99.toFixed(0) + ' ms' : p99}`);
  console.log(`║  Error rate:        ${typeof failRate === 'number' ? (failRate * 100).toFixed(2) + '%' : failRate}`);
  console.log(`║  Connection errors: ${connErrs}`);
  console.log(`║  Timeout errors:    ${timeoutErrs}`);
  console.log('╚══════════════════════════════════════════╝\n');

  return {
    'stdout': textSummary(data, { indent: '  ', enableColors: true }),
    [`load-tests/reports/max-concurrency_${timestamp}.json`]: JSON.stringify(data, null, 2),
  };
}
