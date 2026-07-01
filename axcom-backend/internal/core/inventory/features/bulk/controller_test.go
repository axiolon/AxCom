// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package bulk

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecom-engine/internal/core/inventory/domain"
	bulkdto "ecom-engine/internal/core/inventory/features/bulk/dto"
	apperrors "ecom-engine/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockService struct {
	bulkUpdate func(ctx context.Context, updates []UpdateItem) error
}

func (m *mockService) BulkUpdate(ctx context.Context, updates []UpdateItem) error {
	if m.bulkUpdate != nil {
		return m.bulkUpdate(ctx, updates)
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
	RegisterRoutes(rg, ctrl, mockAuthMiddleware)
	return router
}

// ----------------------------------------------------
// INTEGRATION TESTS (Mocked Service)
// ----------------------------------------------------

func TestController_BulkUpdate_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful bulk update", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			bulkUpdate: func(_ context.Context, updates []UpdateItem) error {
				assert.Len(t, updates, 2)
				assert.Equal(t, "var-1", updates[0].VariantID)
				assert.Equal(t, 10, updates[0].Quantity)
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := bulkdto.BulkUpdateRequest{
			Items: []bulkdto.BulkUpdateItem{
				{VariantID: "var-1", LocationID: "loc-a", Quantity: intPtr(10)},
				{VariantID: "var-2", LocationID: "loc-b", Quantity: intPtr(20)},
			},
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/bulk-update", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("fails - bad request payload", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{}
		router := setupTestRouter(mockSvc)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/bulk-update", bytes.NewBufferString("{invalid_json}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("fails - service error", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			bulkUpdate: func(_ context.Context, _ []UpdateItem) error {
				return apperrors.NewInternal("db transaction failed", nil)
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := bulkdto.BulkUpdateRequest{
			Items: []bulkdto.BulkUpdateItem{
				{VariantID: "var-1", LocationID: "loc-a", Quantity: intPtr(10)},
			},
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/bulk-update", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ----------------------------------------------------
// END-TO-END TESTS (Full Controller + Service + DB Stack)
// ----------------------------------------------------

func TestBulkOperationsFlow_E2E(t *testing.T) {
	t.Parallel()

	repo := &mockBulkRepo{stocks: make(map[string]*domain.StockItem)}
	service := NewService(repo)
	router := setupTestRouter(service)

	// POST bulk updates
	reqBody := bulkdto.BulkUpdateRequest{
		Items: []bulkdto.BulkUpdateItem{
			{VariantID: "v-e2e-1", LocationID: "loc-a", Quantity: intPtr(5)},
			{VariantID: "v-e2e-2", LocationID: "loc-b", Quantity: intPtr(15)},
		},
	}
	jsonBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/inventory/bulk-update", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Validate changes in mock repository
	s1, err := repo.GetStock(context.Background(), "v-e2e-1", "loc-a")
	require.NoError(t, err)
	assert.Equal(t, 5, s1.Quantity)

	s2, err := repo.GetStock(context.Background(), "v-e2e-2", "loc-b")
	require.NoError(t, err)
	assert.Equal(t, 15, s2.Quantity)
}

func intPtr(val int) *int {
	return &val
}
