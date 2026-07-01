// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecom-engine/internal/core/catalog/domain"
	"ecom-engine/internal/events"
	apperrors "ecom-engine/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockCatalogService struct {
	addProduct            func(ctx context.Context, p *domain.Product) error
	getProduct            func(ctx context.Context, id string) (*ProductResponse, error)
	getProductEntity      func(ctx context.Context, id string) (*domain.Product, error)
	getProductByVariantID func(ctx context.Context, variantID string) (*domain.Product, error)
	updateProduct         func(ctx context.Context, p *domain.Product) error
	deleteProduct         func(ctx context.Context, id string) error
	addCategory           func(ctx context.Context, c *domain.Category) error
	getCategory           func(ctx context.Context, id string) (*domain.Category, error)
	updateCategory        func(ctx context.Context, c *domain.Category) error
	deleteCategory        func(ctx context.Context, id string) error
	getProducts           func(ctx context.Context, query *ListProductsQuery) ([]ProductResponse, error)
	getCategories         func(ctx context.Context) ([]domain.Category, error)
	subscribeStockEvents  func(bus events.EventBus)
}

func (m *mockCatalogService) AddProduct(ctx context.Context, p *domain.Product) error {
	if m.addProduct != nil {
		return m.addProduct(ctx, p)
	}
	return nil
}

func (m *mockCatalogService) GetProduct(ctx context.Context, id string) (*ProductResponse, error) {
	if m.getProduct != nil {
		return m.getProduct(ctx, id)
	}
	return nil, nil
}

func (m *mockCatalogService) GetProductEntity(ctx context.Context, id string) (*domain.Product, error) {
	if m.getProductEntity != nil {
		return m.getProductEntity(ctx, id)
	}
	return nil, nil
}

func (m *mockCatalogService) GetProductByVariantID(ctx context.Context, variantID string) (*domain.Product, error) {
	if m.getProductByVariantID != nil {
		return m.getProductByVariantID(ctx, variantID)
	}
	return nil, nil
}

func (m *mockCatalogService) UpdateProduct(ctx context.Context, p *domain.Product) error {
	if m.updateProduct != nil {
		return m.updateProduct(ctx, p)
	}
	return nil
}

func (m *mockCatalogService) DeleteProduct(ctx context.Context, id string) error {
	if m.deleteProduct != nil {
		return m.deleteProduct(ctx, id)
	}
	return nil
}

func (m *mockCatalogService) AddCategory(ctx context.Context, c *domain.Category) error {
	if m.addCategory != nil {
		return m.addCategory(ctx, c)
	}
	return nil
}

func (m *mockCatalogService) GetCategory(ctx context.Context, id string) (*domain.Category, error) {
	if m.getCategory != nil {
		return m.getCategory(ctx, id)
	}
	return nil, nil
}

func (m *mockCatalogService) UpdateCategory(ctx context.Context, c *domain.Category) error {
	if m.updateCategory != nil {
		return m.updateCategory(ctx, c)
	}
	return nil
}

func (m *mockCatalogService) DeleteCategory(ctx context.Context, id string) error {
	if m.deleteCategory != nil {
		return m.deleteCategory(ctx, id)
	}
	return nil
}

func (m *mockCatalogService) GetProducts(ctx context.Context, query *ListProductsQuery) ([]ProductResponse, error) {
	if m.getProducts != nil {
		return m.getProducts(ctx, query)
	}
	return nil, nil
}

func (m *mockCatalogService) GetCategories(ctx context.Context) ([]domain.Category, error) {
	if m.getCategories != nil {
		return m.getCategories(ctx)
	}
	return nil, nil
}

func (m *mockCatalogService) SubscribeStockEvents(bus events.EventBus) {
	if m.subscribeStockEvents != nil {
		m.subscribeStockEvents(bus)
	}
}

func setupTestRouter(svc *mockCatalogService) *gin.Engine {
	router := gin.New()
	rg := router.Group("/api")
	mockAuthMiddleware := func(c *gin.Context) {
		c.Next()
	}
	ctrl := NewController(svc, svc)
	RegisterRoutes(rg, ctrl, mockAuthMiddleware, mockAuthMiddleware)
	return router
}

func TestController_GetProducts(t *testing.T) {
	t.Parallel()

	mockSvc := &mockCatalogService{}
	router := setupTestRouter(mockSvc)

	t.Run("successful products list", func(t *testing.T) {
		mockSvc.getProducts = func(_ context.Context, _ *ListProductsQuery) ([]ProductResponse, error) {
			return []ProductResponse{
				{ID: "prod_1", Name: "Smartphone"},
			}, nil
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/products", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].([]interface{})
		assert.Len(t, data, 1)
		assert.Equal(t, "Smartphone", data[0].(map[string]interface{})["name"])
	})

	t.Run("fails - service error", func(t *testing.T) {
		mockSvc.getProducts = func(_ context.Context, _ *ListProductsQuery) ([]ProductResponse, error) {
			return nil, errors.New("database issue")
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/products", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestController_GetProduct(t *testing.T) {
	t.Parallel()

	mockSvc := &mockCatalogService{}
	router := setupTestRouter(mockSvc)

	t.Run("successful product retrieval", func(t *testing.T) {
		mockSvc.getProduct = func(_ context.Context, id string) (*ProductResponse, error) {
			assert.Equal(t, "prod_123", id)
			return &ProductResponse{
				ID:   "prod_123",
				Name: "Smartphone",
			}, nil
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/products/prod_123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("fails - product not found", func(t *testing.T) {
		mockSvc.getProduct = func(_ context.Context, _ string) (*ProductResponse, error) {
			return nil, apperrors.NewNotFound("product not found", ErrProductNotFound)
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/products/prod_missing", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestController_CreateProduct(t *testing.T) {
	t.Parallel()

	mockSvc := &mockCatalogService{}
	router := setupTestRouter(mockSvc)

	t.Run("successful product creation", func(t *testing.T) {
		mockSvc.addProduct = func(_ context.Context, p *domain.Product) error {
			p.ID = "prod_generated"
			return nil
		}

		reqBody := CreateProductRequest{
			Name:        "Tablet",
			Description: "Pro model",
			CategoryID:  "cat_1",
			Variants: []VariantDTO{
				{SKU: "TAB-01", Name: "128GB", Price: 499.99},
			},
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/products", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("fails - invalid payload", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/products", bytes.NewBufferString("{invalid_json}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestController_UpdateProduct(t *testing.T) {
	t.Parallel()

	mockSvc := &mockCatalogService{}
	router := setupTestRouter(mockSvc)

	t.Run("successful product update", func(t *testing.T) {
		mockSvc.updateProduct = func(_ context.Context, p *domain.Product) error {
			assert.Equal(t, "prod_1", p.ID)
			return nil
		}

		reqBody := UpdateProductRequest{
			Name:       "Updated Tablet",
			CategoryID: "cat_1",
			Variants: []VariantDTO{
				{SKU: "TAB-01", Name: "128GB", Price: 499.99},
			},
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, "/api/products/prod_1", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestController_DeleteProduct(t *testing.T) {
	t.Parallel()

	mockSvc := &mockCatalogService{}
	router := setupTestRouter(mockSvc)

	t.Run("successful product deletion", func(t *testing.T) {
		mockSvc.deleteProduct = func(_ context.Context, id string) error {
			assert.Equal(t, "prod_1", id)
			return nil
		}

		req, _ := http.NewRequest(http.MethodDelete, "/api/products/prod_1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestController_DeleteCategory(t *testing.T) {
	t.Parallel()

	mockSvc := &mockCatalogService{}
	router := setupTestRouter(mockSvc)

	t.Run("fails - delete conflict due to products assigned", func(t *testing.T) {
		mockSvc.deleteCategory = func(_ context.Context, _ string) error {
			return apperrors.NewConflict("cannot delete category: products are assigned to it", nil)
		}

		req, _ := http.NewRequest(http.MethodDelete, "/api/categories/cat_active", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}
