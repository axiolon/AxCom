// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package shipping

import (
	"bytes"
	"context"
	"ecom-engine/internal/core/orders"
	"ecom-engine/pkg/ctxkeys"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type MockShippingService struct {
	CalculateRatesFunc              func(ctx context.Context, req RateRequest) ([]RateResponse, error)
	CreateShipmentFunc              func(ctx context.Context, orderID string, carrier string, trackingNumber string, weight float64, value float64) (*Shipment, error)
	UpdateShipmentStatusFunc        func(ctx context.Context, id string, status ShipmentStatus, trackingNumber string) (*Shipment, error)
	GetShipmentByOrderIDFunc        func(ctx context.Context, orderID string) (*Shipment, error)
	ListAllShipmentsFunc            func(ctx context.Context, limit, offset int) ([]Shipment, error)
	GetShipmentByTrackingNumberFunc func(ctx context.Context, trackingNumber string) (*Shipment, error)
	TrackShipmentFunc               func(ctx context.Context, trackingNumber string) (*Shipment, error)
	DeleteShipmentFunc              func(ctx context.Context, id string) error
}

type MockOrderService struct {
	GetOrderFunc func(ctx context.Context, id string) (*orders.Order, error)
}

func (m *MockOrderService) GetOrder(ctx context.Context, id string) (*orders.Order, error) {
	if m.GetOrderFunc != nil {
		return m.GetOrderFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockShippingService) CalculateRates(ctx context.Context, req RateRequest) ([]RateResponse, error) {
	if m.CalculateRatesFunc != nil {
		return m.CalculateRatesFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockShippingService) CreateShipment(ctx context.Context, orderID string, carrier string, trackingNumber string, weight float64, value float64) (*Shipment, error) {
	if m.CreateShipmentFunc != nil {
		return m.CreateShipmentFunc(ctx, orderID, carrier, trackingNumber, weight, value)
	}
	return nil, nil
}

func (m *MockShippingService) UpdateShipmentStatus(ctx context.Context, id string, status ShipmentStatus, trackingNumber string) (*Shipment, error) {
	if m.UpdateShipmentStatusFunc != nil {
		return m.UpdateShipmentStatusFunc(ctx, id, status, trackingNumber)
	}
	return nil, nil
}

func (m *MockShippingService) GetShipmentByOrderID(ctx context.Context, orderID string) (*Shipment, error) {
	if m.GetShipmentByOrderIDFunc != nil {
		return m.GetShipmentByOrderIDFunc(ctx, orderID)
	}
	return nil, nil
}

func (m *MockShippingService) ListAllShipments(ctx context.Context, limit, offset int) ([]Shipment, error) {
	if m.ListAllShipmentsFunc != nil {
		return m.ListAllShipmentsFunc(ctx, limit, offset)
	}
	return nil, nil
}

func (m *MockShippingService) GetShipmentByTrackingNumber(ctx context.Context, trackingNumber string) (*Shipment, error) {
	if m.GetShipmentByTrackingNumberFunc != nil {
		return m.GetShipmentByTrackingNumberFunc(ctx, trackingNumber)
	}
	return nil, nil
}

func (m *MockShippingService) TrackShipment(ctx context.Context, trackingNumber string) (*Shipment, error) {
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

func setupTestRouter(ctrl *Controller, userID string) *gin.Engine {
	router := gin.New()
	rg := router.Group("/api")

	authMiddleware := func(c *gin.Context) {
		if userID != "" {
			c.Set(string(ctxkeys.UserIDKey), userID)
		}
		c.Next()
	}

	RegisterRoutes(rg, ctrl, authMiddleware)
	return router
}

func TestController_CalculateRates(t *testing.T) {
	t.Parallel()

	t.Run("successful rate calculation", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			CalculateRatesFunc: func(_ context.Context, _ RateRequest) ([]RateResponse, error) {
				return []RateResponse{
					{ProviderName: "Test Carrier", Rate: 12.34},
				}, nil
			},
		}
		ctrl := NewController(svc, &MockOrderService{})
		router := setupTestRouter(ctrl, "")

		reqBody := RateRequest{
			Weight: 5.5,
			Value:  100.0,
		}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/shipping/rates", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		rates := resp["data"].([]interface{})
		assert.Len(t, rates, 1)
		rateMap := rates[0].(map[string]interface{})
		assert.Equal(t, "Test Carrier", rateMap["provider_name"])
		assert.Equal(t, 12.34, rateMap["rate"])
	})

	t.Run("validation error - missing weight", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{}
		ctrl := NewController(svc, &MockOrderService{})
		router := setupTestRouter(ctrl, "")

		// weight is required binding, so 0 or missing will fail validation
		reqBody := map[string]interface{}{
			"value": 100.0,
		}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/shipping/rates", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service failure error", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			CalculateRatesFunc: func(_ context.Context, _ RateRequest) ([]RateResponse, error) {
				return nil, errors.New("service failure")
			},
		}
		ctrl := NewController(svc, &MockOrderService{})
		router := setupTestRouter(ctrl, "")

		reqBody := RateRequest{
			Weight: 5.5,
			Value:  100.0,
		}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/shipping/rates", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestController_GetMyOrderShipment(t *testing.T) {
	t.Parallel()

	t.Run("successful get shipment", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			GetShipmentByOrderIDFunc: func(_ context.Context, orderID string) (*Shipment, error) {
				return &Shipment{
					ID:             "shpm_123",
					OrderID:        orderID,
					Carrier:        "Carrier A",
					TrackingNumber: "TRK123456",
					Status:         StatusInTransit,
				}, nil
			},
		}
		mockOrderSvc := &MockOrderService{
			GetOrderFunc: func(_ context.Context, id string) (*orders.Order, error) {
				return &orders.Order{
					ID:         id,
					CustomerID: "cust_123",
				}, nil
			},
		}
		ctrl := NewController(svc, mockOrderSvc)
		router := setupTestRouter(ctrl, "cust_123")

		req, _ := http.NewRequest("GET", "/api/shipping/order/order_999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		shipment := resp["data"].(map[string]interface{})
		assert.Equal(t, "shpm_123", shipment["id"])
		assert.Equal(t, "order_999", shipment["order_id"])
	})

	t.Run("unauthorized request", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{}
		ctrl := NewController(svc, &MockOrderService{})
		router := setupTestRouter(ctrl, "") // Empty userID

		req, _ := http.NewRequest("GET", "/api/shipping/order/order_999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("shipment not found", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			GetShipmentByOrderIDFunc: func(_ context.Context, _ string) (*Shipment, error) {
				return nil, ErrShipmentNotFound
			},
		}
		ctrl := NewController(svc, &MockOrderService{})
		router := setupTestRouter(ctrl, "cust_123")

		req, _ := http.NewRequest("GET", "/api/shipping/order/order_missing", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("forbidden - order customer ID mismatch", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			GetShipmentByOrderIDFunc: func(_ context.Context, orderID string) (*Shipment, error) {
				return &Shipment{
					ID:      "shpm_123",
					OrderID: orderID,
				}, nil
			},
		}
		mockOrderSvc := &MockOrderService{
			GetOrderFunc: func(_ context.Context, id string) (*orders.Order, error) {
				return &orders.Order{
					ID:         id,
					CustomerID: "another_cust",
				}, nil
			},
		}
		ctrl := NewController(svc, mockOrderSvc)
		router := setupTestRouter(ctrl, "cust_123")

		req, _ := http.NewRequest("GET", "/api/shipping/order/order_999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("associated order not found", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			GetShipmentByOrderIDFunc: func(_ context.Context, orderID string) (*Shipment, error) {
				return &Shipment{
					ID:      "shpm_123",
					OrderID: orderID,
				}, nil
			},
		}
		mockOrderSvc := &MockOrderService{
			GetOrderFunc: func(_ context.Context, _ string) (*orders.Order, error) {
				return nil, errors.New("order DB error")
			},
		}
		ctrl := NewController(svc, mockOrderSvc)
		router := setupTestRouter(ctrl, "cust_123")

		req, _ := http.NewRequest("GET", "/api/shipping/order/order_999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestController_TrackShipment(t *testing.T) {
	t.Parallel()

	t.Run("successful public tracking lookup", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			TrackShipmentFunc: func(_ context.Context, trackingNumber string) (*Shipment, error) {
				return &Shipment{
					ID:             "shpm_123",
					OrderID:        "order_999",
					Carrier:        "USPS",
					TrackingNumber: trackingNumber,
					Status:         StatusInTransit,
				}, nil
			},
		}
		ctrl := NewController(svc, &MockOrderService{})
		router := setupTestRouter(ctrl, "") // public, no auth

		req, _ := http.NewRequest("GET", "/api/shipping/track/TRK123456", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "TRK123456", data["tracking_number"])
		assert.Equal(t, "USPS", data["carrier"])
		assert.Equal(t, "in_transit", data["status"])
		// Verify sensitive fields like order_id are not exposed
		assert.Nil(t, data["order_id"])
	})

	t.Run("shipment not found", func(t *testing.T) {
		t.Parallel()
		svc := &MockShippingService{
			TrackShipmentFunc: func(_ context.Context, _ string) (*Shipment, error) {
				return nil, ErrTrackingNumberNotFound
			},
		}
		ctrl := NewController(svc, &MockOrderService{})
		router := setupTestRouter(ctrl, "")

		req, _ := http.NewRequest("GET", "/api/shipping/track/NOTFOUND", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
