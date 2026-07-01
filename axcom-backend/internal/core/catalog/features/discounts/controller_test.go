// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package discounts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecom-engine/internal/core/catalog/domain"
	apperrors "ecom-engine/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockService mocks the discounts.Service interface
type mockService struct {
	applyDiscount  func(ctx context.Context, productID string, discount *domain.ProductDiscount) error
	removeDiscount func(ctx context.Context, productID string) error
}

func (m *mockService) ApplyDiscount(ctx context.Context, productID string, discount *domain.ProductDiscount) error {
	if m.applyDiscount != nil {
		return m.applyDiscount(ctx, productID, discount)
	}
	return nil
}

func (m *mockService) RemoveDiscount(ctx context.Context, productID string) error {
	if m.removeDiscount != nil {
		return m.removeDiscount(ctx, productID)
	}
	return nil
}

// mockE2ERepository is a simple in-memory repository to enable E2E testing
type mockE2ERepository struct {
	products map[string]*domain.Product
}

func (m *mockE2ERepository) GetProductByID(_ context.Context, id string) (*domain.Product, error) {
	p, exists := m.products[id]
	if !exists {
		return nil, errors.New("product not found")
	}
	return p, nil
}

func (m *mockE2ERepository) UpdateProductDiscount(_ context.Context, id string, discount *domain.ProductDiscount) error {
	p, exists := m.products[id]
	if !exists {
		return errors.New("product not found")
	}
	p.Discount = discount
	return nil
}

func setupTestRouter(svc Service) *gin.Engine {
	router := gin.New()
	rg := router.Group("/api")
	mockAuthMiddleware := func(c *gin.Context) {
		c.Next()
	}
	ctrl := NewController(svc)
	RegisterRoutes(rg, ctrl, mockAuthMiddleware, mockAuthMiddleware)
	return router
}

// ----------------------------------------------------
// INTEGRATION TESTS (Mocked Service)
// ----------------------------------------------------

func TestController_ApplyDiscount_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful discount application", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			applyDiscount: func(_ context.Context, productID string, discount *domain.ProductDiscount) error {
				assert.Equal(t, "prod_1", productID)
				assert.Equal(t, "percentage", discount.Type)
				assert.Equal(t, 20.0, discount.Value)
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := ApplyDiscountRequest{
			Type:  "percentage",
			Value: 20.0,
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/prod_1/discount", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("fails - bad request payload validation", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{}
		router := setupTestRouter(mockSvc)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/prod_1/discount", bytes.NewBufferString("{invalid_json}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("fails - service error mapping", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			applyDiscount: func(_ context.Context, _ string, _ *domain.ProductDiscount) error {
				return apperrors.NewNotFound("product not found", nil)
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := ApplyDiscountRequest{
			Type:  "fixed",
			Value: 10.0,
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/prod_nonexistent/discount", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestController_RemoveDiscount_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful discount removal", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			removeDiscount: func(_ context.Context, productID string) error {
				assert.Equal(t, "prod_1", productID)
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodDelete, "/api/products/prod_1/discount", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
	})
}

// ----------------------------------------------------
// END-TO-END TESTS (Full Controller + Service + DB Stack)
// ----------------------------------------------------

func TestDiscountFlow_E2E(t *testing.T) {
	t.Parallel()

	// 1. Setup real layer components with an in-memory repository mock
	repo := &mockE2ERepository{
		products: map[string]*domain.Product{
			"prod_abc": {
				ID:         "prod_abc",
				Name:       "E2E Wireless Mouse",
				CategoryID: "cat_accessories",
				Variants: []domain.Variant{
					{ID: "var_mouse_1", SKU: "MS-E2E-01", Price: 29.99, Stock: 100},
				},
			},
		},
	}

	service := NewService(repo)
	router := setupTestRouter(service)

	// Verify initially product has no discount
	p, err := repo.GetProductByID(context.Background(), "prod_abc")
	require.NoError(t, err)
	assert.Nil(t, p.Discount)

	// 2. HTTP POST to apply percentage discount
	applyReq := ApplyDiscountRequest{
		Type:  "percentage",
		Value: 15.0,
	}
	applyBytes, _ := json.Marshal(applyReq)

	req1, _ := http.NewRequest(http.MethodPost, "/api/products/prod_abc/discount", bytes.NewBuffer(applyBytes))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	// Verify discount is updated in DB/repo
	p, err = repo.GetProductByID(context.Background(), "prod_abc")
	require.NoError(t, err)
	require.NotNil(t, p.Discount)
	assert.Equal(t, "percentage", p.Discount.Type)
	assert.Equal(t, 15.0, p.Discount.Value)

	// 3. HTTP DELETE to remove discount
	req2, _ := http.NewRequest(http.MethodDelete, "/api/products/prod_abc/discount", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	// Verify discount is nil in DB/repo
	p, err = repo.GetProductByID(context.Background(), "prod_abc")
	require.NoError(t, err)
	assert.Nil(t, p.Discount)
}
