// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reservation

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ecom-engine/internal/core/inventory/domain"
	resdto "ecom-engine/internal/core/inventory/features/reservation/dto"
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
	reserveStock       func(ctx context.Context, variantID string, locationID string, quantity int) (*domain.Reservation, error)
	releaseReservation func(ctx context.Context, reservationID string) error
}

func (m *mockService) ReserveStock(ctx context.Context, variantID string, locationID string, quantity int) (*domain.Reservation, error) {
	if m.reserveStock != nil {
		return m.reserveStock(ctx, variantID, locationID, quantity)
	}
	return nil, nil
}

func (m *mockService) ReleaseReservation(ctx context.Context, reservationID string) error {
	if m.releaseReservation != nil {
		return m.releaseReservation(ctx, reservationID)
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

func TestController_Reserve_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful reservation", func(t *testing.T) {
		t.Parallel()
		expectedRes := &domain.Reservation{
			ID:         "res-123",
			VariantID:  "var-123",
			LocationID: "loc-a",
			Quantity:   2,
			ExpiresAt:  time.Now().Add(15 * time.Minute),
		}
		mockSvc := &mockService{
			reserveStock: func(_ context.Context, variantID string, locationID string, quantity int) (*domain.Reservation, error) {
				assert.Equal(t, "var-123", variantID)
				assert.Equal(t, "loc-a", locationID)
				assert.Equal(t, 2, quantity)
				return expectedRes, nil
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := resdto.ReserveRequest{
			LocationID: "loc-a",
			Quantity:   2,
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/var-123/reserve", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("fails - bad quantity constraint in payload", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{}
		router := setupTestRouter(mockSvc)

		reqBody := resdto.ReserveRequest{
			LocationID: "loc-a",
			Quantity:   0, // minimum is 1
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/inventory/var-123/reserve", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestController_ReleaseReservation_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful release", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			releaseReservation: func(_ context.Context, reservationID string) error {
				assert.Equal(t, "res-123", reservationID)
				return nil
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodDelete, "/api/inventory/var-123/reserve/res-123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("fails - service error", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			releaseReservation: func(_ context.Context, _ string) error {
				return apperrors.NewNotFound("reservation not found", nil)
			},
		}

		router := setupTestRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodDelete, "/api/inventory/var-123/reserve/res-nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// ----------------------------------------------------
// END-TO-END TESTS (Full Controller + Service + DB Stack)
// ----------------------------------------------------

func TestReservationOperationsFlow_E2E(t *testing.T) {
	t.Parallel()

	repo := &mockReservationRepo{
		stocks: make(map[string]*domain.StockItem),
		res:    make(map[string]*domain.Reservation),
	}
	bus := events.NewLocalEventBus()
	service := NewService(repo, bus, nil)
	router := setupTestRouter(service)

	variantID := "v-res-e2e"
	locationID := "default"

	// Initialize database state directly in repository
	repo.stocks[variantID+":"+locationID] = &domain.StockItem{
		VariantID:         variantID,
		LocationID:        locationID,
		Quantity:          10,
		LowStockThreshold: 2,
		AllowBackorders:   false,
		BackorderLimit:    0,
	}

	// 1. Reserve 3 items via HTTP POST
	reqBody := resdto.ReserveRequest{
		LocationID: locationID,
		Quantity:   3,
	}
	jsonBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/inventory/"+variantID+"/reserve", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var reserveResp struct {
		Success bool `json:"success"`
		Data    struct {
			ReservationID string `json:"reservation_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &reserveResp)
	assert.NotEmpty(t, reserveResp.Data.ReservationID)

	// Verify quantity decr in DB (10 - 3 = 7)
	s, err := repo.GetStock(context.Background(), variantID, locationID)
	require.NoError(t, err)
	assert.Equal(t, 7, s.Quantity)

	// 2. Release reservation via HTTP DELETE
	req2, _ := http.NewRequest(http.MethodDelete, "/api/inventory/"+variantID+"/reserve/"+reserveResp.Data.ReservationID, nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	// Verify quantity restored in DB (7 + 3 = 10)
	s2, err := repo.GetStock(context.Background(), variantID, locationID)
	require.NoError(t, err)
	assert.Equal(t, 10, s2.Quantity)
}
