/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 */

import { check, sleep, group } from 'k6';
import { generateUserCredentials, randomString, randomNum, safeJson, thinkTime } from '../helpers/data.js';

export default function runCheckout(api, setupData) {
  group('checkout', () => {
    if (!setupData || !setupData.products || setupData.products.length === 0) {
      sleep(2);
      return;
    }

    // Pick a random product & variant
    const product = setupData.products[Math.floor(Math.random() * setupData.products.length)];
    if (!product.variants || product.variants.length === 0) {
      sleep(2);
      return;
    }

    const variant = product.variants[Math.floor(Math.random() * product.variants.length)];
    const quantity = randomNum(1, 3);
    const price = variant.price;

    // Decide checkout type (40% guest, 60% authenticated customer)
    const isGuest = Math.random() < 0.4;

    if (isGuest) {
      // 1. Guest Checkout
      const uniq = randomString(5);
      const guestInfo = {
        name: `Guest User ${uniq}`,
        email: `guest_${uniq}@example.com`,
        contact_number: `+1555${randomNum(1000000, 9999999)}`,
      };

      const items = [
        {
          variant_id: variant.id,
          quantity: quantity,
          price: price,
        },
      ];

      const guestRes = api.createGuestOrder(guestInfo, items);
      check(guestRes, {
        'guest checkout status is 200': (r) => r.status === 200,
        'guest checkout order ID exists': (r) => {
          const body = safeJson(r);
          return body && body.success && body.data && body.data.order_id !== undefined;
        },
      });
    } else {
      // 2. Authenticated Customer Checkout & Admin Fulfillment
      const creds = generateUserCredentials();
      const registerRes = api.register(creds.email, creds.password);
      
      if (registerRes.status !== 200) {
        sleep(2);
        return;
      }

      sleep(0.6); // rate limit guard

      const loginRes = api.login(creds.email, creds.password);
      if (loginRes.status !== 200) {
        sleep(2);
        return;
      }

      const loginBody = safeJson(loginRes);
      let token = loginBody && loginBody.data ? loginBody.data.access_token : null;
      const refreshToken = loginBody && loginBody.data ? loginBody.data.refresh_token : null;
      if (!token) {
        sleep(2);
        return;
      }
      sleep(thinkTime(0.5, 1.5));

      // 2.1 Refresh Token Flow (20% chance) [NEW]
      if (Math.random() < 0.2 && refreshToken) {
        const refreshRes = api.refresh(refreshToken);
        const refreshed = check(refreshRes, {
          'token refresh status is 200': (r) => r.status === 200,
          'token refresh returns new access token': (r) => {
            const body = safeJson(r);
            return body && body.success && body.data && body.data.access_token !== undefined;
          },
        });
        if (refreshed) {
          const refreshBody = safeJson(refreshRes);
          token = refreshBody && refreshBody.data ? refreshBody.data.access_token : token;
          sleep(0.6); // rate limit guard
        }
      }

      const items = [
        {
          variant_id: variant.id,
          quantity: quantity,
          price: price,
        },
      ];

      // Create customer order
      const orderRes = api.createCustomerOrder(token, items);
      const orderCreated = check(orderRes, {
        'customer checkout status is 200': (r) => r.status === 200,
        'customer checkout order ID exists': (r) => {
          const body = safeJson(r);
          return body && body.success && body.data && body.data.id !== undefined;
        },
      });

      const orderBody = safeJson(orderRes);
      if (orderCreated && orderBody && orderBody.success) {
        const orderId = orderBody.data.id;
        sleep(thinkTime(1, 2));

        // Fetch customer orders list
        const listRes = api.getCustomerOrders(token);
        check(listRes, {
          'get customer orders status is 200': (r) => r.status === 200,
        });

        sleep(thinkTime(0.5, 1.5));

        // Fetch specific order details
        const detailRes = api.getCustomerOrder(token, orderId);
        check(detailRes, {
          'get customer order detail status is 200': (r) => r.status === 200,
        });

        const randAction = Math.random();

        if (randAction < 0.2) {
          // 2.2 Customer Order Cancellation [NEW] (20% chance)
          sleep(thinkTime(1, 2));
          const cancelRes = api.cancelOrder(token, orderId);
          check(cancelRes, {
            'cancel order status is 200': (r) => r.status === 200,
          });
        } else if (randAction < 0.5) {
          // 2.3 Administrative Order Fulfillment Transitions [NEW] (30% chance)
          // Mint a short-lived admin token inline to avoid expiry during long soak runs
          sleep(thinkTime(1, 3));

          const adminCreds = generateUserCredentials();
          adminCreds.role = 'admin';
          const adminRegRes = api.register(adminCreds.email, adminCreds.password, adminCreds.role);
          if (adminRegRes.status !== 200) {
            sleep(2);
            return;
          }
          sleep(0.6); // rate limit guard

          const adminLoginRes = api.login(adminCreds.email, adminCreds.password);
          if (adminLoginRes.status !== 200) {
            sleep(2);
            return;
          }

          const adminBody = safeJson(adminLoginRes);
          const adminToken = adminBody && adminBody.data ? adminBody.data.access_token : null;
          if (!adminToken) {
            sleep(2);
            return;
          }
        
          // Fetch order from admin view
          const adminGetRes = api.adminGetOrder(adminToken, orderId);
          const gotOrder = check(adminGetRes, {
            'admin get order status is 200': (r) => r.status === 200,
          });

          if (gotOrder) {
            // Transition: pending -> paid
            sleep(thinkTime(0.5, 1.5));
            const t1Res = api.adminTransitionOrder(adminToken, orderId, 'pay');
            check(t1Res, {
              'transition to paid status is 200': (r) => r.status === 200,
            });

            // Transition: paid -> shipped
            sleep(thinkTime(0.5, 1.5));
            const t2Res = api.adminTransitionOrder(adminToken, orderId, 'ship');
            check(t2Res, {
              'transition to shipped status is 200': (r) => r.status === 200,
            });

            // Transition: shipped -> completed
            sleep(thinkTime(0.5, 1.5));
            const t3Res = api.adminTransitionOrder(adminToken, orderId, 'complete');
            check(t3Res, {
              'transition to completed status is 200': (r) => r.status === 200,
            });
          }
        }
      }

      // 2.4 Logout flow [NEW]
      sleep(thinkTime(1, 2));
      const logoutRes = api.logout(refreshToken, token);
      check(logoutRes, {
        'logout status is 200': (r) => r.status === 200,
      });
    }

    sleep(thinkTime(2, 4));
  });
}
