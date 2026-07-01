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

// catalogCollections are the collections touched by catalog tests.
var catalogCollections = []string{"users", "refresh_tokens", "products", "categories"}

// adminCreds holds the test admin credentials seeded once per test.
const (
	adminEmail    = "catalog_admin@example.com"
	adminPassword = "Admin123!"
)

// setupCatalogAdmin seeds an admin user and returns an access token.
func setupCatalogAdmin(t *testing.T) string {
	t.Helper()
	harness.SeedUser(t, adminEmail, adminPassword, "admin")
	token, _ := harness.LoginAs(t, adminEmail, adminPassword)
	return token
}

func TestCatalog_CategoryCRUD(t *testing.T) {
	harness.Truncate(t, catalogCollections...)
	adminToken := setupCatalogAdmin(t)

	var categoryID string

	t.Run("public list categories returns empty initially", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/categories", nil, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data []interface{} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Empty(t, body.Data)
	})

	t.Run("unauthenticated create returns 401", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/categories", map[string]string{
			"name": "Electronics",
			"slug": "electronics",
		}, "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("non-admin create returns 403", func(t *testing.T) {
		// Register and login as a regular customer
		reg := harness.Do(t, http.MethodPost, "/api/auth/register", map[string]string{
			"email":    "customer_catalog@example.com",
			"password": "Password123!",
		}, "")
		require.Equal(t, http.StatusOK, reg.StatusCode)
		reg.Body.Close()

		custToken, _ := harness.LoginAs(t, "customer_catalog@example.com", "Password123!")

		resp := harness.Do(t, http.MethodPost, "/api/categories", map[string]string{
			"name": "Electronics",
			"slug": "electronics",
		}, custToken)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("admin creates category", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/categories", map[string]string{
			"name": "Electronics",
			"slug": "electronics",
		}, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Slug string `json:"slug"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.NotEmpty(t, body.Data.ID)
		assert.Equal(t, "Electronics", body.Data.Name)
		assert.Equal(t, "electronics", body.Data.Slug)
		categoryID = body.Data.ID
	})

	t.Run("public list categories shows the new category", func(t *testing.T) {
		require.NotEmpty(t, categoryID, "depends on 'admin creates category'")

		resp := harness.Do(t, http.MethodGet, "/api/categories", nil, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		require.Len(t, body.Data, 1)
		assert.Equal(t, categoryID, body.Data[0].ID)
	})

	t.Run("admin updates category", func(t *testing.T) {
		require.NotEmpty(t, categoryID, "depends on 'admin creates category'")

		resp := harness.Do(t, http.MethodPut, "/api/categories/"+categoryID, map[string]string{
			"name": "Consumer Electronics",
			"slug": "consumer-electronics",
		}, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct{ Name string `json:"name"` } `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Equal(t, "Consumer Electronics", body.Data.Name)
	})

	t.Run("admin deletes category", func(t *testing.T) {
		require.NotEmpty(t, categoryID, "depends on 'admin creates category'")

		resp := harness.Do(t, http.MethodDelete, "/api/categories/"+categoryID, nil, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// Confirm it is gone
		listResp := harness.Do(t, http.MethodGet, "/api/categories", nil, "")
		require.Equal(t, http.StatusOK, listResp.StatusCode)

		var body struct {
			Data []interface{} `json:"data"`
		}
		testutil.Decode(t, listResp, &body)
		assert.Empty(t, body.Data)
	})
}

func TestCatalog_ProductCRUD(t *testing.T) {
	harness.Truncate(t, catalogCollections...)
	adminToken := setupCatalogAdmin(t)

	// Create a category to attach products to
	catResp := harness.Do(t, http.MethodPost, "/api/categories", map[string]string{
		"name": "Clothing",
		"slug": "clothing",
	}, adminToken)
	require.Equal(t, http.StatusOK, catResp.StatusCode)

	var catBody struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	testutil.Decode(t, catResp, &catBody)
	categoryID := catBody.Data.ID
	require.NotEmpty(t, categoryID)

	var productID string

	t.Run("public list products is empty initially", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/products", nil, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data []interface{} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Empty(t, body.Data)
	})

	t.Run("admin creates product with variant", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/products", map[string]interface{}{
			"name":        "Classic T-Shirt",
			"description": "A timeless wardrobe staple",
			"category_id": categoryID,
			"variants": []map[string]interface{}{
				{
					"sku":   "TSHIRT-M-BLK",
					"name":  "Medium Black",
					"price": 29.99,
					"attributes": map[string]string{
						"size":  "M",
						"color": "black",
					},
				},
			},
		}, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				CategoryID string `json:"category_id"`
				Variants   []struct {
					SKU   string  `json:"sku"`
					Price float64 `json:"price"`
				} `json:"variants"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.NotEmpty(t, body.Data.ID)
		assert.Equal(t, "Classic T-Shirt", body.Data.Name)
		assert.Equal(t, categoryID, body.Data.CategoryID)
		require.Len(t, body.Data.Variants, 1)
		assert.Equal(t, "TSHIRT-M-BLK", body.Data.Variants[0].SKU)
		assert.Equal(t, 29.99, body.Data.Variants[0].Price)
		productID = body.Data.ID
	})

	t.Run("product requires at least one variant", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/products", map[string]interface{}{
			"name":        "No-Variant Product",
			"category_id": categoryID,
			"variants":    []interface{}{},
		}, adminToken)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("get product by ID", func(t *testing.T) {
		require.NotEmpty(t, productID, "depends on 'admin creates product'")

		resp := harness.Do(t, http.MethodGet, "/api/products/"+productID, nil, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Equal(t, productID, body.Data.ID)
		assert.Equal(t, "Classic T-Shirt", body.Data.Name)
	})

	t.Run("get non-existent product returns 404", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/products/prod_doesnotexist", nil, "")
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("list products returns the created product", func(t *testing.T) {
		require.NotEmpty(t, productID, "depends on 'admin creates product'")

		resp := harness.Do(t, http.MethodGet, "/api/products", nil, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		require.Len(t, body.Data, 1)
		assert.Equal(t, productID, body.Data[0].ID)
	})

	t.Run("list products filtered by category", func(t *testing.T) {
		require.NotEmpty(t, productID, "depends on 'admin creates product'")

		resp := harness.Do(t, http.MethodGet, "/api/products?category_id="+categoryID, nil, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data []struct{ ID string `json:"id"` } `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Len(t, body.Data, 1)
		resp.Body.Close()

		// Different category should return empty
		resp2 := harness.Do(t, http.MethodGet, "/api/products?category_id=cat_other", nil, "")
		require.Equal(t, http.StatusOK, resp2.StatusCode)
		var body2 struct {
			Data []interface{} `json:"data"`
		}
		testutil.Decode(t, resp2, &body2)
		assert.Empty(t, body2.Data)
	})

	t.Run("admin updates product", func(t *testing.T) {
		require.NotEmpty(t, productID, "depends on 'admin creates product'")

		resp := harness.Do(t, http.MethodPut, "/api/products/"+productID, map[string]interface{}{
			"name":        "Premium T-Shirt",
			"description": "Updated description",
			"category_id": categoryID,
			"variants": []map[string]interface{}{
				{
					"sku":   "TSHIRT-M-BLK",
					"name":  "Medium Black",
					"price": 39.99,
					"attributes": map[string]string{
						"size":  "M",
						"color": "black",
					},
				},
			},
		}, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				Name     string `json:"name"`
				Variants []struct{ Price float64 `json:"price"` } `json:"variants"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Equal(t, "Premium T-Shirt", body.Data.Name)
		require.Len(t, body.Data.Variants, 1)
		assert.Equal(t, 39.99, body.Data.Variants[0].Price)
	})

	t.Run("admin deletes product", func(t *testing.T) {
		require.NotEmpty(t, productID, "depends on 'admin creates product'")

		resp := harness.Do(t, http.MethodDelete, "/api/products/"+productID, nil, adminToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// Confirm gone
		getResp := harness.Do(t, http.MethodGet, "/api/products/"+productID, nil, "")
		assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
		getResp.Body.Close()
	})
}

func TestCatalog_SearchProducts(t *testing.T) {
	harness.Truncate(t, catalogCollections...)
	adminToken := setupCatalogAdmin(t)

	// Seed a category and two products
	catResp := harness.Do(t, http.MethodPost, "/api/categories", map[string]string{
		"name": "Footwear",
		"slug": "footwear",
	}, adminToken)
	require.Equal(t, http.StatusOK, catResp.StatusCode)
	var catBody struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	testutil.Decode(t, catResp, &catBody)
	catID := catBody.Data.ID

	for _, p := range []struct {
		name  string
		sku   string
		price float64
	}{
		{"Running Shoes", "SHOE-RUN-42", 89.99},
		{"Hiking Boots", "BOOT-HIK-42", 149.99},
	} {
		r := harness.Do(t, http.MethodPost, "/api/products", map[string]interface{}{
			"name":        p.name,
			"category_id": catID,
			"variants": []map[string]interface{}{
				{"sku": p.sku, "name": p.name, "price": p.price},
			},
		}, adminToken)
		require.Equal(t, http.StatusOK, r.StatusCode)
		r.Body.Close()
	}

	t.Run("full text search by name", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/products?q=running", nil, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data []struct{ Name string `json:"name"` } `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		require.Len(t, body.Data, 1)
		assert.Equal(t, "Running Shoes", body.Data[0].Name)
	})

	t.Run("price range filter", func(t *testing.T) {
		resp := harness.Do(t, http.MethodGet, "/api/products?price_min=100", nil, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data []struct{ Name string `json:"name"` } `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		require.Len(t, body.Data, 1)
		assert.Equal(t, "Hiking Boots", body.Data[0].Name)
	})
}
