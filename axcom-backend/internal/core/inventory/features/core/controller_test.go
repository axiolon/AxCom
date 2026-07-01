// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecom-engine/internal/core/inventory/domain"
	sharedDTO "ecom-engine/internal/core/inventory/dto"
	coredto "ecom-engine/internal/core/inventory/features/core/dto"
	apperrors "ecom-engine/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ----------------------------------------------------
// MOCK SERVICE FOR INTEGRATION TESTING
// ----------------------------------------------------

type mockService struct {
	checkStock     func(ctx context.Context, variantID string, locationID string) (int, error)
	updateStock    func(ctx context.Context, variantID string, locationID string, quantity int) error
	listStock      func(ctx context.Context, filter ListStockFilter) ([]*domain.StockItem, error)
	deleteStock    func(ctx context.Context, variantID string, locationID string) error
	listAlerts     func(ctx context.Context, limit, offset int) ([]*domain.Alert, error)
	configureStock func(ctx context.Context, settings ConfigureStockSettings) error
}

func (m *mockService) CheckStock(ctx context.Context, variantID string, locationID string) (int, error) {
	if m.checkStock != nil {
		return m.checkStock(ctx, variantID, locationID)
	}
	return 0, nil
}

func (m *mockService) UpdateStock(ctx context.Context, variantID string, locationID string, quantity int) error {
	if m.updateStock != nil {
		return m.updateStock(ctx, variantID, locationID, quantity)
	}
	return nil
}

func (m *mockService) ListStock(ctx context.Context, filter ListStockFilter) ([]*domain.StockItem, error) {
	if m.listStock != nil {
		return m.listStock(ctx, filter)
	}
	return nil, nil
}

func (m *mockService) DeleteStock(ctx context.Context, variantID string, locationID string) error {
	if m.deleteStock != nil {
		return m.deleteStock(ctx, variantID, locationID)
	}
	return nil
}

func (m *mockService) ListAlerts(ctx context.Context, limit, offset int) ([]*domain.Alert, error) {
	if m.listAlerts != nil {
		return m.listAlerts(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockService) ConfigureStock(ctx context.Context, settings ConfigureStockSettings) error {
	if m.configureStock != nil {
		return m.configureStock(ctx, settings)
	}
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

func TestController_CheckStock_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful check stock", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			checkStock: func(_ context.Context, variantID string, locationID string) (int, error) {
				assert.Equal(t, "var_123", variantID)
				assert.Equal(t, "loc_a", locationID)
				return 42, nil
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/inventory/var_123?location_id=loc_a", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				VariantID  string `json:"variant_id"`
				LocationID string `json:"location_id"`
				Quantity   int    `json:"quantity"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.True(t, resp.Success)
		assert.Equal(t, "var_123", resp.Data.VariantID)
		assert.Equal(t, 42, resp.Data.Quantity)
	})

	t.Run("fails - service error", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			checkStock: func(_ context.Context, _, _ string) (int, error) {
				return 0, apperrors.NewNotFound("stock not found", nil)
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/inventory/var_nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestController_UpdateStock_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful update stock", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			updateStock: func(_ context.Context, variantID string, _ string, quantity int) error {
				assert.Equal(t, "var_123", variantID)
				assert.Equal(t, 10, quantity)
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := coredto.UpdateStockRequest{
			VariantID:  "var_123",
			LocationID: "default",
			Quantity:   intPtr(10),
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/update", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("fails - invalid payload", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{}
		router := setupTestRouter(mockSvc)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/update", bytes.NewBufferString("{invalid}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestController_ConfigureStock_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful configure stock", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			configureStock: func(_ context.Context, settings ConfigureStockSettings) error {
				assert.Equal(t, "var_123", settings.VariantID)
				assert.Equal(t, 5, *settings.LowStockThreshold)
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := coredto.ConfigureStockRequest{
			VariantID:         "var_123",
			LocationID:        "default",
			LowStockThreshold: intPtr(5),
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/configure", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestController_ListStock_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful list stock with default pagination", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			listStock: func(_ context.Context, filter ListStockFilter) ([]*domain.StockItem, error) {
				assert.Equal(t, int64(20), filter.Limit)
				assert.Equal(t, int64(0), filter.Offset)
				return []*domain.StockItem{
					{
						VariantID:         "var_1",
						LocationID:        "default",
						Quantity:          2,
						LowStockThreshold: 5,
					},
				}, nil
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/inventory?status=low_stock", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Items  []sharedDTO.StockResponse `json:"items"`
				Limit  int64                     `json:"limit"`
				Offset int64                     `json:"offset"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.True(t, resp.Success)
		require.Len(t, resp.Data.Items, 1)
		assert.Equal(t, "var_1", resp.Data.Items[0].VariantID)
		assert.True(t, resp.Data.Items[0].IsLowStock)
		assert.Equal(t, int64(20), resp.Data.Limit)
		assert.Equal(t, int64(0), resp.Data.Offset)
	})

	t.Run("successful list stock with custom pagination", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			listStock: func(_ context.Context, filter ListStockFilter) ([]*domain.StockItem, error) {
				assert.Equal(t, int64(5), filter.Limit)
				assert.Equal(t, int64(10), filter.Offset)
				return []*domain.StockItem{}, nil
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/inventory?limit=5&offset=10", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Items  []sharedDTO.StockResponse `json:"items"`
				Limit  int64                     `json:"limit"`
				Offset int64                     `json:"offset"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.True(t, resp.Success)
		assert.Equal(t, int64(5), resp.Data.Limit)
		assert.Equal(t, int64(10), resp.Data.Offset)
	})
}

func TestController_ListAlerts_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful list alerts", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			listAlerts: func(_ context.Context, _, _ int) ([]*domain.Alert, error) {
				return []*domain.Alert{
					{
						ID:        "alt_1",
						Type:      "LOW_STOCK",
						Message:   "below threshold",
						VariantID: "var_1",
					},
				}, nil
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/inventory/alerts", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestController_DeleteStock_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful delete stock", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			deleteStock: func(_ context.Context, variantID string, locationID string) error {
				assert.Equal(t, "var_123", variantID)
				assert.Equal(t, "loc_a", locationID)
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodDelete, "/api/inventory/var_123?location_id=loc_a", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ----------------------------------------------------
// END-TO-END TESTS (Full Controller + Service + DB Stack)
// ----------------------------------------------------

func TestInventoryOperationsFlow_E2E(t *testing.T) {
	t.Parallel()

	repo := &mockInventoryRepo{
		stocks: make(map[string]*domain.StockItem),
	}
	dispatcher := NewDashboardAlertDispatcher(repo)
	service := NewService(repo, dispatcher)
	router := setupTestRouter(service)

	variantID := "v-e2e-123"
	locationID := "default"

	// 1. Check stock initially (should return 200 with quantity 0 since not configured/created yet)
	req1, _ := http.NewRequest(http.MethodGet, "/api/inventory/"+variantID+"?location_id="+locationID, nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	var checkResp1 struct {
		Data struct {
			Quantity int `json:"quantity"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w1.Body.Bytes(), &checkResp1)
	assert.Equal(t, 0, checkResp1.Data.Quantity)

	// 2. Configure stock settings (initializes the stock item)
	cfgReq := coredto.ConfigureStockRequest{
		VariantID:         variantID,
		LocationID:        locationID,
		Quantity:          intPtr(10),
		LowStockThreshold: intPtr(5),
	}
	cfgBytes, _ := json.Marshal(cfgReq)
	req2, _ := http.NewRequest(http.MethodPost, "/api/inventory/configure", bytes.NewBuffer(cfgBytes))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Check stock now (should be 10)
	req3, _ := http.NewRequest(http.MethodGet, "/api/inventory/"+variantID+"?location_id="+locationID, nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	var checkResp struct {
		Data struct {
			Quantity int `json:"quantity"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w3.Body.Bytes(), &checkResp)
	assert.Equal(t, 10, checkResp.Data.Quantity)

	// 3. Update stock to a value below threshold (3) -> should trigger alert
	updReq := coredto.UpdateStockRequest{
		VariantID:  variantID,
		LocationID: locationID,
		Quantity:   intPtr(3),
	}
	updBytes, _ := json.Marshal(updReq)
	req4, _ := http.NewRequest(http.MethodPost, "/api/inventory/update", bytes.NewBuffer(updBytes))
	req4.Header.Set("Content-Type", "application/json")
	w4 := httptest.NewRecorder()
	router.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)

	// 4. GET inventory alerts via API
	req5, _ := http.NewRequest(http.MethodGet, "/api/inventory/alerts", nil)
	w5 := httptest.NewRecorder()
	router.ServeHTTP(w5, req5)
	assert.Equal(t, http.StatusOK, w5.Code)

	var alertResp struct {
		Data struct {
			Alerts []*domain.Alert `json:"alerts"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w5.Body.Bytes(), &alertResp)
	assert.Len(t, alertResp.Data.Alerts, 1)
	assert.Equal(t, variantID, alertResp.Data.Alerts[0].VariantID)

	// 5. DELETE the stock item
	req6, _ := http.NewRequest(http.MethodDelete, "/api/inventory/"+variantID+"?location_id="+locationID, nil)
	w6 := httptest.NewRecorder()
	router.ServeHTTP(w6, req6)
	assert.Equal(t, http.StatusOK, w6.Code)

	// Check stock again (should be 0 now after deletion)
	req7, _ := http.NewRequest(http.MethodGet, "/api/inventory/"+variantID+"?location_id="+locationID, nil)
	w7 := httptest.NewRecorder()
	router.ServeHTTP(w7, req7)
	assert.Equal(t, http.StatusOK, w7.Code)

	var checkResp2 struct {
		Data struct {
			Quantity int `json:"quantity"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w7.Body.Bytes(), &checkResp2)
	assert.Equal(t, 0, checkResp2.Data.Quantity)
}

func intPtr(val int) *int {
	return &val
}
