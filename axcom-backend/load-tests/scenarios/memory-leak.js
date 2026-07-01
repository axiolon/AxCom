/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 *
 * MEMORY LEAK DETECTION TEST
 * ==========================
 * Runs moderate load for an extended period (30 minutes) to detect:
 *   - Memory leaks (goroutine/heap growth over time)
 *   - GC pressure under sustained load
 *   - Connection pool drift (connections not being returned)
 *   - Latency degradation over time
 *
 * Run this alongside `docker stats` to watch the container's MEM USAGE.
 * A healthy Go service should reach a plateau within 5-10 minutes.
 * If memory keeps climbing linearly, there's a leak.
 *
 * Usage:
 *   k6 run load-tests/scenarios/memory-leak.js
 *   k6 run -e DURATION=60m load-tests/scenarios/memory-leak.js
 *
 * Monitor alongside:
 *   docker stats --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"
 */

import { check, group, sleep } from 'k6';
import http from 'k6/http';
import { Counter, Trend } from 'k6/metrics';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.3/index.js';

// Custom metrics to track drift over time
const iterationLatency = new Trend('iteration_total_latency', true);
const healthcheckLatency = new Trend('healthcheck_latency', true);
const memoryPressureErrors = new Counter('memory_pressure_errors');

const DURATION = __ENV.DURATION || '30m';
const TARGET_VUS = parseInt(__ENV.VUS || '50');

export const options = {
  scenarios: {
    sustained_load: {
      executor: 'constant-vus',
      vus: TARGET_VUS,
      duration: DURATION,
    },
  },
  thresholds: {
    // Lenient — we're looking for drift, not absolute numbers
    http_req_failed:   [{ threshold: 'rate<0.05', abortOnFail: false }],
    http_req_duration: [{ threshold: 'p(95)<2000', abortOnFail: false }],
    // Abort if the container is clearly dead
    http_req_failed:   [{ threshold: 'rate<0.30', abortOnFail: true, delayAbortEval: '30s' }],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
let iterationCount = 0;

// Each VU registers once
let userToken = null;
let adminToken = null;
let vuInitialized = false;

function initVU() {
  if (vuInitialized) return;

  const uniq = `memleak_${__VU}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;

  // Register customer
  const email = `${uniq}@loadtest.local`;
  const password = `Pass${uniq}!1`;
  const regRes = http.post(`${BASE_URL}/api/auth/register`,
    JSON.stringify({ email, password, role: 'customer' }),
    { headers: { 'Content-Type': 'application/json' }, timeout: '10s', tags: { name: 'POST /api/auth/register' } }
  );

  if (regRes.status === 200) {
    sleep(0.5);
    const loginRes = http.post(`${BASE_URL}/api/auth/login`,
      JSON.stringify({ email, password }),
      { headers: { 'Content-Type': 'application/json' }, timeout: '10s', tags: { name: 'POST /api/auth/login' } }
    );
    if (loginRes.status === 200) {
      try { userToken = loginRes.json().data.access_token; } catch (_) {}
    }
  }

  vuInitialized = true;
}

function headers(token) {
  const h = { 'Content-Type': 'application/json', 'Accept': 'application/json' };
  if (token) h['Authorization'] = `Bearer ${token}`;
  return h;
}

export default function () {
  initVU();
  iterationCount++;
  const iterStart = Date.now();

  // Weighted scenario distribution (same as production traffic)
  const rand = Math.random();

  if (rand < 0.60) {
    // --- Browsing (60%) ---
    group('sustained: browsing', function () {
      const res = http.get(`${BASE_URL}/api/products`, {
        headers: headers(),
        timeout: '10s',
        tags: { name: 'GET /api/products' },
      });

      check(res, { 'browse ok': (r) => r.status === 200 });
      trackMemoryPressure(res);

      if (res.status === 200) {
        try {
          const products = res.json().data;
          if (products && products.length > 0) {
            const p = products[Math.floor(Math.random() * products.length)];
            const detRes = http.get(`${BASE_URL}/api/products/${p.id}`, {
              headers: headers(),
              timeout: '10s',
              tags: { name: 'GET /api/products/:id' },
            });
            check(detRes, { 'detail ok': (r) => r.status === 200 });
          }
        } catch (_) {}
      }
    });
  } else if (rand < 0.90) {
    // --- Cart (30%) ---
    group('sustained: cart', function () {
      if (!userToken) { readFallback(); return; }

      const listRes = http.get(`${BASE_URL}/api/products`, {
        headers: headers(),
        timeout: '10s',
        tags: { name: 'GET /api/products' },
      });

      if (listRes.status === 200) {
        try {
          const products = listRes.json().data;
          if (products && products.length > 0) {
            const p = products[Math.floor(Math.random() * products.length)];
            if (p.variants && p.variants.length > 0) {
              const v = p.variants[Math.floor(Math.random() * p.variants.length)];
              const addRes = http.post(`${BASE_URL}/api/cart`,
                JSON.stringify({ variant_id: v.id, quantity: 1 }),
                { headers: headers(userToken), timeout: '10s', tags: { name: 'POST /api/cart' } }
              );
              check(addRes, { 'cart add ok': (r) => r.status === 200 || r.status === 400 });
              trackMemoryPressure(addRes);
            }
          }
        } catch (_) {}
      }

      const cartRes = http.get(`${BASE_URL}/api/cart`, {
        headers: headers(userToken),
        timeout: '10s',
        tags: { name: 'GET /api/cart' },
      });
      check(cartRes, { 'cart read ok': (r) => r.status === 200 || r.status === 404 });
    });
  } else {
    // --- Checkout (10%) ---
    group('sustained: checkout', function () {
      if (!userToken) { readFallback(); return; }

      const listRes = http.get(`${BASE_URL}/api/products`, {
        headers: headers(),
        timeout: '10s',
        tags: { name: 'GET /api/products' },
      });

      if (listRes.status === 200) {
        try {
          const products = listRes.json().data;
          if (products && products.length > 0) {
            const p = products[Math.floor(Math.random() * products.length)];
            if (p.variants && p.variants.length > 0) {
              const v = p.variants[Math.floor(Math.random() * p.variants.length)];

              // Clear cart first
              http.del(`${BASE_URL}/api/cart`, null, {
                headers: headers(userToken),
                timeout: '10s',
                tags: { name: 'DELETE /api/cart' },
              });

              // Add to cart
              http.post(`${BASE_URL}/api/cart`,
                JSON.stringify({ variant_id: v.id, quantity: 1 }),
                { headers: headers(userToken), timeout: '10s', tags: { name: 'POST /api/cart' } }
              );

              // Place order
              const orderRes = http.post(`${BASE_URL}/api/orders`,
                JSON.stringify({ items: [{ variant_id: v.id, quantity: 1 }] }),
                { headers: headers(userToken), timeout: '15s', tags: { name: 'POST /api/orders' } }
              );
              check(orderRes, { 'order ok': (r) => r.status === 200 || r.status === 400 });
              trackMemoryPressure(orderRes);
            }
          }
        } catch (_) {}
      }
    });
  }

  // Periodic healthcheck heartbeat (every ~10 iterations)
  if (iterationCount % 10 === 0) {
    const hRes = http.get(`${BASE_URL}/healthz`, { timeout: '5s', tags: { name: 'GET /healthz' } });
    healthcheckLatency.add(hRes.timings.duration);
    check(hRes, { 'healthz ok': (r) => r.status === 200 });
  }

  // Track total iteration time
  iterationLatency.add(Date.now() - iterStart);

  // Realistic think time
  sleep(1 + Math.random() * 2);
}

function readFallback() {
  const res = http.get(`${BASE_URL}/api/products`, {
    headers: headers(),
    timeout: '10s',
    tags: { name: 'GET /api/products' },
  });
  check(res, { 'fallback ok': (r) => r.status === 200 });
}

function trackMemoryPressure(res) {
  // 503 or specific error strings indicate memory/resource pressure
  if (res.status === 503 || res.status === 502 ||
      (res.error && (res.error.includes('connection reset') || res.error.includes('EOF')))) {
    memoryPressureErrors.add(1);
  }
}

export function handleSummary(data) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').split('T')[0];

  const p95 = data.metrics.http_req_duration ? data.metrics.http_req_duration.values['p(95)'] : 'N/A';
  const failRate = data.metrics.http_req_failed ? data.metrics.http_req_failed.values.rate : 'N/A';
  const memErrs = data.metrics.memory_pressure_errors ? data.metrics.memory_pressure_errors.values.count : 0;
  const totalIters = data.metrics.iterations ? data.metrics.iterations.values.count : 'N/A';

  const iterP50 = data.metrics.iteration_total_latency ? data.metrics.iteration_total_latency.values['p(50)'] : 'N/A';
  const iterP95 = data.metrics.iteration_total_latency ? data.metrics.iteration_total_latency.values['p(95)'] : 'N/A';

  console.log('\n╔══════════════════════════════════════════╗');
  console.log('║    MEMORY LEAK DETECTION RESULTS         ║');
  console.log('╠══════════════════════════════════════════╣');
  console.log(`║  Duration:          ${DURATION}`);
  console.log(`║  Total iterations:  ${totalIters}`);
  console.log(`║  HTTP p95:          ${typeof p95 === 'number' ? p95.toFixed(0) + ' ms' : p95}`);
  console.log(`║  Iter p50:          ${typeof iterP50 === 'number' ? iterP50.toFixed(0) + ' ms' : iterP50}`);
  console.log(`║  Iter p95:          ${typeof iterP95 === 'number' ? iterP95.toFixed(0) + ' ms' : iterP95}`);
  console.log(`║  Error rate:        ${typeof failRate === 'number' ? (failRate * 100).toFixed(2) + '%' : failRate}`);
  console.log(`║  Mem pressure errs: ${memErrs}`);
  if (memErrs > 10) {
    console.log('║  ⚠️  High memory pressure detected!');
    console.log('║     Check docker stats output for OOM patterns');
  }
  console.log('║');
  console.log('║  📊 Compare iter_p95 at start vs end of test.');
  console.log('║     If it climbs steadily → possible memory leak.');
  console.log('║     If it plateaus → healthy GC behaviour.');
  console.log('╚══════════════════════════════════════════╝\n');

  return {
    'stdout': textSummary(data, { indent: '  ', enableColors: true }),
    [`load-tests/reports/memory-leak_${timestamp}.json`]: JSON.stringify(data, null, 2),
  };
}
