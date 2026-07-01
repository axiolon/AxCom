// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package admin

import (
	"bytes"
	"context"
	"ecom-engine/internal/core/shipping"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type MockShippingService struct {
	CalculateRatesFunc              func(ctx context.Context, req shipping.RateRequest) ([]shipping.RateResponse, error)
	CreateShipmentFunc              func(ctx context.Context, orderID string, carrier string, trackingNumber string, weight float64, value float64) (*shipping.Shipment, error)
	UpdateShipmentStatusFunc        func(ctx context.Context, id string, status shipping.ShipmentStatus, trackingNumber string) (*shipping.Shipment, error)
	GetShipmentByOrderIDFunc        func(ctx context.Context, orderID string) (*shipping.Shipment, error)
	ListAllShipmentsFunc            func(ctx context.Context, limit, offset int) ([]shipping.Shipment, error)
	GetShipmentByTrackingNumberFunc func(ctx context.Context, trackingNumber string) (*shipping.Shipment, error)
	TrackShipmentFunc               func(ctx context.Context, trackingNumber string) (*shipping.Shipment, error)
	DeleteShipmentFunc              func(ctx context.Context, id string) error
}

func (m *MockShippingService) CalculateRates(ctx context.Context, req shipping.RateRequest) ([]shipping.RateResponse, error) {
	if m.CalculateRatesFunc != nil {
		return m.CalculateRatesFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockShippingService) CreateShipment(ctx context.Context, orderID string, carrier string, trackingNumber string, weight float64, value float64) (*shipping.Shipment, error) {
	if m.CreateShipmentFunc != nil {
		return m.CreateShipmentFunc(ctx, orderID, carrier, trackingNumber, weight, value)
	}
	return nil, nil
}

func (m *MockShippingService) UpdateShipmentStatus(ctx context.Context, id string, status shipping.ShipmentStatus, trackingNumber string) (*shipping.Shipment, error) {
	if m.UpdateShipmentStatusFunc != nil {
		return m.UpdateShipmentStatusFunc(ctx, id, status, trackingNumber)
	}
	return nil, nil
}

func (m *MockShippingService) GetShipmentByOrderID(ctx context.Context, orderID string) (*shipping.Shipment, error) {
	if m.GetShipmentByOrderIDFunc != nil {
		return m.GetShipmentByOrderIDFunc(ctx, orderID)
	}
	return nil, nil
}

func (m *MockShippingService) ListAllShipments(ctx context.Context, limit, offset int) ([]shipping.Shipment, error) {
	if m.ListAllShipmentsFunc != nil {
		return m.ListAllShipmentsFunc(ctx, limit, offset)
	}
	return nil, nil
}

func (m *MockShippingService) GetShipmentByTrackingNumber(ctx context.Context, trackingNumber string) (*shipping.Shipment, error) {
	if m.GetShipmentByTrackingNumberFunc != nil {
		return m.GetShipmentByTrackingNumberFunc(ctx, trackingNumber)
	}
	return nil, nil
}

func (m *MockShippingService) TrackShipment(ctx context.Context, trackingNumber string) (*shipping.Shipment, error) {
	if m.TrackShipmentFunc != nil {
		return m.TrackShipmentFunc(ctx, trackingNumber)
	}
	return nil, nil
}

func (m *MockShippingService) DeleteShipment(ctx context.Context, id string) error {
	if m.DeleteShipmentFunc != nil {
		return m.DeleteShipmentFunc(ctx, id)
	}
	return nil
}

func setupTestRouter(ctrl *Controller) *gin.Engine {
	router := gin.New()
	rg := router.Group("/api")
	RegisterAdminRoutes(rg, ctrl)
	return router
}

func TestController_ListShipments(t *testing.T) {
	t.Parallel()

	t.Run("successful list", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			ListAllShipmentsFunc: func(_ context.Context, _, _ int) ([]shipping.Shipment, error) {
				return []shipping.Shipment{
					{ID: "shpm_1", OrderID: "ord_1", Carrier: "Carrier A", Status: shipping.StatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()},
					{ID: "shpm_2", OrderID: "ord_2", Carrier: "Carrier B", Status: shipping.StatusInTransit, CreatedAt: time.Now(), UpdatedAt: time.Now()},
				}, nil
			},
		}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl)

		req, _ := http.NewRequest("GET", "/api/admin/shipping", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(2), data["count"])
		shipments := data["shipments"].([]interface{})
		assert.Len(t, shipments, 2)
	})

	t.Run("service failure", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			ListAllShipmentsFunc: func(_ context.Context, _, _ int) ([]shipping.Shipment, error) {
				return nil, errors.New("database connection lost")
			},
		}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl)

		req, _ := http.NewRequest("GET", "/api/admin/shipping", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestController_CreateShipment(t *testing.T) {
	t.Parallel()

	t.Run("successful creation", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			CreateShipmentFunc: func(_ context.Context, orderID string, carrier string, trackingNumber string, weight float64, value float64) (*shipping.Shipment, error) {
				return &shipping.Shipment{
					ID:             "shpm_created",
					OrderID:        orderID,
					Carrier:        carrier,
					TrackingNumber: trackingNumber,
					Status:         shipping.StatusPending,
					Weight:         weight,
					Value:          value,
					CreatedAt:      time.Now(),
					UpdatedAt:      time.Now(),
				}, nil
			},
		}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl)

		reqBody := CreateShipmentRequest{
			OrderID:        "ord_123",
			Carrier:        "Carrier X",
			TrackingNumber: "TRK123",
			Weight:         10.5,
			Value:          150.0,
		}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/admin/shipping", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		shipment := resp["data"].(map[string]interface{})
		assert.Equal(t, "shpm_created", shipment["id"])
		assert.Equal(t, "ord_123", shipment["order_id"])
	})

	t.Run("validation error - missing carrier", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl)

		reqBody := map[string]interface{}{
			"order_id": "ord_123",
			"weight":   10.5,
		}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/admin/shipping", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service failure", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			CreateShipmentFunc: func(_ context.Context, _ string, _ string, _ string, _ float64, _ float64) (*shipping.Shipment, error) {
				return nil, errors.New("cannot write record")
			},
		}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl)

		reqBody := CreateShipmentRequest{
			OrderID:        "ord_123",
			Carrier:        "Carrier X",
			TrackingNumber: "TRK123",
			Weight:         10.5,
			Value:          150.0,
		}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/admin/shipping", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestController_UpdateShipmentStatus(t *testing.T) {
	t.Parallel()

	t.Run("successful update", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			UpdateShipmentStatusFunc: func(_ context.Context, id string, status shipping.ShipmentStatus, trackingNumber string) (*shipping.Shipment, error) {
				return &shipping.Shipment{
					ID:             id,
					Status:         status,
					TrackingNumber: trackingNumber,
					UpdatedAt:      time.Now(),
				}, nil
			},
		}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl)

		reqBody := UpdateShipmentRequest{
			Status:         string(shipping.StatusDelivered),
			TrackingNumber: "TRK456",
		}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/admin/shipping/shpm_1", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		shipment := resp["data"].(map[string]interface{})
		assert.Equal(t, "shpm_1", shipment["id"])
		assert.Equal(t, "delivered", shipment["status"])
	})

	t.Run("validation error - missing status", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl)

		reqBody := map[string]interface{}{
			"tracking_number": "TRK456",
		}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/admin/shipping/shpm_1", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("shipment not found", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			UpdateShipmentStatusFunc: func(_ context.Context, _ string, _ shipping.ShipmentStatus, _ string) (*shipping.Shipment, error) {
				return nil, shipping.ErrShipmentNotFound
			},
		}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl)

		reqBody := UpdateShipmentRequest{
			Status: "delivered",
		}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/admin/shipping/shpm_nonexistent", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("service failure", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			UpdateShipmentStatusFunc: func(_ context.Context, _ string, _ shipping.ShipmentStatus, _ string) (*shipping.Shipment, error) {
				return nil, errors.New("update error")
			},
		}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl)

		reqBody := UpdateShipmentRequest{
			Status: "delivered",
		}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/admin/shipping/shpm_1", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestController_DeleteShipment(t *testing.T) {
	t.Parallel()

	t.Run("successful deletion", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			DeleteShipmentFunc: func(_ context.Context, _ string) error {
				return nil
			},
		}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl)

		req, _ := http.NewRequest("DELETE", "/api/admin/shipping/shpm_1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("shipment not found", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			DeleteShipmentFunc: func(_ context.Context, _ string) error {
				return shipping.ErrShipmentNotFound
			},
		}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl)

		req, _ := http.NewRequest("DELETE", "/api/admin/shipping/shpm_nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
