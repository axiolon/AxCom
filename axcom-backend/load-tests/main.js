/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 */

import { sleep } from 'k6';
import { ApiClient } from './helpers/api.js';
import { getOptions } from './config/profiles.js';
import { generateUserCredentials, generateMockProduct } from './helpers/data.js';

import runBrowsing from './scenarios/browsing.js';
import runCart from './scenarios/cart.js';
import runCheckout from './scenarios/checkout.js';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.3/index.js';

// Resolve configuration options based on target profile
const profile = __ENV.PROFILE || 'smoke';
export const options = getOptions(profile);

// Setup function to seed database if empty
export function setup() {
  const baseUrl = __ENV.BASE_URL || 'http://localhost:8080';
  const api = new ApiClient(baseUrl);

  const stockByProfile = {
    smoke: 100,
    load: 5000,
    stress: 50000,
    spike: 50000,
    soak: 10000,
  };
  const seedStock = stockByProfile[profile] || 5000;

  console.log(`[setup] Starting test setup on target: ${baseUrl}`);
  console.log(`[setup] Checking for existing products...`);

  let products = [];
  try {
    const productsRes = api.getProducts();
    if (productsRes.status === 200 && productsRes.json() && productsRes.json().success) {
      products = productsRes.json().data || [];
    }
  } catch (err) {
    console.log(`[setup] Warning: failed to fetch products: ${err}`);
  }

  // Create an admin user for admin operations (order transition, product creation, etc.)
  const merchantCreds = generateUserCredentials();
  merchantCreds.role = 'admin';
  let merchantToken = '';

  console.log(`[setup] Creating admin user for administrative actions: ${merchantCreds.email}`);
  const regRes = api.register(merchantCreds.email, merchantCreds.password, merchantCreds.role);
  if (regRes.status !== 200) {
    throw new Error(`Setup failed: could not register admin user. Code: ${regRes.status}. Body: ${regRes.body}`);
  }
  sleep(0.6); // rate limit guard

  const loginRes = api.login(merchantCreds.email, merchantCreds.password);
  if (loginRes.status !== 200) {
    throw new Error(`Setup failed: could not login merchant user. Code: ${loginRes.status}`);
  }
  merchantToken = loginRes.json().data.access_token;
  sleep(0.6); // rate limit guard

  // If DB is empty, perform automated data seeding
  if (products.length === 0) {
    console.log(`[setup] No products found. Seeding test catalog...`);

    // Create a seed category
    const categoryName = 'Seeded Loadtest Category';
    const catRes = api.createCategory(merchantToken, categoryName, 'seeded-loadtest');
    if (catRes.status !== 200) {
      throw new Error(`Setup failed: could not create category. Code: ${catRes.status}`);
    }

    const categoryId = catRes.json().data.id;
    sleep(0.6); // rate limit guard

    // Create 3 seed products
    for (let i = 0; i < 3; i++) {
      const productPayload = generateMockProduct(categoryId);
      const prodRes = api.createProduct(merchantToken, productPayload);
      if (prodRes.status !== 200) {
        console.log(`[setup] Warning: failed to seed product ${i}: ${prodRes.body}`);
      } else {
        const createdProduct = prodRes.json().data;
        if (createdProduct && createdProduct.variants) {
          for (let j = 0; j < createdProduct.variants.length; j++) {
            const variant = createdProduct.variants[j];
            const invRes = api.configureInventory(merchantToken, variant.id, 0);
            if (invRes.status === 200) {
              const adjRes = api.adjustInventory(merchantToken, variant.id, seedStock);
              if (adjRes.status !== 200) {
                console.log(`[setup] Warning: failed to adjust inventory for variant ${variant.id}: ${adjRes.body}`);
              }
            } else {
              console.log(`[setup] Warning: failed to configure inventory for variant ${variant.id}: ${invRes.body}`);
            }
            sleep(0.6); // rate limit guard
          }
        }
      }
      sleep(0.6); // rate limit guard
    }

    // Retrieve fresh product list
    const finalProductsRes = api.getProducts();
    if (finalProductsRes.status === 200 && finalProductsRes.json() && finalProductsRes.json().success) {
      products = finalProductsRes.json().data || [];
    }
    console.log(`[setup] Seeding complete. Seeded ${products.length} products.`);
    var activeProducts = products;
  } else {
    var activeProducts = products.slice(0, 5);
    console.log(`[setup] Found ${products.length} existing products. Ensuring variant stock for active ${activeProducts.length} products...`);
    for (let i = 0; i < activeProducts.length; i++) {
      const p = activeProducts[i];
      if (p.variants && p.variants.length > 0) {
        for (let j = 0; j < p.variants.length; j++) {
          const variant = p.variants[j];
          api.configureInventory(merchantToken, variant.id, 0);
          sleep(0.6); // rate limit guard
          api.adjustInventory(merchantToken, variant.id, seedStock);
          sleep(0.6); // rate limit guard
        }
      }
    }
  }

  return {
    products: activeProducts,
    baseUrl: baseUrl,
    merchantToken: merchantToken,
  };
}

// Default VU entrypoint
export default function (setupData) {
  const api = new ApiClient(setupData.baseUrl);
  
  // Allow overriding via environment variables to run a single scenario
  const scenario = __ENV.SCENARIO;

  if (scenario === 'browsing') {
    runBrowsing(api, setupData);
  } else if (scenario === 'cart') {
    runCart(api, setupData);
  } else if (scenario === 'checkout') {
    runCheckout(api, setupData);
  } else {
    // Realistic multi-scenario traffic distribution:
    // - 60% Browsing / Searching
    // - 30% Cart Updates / Shopping
    // - 10% Checkout / Purchasing
    const rand = Math.random();
    if (rand < 0.60) {
      runBrowsing(api, setupData);
    } else if (rand < 0.90) {
      runCart(api, setupData);
    } else {
      runCheckout(api, setupData);
    }
  }
}

export function handleSummary(data) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').split('T')[0];
  return {
    'stdout': textSummary(data, { indent: '  ', enableColors: true }),
    [`load-tests/reports/${profile}_${timestamp}.json`]: JSON.stringify(data, null, 2),
  };
}

export function teardown(data) {
  console.log(`[teardown] Test run complete.`);
  console.log(`[teardown] Profile: ${__ENV.PROFILE || 'smoke'}`);
  console.log(`[teardown] Target: ${data.baseUrl}`);
  console.log(`[teardown] Products used: ${data.products ? data.products.length : 0}`);
  console.log(`[teardown] ⚠️  Note: Test users, carts, and orders created during this run are NOT cleaned up.`);
  console.log(`[teardown] For a clean slate, reset the database before the next run.`);
}
