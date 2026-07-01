/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 */

import { check, sleep, group } from 'k6';
import { generateUserCredentials, safeJson, thinkTime } from '../helpers/data.js';

export default function runCart(api, setupData) {
  group('cart', () => {
    // 1. Create a unique customer for this VU session to prevent cart state pollution
    const creds = generateUserCredentials();
    const registerRes = api.register(creds.email, creds.password);
    
    const registered = check(registerRes, {
      'cart user register status is 200': (r) => r.status === 200,
    });

    if (!registered) {
      sleep(2);
      return;
    }

    sleep(0.6); // rate limit guard

    // 2. Login to get token
    const loginRes = api.login(creds.email, creds.password);
    const loggedIn = check(loginRes, {
      'cart user login status is 200': (r) => r.status === 200,
      'cart user has access token': (r) => {
        const body = safeJson(r);
        return body && body.success && body.data && body.data.access_token !== undefined;
      },
    });

    if (!loggedIn) {
      sleep(2);
      return;
    }

    const loginBody = safeJson(loginRes);
    const token = loginBody && loginBody.data ? loginBody.data.access_token : null;
    if (!token) {
      sleep(2);
      return;
    }
    sleep(thinkTime(0.5, 1.5));

    // 3. Get initial cart (should be empty)
    const cartRes = api.getCart(token);
    check(cartRes, {
      'get cart status is 200': (r) => r.status === 200,
    });

    // 3.1 Check initial cart count (should be 0) [NEW]
    const countRes = api.getCartCount(token);
    check(countRes, {
      'get cart count status is 200': (r) => r.status === 200,
      'initial cart count is 0': (r) => {
        const body = safeJson(r);
        return body && body.success && body.data && body.data.count === 0;
      },
    });

    // Pick a random product/variant from setupData
    if (setupData && setupData.products && setupData.products.length > 0) {
      const randomProduct = setupData.products[Math.floor(Math.random() * setupData.products.length)];
      if (randomProduct.variants && randomProduct.variants.length > 0) {
        const variant = randomProduct.variants[Math.floor(Math.random() * randomProduct.variants.length)];

        sleep(thinkTime(1, 2));

        // 4. Add item to cart
        const addRes = api.addToCart(token, variant.id, 1);
        const added = check(addRes, {
          'add to cart status is 200': (r) => r.status === 200,
        });

        if (added) {
          sleep(thinkTime(1, 2));

          // 4.1 Check cart count after adding (should be > 0 under concurrent load)
          const countRes2 = api.getCartCount(token);
          check(countRes2, {
            'cart count after add is > 0': (r) => {
              const body = safeJson(r);
              return body && body.success && body.data && body.data.count > 0;
            },
          });

          // 5. Update cart item quantity
          const updateRes = api.updateCart(token, variant.id, 3);
          const updated = check(updateRes, {
            'update cart status is 200': (r) => r.status === 200,
          });

          if (updated) {
            sleep(thinkTime(1, 2));

            // 5.1 Check cart count after updating (should be > 0 under concurrent load)
            const countRes3 = api.getCartCount(token);
            check(countRes3, {
              'cart count after update is > 0': (r) => {
                const body = safeJson(r);
                return body && body.success && body.data && body.data.count > 0;
              },
            });

            // 5.2 Remove item from cart [NEW]
            const removeRes = api.removeFromCart(token, variant.id);
            const removed = check(removeRes, {
              'remove from cart status is 200': (r) => r.status === 200,
            });

            if (removed) {
              sleep(thinkTime(0.5, 1.5));
              // Verify count is back to 0
              const countRes4 = api.getCartCount(token);
              check(countRes4, {
                'cart count after remove is 0': (r) => {
                  const body = safeJson(r);
                  return body && body.success && body.data && body.data.count === 0;
                },
              });
            }
          }
        }

        // 6. Test clear cart logic [NEW]
        sleep(thinkTime(1, 2));
        // Add item again
        const addRes2 = api.addToCart(token, variant.id, 2);
        if (addRes2.status === 200) {
          sleep(thinkTime(0.5, 1.5));
          // Clear all items
          const clearRes = api.clearCart(token);
          const cleared = check(clearRes, {
            'clear cart status is 200': (r) => r.status === 200,
          });

          if (cleared) {
            sleep(thinkTime(0.5, 1.5));
            const countRes5 = api.getCartCount(token);
            check(countRes5, {
              'cart count after clear is 0': (r) => {
                const body = safeJson(r);
                return body && body.success && body.data && body.data.count === 0;
              },
            });
          }
        }
      }
    }

    sleep(thinkTime(2, 4));
  });
}
