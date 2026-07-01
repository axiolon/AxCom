// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package adjustment

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecom-engine/internal/core/inventory/domain"
	adjdto "ecom-engine/internal/core/inventory/features/adjustment/dto"
	"ecom-engine/internal/events"
	apperrors "ecom-engine/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockService struct {
	adjustStock func(ctx context.Context, variantID string, locationID string, qty int, reason string) error
}

func (m *mockService) AdjustStock(ctx context.Context, variantID string, locationID string, qty int, reason string) error {
	if m.adjustStock != nil {
		return m.adjustStock(ctx, variantID, locationID, qty, reason)
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

func TestController_Adjust_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful adjustment", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			adjustStock: func(_ context.Context, variantID string, locationID string, qty int, reason string) error {
				assert.Equal(t, "var-123", variantID)
				assert.Equal(t, "loc-a", locationID)
				assert.Equal(t, 5, qty)
				assert.Equal(t, "audit", reason)
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := adjdto.AdjustRequest{
			LocationID: "loc-a",
			Adjustment: intPtr(5),
			Reason:     "audit",
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/var-123/adjust", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("fails - missing validation binding reason", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{}
		router := setupTestRouter(mockSvc)

		reqBody := map[string]interface{}{
			"location_id": "loc-a",
			"adjustment":  5,
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/var-123/adjust", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("fails - service error", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			adjustStock: func(_ context.Context, _ string, _ string, _ int, _ string) error {
				return apperrors.NewBadRequest("insufficient stock", nil)
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := adjdto.AdjustRequest{
			LocationID: "loc-a",
			Adjustment: intPtr(-10),
			Reason:     "shrinkage",
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/var-123/adjust", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// ----------------------------------------------------
// END-TO-END TESTS (Full Controller + Service + DB Stack)
// ----------------------------------------------------

func TestAdjustmentOperationsFlow_E2E(t *testing.T) {
	t.Parallel()

	repo := &mockAdjustmentRepo{stocks: make(map[string]*domain.StockItem)}
	bus := &mockEventBus{}
	service := NewService(repo, bus, nil)
	router := setupTestRouter(service)

	variantID := "v-adj-e2e"
	locationID := "default"

	// Create initial stock level
	_ = repo.SaveStock(context.Background(), &domain.StockItem{
		VariantID:  variantID,
		LocationID: locationID,
		Quantity:   20,
	})

	// POST adjustment (-5)
	reqBody := adjdto.AdjustRequest{
		LocationID: locationID,
		Adjustment: intPtr(-5),
		Reason:     "sales",
	}
	jsonBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/inventory/"+variantID+"/adjust", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify stock is now 15
	s, err := repo.GetStock(context.Background(), variantID, locationID)
	require.NoError(t, err)
	assert.Equal(t, 15, s.Quantity)

	// Verify event bus published event
	require.Len(t, bus.publishedEvents, 1)
	assert.Equal(t, events.InventoryStockChangedTopic, bus.publishedEvents[0].Topic)
}

func intPtr(val int) *int {
	return &val
}
