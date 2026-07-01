// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package variants

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
	getVariants   func(ctx context.Context, productID string) ([]domain.Variant, error)
	addVariant    func(ctx context.Context, productID string, v *domain.Variant) error
	updateVariant func(ctx context.Context, productID string, v *domain.Variant) error
	deleteVariant func(ctx context.Context, productID string, variantID string) error
}

func (m *mockService) GetVariants(ctx context.Context, productID string) ([]domain.Variant, error) {
	if m.getVariants != nil {
		return m.getVariants(ctx, productID)
	}
	return nil, nil
}

func (m *mockService) AddVariant(ctx context.Context, productID string, v *domain.Variant) error {
	if m.addVariant != nil {
		return m.addVariant(ctx, productID, v)
	}
	return nil
}

func (m *mockService) UpdateVariant(ctx context.Context, productID string, v *domain.Variant) error {
	if m.updateVariant != nil {
		return m.updateVariant(ctx, productID, v)
	}
	return nil
}

func (m *mockService) DeleteVariant(ctx context.Context, productID string, variantID string) error {
	if m.deleteVariant != nil {
		return m.deleteVariant(ctx, productID, variantID)
	}
	return nil
}

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

func (m *mockE2ERepository) UpdateProductVariants(_ context.Context, id string, variants []domain.Variant) error {
	p, exists := m.products[id]
	if !exists {
		return errors.New("product not found")
	}
	p.Variants = variants
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

func TestController_GetVariants_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful get variants", func(t *testing.T) {
		t.Parallel()
		expectedVariants := []domain.Variant{
			{ID: "var_1", SKU: "SKU-1", Name: "V1", Price: 10.0},
		}
		mockSvc := &mockService{
			getVariants: func(_ context.Context, productID string) ([]domain.Variant, error) {
				assert.Equal(t, "prod_1", productID)
				return expectedVariants, nil
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/products/prod_1/variants", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool             `json:"success"`
			Data    []domain.Variant `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.True(t, resp.Success)
		assert.Equal(t, expectedVariants, resp.Data)
	})

	t.Run("fails - product not found", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			getVariants: func(_ context.Context, _ string) ([]domain.Variant, error) {
				return nil, apperrors.NewNotFound("product not found", nil)
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/products/prod_nonexistent/variants", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestController_AddVariant_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful add variant", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			addVariant: func(_ context.Context, productID string, v *domain.Variant) error {
				assert.Equal(t, "prod_1", productID)
				assert.Equal(t, "SKU-NEW", v.SKU)
				v.ID = "var_generated"
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := CreateVariantRequest{
			SKU:   "SKU-NEW",
			Name:  "New Variant",
			Price: 25.0,
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/prod_1/variants", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool           `json:"success"`
			Data    domain.Variant `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.True(t, resp.Success)
		assert.Equal(t, "var_generated", resp.Data.ID)
		assert.Equal(t, "SKU-NEW", resp.Data.SKU)
	})

	t.Run("fails - missing binding fields", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{}
		router := setupTestRouter(mockSvc)

		// SKU and Name are required, price cannot be negative
		reqBody := map[string]interface{}{
			"price": -10.0,
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/prod_1/variants", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestController_UpdateVariant_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful update variant", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			updateVariant: func(_ context.Context, productID string, v *domain.Variant) error {
				assert.Equal(t, "prod_1", productID)
				assert.Equal(t, "var_1", v.ID)
				assert.Equal(t, "SKU-UPDATED", v.SKU)
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := UpdateVariantRequest{
			SKU:   "SKU-UPDATED",
			Name:  "Updated Variant",
			Price: 15.0,
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, "/api/products/prod_1/variants/var_1", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestController_DeleteVariant_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful delete variant", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			deleteVariant: func(_ context.Context, productID string, variantID string) error {
				assert.Equal(t, "prod_1", productID)
				assert.Equal(t, "var_1", variantID)
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodDelete, "/api/products/prod_1/variants/var_1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ----------------------------------------------------
// END-TO-END TESTS (Full Controller + Service + DB Stack)
// ----------------------------------------------------

func TestVariantOperationsFlow_E2E(t *testing.T) {
	t.Parallel()

	repo := &mockE2ERepository{
		products: map[string]*domain.Product{
			"prod_electronics": {
				ID:          "prod_electronics",
				Name:        "Smartphone",
				CategoryID:  "cat_phones",
				Description: "Latest brand model",
				Variants: []domain.Variant{
					{ID: "var_base", SKU: "SP-BASE", Name: "Base Variant", Price: 499.99},
				},
			},
		},
	}

	service := NewService(repo)
	router := setupTestRouter(service)

	// 1. GET variants (should only contain var_base)
	req1, _ := http.NewRequest(http.MethodGet, "/api/products/prod_electronics/variants", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	var getResp struct {
		Success bool             `json:"success"`
		Data    []domain.Variant `json:"data"`
	}
	err := json.Unmarshal(w1.Body.Bytes(), &getResp)
	require.NoError(t, err)
	require.Len(t, getResp.Data, 1)
	assert.Equal(t, "var_base", getResp.Data[0].ID)

	// 2. ADD new variant (e.g. Pro model)
	addReq := CreateVariantRequest{
		SKU:   "SP-PRO",
		Name:  "Pro Model Variant",
		Price: 799.99,
		Attributes: map[string]string{
			"storage": "256GB",
			"color":   "Titanium",
		},
	}
	addBytes, _ := json.Marshal(addReq)
	req2, _ := http.NewRequest(http.MethodPost, "/api/products/prod_electronics/variants", bytes.NewBuffer(addBytes))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	var addResp struct {
		Success bool           `json:"success"`
		Data    domain.Variant `json:"data"`
	}
	err = json.Unmarshal(w2.Body.Bytes(), &addResp)
	require.NoError(t, err)
	assert.NotEmpty(t, addResp.Data.ID)
	assert.Equal(t, "SP-PRO", addResp.Data.SKU)
	proID := addResp.Data.ID

	// Verify it's now in the database
	assert.Len(t, repo.products["prod_electronics"].Variants, 2)

	// 3. UPDATE the newly added Pro model variant
	updateReq := UpdateVariantRequest{
		SKU:   "SP-PRO-UPDATED",
		Name:  "Pro Model 512GB",
		Price: 849.99,
		Attributes: map[string]string{
			"storage": "512GB",
			"color":   "Titanium",
		},
	}
	updateBytes, _ := json.Marshal(updateReq)
	req3, _ := http.NewRequest(http.MethodPut, "/api/products/prod_electronics/variants/"+proID, bytes.NewBuffer(updateBytes))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusOK, w3.Code)
	assert.Equal(t, "SP-PRO-UPDATED", repo.products["prod_electronics"].Variants[1].SKU)
	assert.Equal(t, 849.99, repo.products["prod_electronics"].Variants[1].Price)

	// 4. DELETE the base variant
	req4, _ := http.NewRequest(http.MethodDelete, "/api/products/prod_electronics/variants/var_base", nil)
	w4 := httptest.NewRecorder()
	router.ServeHTTP(w4, req4)

	assert.Equal(t, http.StatusOK, w4.Code)
	assert.Len(t, repo.products["prod_electronics"].Variants, 1)
	assert.Equal(t, proID, repo.products["prod_electronics"].Variants[0].ID)
}
