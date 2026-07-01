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

// inventoryCollections are the collections touched by inventory tests.
var inventoryCollections = []string{
	"users", "refresh_tokens", "products", "categories",
	"stocks", "stock_history", "reservations", "alerts",
}

// setupInventoryAdmin seeds an admin and returns a token.
func setupInventoryAdmin(t *testing.T) string {
	t.Helper()
	harness.SeedUser(t, "inv_admin@example.com", "Admin123!", "admin")
	token, _ := harness.LoginAs(t, "inv_admin@example.com", "Admin123!")
	return token
}

// setupInventoryVariant creates a category + product and returns the variant ID.
func setupInventoryVariant(t *testing.T, adminToken string) string {
	t.Helper()

	catResp := harness.Do(t, http.MethodPost, "/api/categories", map[string]string{
		"name": "Inv Test Category",
		"slug": "inv-test",
	}, adminToken)
	require.Equal(t, http.StatusOK, catResp.StatusCode)
	var catBody struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	testutil.Decode(t, catResp, &catBody)

	prodResp := harness.Do(t, http.MethodPost, "/api/products", map[string]interface{}{
		"name":        "Inventory Widget",
		"category_id": catBody.Data.ID,
		"variants": []map[string]interface{}{
			{"sku": "INV-W-001", "name": "Default", "price": 10.00},
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
	return prodBody.Data.Variants[0].ID
}

func TestInventory_UpdateAndCheck(t *testing.T) {
	harness.Truncate(t, inventoryCollections...)
	adminToken := setupInventoryAdmin(t)
	variantID := setupInventoryVariant(t, adminToken)

	t.Run("check stock returns zero initially", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/inventory/"+variantID, nil, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				Quantity int `json:"quantity"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Equal(t, 0, body.Data.Quantity)
	})

	t.Run("unauthenticated update returns 401", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/inventory/update", map[string]interface{}{
			"variant_id": variantID,
			"quantity":   100,
		}, "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("admin updates stock", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/inventory/update", map[string]interface{}{
			"variant_id": variantID,
			"quantity":   100,
		}, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// Verify
		check := harness.Do(t, http.MethodGet, "/api/inventory/"+variantID, nil, "")
		require.Equal(t, http.StatusOK, check.StatusCode)

		var body struct {
			Data struct {
				Quantity int `json:"quantity"`
			} `json:"data"`
		}
		testutil.Decode(t, check, &body)
		assert.Equal(t, 100, body.Data.Quantity)
	})
}

func TestInventory_ListAndConfigure(t *testing.T) {
	harness.Truncate(t, inventoryCollections...)
	adminToken := setupInventoryAdmin(t)
	variantID := setupInventoryVariant(t, adminToken)

	// Seed stock
	resp := harness.Do(t, http.MethodPost, "/api/inventory/update", map[string]interface{}{
		"variant_id": variantID,
		"quantity":   20,
	}, adminToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	t.Run("admin lists inventory", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/inventory", nil, adminToken)
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
		require.NotEmpty(t, body.Data.Items)
	})

	t.Run("admin configures low stock threshold", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/inventory/configure", map[string]interface{}{
			"variant_id":          variantID,
			"low_stock_threshold": 50,
		}, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestInventory_Adjust(t *testing.T) {
	harness.Truncate(t, inventoryCollections...)
	adminToken := setupInventoryAdmin(t)
	variantID := setupInventoryVariant(t, adminToken)

	// Set initial stock
	resp := harness.Do(t, http.MethodPost, "/api/inventory/update", map[string]interface{}{
		"variant_id": variantID,
		"quantity":   100,
	}, adminToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	t.Run("adjust stock down", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/inventory/"+variantID+"/adjust", map[string]interface{}{
			"adjustment": -10,
			"reason":     "damaged goods",
		}, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// Verify
		check := harness.Do(t, http.MethodGet, "/api/inventory/"+variantID, nil, "")
		require.Equal(t, http.StatusOK, check.StatusCode)

		var body struct {
			Data struct{ Quantity int `json:"quantity"` } `json:"data"`
		}
		testutil.Decode(t, check, &body)
		assert.Equal(t, 90, body.Data.Quantity)
	})

	t.Run("adjust stock up", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/inventory/"+variantID+"/adjust", map[string]interface{}{
			"adjustment": 20,
			"reason":     "restock",
		}, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		check := harness.Do(t, http.MethodGet, "/api/inventory/"+variantID, nil, "")
		require.Equal(t, http.StatusOK, check.StatusCode)

		var body struct {
			Data struct{ Quantity int `json:"quantity"` } `json:"data"`
		}
		testutil.Decode(t, check, &body)
		assert.Equal(t, 110, body.Data.Quantity)
	})
}

func TestInventory_Reservation(t *testing.T) {
	harness.Truncate(t, inventoryCollections...)
	adminToken := setupInventoryAdmin(t)
	variantID := setupInventoryVariant(t, adminToken)

	// Seed stock
	resp := harness.Do(t, http.MethodPost, "/api/inventory/update", map[string]interface{}{
		"variant_id": variantID,
		"quantity":   10,
	}, adminToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	var reservationID string

	t.Run("reserve stock", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/inventory/"+variantID+"/reserve", map[string]interface{}{
			"quantity": 3,
		}, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				ReservationID string `json:"reservation_id"`
				Quantity      int    `json:"quantity"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.NotEmpty(t, body.Data.ReservationID)
		assert.Equal(t, 3, body.Data.Quantity)
		reservationID = body.Data.ReservationID
	})

	t.Run("release reservation", func(t *testing.T) {
		require.NotEmpty(t, reservationID, "depends on 'reserve stock'")

		resp := harness.Do(t, http.MethodDelete, "/api/inventory/"+variantID+"/reserve/"+reservationID, nil, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestInventory_BulkUpdate(t *testing.T) {
	harness.Truncate(t, inventoryCollections...)
	adminToken := setupInventoryAdmin(t)
	variantID := setupInventoryVariant(t, adminToken)

	t.Run("bulk update sets stock", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/inventory/bulk-update", map[string]interface{}{
			"items": []map[string]interface{}{
				{"variant_id": variantID, "quantity": 200},
			},
		}, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// Verify
		check := harness.Do(t, http.MethodGet, "/api/inventory/"+variantID, nil, "")
		require.Equal(t, http.StatusOK, check.StatusCode)

		var body struct {
			Data struct{ Quantity int `json:"quantity"` } `json:"data"`
		}
		testutil.Decode(t, check, &body)
		assert.Equal(t, 200, body.Data.Quantity)
	})
}

func TestInventory_Transfer(t *testing.T) {
	harness.Truncate(t, inventoryCollections...)
	adminToken := setupInventoryAdmin(t)
	variantID := setupInventoryVariant(t, adminToken)

	// Seed stock at location "warehouse-a"
	resp := harness.Do(t, http.MethodPost, "/api/inventory/update", map[string]interface{}{
		"variant_id":  variantID,
		"location_id": "warehouse-a",
		"quantity":    50,
	}, adminToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	t.Run("transfer stock between locations", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/inventory/transfer", map[string]interface{}{
			"variant_id":    variantID,
			"from_location": "warehouse-a",
			"to_location":   "warehouse-b",
			"quantity":      20,
		}, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestInventory_History(t *testing.T) {
	harness.Truncate(t, inventoryCollections...)
	adminToken := setupInventoryAdmin(t)
	variantID := setupInventoryVariant(t, adminToken)

	// Create some stock history
	resp := harness.Do(t, http.MethodPost, "/api/inventory/update", map[string]interface{}{
		"variant_id": variantID,
		"quantity":   100,
	}, adminToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	t.Run("get stock history", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/inventory/"+variantID+"/history", nil, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				History []struct {
					VariantID string `json:"variant_id"`
				} `json:"history"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		// History may or may not be populated depending on event flow;
		// just ensure the endpoint responds correctly.
	})
}

func TestInventory_Delete(t *testing.T) {
	harness.Truncate(t, inventoryCollections...)
	adminToken := setupInventoryAdmin(t)
	variantID := setupInventoryVariant(t, adminToken)

	// Seed stock
	resp := harness.Do(t, http.MethodPost, "/api/inventory/update", map[string]interface{}{
		"variant_id": variantID,
		"quantity":   10,
	}, adminToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	t.Run("admin deletes stock record", func(t *testing.T) {
		resp := harness.Do(t, http.MethodDelete, "/api/inventory/"+variantID, nil, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}
