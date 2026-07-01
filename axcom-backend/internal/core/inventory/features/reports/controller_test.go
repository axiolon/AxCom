// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecom-engine/internal/core/inventory/domain"
	apperrors "ecom-engine/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockService struct {
	getLowStockReport  func(ctx context.Context) ([]*domain.StockItem, error)
	exportInventoryCSV func(ctx context.Context) ([]byte, error)
}

func (m *mockService) GetLowStockReport(ctx context.Context) ([]*domain.StockItem, error) {
	if m.getLowStockReport != nil {
		return m.getLowStockReport(ctx)
	}
	return nil, nil
}

func (m *mockService) ExportInventoryCSV(ctx context.Context) ([]byte, error) {
	if m.exportInventoryCSV != nil {
		return m.exportInventoryCSV(ctx)
	}
	return nil, nil
}

func setupTestRouter(svc Service) *gin.Engine {
	router := gin.New()
	rg := router.Group("/api")
	mockAuthMiddleware := func(c *gin.Context) {
		c.Next()
	}
	ctrl := NewController(svc)
	RegisterRoutes(rg, ctrl, mockAuthMiddleware)
	return router
}

// ----------------------------------------------------
// INTEGRATION TESTS (Mocked Service)
// ----------------------------------------------------

func TestController_LowStock_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful low stock list retrieval", func(t *testing.T) {
		t.Parallel()
		expectedItems := []*domain.StockItem{
			{VariantID: "var-1", LocationID: "default", Quantity: 2},
		}
		mockSvc := &mockService{
			getLowStockReport: func(_ context.Context) ([]*domain.StockItem, error) {
				return expectedItems, nil
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/inventory/low-stock", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("fails - service error", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			getLowStockReport: func(_ context.Context) ([]*domain.StockItem, error) {
				return nil, apperrors.NewInternal("service down", nil)
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/inventory/low-stock", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestController_Export_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful CSV export download", func(t *testing.T) {
		t.Parallel()
		expectedCSV := []byte("Variant ID,Location ID,Quantity,Low Stock Threshold,Allow Backorders,Backorder Limit\nvar-1,loc-a,10,5,true,20\n")
		mockSvc := &mockService{
			exportInventoryCSV: func(_ context.Context) ([]byte, error) {
				return expectedCSV, nil
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/inventory/export", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
		assert.Equal(t, "attachment; filename=inventory.csv", w.Header().Get("Content-Disposition"))
		assert.Equal(t, expectedCSV, w.Body.Bytes())
	})

	t.Run("fails - service error", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			exportInventoryCSV: func(_ context.Context) ([]byte, error) {
				return nil, errors.New("CSV writing error")
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/inventory/export", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ----------------------------------------------------
// END-TO-END TESTS (Full Controller + Service + DB Stack)
// ----------------------------------------------------

func TestReportsOperationsFlow_E2E(t *testing.T) {
	t.Parallel()

	repo := &mockReportsRepo{
		lowStockItems: []*domain.StockItem{
			{VariantID: "var-low", LocationID: "default", Quantity: 1, LowStockThreshold: 5},
		},
		allStockItems: []*domain.StockItem{
			{VariantID: "var-low", LocationID: "default", Quantity: 1, LowStockThreshold: 5},
			{VariantID: "var-high", LocationID: "default", Quantity: 100, LowStockThreshold: 5},
		},
	}

	service := NewService(repo)
	router := setupTestRouter(service)

	// 1. GET low stock report
	req1, _ := http.NewRequest(http.MethodGet, "/api/inventory/low-stock", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	// 2. GET export CSV download
	req2, _ := http.NewRequest(http.MethodGet, "/api/inventory/export", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "text/csv", w2.Header().Get("Content-Type"))
	csvData := w2.Body.String()
	require.Contains(t, csvData, "var-low")
	require.Contains(t, csvData, "var-high")
}
