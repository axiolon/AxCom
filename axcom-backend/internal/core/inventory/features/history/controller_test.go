// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ecom-engine/internal/core/inventory/domain"
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
	getHistory    func(ctx context.Context, variantID string, limit, offset int) ([]*domain.StockHistory, error)
	recordHistory func(ctx context.Context, h *domain.StockHistory) error
}

func (m *mockService) GetHistory(ctx context.Context, variantID string, limit, offset int) ([]*domain.StockHistory, error) {
	if m.getHistory != nil {
		return m.getHistory(ctx, variantID, limit, offset)
	}
	return nil, nil
}

func (m *mockService) RecordHistory(ctx context.Context, h *domain.StockHistory) error {
	if m.recordHistory != nil {
		return m.recordHistory(ctx, h)
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

func TestController_History_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful history retrieval", func(t *testing.T) {
		t.Parallel()
		expectedHist := []*domain.StockHistory{
			{ID: "hist_1", VariantID: "var_123", NewQuantity: 5},
		}
		mockSvc := &mockService{
			getHistory: func(_ context.Context, variantID string, _ int, _ int) ([]*domain.StockHistory, error) {
				assert.Equal(t, "var_123", variantID)
				return expectedHist, nil
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/inventory/var_123/history", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				History []*domain.StockHistory `json:"history"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.True(t, resp.Success)
		require.Len(t, resp.Data.History, 1)
		assert.Equal(t, "hist_1", resp.Data.History[0].ID)
	})

	t.Run("fails - service error", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			getHistory: func(_ context.Context, _ string, _ int, _ int) ([]*domain.StockHistory, error) {
				return nil, apperrors.NewInternal("db failure", nil)
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/inventory/var_error/history", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ----------------------------------------------------
// END-TO-END TESTS (Full Controller + Service + DB Stack)
// ----------------------------------------------------

func TestHistoryOperationsFlow_E2E(t *testing.T) {
	t.Parallel()

	repo := &mockHistoryRepo{}
	bus := events.NewLocalEventBus()
	service := NewService(repo, bus)
	router := setupTestRouter(service)

	variantID := "v-hist-e2e"

	// 1. Fetch history initially (should be empty)
	req1, _ := http.NewRequest(http.MethodGet, "/api/inventory/"+variantID+"/history", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	var resp1 struct {
		Data struct {
			History []*domain.StockHistory `json:"history"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
	assert.Empty(t, resp1.Data.History)

	// 2. Publish stock changed event
	payload := &events.StockChangedPayload{
		VariantID:    variantID,
		LocationID:   "default",
		OldQuantity:  10,
		NewQuantity:  15,
		ChangeReason: "Restock",
		ChangedBy:    "admin-user",
	}
	event := events.NewEvent(events.InventoryStockChangedTopic, "inventory-service", payload)
	bus.Publish(event)

	// Sleep slightly to let asynchronous subscription goroutine complete
	time.Sleep(50 * time.Millisecond)

	// 3. Fetch history again (should contain 1 record)
	req2, _ := http.NewRequest(http.MethodGet, "/api/inventory/"+variantID+"/history", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	var resp2 struct {
		Data struct {
			History []*domain.StockHistory `json:"history"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)

	require.Len(t, resp2.Data.History, 1)
	assert.Equal(t, variantID, resp2.Data.History[0].VariantID)
	assert.Equal(t, 10, resp2.Data.History[0].OldQuantity)
	assert.Equal(t, 15, resp2.Data.History[0].NewQuantity)
	assert.Equal(t, "Restock", resp2.Data.History[0].ChangeReason)
}
