// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ecom-engine/tests/e2e/testutil"
)

// cartCollections are the collections touched by cart tests.
var cartCollections = []string{"users", "refresh_tokens", "products", "categories", "carts", "stocks"}

// setupCartFixtures seeds an admin, a category, a product with a variant, inventory stock,
// and a customer user. Returns (adminToken, customerToken, variantSKU).
func setupCartFixtures(t *testing.T) (adminToken, customerToken, variantID string) {
	t.Helper()

	// Admin
	harness.SeedUser(t, "cart_admin@example.com", "Admin123!", "admin")
	adminToken, _ = harness.LoginAs(t, "cart_admin@example.com", "Admin123!")

	// Category
	catResp := harness.Do(t, http.MethodPost, "/api/categories", map[string]string{
		"name": "Cart Test Category",
		"slug": "cart-test",
	}, adminToken)
	require.Equal(t, http.StatusOK, catResp.StatusCode)
	var catBody struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	testutil.Decode(t, catResp, &catBody)

	// Product with variant
	prodResp := harness.Do(t, http.MethodPost, "/api/products", map[string]interface{}{
		"name":        "Cart Widget",
		"category_id": catBody.Data.ID,
		"variants": []map[string]interface{}{
			{"sku": "CART-W-001", "name": "Default", "price": 19.99},
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

	// Seed inventory so the cart can validate stock
	stockResp := harness.Do(t, http.MethodPost, "/api/inventory/sync", map[string]interface{}{
		"variant_id": variantID,
		"quantity":   50,
	}, adminToken)
	require.Equal(t, http.StatusOK, stockResp.StatusCode)
	stockResp.Body.Close()

	// Wait for async stock synchronization to the catalog to complete
	time.Sleep(100 * time.Millisecond)

	// Customer
	reg := harness.Do(t, http.MethodPost, "/api/auth/register", map[string]string{
		"email":    "cart_customer@example.com",
		"password": "Password123!",
	}, "")
	require.Equal(t, http.StatusOK, reg.StatusCode)
	reg.Body.Close()
	customerToken, _ = harness.LoginAs(t, "cart_customer@example.com", "Password123!")

	return adminToken, customerToken, variantID
}

func TestCart_AddAndGet(t *testing.T) {
	harness.Truncate(t, cartCollections...)
	_, custToken, variantID := setupCartFixtures(t)

	t.Run("empty cart initially", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/cart", nil, custToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				Items []interface{} `json:"items"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Empty(t, body.Data.Items)
	})

	t.Run("unauthenticated add returns 401", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/cart", map[string]interface{}{
			"variant_id": variantID,
			"quantity":   1,
		}, "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("add item to cart", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/cart", map[string]interface{}{
			"variant_id": variantID,
			"quantity":   2,
		}, custToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				Items []struct {
					VariantID string `json:"variant_id"`
					Quantity  int    `json:"quantity"`
				} `json:"items"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		require.Len(t, body.Data.Items, 1)
		assert.Equal(t, variantID, body.Data.Items[0].VariantID)
		assert.Equal(t, 2, body.Data.Items[0].Quantity)
	})

	t.Run("get cart count", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/cart/count", nil, custToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				Count         int `json:"count"`
				DistinctCount int `json:"distinct_count"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Equal(t, 2, body.Data.Count)
		assert.Equal(t, 1, body.Data.DistinctCount)
	})
}

func TestCart_UpdateAndRemove(t *testing.T) {
	harness.Truncate(t, cartCollections...)
	_, custToken, variantID := setupCartFixtures(t)

	// Add item first
	addResp := harness.Do(t, http.MethodPost, "/api/cart", map[string]interface{}{
		"variant_id": variantID,
		"quantity":   1,
	}, custToken)
	require.Equal(t, http.StatusOK, addResp.StatusCode)
	addResp.Body.Close()

	t.Run("update item quantity", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPut, "/api/cart/items/"+variantID, map[string]interface{}{
			"quantity": 5,
		}, custToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				Items []struct {
					VariantID string `json:"variant_id"`
					Quantity  int    `json:"quantity"`
				} `json:"items"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		require.Len(t, body.Data.Items, 1)
		assert.Equal(t, 5, body.Data.Items[0].Quantity)
	})

	t.Run("remove item from cart", func(t *testing.T) {
		resp := harness.Do(t, http.MethodDelete, "/api/cart/items/"+variantID, nil, custToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				Items []interface{} `json:"items"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Empty(t, body.Data.Items)
	})
}

func TestCart_Clear(t *testing.T) {
	harness.Truncate(t, cartCollections...)
	_, custToken, variantID := setupCartFixtures(t)

	// Add item
	addResp := harness.Do(t, http.MethodPost, "/api/cart", map[string]interface{}{
		"variant_id": variantID,
		"quantity":   3,
	}, custToken)
	require.Equal(t, http.StatusOK, addResp.StatusCode)
	addResp.Body.Close()

	t.Run("clear cart removes all items", func(t *testing.T) {
		resp := harness.Do(t, http.MethodDelete, "/api/cart", nil, custToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// Verify cart is empty
		getResp := harness.Do(t, http.MethodGet, "/api/cart", nil, custToken)
		require.Equal(t, http.StatusOK, getResp.StatusCode)

		var body struct {
			Data struct {
				Items []interface{} `json:"items"`
			} `json:"data"`
		}
		testutil.Decode(t, getResp, &body)
		assert.Empty(t, body.Data.Items)
	})
}
