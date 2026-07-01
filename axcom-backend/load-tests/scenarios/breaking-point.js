/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 *
 * Breaking-Point Test
 * -------------------
 * Uses ramping-arrival-rate to drive a fixed RPS regardless of response time.
 * This finds the exact throughput at which the system violates its SLOs.
 *
 * Usage:
 *   k6 run -e BASE_URL=http://localhost:8080 load-tests/scenarios/breaking-point.js
 *
 * Watch for:
 *   - http_req_duration{p(95)} crossing 350ms  → latency SLO breach
 *   - http_req_failed rate crossing 1%          → error SLO breach
 *   - dropped_iterations appearing              → VU pool exhausted (saturation)
 */

import { check, group, sleep } from 'k6';
import { ApiClient } from '../helpers/api.js';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.3/index.js';

export const options = {
  scenarios: {
    break_point: {
      executor: 'ramping-arrival-rate',

      // Starting request rate
      startRate: 10,
      timeUnit: '1s',

      // VU pool – pre-allocate generously, hard cap at your hardware limit
      preAllocatedVUs: 50,
      maxVUs: 300,

      // Each stage holds a target RPS for 2 minutes, giving the system time to stabilise
      stages: [
        { duration: '2m', target: 50  },  // 10  → 50  req/s
        { duration: '2m', target: 100 },  // 50  → 100 req/s
        { duration: '2m', target: 150 },  // 100 → 150 req/s
        { duration: '2m', target: 200 },  // 150 → 200 req/s
        { duration: '2m', target: 250 },  // 200 → 250 req/s
        { duration: '2m', target: 300 },  // 250 → 300 req/s (max)
        { duration: '1m', target: 0   },  // graceful ramp-down
      ],
    },
  },

  thresholds: {
    // Soft limits – let them cross so we can see the breaking point in the report
    http_req_failed:   [{ threshold: 'rate<0.05',  abortOnFail: false }],
    http_req_duration: [{ threshold: 'p(95)<1000', abortOnFail: false }],

    // Hard abort: if error rate hits 20%, the system has collapsed – stop cleanly
    'http_req_failed{scenario:break_point}': [
      { threshold: 'rate<0.20', abortOnFail: true, delayAbortEval: '10s' },
    ],
  },
};

export default function () {
  const baseUrl = __ENV.BASE_URL || 'http://localhost:8080';
  const api = new ApiClient(baseUrl);

  // Focused on the read path (browse) – highest-volume, most representative endpoint
  group('browse products', function () {
    const res = api.getProducts();
    check(res, {
      'status 200':    (r) => r.status === 200,
      'has data':      (r) => {
        try { return r.json().data !== undefined; } catch { return false; }
      },
      'p95 < 350 ms':  (r) => r.timings.duration < 350,
    });
  });

  // Minimal think time to allow more realistic concurrency without overwhelming
  sleep(Math.random() * 0.5);
}

export function handleSummary(data) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').split('T')[0];
  return {
    'stdout': textSummary(data, { indent: '  ', enableColors: true }),
    [`load-tests/reports/breaking-point_${timestamp}.json`]: JSON.stringify(data, null, 2),
  };
}
