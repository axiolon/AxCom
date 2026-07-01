/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 *
 * RACE CONDITIONS TEST
 * ====================
 * Tests inventory atomic guarantees and transaction isolation under load.
 *
 * 100 VUs concurrently purchase from the same 3 product variants.
 * With limited stock (50 units per variant, total 150 units) and
 * total requested units exceeding available stock (100 VUs * 2 units = 200 units),
 * some checkouts MUST succeed (HTTP 200) and some MUST fail with insufficient stock (HTTP 400).
 *
 * No negative inventory or HTTP 500 errors should occur.
 *
 * Usage:
 *   k6 run load-tests/scenarios/race-conditions.js
 *   k6 run -e BASE_URL=http://localhost:8080 load-tests/scenarios/race-conditions.js
 */

import { check, group, sleep } from 'k6';
import http from 'k6/http';
import { Counter } from 'k6/metrics';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.3/index.js';
import { generateUserCredentials, safeJson } from '../helpers/data.js';
import { ApiClient } from '../helpers/api.js';

// Custom metrics
const successfulOrders = new Counter('successful_orders');
const insufficientStockErrors = new Counter('insufficient_stock_errors');
const server5xxErrors = new Counter('server_5xx_errors');
const otherErrors = new Counter('other_errors');

export const options = {
  scenarios: {
    race_conditions: {
      executor: 'constant-vus',
      vus: 100,
      duration: '1m',
    },
  },
  thresholds: {
    http_req_failed: [{ threshold: 'rate<0.05', abortOnFail: false }],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export function setup() {
  const api = new ApiClient(BASE_URL);

  console.log(`[setup] Starting race conditions setup against: ${BASE_URL}`);

  // Create an admin user to configure inventory
  const adminCreds = generateUserCredentials();
  adminCreds.role = 'admin';
  
  const regRes = api.register(adminCreds.email, adminCreds.password, adminCreds.role);
  if (regRes.status !== 200) {
    throw new Error(`Race setup failed: could not register admin user. Code: ${regRes.status}`);
  }
  sleep(0.5);

  const loginRes = api.login(adminCreds.email, adminCreds.password);
  if (loginRes.status !== 200) {
    throw new Error(`Race setup failed: could not log in admin user. Code: ${loginRes.status}`);
  }
  const adminToken = safeJson(loginRes)?.data?.access_token;
  if (!adminToken) {
    throw new Error('Race setup failed: access token was missing');
  }
  sleep(0.5);

  // Fetch existing products
  let products = [];
  const prodRes = api.getProducts();
  if (prodRes.status === 200 && safeJson(prodRes)?.success) {
    products = safeJson(prodRes).data || [];
  }

  if (products.length === 0) {
    throw new Error('No products found to run race-conditions test. Please run main.js first to seed products.');
  }

  // Pick up to 3 target variants
  const targetVariants = [];
  for (const product of products) {
    if (product.variants && product.variants.length > 0) {
      for (const variant of product.variants) {
        targetVariants.push({
          id: variant.id,
          sku: variant.sku,
          price: variant.price,
        });
        if (targetVariants.length >= 3) break;
      }
    }
    if (targetVariants.length >= 3) break;
  }

  console.log(`[setup] Selected ${targetVariants.length} target variants for concurrent race condition testing:`);
  targetVariants.forEach(v => console.log(`  - Variant: ${v.sku} (${v.id})`));

  // Seed exactly 50 units of stock for each target variant
  for (const v of targetVariants) {
    api.configureInventory(adminToken, v.id, 0);
    sleep(0.5);
    api.adjustInventory(adminToken, v.id, 50, 'Reset stock to 50 for race condition test');
    sleep(0.5);
  }

  return {
    variants: targetVariants,
  };
}

export default function (setupData) {
  const api = new ApiClient(BASE_URL);
  const variants = setupData.variants;

  if (!variants || variants.length === 0) {
    return;
  }

  group('race condition checkout iteration', () => {
    // 1. Register a unique user
    const creds = generateUserCredentials();
    const regRes = api.register(creds.email, creds.password);
    if (regRes.status !== 200) {
      otherErrors.add(1);
      return;
    }
    sleep(0.1);

    // 2. Login
    const loginRes = api.login(creds.email, creds.password);
    if (loginRes.status !== 200) {
      otherErrors.add(1);
      return;
    }
    const token = safeJson(loginRes)?.data?.access_token;
    if (!token) {
      otherErrors.add(1);
      return;
    }

    // 3. Select a target variant based on random choice
    const targetVariant = variants[Math.floor(Math.random() * variants.length)];

    // 4. Add to cart (quantity 2)
    const addRes = api.addToCart(token, targetVariant.id, 2);
    const added = check(addRes, {
      'added to cart status is 200': (r) => r.status === 200,
    });

    if (!added) {
      otherErrors.add(1);
      return;
    }
    sleep(0.1);

    // 5. Checkout
    const orderRes = api.createCustomerOrder(token, [
      { variant_id: targetVariant.id, quantity: 2, price: targetVariant.price }
    ]);

    const orderBody = safeJson(orderRes);
    const status = orderRes.status;

    if (status === 200) {
      successfulOrders.add(1);
    } else if (status === 400 && orderBody?.message && orderBody.message.includes('insufficient stock')) {
      insufficientStockErrors.add(1);
    } else if (status >= 500) {
      server5xxErrors.add(1);
    } else {
      otherErrors.add(1);
    }

    check(orderRes, {
      'order checkout status is 200 or 400': (r) => r.status === 200 || r.status === 400,
      'no server 5xx error': (r) => r.status < 500,
    });

    // Small random think time
    sleep(Math.random() * 0.4 + 0.1);
  });
}

export function teardown(setupData) {
  const api = new ApiClient(BASE_URL);
  console.log('[teardown] Race conditions test completed.');
  console.log('[teardown] Fetching remaining stock counts to verify no double-spend / negative stock...');

  if (!setupData || !setupData.variants) return;

  setupData.variants.forEach(v => {
    const invRes = api.getInventory(v.id);
    if (invRes.status === 200) {
      const invBody = safeJson(invRes);
      const remainingStock = invBody?.data?.quantity;
      console.log(`  - Variant SKU: ${v.sku} (${v.id}): Remaining Stock = ${remainingStock}`);
      if (remainingStock < 0) {
        console.log(`    ❌ ERROR: NEGATIVE STOCK DETECTED FOR SKU ${v.sku}! Race condition guard failed.`);
      } else {
        console.log(`    ✓ Stock value is valid (non-negative).`);
      }
    } else {
      console.log(`  - Variant SKU: ${v.sku} (${v.id}): Failed to fetch inventory status (HTTP ${invRes.status})`);
    }
    sleep(0.2);
  });
}

export function handleSummary(data) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').split('T')[0];

  const p95 = data.metrics.http_req_duration ? data.metrics.http_req_duration.values['p(95)'] : 'N/A';
  const failRate = data.metrics.http_req_failed ? data.metrics.http_req_failed.values.rate : 'N/A';
  const successOrdersCount = data.metrics.successful_orders ? data.metrics.successful_orders.values.count : 0;
  const stockErrorsCount = data.metrics.insufficient_stock_errors ? data.metrics.insufficient_stock_errors.values.count : 0;
  const server5xxCount = data.metrics.server_5xx_errors ? data.metrics.server_5xx_errors.values.count : 0;
  const otherErrorsCount = data.metrics.other_errors ? data.metrics.other_errors.values.count : 0;

  console.log('\n╔══════════════════════════════════════════╗');
  console.log('║        RACE CONDITIONS TEST RESULTS      ║');
  console.log('╠══════════════════════════════════════════╣');
  console.log(`║  Successful orders:   ${successOrdersCount}`);
  console.log(`║  Insufficient stock:  ${stockErrorsCount}`);
  console.log(`║  Server 5xx errors:   ${server5xxCount}`);
  console.log(`║  Other errors:        ${otherErrorsCount}`);
  console.log(`║  Overall p95 latency: ${typeof p95 === 'number' ? p95.toFixed(0) + ' ms' : p95}`);
  console.log(`║  HTTP Failure Rate:   ${typeof failRate === 'number' ? (failRate * 100).toFixed(2) + '%' : failRate}`);
  if (server5xxCount > 0) {
    console.log('║  ⚠️  Server 5xx errors detected! Potential db lockups.');
  }
  console.log('╚══════════════════════════════════════════╝\n');

  return {
    'stdout': textSummary(data, { indent: '  ', enableColors: true }),
    [`load-tests/reports/race-conditions_${timestamp}.json`]: JSON.stringify(data, null, 2),
  };
}
