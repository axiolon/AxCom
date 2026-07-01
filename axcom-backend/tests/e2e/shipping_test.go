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

// shippingCollections are the collections touched by shipping tests.
var shippingCollections = []string{
	"users", "refresh_tokens", "products", "categories",
	"stocks", "orders", "shipments",
}

// setupShippingFixtures seeds a full order and returns tokens, order ID, and variant ID.
func setupShippingFixtures(t *testing.T) (adminToken, custToken, orderID, variantID string) {
	t.Helper()

	harness.SeedUser(t, "ship_admin@example.com", "Admin123!", "admin")
	adminToken, _ = harness.LoginAs(t, "ship_admin@example.com", "Admin123!")

	// Category + Product + Inventory
	catResp := harness.Do(t, http.MethodPost, "/api/categories", map[string]string{
		"name": "Ship Test Category",
		"slug": "ship-test",
	}, adminToken)
	require.Equal(t, http.StatusOK, catResp.StatusCode)
	var catBody struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	testutil.Decode(t, catResp, &catBody)

	prodResp := harness.Do(t, http.MethodPost, "/api/products", map[string]interface{}{
		"name":        "Ship Widget",
		"category_id": catBody.Data.ID,
		"variants": []map[string]interface{}{
			{"sku": "SHIP-W-001", "name": "Default", "price": 29.99},
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

	stockResp := harness.Do(t, http.MethodPost, "/api/inventory/update", map[string]interface{}{
		"variant_id": variantID,
		"quantity":   100,
	}, adminToken)
	require.Equal(t, http.StatusOK, stockResp.StatusCode)
	stockResp.Body.Close()

	// Wait for async stock synchronization to the catalog to complete
	time.Sleep(100 * time.Millisecond)

	// Customer + Order
	reg := harness.Do(t, http.MethodPost, "/api/auth/register", map[string]string{
		"email":    "ship_customer@example.com",
		"password": "Password123!",
	}, "")
	require.Equal(t, http.StatusOK, reg.StatusCode)
	reg.Body.Close()
	custToken, _ = harness.LoginAs(t, "ship_customer@example.com", "Password123!")

	orderResp := harness.Do(t, http.MethodPost, "/api/orders", map[string]interface{}{
		"items": []map[string]interface{}{
			{"variant_id": variantID, "quantity": 1, "price": 29.99},
		},
	}, custToken)
	require.Equal(t, http.StatusOK, orderResp.StatusCode)
	var orderBody struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	testutil.Decode(t, orderResp, &orderBody)
	orderID = orderBody.Data.ID

	return adminToken, custToken, orderID, variantID
}

func TestShipping_CalculateRates(t *testing.T) {
	harness.Truncate(t, shippingCollections...)

	t.Run("public rate calculation", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/shipping/rates", map[string]interface{}{
			"weight": 2.5,
			"value":  50.00,
		}, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data []struct {
				ProviderName string  `json:"provider_name"`
				Rate         float64 `json:"rate"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		require.NotEmpty(t, body.Data)
		assert.NotEmpty(t, body.Data[0].ProviderName)
		assert.Greater(t, body.Data[0].Rate, 0.0)
	})
}

func TestShipping_AdminCRUD(t *testing.T) {
	harness.Truncate(t, shippingCollections...)
	adminToken, custToken, orderID, variantID := setupShippingFixtures(t)

	var shipmentID string

	t.Run("admin creates shipment", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/admin/shipping", map[string]interface{}{
			"order_id": orderID,
			"carrier":  "FedEx",
			"weight":   1.5,
			"value":    29.99,
		}, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				ID             string `json:"id"`
				OrderID        string `json:"order_id"`
				Carrier        string `json:"carrier"`
				TrackingNumber string `json:"tracking_number"`
				Status         string `json:"status"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.NotEmpty(t, body.Data.ID)
		assert.Equal(t, orderID, body.Data.OrderID)
		assert.Equal(t, "FedEx", body.Data.Carrier)
		assert.Empty(t, body.Data.TrackingNumber)
		assert.Equal(t, "pending", body.Data.Status)
		shipmentID = body.Data.ID
	})

	t.Run("admin lists shipments", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/admin/shipping", nil, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				Shipments []struct{ ID string `json:"id"` } `json:"shipments"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		require.NotEmpty(t, body.Data.Shipments)
	})

	t.Run("admin updates shipment status to in_transit", func(t *testing.T) {
		require.NotEmpty(t, shipmentID, "depends on 'admin creates shipment'")

		resp := harness.Do(t, http.MethodPut, "/api/admin/shipping/"+shipmentID, map[string]string{
			"status":          "in_transit",
			"tracking_number": "FX123456789",
		}, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Equal(t, "in_transit", body.Data.Status)
	})

	t.Run("public tracking by tracking number", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/shipping/track/FX123456789", nil, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				TrackingNumber string `json:"tracking_number"`
				Carrier        string `json:"carrier"`
				Status         string `json:"status"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Equal(t, "FX123456789", body.Data.TrackingNumber)
		assert.Equal(t, "in_transit", body.Data.Status)
	})

	t.Run("customer gets their order shipment", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/shipping/order/"+orderID, nil, custToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				OrderID string `json:"order_id"`
				Carrier string `json:"carrier"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Equal(t, orderID, body.Data.OrderID)
	})

	t.Run("non-admin cannot create shipments", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/admin/shipping", map[string]interface{}{
			"order_id": orderID,
			"carrier":  "UPS",
			"weight":   1.0,
		}, custToken)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("admin deletes shipment", func(t *testing.T) {
		// Create a new order to attach the pending shipment to
		orderResp := harness.Do(t, http.MethodPost, "/api/orders", map[string]interface{}{
			"items": []map[string]interface{}{
				{"variant_id": variantID, "quantity": 1, "price": 29.99},
			},
		}, custToken)
		require.Equal(t, http.StatusOK, orderResp.StatusCode)
		var orderBody struct {
			Data struct{ ID string `json:"id"` } `json:"data"`
		}
		testutil.Decode(t, orderResp, &orderBody)
		newOrderID := orderBody.Data.ID

		// Create a pending shipment for this order (no tracking number)
		shipResp := harness.Do(t, http.MethodPost, "/api/admin/shipping", map[string]interface{}{
			"order_id": newOrderID,
			"carrier":  "FedEx",
			"weight":   1.5,
			"value":    29.99,
		}, adminToken)
		require.Equal(t, http.StatusOK, shipResp.StatusCode)
		var shipBody struct {
			Data struct{ ID string `json:"id"` } `json:"data"`
		}
		testutil.Decode(t, shipResp, &shipBody)
		tempShipmentID := shipBody.Data.ID

		// Delete this pending shipment
		resp := harness.Do(t, http.MethodDelete, "/api/admin/shipping/"+tempShipmentID, nil, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}
