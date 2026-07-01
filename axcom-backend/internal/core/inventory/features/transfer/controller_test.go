// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecom-engine/internal/core/inventory/domain"
	transferdto "ecom-engine/internal/core/inventory/features/transfer/dto"
	apperrors "ecom-engine/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockService struct {
	transferStock func(ctx context.Context, variantID string, fromLoc string, toLoc string, qty int) error
}

func (m *mockService) TransferStock(ctx context.Context, variantID string, fromLoc string, toLoc string, qty int) error {
	if m.transferStock != nil {
		return m.transferStock(ctx, variantID, fromLoc, toLoc, qty)
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

func TestController_Transfer_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful transfer", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			transferStock: func(_ context.Context, variantID string, fromLoc string, toLoc string, qty int) error {
				assert.Equal(t, "var-123", variantID)
				assert.Equal(t, "loc-a", fromLoc)
				assert.Equal(t, "loc-b", toLoc)
				assert.Equal(t, 5, qty)
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := transferdto.TransferRequest{
			VariantID:    "var-123",
			FromLocation: "loc-a",
			ToLocation:   "loc-b",
			Quantity:     intPtr(5),
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/transfer", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("fails - invalid payload binding", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{}
		router := setupTestRouter(mockSvc)

		reqBody := map[string]interface{}{
			"variant_id": "var-123",
			"quantity":   -5,
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/transfer", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("fails - service conflict error", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			transferStock: func(_ context.Context, _ string, _ string, _ string, _ int) error {
				return apperrors.NewConflict("insufficient stock", nil)
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := transferdto.TransferRequest{
			VariantID:    "var-123",
			FromLocation: "loc-a",
			ToLocation:   "loc-b",
			Quantity:     intPtr(15),
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/transfer", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

// ----------------------------------------------------
// END-TO-END TESTS (Full Controller + Service + DB Stack)
// ----------------------------------------------------

func TestTransferOperationsFlow_E2E(t *testing.T) {
	t.Parallel()

	repo := &mockTransferRepo{stocks: make(map[string]*domain.StockItem)}
	bus := &mockEventBus{}
	service := NewService(repo, bus, nil)
	router := setupTestRouter(service)

	variantID := "v-trsf-e2e"
	fromLoc := "loc-a"
	toLoc := "loc-b"

	// Set initial values
	_ = repo.SaveStock(context.Background(), &domain.StockItem{
		VariantID:  variantID,
		LocationID: fromLoc,
		Quantity:   20,
	})
	_ = repo.SaveStock(context.Background(), &domain.StockItem{
		VariantID:  variantID,
		LocationID: toLoc,
		Quantity:   5,
	})

	// POST transfer (5 items)
	reqBody := transferdto.TransferRequest{
		VariantID:    variantID,
		FromLocation: fromLoc,
		ToLocation:   toLoc,
		Quantity:     intPtr(5),
	}
	jsonBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/inventory/transfer", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Validate DB state (loc-a: 15, loc-b: 10)
	sA, err := repo.GetStock(context.Background(), variantID, fromLoc)
	require.NoError(t, err)
	assert.Equal(t, 15, sA.Quantity)

	sB, err := repo.GetStock(context.Background(), variantID, toLoc)
	require.NoError(t, err)
	assert.Equal(t, 10, sB.Quantity)

	// Validate dual events published
	require.Len(t, bus.publishedEvents, 2)
}

func intPtr(val int) *int {
	return &val
}
