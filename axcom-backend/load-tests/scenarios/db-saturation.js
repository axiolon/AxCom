/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 *
 * DATABASE SATURATION TEST
 * ========================
 * Pushes the database connection pool to exhaustion by hammering
 * multiple DB-heavy endpoints with zero think time.
 *
 * The loadtest Docker config uses max_pool_size: 50 and
 * pool_acquire_timeout: 3s. When all 50 connections are busy,
 * new requests wait up to 3s for a connection — then fail.
 *
 * Works against both MongoDB and PostgreSQL backends. This test
 * deliberately combines reads AND writes to create maximum contention
 * on the connection pool and database-level locks.
 *
 * Usage:
 *   k6 run load-tests/scenarios/db-saturation.js
 *   k6 run -e BASE_URL=http://localhost:8080 -e VUS=300 load-tests/scenarios/db-saturation.js
 *
 * Watch for:
 *   - pool_timeout_errors counter climbing     → pool exhaustion
 *   - http_req_duration suddenly spiking       → connections queuing
 *   - 503/500 errors appearing                 → server can't get DB conn
 */

import { check, group, sleep } from 'k6';
import http from 'k6/http';
import { Counter, Trend } from 'k6/metrics';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.3/index.js';

// Custom metrics
const poolTimeoutErrors = new Counter('pool_timeout_errors');
const serverErrors = new Counter('server_5xx_errors');
const slowQueries = new Counter('slow_queries_over_1s');
const dbReadLatency = new Trend('db_read_latency', true);
const dbWriteLatency = new Trend('db_write_latency', true);

const TARGET_VUS = parseInt(__ENV.VUS || '200');

export const options = {
  scenarios: {
    db_saturation: {
      executor: 'constant-vus',
      vus: TARGET_VUS,
      duration: '5m',
    },
  },
  thresholds: {
    // Soft — observe where they cross
    http_req_failed:   [{ threshold: 'rate<0.15', abortOnFail: false }],
    http_req_duration: [{ threshold: 'p(95)<3000', abortOnFail: false }],
    // Hard abort: if > 50% fail, the DB has fully collapsed
    http_req_failed:   [{ threshold: 'rate<0.50', abortOnFail: true, delayAbortEval: '20s' }],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Each VU registers once and reuses the token
let userToken = null;
let userRegistered = false;

function ensureAuth() {
  if (userRegistered) return userToken;

  const uniq = `dbsat_${__VU}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;
  const email = `${uniq}@loadtest.local`;
  const password = `Pass${uniq}!1`;

  const regRes = http.post(`${BASE_URL}/api/auth/register`,
    JSON.stringify({ email, password, role: 'customer' }),
    { headers: { 'Content-Type': 'application/json' }, timeout: '10s', tags: { name: 'POST /api/auth/register' } }
  );

  if (regRes.status === 200) {
    const loginRes = http.post(`${BASE_URL}/api/auth/login`,
      JSON.stringify({ email, password }),
      { headers: { 'Content-Type': 'application/json' }, timeout: '10s', tags: { name: 'POST /api/auth/login' } }
    );
    if (loginRes.status === 200) {
      try {
        userToken = loginRes.json().data.access_token;
      } catch (_) {}
    }
  }

  userRegistered = true;
  return userToken;
}

function headers(token) {
  const h = { 'Content-Type': 'application/json', 'Accept': 'application/json' };
  if (token) h['Authorization'] = `Bearer ${token}`;
  return h;
}

export default function () {
  const token = ensureAuth();

  // Cycle through 5 different DB-heavy operations — no sleep between them
  const ops = [
    { weight: 0.25, fn: readProducts },
    { weight: 0.45, fn: readProductDetail },
    { weight: 0.60, fn: readCategories },
    { weight: 0.80, fn: readCart },
    { weight: 1.00, fn: writeCart },
  ];

  const rand = Math.random();
  for (const op of ops) {
    if (rand < op.weight) {
      op.fn(token);
      break;
    }
  }

  // ZERO think time — maximum DB pressure
}

function readProducts(token) {
  group('db-read: products', function () {
    const res = http.get(`${BASE_URL}/api/products`, {
      headers: headers(),
      timeout: '10s',
      tags: { name: 'GET /api/products' },
    });
    dbReadLatency.add(res.timings.duration);
    trackErrors(res);
    check(res, { 'products 200': (r) => r.status === 200 });
  });
}

function readProductDetail(token) {
  group('db-read: product detail', function () {
    // First get product list to pick a real ID
    const listRes = http.get(`${BASE_URL}/api/products`, {
      headers: headers(),
      timeout: '10s',
      tags: { name: 'GET /api/products' },
    });

    if (listRes.status === 200) {
      try {
        const products = listRes.json().data;
        if (products && products.length > 0) {
          const product = products[Math.floor(Math.random() * products.length)];
          const detailRes = http.get(`${BASE_URL}/api/products/${product.id}`, {
            headers: headers(),
            timeout: '10s',
            tags: { name: 'GET /api/products/:id' },
          });
          dbReadLatency.add(detailRes.timings.duration);
          trackErrors(detailRes);
          check(detailRes, { 'product detail 200': (r) => r.status === 200 });
        }
      } catch (_) {}
    }
  });
}

function readCategories(token) {
  group('db-read: categories', function () {
    const res = http.get(`${BASE_URL}/api/categories`, {
      headers: headers(),
      timeout: '10s',
      tags: { name: 'GET /api/categories' },
    });
    dbReadLatency.add(res.timings.duration);
    trackErrors(res);
    check(res, { 'categories 200': (r) => r.status === 200 });
  });
}

function readCart(token) {
  if (!token) return readProducts(token); // fallback if auth failed

  group('db-read: cart', function () {
    const res = http.get(`${BASE_URL}/api/cart`, {
      headers: headers(token),
      timeout: '10s',
      tags: { name: 'GET /api/cart' },
    });
    dbReadLatency.add(res.timings.duration);
    trackErrors(res);
    check(res, { 'cart read ok': (r) => r.status === 200 || r.status === 404 });
  });
}

function writeCart(token) {
  if (!token) return readProducts(token); // fallback if auth failed

  group('db-write: add to cart', function () {
    // Get a real variant ID
    const listRes = http.get(`${BASE_URL}/api/products`, {
      headers: headers(),
      timeout: '10s',
      tags: { name: 'GET /api/products' },
    });

    if (listRes.status === 200) {
      try {
        const products = listRes.json().data;
        if (products && products.length > 0) {
          const product = products[Math.floor(Math.random() * products.length)];
          if (product.variants && product.variants.length > 0) {
            const variant = product.variants[Math.floor(Math.random() * product.variants.length)];
            const addRes = http.post(`${BASE_URL}/api/cart`,
              JSON.stringify({ variant_id: variant.id, quantity: 1 }),
              {
                headers: headers(token),
                timeout: '10s',
                tags: { name: 'POST /api/cart' },
              }
            );
            dbWriteLatency.add(addRes.timings.duration);
            trackErrors(addRes);
            check(addRes, { 'cart write ok': (r) => r.status === 200 || r.status === 400 });
          }
        }
      } catch (_) {}
    }
  });
}

function trackErrors(res) {
  if (res.timings.duration > 1000) {
    slowQueries.add(1);
  }
  if (res.status >= 500) {
    serverErrors.add(1);
  }
  // Pool timeout typically surfaces as 503 or connection-related errors
  if (res.status === 503 || (res.error && res.error.includes('timeout'))) {
    poolTimeoutErrors.add(1);
  }
}

export function handleSummary(data) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').split('T')[0];

  const p95 = data.metrics.http_req_duration ? data.metrics.http_req_duration.values['p(95)'] : 'N/A';
  const failRate = data.metrics.http_req_failed ? data.metrics.http_req_failed.values.rate : 'N/A';
  const poolErrs = data.metrics.pool_timeout_errors ? data.metrics.pool_timeout_errors.values.count : 0;
  const fivexx = data.metrics.server_5xx_errors ? data.metrics.server_5xx_errors.values.count : 0;
  const slow = data.metrics.slow_queries_over_1s ? data.metrics.slow_queries_over_1s.values.count : 0;

  const readP95 = data.metrics.db_read_latency ? data.metrics.db_read_latency.values['p(95)'] : 'N/A';
  const writeP95 = data.metrics.db_write_latency ? data.metrics.db_write_latency.values['p(95)'] : 'N/A';

  console.log('\n╔══════════════════════════════════════════╗');
  console.log('║     DB SATURATION TEST RESULTS           ║');
  console.log('╠══════════════════════════════════════════╣');
  console.log(`║  Overall p95:       ${typeof p95 === 'number' ? p95.toFixed(0) + ' ms' : p95}`);
  console.log(`║  DB read p95:       ${typeof readP95 === 'number' ? readP95.toFixed(0) + ' ms' : readP95}`);
  console.log(`║  DB write p95:      ${typeof writeP95 === 'number' ? writeP95.toFixed(0) + ' ms' : writeP95}`);
  console.log(`║  Error rate:        ${typeof failRate === 'number' ? (failRate * 100).toFixed(2) + '%' : failRate}`);
  console.log(`║  Pool timeouts:     ${poolErrs}`);
  console.log(`║  5xx errors:        ${fivexx}`);
  console.log(`║  Slow queries (>1s): ${slow}`);
  if (poolErrs > 0) {
    console.log('║  ⚠️  Pool timeouts detected!');
    console.log('║     max_pool_size (50) was exhausted');
  }
  console.log('╚══════════════════════════════════════════╝\n');

  return {
    'stdout': textSummary(data, { indent: '  ', enableColors: true }),
    [`load-tests/reports/db-saturation_${timestamp}.json`]: JSON.stringify(data, null, 2),
  };
}
