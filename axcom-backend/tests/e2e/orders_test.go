// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ecom-engine/tests/e2e/testutil"
)

// orderCollections are the collections touched by order tests.
var orderCollections = []string{
	"users", "refresh_tokens", "products", "categories",
	"stocks", "carts", "orders",
}

// setupOrderFixtures seeds catalog + inventory + users for order tests.
// Returns (adminToken, customerToken, variantID, price).
func setupOrderFixtures(t *testing.T) (adminToken, customerToken, variantID string, price float64) {
	t.Helper()

	harness.SeedUser(t, "order_admin@example.com", "Admin123!", "admin")
	adminToken, _ = harness.LoginAs(t, "order_admin@example.com", "Admin123!")

	// Category
	catResp := harness.Do(t, http.MethodPost, "/api/categories", map[string]string{
		"name": "Order Test Category",
		"slug": "order-test",
	}, adminToken)
	require.Equal(t, http.StatusOK, catResp.StatusCode)
	var catBody struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	testutil.Decode(t, catResp, &catBody)

	// Product
	price = 49.99
	prodResp := harness.Do(t, http.MethodPost, "/api/products", map[string]interface{}{
		"name":        "Order Widget",
		"category_id": catBody.Data.ID,
		"variants": []map[string]interface{}{
			{"sku": "ORD-W-001", "name": "Default", "price": price},
		},
	}, adminToken)
	require.Equal(t, http.StatusOK, prodResp.StatusCode)
	var prodBody struct {
		Data struct {
			Variants []struct{ ID string `json:"id"` } `json:"variants"`
		} `json:"data"`
	}
	testutil.Decode(t, prodResp, &prodBody)
	require.Len(t, prodBody.Data.Variants, 1)
	variantID = prodBody.Data.Variants[0].ID

	// Inventory
	stockResp := harness.Do(t, http.MethodPost, "/api/inventory/update", map[string]interface{}{
		"variant_id": variantID,
		"quantity":   100,
	}, adminToken)
	require.Equal(t, http.StatusOK, stockResp.StatusCode)
	stockResp.Body.Close()

	// Customer
	reg := harness.Do(t, http.MethodPost, "/api/auth/register", map[string]string{
		"email":    "order_customer@example.com",
		"password": "Password123!",
	}, "")
	require.Equal(t, http.StatusOK, reg.StatusCode)
	reg.Body.Close()
	customerToken, _ = harness.LoginAs(t, "order_customer@example.com", "Password123!")

	return
}

func TestOrders_CustomerFlow(t *testing.T) {
	harness.Truncate(t, orderCollections...)
	adminToken, custToken, variantID, price := setupOrderFixtures(t)
	_ = adminToken

	var orderID string

	t.Run("unauthenticated create returns 401", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/orders", map[string]interface{}{
			"items": []map[string]interface{}{
				{"variant_id": variantID, "quantity": 1, "price": price},
			},
		}, "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("customer creates order", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/orders", map[string]interface{}{
			"items": []map[string]interface{}{
				{"variant_id": variantID, "quantity": 2, "price": price},
			},
		}, custToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				ID     string  `json:"id"`
				Status string  `json:"status"`
				Total  float64 `json:"total"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.NotEmpty(t, body.Data.ID)
		assert.Equal(t, "pending", body.Data.Status)
		assert.InDelta(t, price*2, body.Data.Total, 0.01)
		orderID = body.Data.ID
	})

	t.Run("customer lists their orders", func(t *testing.T) {
		require.NotEmpty(t, orderID, "depends on 'customer creates order'")

		resp := harness.Do(t, http.MethodGet, "/api/orders", nil, custToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				Orders []struct{ ID string `json:"id"` } `json:"orders"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		require.Len(t, body.Data.Orders, 1)
		assert.Equal(t, orderID, body.Data.Orders[0].ID)
	})

	t.Run("customer gets order by ID", func(t *testing.T) {
		require.NotEmpty(t, orderID, "depends on 'customer creates order'")

		resp := harness.Do(t, http.MethodGet, "/api/orders/"+orderID, nil, custToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Equal(t, orderID, body.Data.ID)
	})

	t.Run("customer cancels order", func(t *testing.T) {
		require.NotEmpty(t, orderID, "depends on 'customer creates order'")

		resp := harness.Do(t, http.MethodPost, "/api/orders/"+orderID+"/cancel", nil, custToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// Verify status
		getResp := harness.Do(t, http.MethodGet, "/api/orders/"+orderID, nil, custToken)
		require.Equal(t, http.StatusOK, getResp.StatusCode)

		var body struct {
			Data struct{ Status string `json:"status"` } `json:"data"`
		}
		testutil.Decode(t, getResp, &body)
		assert.Equal(t, "canceled", body.Data.Status)
	})
}

func TestOrders_GuestCheckout(t *testing.T) {
	harness.Truncate(t, orderCollections...)
	adminToken, _, variantID, price := setupOrderFixtures(t)
	_ = adminToken

	t.Run("guest creates order", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/orders/guest", map[string]interface{}{
			"guest_info": map[string]string{
				"name":           "Jane Doe",
				"email":          "jane@example.com",
				"contact_number": "+1234567890",
			},
			"items": []map[string]interface{}{
				{"variant_id": variantID, "quantity": 1, "price": price},
			},
		}, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				OrderID string `json:"order_id"`
				Status  string `json:"status"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.NotEmpty(t, body.Data.OrderID)
		assert.Equal(t, "pending", body.Data.Status)
	})

	t.Run("guest order requires guest info", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/orders/guest", map[string]interface{}{
			"items": []map[string]interface{}{
				{"variant_id": variantID, "quantity": 1, "price": price},
			},
		}, "")
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestOrders_AdminManagement(t *testing.T) {
	harness.Truncate(t, orderCollections...)
	adminToken, custToken, variantID, price := setupOrderFixtures(t)

	// Customer creates an order for admin to manage
	createResp := harness.Do(t, http.MethodPost, "/api/orders", map[string]interface{}{
		"items": []map[string]interface{}{
			{"variant_id": variantID, "quantity": 1, "price": price},
		},
	}, custToken)
	require.Equal(t, http.StatusOK, createResp.StatusCode)
	var createBody struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	testutil.Decode(t, createResp, &createBody)
	orderID := createBody.Data.ID

	t.Run("admin lists all orders", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/admin/orders", nil, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				Orders []struct{ ID string `json:"id"` } `json:"orders"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		require.NotEmpty(t, body.Data.Orders)
	})

	t.Run("admin gets order by ID", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/admin/orders/"+orderID, nil, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				ID         string `json:"id"`
				CustomerID string `json:"customer_id"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Equal(t, orderID, body.Data.ID)
		assert.NotEmpty(t, body.Data.CustomerID)
	})

	t.Run("admin transitions order status", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/admin/orders/"+orderID+"/transition", map[string]string{
			"action": "pay",
		}, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("non-admin cannot access admin orders", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/admin/orders", nil, custToken)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		resp.Body.Close()
	})
}
