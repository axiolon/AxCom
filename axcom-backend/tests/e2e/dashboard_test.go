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

// dashboardCollections are the collections touched by dashboard tests.
var dashboardCollections = []string{
	"users", "refresh_tokens", "products", "categories",
	"stocks", "orders",
}

func TestDashboard_AdminStats(t *testing.T) {
	harness.Truncate(t, dashboardCollections...)

	// Seed admin
	harness.SeedUser(t, "dash_admin@example.com", "Admin123!", "admin")
	adminToken, _ := harness.LoginAs(t, "dash_admin@example.com", "Admin123!")

	// Seed a customer + category + product + inventory + order for non-empty stats
	catResp := harness.Do(t, http.MethodPost, "/api/categories", map[string]string{
		"name": "Dash Test Category",
		"slug": "dash-test",
	}, adminToken)
	require.Equal(t, http.StatusOK, catResp.StatusCode)
	var catBody struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	testutil.Decode(t, catResp, &catBody)

	prodResp := harness.Do(t, http.MethodPost, "/api/products", map[string]interface{}{
		"name":        "Dash Widget",
		"category_id": catBody.Data.ID,
		"variants": []map[string]interface{}{
			{"sku": "DASH-W-001", "name": "Default", "price": 25.00},
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
	variantID := prodBody.Data.Variants[0].ID

	stockResp := harness.Do(t, http.MethodPost, "/api/inventory/update", map[string]interface{}{
		"variant_id": variantID,
		"quantity":   50,
	}, adminToken)
	require.Equal(t, http.StatusOK, stockResp.StatusCode)
	stockResp.Body.Close()

	reg := harness.Do(t, http.MethodPost, "/api/auth/register", map[string]string{
		"email":    "dash_customer@example.com",
		"password": "Password123!",
	}, "")
	require.Equal(t, http.StatusOK, reg.StatusCode)
	reg.Body.Close()
	custToken, _ := harness.LoginAs(t, "dash_customer@example.com", "Password123!")

	orderResp := harness.Do(t, http.MethodPost, "/api/orders", map[string]interface{}{
		"items": []map[string]interface{}{
			{"variant_id": variantID, "quantity": 2, "price": 25.00},
		},
	}, custToken)
	require.Equal(t, http.StatusOK, orderResp.StatusCode)
	orderResp.Body.Close()

	t.Run("admin gets dashboard stats", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/admin/dashboard", nil, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				Tier           string             `json:"tier"`
				RevenueToday   float64            `json:"revenue_today"`
				OrdersByStatus map[string]int64   `json:"orders_by_status"`
				RecentOrders   []interface{}      `json:"recent_orders"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Equal(t, "small", body.Data.Tier)
		assert.NotNil(t, body.Data.OrdersByStatus)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/admin/dashboard", nil, "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("non-admin returns 403", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/admin/dashboard", nil, custToken)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		resp.Body.Close()
	})
}