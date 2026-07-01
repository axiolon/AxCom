/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 */

import http from 'k6/http';

export class ApiClient {
  constructor(baseUrl) {
    this.baseUrl = baseUrl;
    this.defaultTimeout = '10s';
  }

  getHeaders(token = null) {
    const headers = {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
    return headers;
  }

  getParams(token = null) {
    return {
      headers: this.getHeaders(token),
      timeout: this.defaultTimeout,
    };
  }

  healthz() {
    return http.get(`${this.baseUrl}/healthz`, this.getParams());
  }

  readyz() {
    return http.get(`${this.baseUrl}/readyz`, this.getParams());
  }

  register(email, password, role = 'customer') {
    const payload = JSON.stringify({ email, password, role });
    return http.post(`${this.baseUrl}/api/auth/register`, payload, this.getParams());
  }

  login(email, password) {
    const payload = JSON.stringify({ email, password });
    return http.post(`${this.baseUrl}/api/auth/login`, payload, this.getParams());
  }

  getCategories() {
    return http.get(`${this.baseUrl}/api/categories`, this.getParams());
  }

  createCategory(token, name, slug = '') {
    const payload = JSON.stringify({ name, slug });
    return http.post(`${this.baseUrl}/api/categories`, payload, this.getParams(token));
  }

  getProducts(queryParams = '') {
    return http.get(`${this.baseUrl}/api/products${queryParams}`, this.getParams());
  }

  getProduct(id) {
    return http.get(`${this.baseUrl}/api/products/${id}`, {
      ...this.getParams(),
      tags: { name: 'GET /api/products/:id' },
    });
  }

  createProduct(token, productPayload) {
    return http.post(`${this.baseUrl}/api/products`, JSON.stringify(productPayload), this.getParams(token));
  }

  getCart(token) {
    return http.get(`${this.baseUrl}/api/cart`, this.getParams(token));
  }

  addToCart(token, variantId, quantity) {
    const payload = JSON.stringify({ variant_id: variantId, quantity: quantity });
    return http.post(`${this.baseUrl}/api/cart`, payload, this.getParams(token));
  }

  updateCart(token, variantId, quantity) {
    const payload = JSON.stringify({ quantity: quantity });
    return http.put(`${this.baseUrl}/api/cart/items/${variantId}`, payload, {
      ...this.getParams(token),
      tags: { name: 'PUT /api/cart/items/:variantId' },
    });
  }

  createGuestOrder(guestInfo, items) {
    const payload = JSON.stringify({ guest_info: guestInfo, items: items });
    return http.post(`${this.baseUrl}/api/orders/guest`, payload, this.getParams());
  }

  createCustomerOrder(token, items) {
    const payload = JSON.stringify({ items: items });
    return http.post(`${this.baseUrl}/api/orders`, payload, this.getParams(token));
  }

  getCustomerOrders(token) {
    return http.get(`${this.baseUrl}/api/orders`, this.getParams(token));
  }

  getCustomerOrder(token, id) {
    return http.get(`${this.baseUrl}/api/orders/${id}`, {
      ...this.getParams(token),
      tags: { name: 'GET /api/orders/:id' },
    });
  }

  getCategory(id) {
    return http.get(`${this.baseUrl}/api/categories/${id}`, {
      ...this.getParams(),
      tags: { name: 'GET /api/categories/:id' },
    });
  }

  getCartCount(token) {
    return http.get(`${this.baseUrl}/api/cart/count`, this.getParams(token));
  }

  removeFromCart(token, variantId) {
    return http.del(`${this.baseUrl}/api/cart/items/${variantId}`, null, {
      ...this.getParams(token),
      tags: { name: 'DELETE /api/cart/items/:variantId' },
    });
  }

  clearCart(token) {
    return http.del(`${this.baseUrl}/api/cart`, null, this.getParams(token));
  }

  cancelOrder(token, orderId) {
    return http.post(`${this.baseUrl}/api/orders/${orderId}/cancel`, null, {
      ...this.getParams(token),
      tags: { name: 'POST /api/orders/:id/cancel' },
    });
  }

  adminListOrders(token) {
    return http.get(`${this.baseUrl}/api/admin/orders`, this.getParams(token));
  }

  adminGetOrder(token, orderId) {
    return http.get(`${this.baseUrl}/api/admin/orders/${orderId}`, {
      ...this.getParams(token),
      tags: { name: 'GET /api/admin/orders/:id' },
    });
  }

  adminTransitionOrder(token, orderId, action) {
    const payload = JSON.stringify({ action });
    return http.post(`${this.baseUrl}/api/admin/orders/${orderId}/transition`, payload, {
      ...this.getParams(token),
      tags: { name: 'POST /api/admin/orders/:id/transition' },
    });
  }

  refresh(refreshToken) {
    const payload = JSON.stringify({ refresh_token: refreshToken });
    return http.post(`${this.baseUrl}/api/auth/refresh`, payload, this.getParams());
  }

  logout(refreshToken, token) {
    const payload = JSON.stringify({ refresh_token: refreshToken });
    return http.post(`${this.baseUrl}/api/auth/logout`, payload, this.getParams(token));
  }

  configureInventory(token, variantId, quantity = 100) {
    const payload = JSON.stringify({
      variant_id: variantId,
      location_id: 'default',
      quantity: quantity,
      low_stock_threshold: 5,
      allow_backorders: false,
    });
    return http.post(`${this.baseUrl}/api/inventory/configure`, payload, this.getParams(token));
  }

  adjustInventory(token, variantId, adjustment = 100, reason = 'Seeding load test stock') {
    const payload = JSON.stringify({
      location_id: 'default',
      adjustment: adjustment,
      reason: reason,
    });
    return http.post(`${this.baseUrl}/api/inventory/${variantId}/adjust`, payload, {
      ...this.getParams(token),
      tags: { name: 'POST /api/inventory/:variantId/adjust' },
    });
  }

  getInventory(variantId) {
    return http.get(`${this.baseUrl}/api/inventory/${variantId}`, {
      ...this.getParams(),
      tags: { name: 'GET /api/inventory/:variantId' },
    });
  }
}
