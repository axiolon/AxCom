/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 */

export function randomString(length = 8) {
  const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

export function randomNum(min = 1, max = 1000) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

export function generateUserCredentials() {
  const uniq = randomString(6) + randomNum(100, 999);
  return {
    email: `testuser_${uniq}@example.com`,
    password: `Pass${uniq}!`, // satisfies length >= 8, has letters, numbers, and special characters
    role: 'customer',
  };
}

export function generateMockProduct(categoryId) {
  const uniq = randomString(4);
  const price = parseFloat((Math.random() * 90 + 10).toFixed(2));
  return {
    name: `Product ${uniq}`,
    description: `This is a high-performance loaded product of type ${uniq}.`,
    category_id: categoryId,
    variants: [
      {
        sku: `SKU-${uniq}-1`,
        name: `Standard ${uniq}`,
        price: price,
        attributes: {
          size: 'M',
          color: 'Blue',
        },
      },
      {
        sku: `SKU-${uniq}-2`,
        name: `Premium ${uniq}`,
        price: price + 15.00,
        attributes: {
          size: 'L',
          color: 'Gold',
        },
      },
    ],
  };
}

/**
 * Safely parse a k6 HTTP response body as JSON.
 * Returns null if the body is empty, not JSON, or the parse throws.
 */
export function safeJson(response) {
  try {
    if (!response || !response.body) return null;
    return response.json();
  } catch (_) {
    return null;
  }
}

/**
 * Simulate realistic user think-time with random variation.
 * @param {number} min - Minimum seconds to sleep
 * @param {number} max - Maximum seconds to sleep
 */
export function thinkTime(min = 0.5, max = 2.5) {
  return Math.random() * (max - min) + min;
}
