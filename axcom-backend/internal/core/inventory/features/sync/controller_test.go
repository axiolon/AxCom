// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecom-engine/internal/core/inventory/domain"
	syncdto "ecom-engine/internal/core/inventory/features/sync/dto"
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
	syncStock func(ctx context.Context, variantID string, locationID string, qty int) error
}

func (m *mockService) SyncStock(ctx context.Context, variantID string, locationID string, qty int) error {
	if m.syncStock != nil {
		return m.syncStock(ctx, variantID, locationID, qty)
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

func TestController_Sync_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful sync", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			syncStock: func(_ context.Context, variantID string, locationID string, qty int) error {
				assert.Equal(t, "var-123", variantID)
				assert.Equal(t, "loc-a", locationID)
				assert.Equal(t, 25, qty)
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := syncdto.SyncRequest{
			VariantID:  "var-123",
			LocationID: "loc-a",
			Quantity:   intPtr(25),
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/sync", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("fails - missing validation fields", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{}
		router := setupTestRouter(mockSvc)

		reqBody := map[string]interface{}{
			"location_id": "loc-a",
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/sync", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("fails - service error", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			syncStock: func(_ context.Context, _ string, _ string, _ int) error {
				return apperrors.NewInternal("sync update failed", nil)
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := syncdto.SyncRequest{
			VariantID:  "var-123",
			LocationID: "loc-a",
			Quantity:   intPtr(25),
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/sync", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ----------------------------------------------------
// END-TO-END TESTS (Full Controller + Service + DB Stack)
// ----------------------------------------------------

func TestSyncOperationsFlow_E2E(t *testing.T) {
	t.Parallel()

	repo := &mockSyncRepo{stocks: make(map[string]*domain.StockItem)}
	bus := &mockEventBus{}
	service := NewService(repo, bus, nil)
	router := setupTestRouter(service)

	variantID := "v-sync-e2e"
	locationID := "default"

	// Create initial stock level (10)
	_ = repo.SaveStock(context.Background(), &domain.StockItem{
		VariantID:  variantID,
		LocationID: locationID,
		Quantity:   10,
	})

	// POST sync (overwrite to 35)
	reqBody := syncdto.SyncRequest{
		VariantID:  variantID,
		LocationID: locationID,
		Quantity:   intPtr(35),
	}
	jsonBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/inventory/sync", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify stock is now 35
	s, err := repo.GetStock(context.Background(), variantID, locationID)
	require.NoError(t, err)
	assert.Equal(t, 35, s.Quantity)

	// Verify event bus published event
	require.Len(t, bus.publishedEvents, 1)
	assert.Equal(t, events.InventoryStockChangedTopic, bus.publishedEvents[0].Topic)
}

func intPtr(val int) *int {
	return &val
}
