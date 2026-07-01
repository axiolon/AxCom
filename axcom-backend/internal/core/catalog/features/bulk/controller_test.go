// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package bulk

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

type mockService struct {
	bulkCreate func(ctx context.Context, products []*domain.Product) error
	bulkUpdate func(ctx context.Context, products []*domain.Product) error
	bulkDelete func(ctx context.Context, ids []string) error
}

func (m *mockService) BulkCreate(ctx context.Context, products []*domain.Product) error {
	if m.bulkCreate != nil {
		return m.bulkCreate(ctx, products)
	}
	return nil
}

func (m *mockService) BulkUpdate(ctx context.Context, products []*domain.Product) error {
	if m.bulkUpdate != nil {
		return m.bulkUpdate(ctx, products)
	}
	return nil
}

func (m *mockService) BulkDelete(ctx context.Context, ids []string) error {
	if m.bulkDelete != nil {
		return m.bulkDelete(ctx, ids)
	}
	return nil
}

type mockE2ERepository struct {
	products map[string]*domain.Product
}

func (m *mockE2ERepository) BulkCreate(_ context.Context, products []*domain.Product) error {
	for _, p := range products {
		m.products[p.ID] = p
	}
	return nil
}

func (m *mockE2ERepository) BulkUpdate(_ context.Context, products []*domain.Product) error {
	for _, p := range products {
		if _, exists := m.products[p.ID]; !exists {
			return errors.New("product not found")
		}
		m.products[p.ID] = p
	}
	return nil
}

func (m *mockE2ERepository) BulkDelete(_ context.Context, ids []string) error {
	for _, id := range ids {
		delete(m.products, id)
	}
	return nil
}

func (m *mockE2ERepository) GetCategoryByID(_ context.Context, id string) (*domain.Category, error) {
	if id == "invalid_cat" {
		return nil, errors.New("category not found")
	}
	return &domain.Category{ID: id, Name: "Category " + id}, nil
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

func TestController_BulkCreate_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful bulk create", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			bulkCreate: func(_ context.Context, products []*domain.Product) error {
				assert.Len(t, products, 1)
				assert.Equal(t, "Prod Name", products[0].Name)
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := []BulkCreateProductRequest{
			{
				Name:        "Prod Name",
				Description: "Prod Desc",
				CategoryID:  "cat_123",
				Variants: []BulkVariantDTO{
					{
						SKU:   "SKU-ABC",
						Name:  "Variant ABC",
						Price: 49.99,
					},
				},
			},
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/bulk", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("fails - invalid json payload", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{}
		router := setupTestRouter(mockSvc)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/bulk", bytes.NewBufferString("{invalid}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("fails - service error propagation", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			bulkCreate: func(_ context.Context, _ []*domain.Product) error {
				return apperrors.NewNotFound("category not found", nil)
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := []BulkCreateProductRequest{
			{
				Name:       "Prod Name",
				CategoryID: "cat_invalid",
				Variants: []BulkVariantDTO{
					{
						SKU:   "SKU-ABC",
						Name:  "Variant ABC",
						Price: 49.99,
					},
				},
			},
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/bulk", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestController_BulkUpdate_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful bulk update", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			bulkUpdate: func(_ context.Context, products []*domain.Product) error {
				assert.Len(t, products, 1)
				assert.Equal(t, "prod_123", products[0].ID)
				assert.Equal(t, "Updated Name", products[0].Name)
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := []BulkUpdateProductRequest{
			{
				ID:          "prod_123",
				Name:        "Updated Name",
				Description: "Updated Desc",
				CategoryID:  "cat_123",
				Variants: []BulkVariantDTO{
					{
						SKU:   "SKU-ABC",
						Name:  "Variant ABC",
						Price: 59.99,
					},
				},
			},
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, "/api/products/bulk", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestController_BulkDelete_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful bulk delete", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			bulkDelete: func(_ context.Context, ids []string) error {
				assert.Equal(t, []string{"prod_1", "prod_2"}, ids)
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := BulkDeleteRequest{
			IDs: []string{"prod_1", "prod_2"},
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodDelete, "/api/products/bulk", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ----------------------------------------------------
// END-TO-END TESTS (Full Controller + Service + DB Stack)
// ----------------------------------------------------

func TestBulkOperationsFlow_E2E(t *testing.T) {
	t.Parallel()

	repo := &mockE2ERepository{
		products: make(map[string]*domain.Product),
	}

	service := NewService(repo)
	router := setupTestRouter(service)

	// 1. Bulk Create
	createReq := []BulkCreateProductRequest{
		{
			Name:        "Wireless Mouse",
			Description: "Ergonomic wireless mouse",
			CategoryID:  "cat_accessories",
			Variants: []BulkVariantDTO{
				{SKU: "MS-WIRELESS-01", Name: "Gray Mouse", Price: 25.0},
			},
		},
		{
			Name:        "Mechanical Keyboard",
			Description: "RGB keyboard",
			CategoryID:  "cat_accessories",
			Variants: []BulkVariantDTO{
				{SKU: "KB-RGB-01", Name: "Blue Switch", Price: 85.0},
			},
		},
	}
	createBytes, _ := json.Marshal(createReq)

	req1, _ := http.NewRequest(http.MethodPost, "/api/products/bulk", bytes.NewBuffer(createBytes))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	var resp struct {
		Success bool              `json:"success"`
		Data    []*domain.Product `json:"data"`
	}
	err := json.Unmarshal(w1.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.True(t, resp.Success)
	createdProducts := resp.Data
	require.Len(t, createdProducts, 2)
	assert.NotEmpty(t, createdProducts[0].ID)
	assert.NotEmpty(t, createdProducts[1].ID)

	p1ID := createdProducts[0].ID
	p2ID := createdProducts[1].ID

	// 2. Bulk Update
	updateReq := []BulkUpdateProductRequest{
		{
			ID:          p1ID,
			Name:        "Wireless Mouse Pro",
			Description: "Advanced ergonomic wireless mouse",
			CategoryID:  "cat_accessories",
			Variants: []BulkVariantDTO{
				{ID: createdProducts[0].Variants[0].ID, SKU: "MS-WIRELESS-01", Name: "Black Mouse", Price: 35.0},
			},
		},
	}
	updateBytes, _ := json.Marshal(updateReq)

	req2, _ := http.NewRequest(http.MethodPut, "/api/products/bulk", bytes.NewBuffer(updateBytes))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "Wireless Mouse Pro", repo.products[p1ID].Name)
	assert.Equal(t, 35.0, repo.products[p1ID].Variants[0].Price)

	// 3. Bulk Delete
	deleteReq := BulkDeleteRequest{
		IDs: []string{p1ID, p2ID},
	}
	deleteBytes, _ := json.Marshal(deleteReq)

	req3, _ := http.NewRequest(http.MethodDelete, "/api/products/bulk", bytes.NewBuffer(deleteBytes))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusOK, w3.Code)
	assert.Nil(t, repo.products[p1ID])
	assert.Nil(t, repo.products[p2ID])
}
