/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 */

import { check, sleep, group } from 'k6';
import { thinkTime } from '../helpers/data.js';

export default function runBrowsing(api, data) {
  group('browsing', () => {
    // 1. Liveness check
    const healthRes = api.healthz();
    check(healthRes, {
      'healthz status is 200': (r) => r.status === 200,
      'healthz response status is UP': (r) => r.json() && r.json().status === 'UP',
    });

    sleep(thinkTime(0.5, 1.5));

    // 2. Fetch categories
    const categoriesRes = api.getCategories();
    const isCategoriesOk = check(categoriesRes, {
      'categories status is 200': (r) => r.status === 200,
    });

    let categoryId = '';
    if (isCategoriesOk && categoriesRes.json() && categoriesRes.json().success) {
      const categories = categoriesRes.json().data || [];
      if (categories.length > 0) {
        const randomCategory = categories[Math.floor(Math.random() * categories.length)];
        categoryId = randomCategory.id;

        sleep(thinkTime(0.5, 1.5));

        // 2.1 Fetch specific category detail [NEW]
        const catDetailRes = api.getCategory(categoryId);
        check(catDetailRes, {
          'category detail status is 200': (r) => r.status === 200,
          'category detail ID matches': (r) => r.json() && r.json().success && r.json().data.id === categoryId,
        });
      }
    }

    sleep(thinkTime(1, 2));

    // 3. Fetch products list with filtering/searching [NEW]
    let queryParams = '';
    if (categoryId) {
      // 50% chance of adding text search parameter or category constraints
      if (Math.random() < 0.5) {
        queryParams = `?category_id=${categoryId}&price_min=10&price_max=200`;
      } else {
        queryParams = `?category_id=${categoryId}&q=Product`;
      }
    } else {
      queryParams = '?q=Product';
    }

    const productsRes = api.getProducts(queryParams);
    const isProductsOk = check(productsRes, {
      'products list status is 200': (r) => r.status === 200,
    });

    if (isProductsOk && productsRes.json() && productsRes.json().success) {
      const products = productsRes.json().data;
      if (Array.isArray(products) && products.length > 0) {
        // Pick a random product
        const randomIndex = Math.floor(Math.random() * products.length);
        const product = products[randomIndex];
        
        sleep(thinkTime(1, 2));
        
        // 4. Fetch specific product details
        const detailRes = api.getProduct(product.id);
        check(detailRes, {
          'product detail status is 200': (r) => r.status === 200,
          'product detail ID matches': (r) => r.json() && r.json().success && r.json().data.id === product.id,
        });
      }
    }

    sleep(thinkTime(2, 4));
  });
}
