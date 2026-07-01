// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ecom-engine/internal/core/orders"
	"ecom-engine/internal/core/orders/domain"
	"ecom-engine/pkg/ctxkeys"
	apperrors "ecom-engine/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockOrderService struct {
	createOrder       func(ctx context.Context, customerID string, customerSnapshot orders.OrderCustomerSnapshot, items []orders.OrderItem) (*orders.Order, error)
	transitionOrder   func(ctx context.Context, id string, action string) (*orders.Order, error)
	getOrder          func(ctx context.Context, id string) (*orders.Order, error)
	getCustomerOrders func(ctx context.Context, customerID string, limit, offset int) ([]orders.Order, error)
	getAllOrders      func(ctx context.Context, limit, offset int) ([]orders.Order, error)
}

func (m *mockOrderService) CreateOrder(ctx context.Context, customerID string, customerSnapshot orders.OrderCustomerSnapshot, items []orders.OrderItem) (*orders.Order, error) {
	if m.createOrder != nil {
		return m.createOrder(ctx, customerID, customerSnapshot, items)
	}
	return nil, nil
}

func (m *mockOrderService) TransitionOrder(ctx context.Context, id string, action string) (*orders.Order, error) {
	if m.transitionOrder != nil {
		return m.transitionOrder(ctx, id, action)
	}
	return nil, nil
}

func (m *mockOrderService) GetOrder(ctx context.Context, id string) (*orders.Order, error) {
	if m.getOrder != nil {
		return m.getOrder(ctx, id)
	}
	return nil, nil
}

func (m *mockOrderService) GetCustomerOrders(ctx context.Context, customerID string, limit, offset int) ([]orders.Order, error) {
	if m.getCustomerOrders != nil {
		return m.getCustomerOrders(ctx, customerID, limit, offset)
	}
	return nil, nil
}

func (m *mockOrderService) GetAllOrders(ctx context.Context, limit, offset int) ([]orders.Order, error) {
	if m.getAllOrders != nil {
		return m.getAllOrders(ctx, limit, offset)
	}
	return nil, nil
}

func setupTestRouter(svc orders.Service, userID string) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if userID != "" {
			c.Set(string(ctxkeys.UserIDKey), userID)
		}
		c.Next()
	})
	rg := router.Group("/api")
	RegisterUserRoutes(rg, svc)
	return router
}

func TestController_Create(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T, userID string) (*mockOrderService, *gin.Engine) {
		mockSvc := &mockOrderService{}
		router := setupTestRouter(mockSvc, userID)
		return mockSvc, router
	}

	t.Run("successful order creation", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "cust_123")

		mockSvc.createOrder = func(_ context.Context, customerID string, _ orders.OrderCustomerSnapshot, items []orders.OrderItem) (*orders.Order, error) {
			assert.Equal(t, "cust_123", customerID)
			assert.Len(t, items, 1)
			assert.Equal(t, "v_1", items[0].VariantID)
			return &orders.Order{
				ID:         "ord_999",
				CustomerID: customerID,
				Items:      items,
				Total:      25.00,
				Status:     orders.StatusPending,
				CreatedAt:  time.Now(),
			}, nil
		}

		reqBody := CreateOrderRequest{
			Items: []OrderItemRequest{
				{VariantID: "v_1", Quantity: 1, Price: 25.00},
			},
		}
		jsonBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodPost, "/api/orders", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "ord_999", data["id"])
		assert.Equal(t, float64(25), data["total"])
	})

	t.Run("fails - unauthorized when user context missing", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t, "") // empty user ID

		reqBody := CreateOrderRequest{
			Items: []OrderItemRequest{
				{VariantID: "v_1", Quantity: 1, Price: 25.00},
			},
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/orders", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("fails - invalid request JSON payload", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t, "cust_123")

		req, _ := http.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString("{invalid_json}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("fails - service returns empty order error", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "cust_123")
		mockSvc.createOrder = func(_ context.Context, _ string, _ orders.OrderCustomerSnapshot, _ []orders.OrderItem) (*orders.Order, error) {
			return nil, apperrors.NewBadRequest("order must contain at least one item", domain.ErrEmptyOrder)
		}

		reqBody := CreateOrderRequest{
			Items: []OrderItemRequest{},
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/orders", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestController_GetMyOrder(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T, userID string) (*mockOrderService, *gin.Engine) {
		mockSvc := &mockOrderService{}
		router := setupTestRouter(mockSvc, userID)
		return mockSvc, router
	}

	t.Run("successful order retrieval", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "cust_123")

		mockSvc.getOrder = func(_ context.Context, id string) (*orders.Order, error) {
			assert.Equal(t, "ord_1", id)
			return &orders.Order{
				ID:         "ord_1",
				CustomerID: "cust_123",
				Status:     orders.StatusPending,
			}, nil
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/orders/ord_1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("fails - unauthorized", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t, "")

		req, _ := http.NewRequest(http.MethodGet, "/api/orders/ord_1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("fails - forbidden ownership", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "cust_another")

		mockSvc.getOrder = func(_ context.Context, _ string) (*orders.Order, error) {
			return &orders.Order{
				ID:         "ord_1",
				CustomerID: "cust_123", // belongs to cust_123, not cust_another
			}, nil
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/orders/ord_1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("fails - order not found", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "cust_123")

		mockSvc.getOrder = func(_ context.Context, _ string) (*orders.Order, error) {
			return nil, apperrors.NewNotFound("order not found", domain.ErrOrderNotFound)
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/orders/ord_missing", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestController_ListMyOrders(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T, userID string) (*mockOrderService, *gin.Engine) {
		mockSvc := &mockOrderService{}
		router := setupTestRouter(mockSvc, userID)
		return mockSvc, router
	}

	t.Run("successful list orders", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "cust_123")

		mockSvc.getCustomerOrders = func(_ context.Context, customerID string, limit, offset int) ([]orders.Order, error) {
			assert.Equal(t, "cust_123", customerID)
			assert.Equal(t, 5, limit)
			assert.Equal(t, 2, offset)
			return []orders.Order{
				{ID: "ord_1", CustomerID: customerID, Status: orders.StatusPending},
				{ID: "ord_2", CustomerID: customerID, Status: orders.StatusPaid},
			}, nil
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/orders?limit=5&offset=2", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(2), data["count"])
	})

	t.Run("fails - unauthorized", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t, "")

		req, _ := http.NewRequest(http.MethodGet, "/api/orders", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("fails - service error", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "cust_123")
		mockSvc.getCustomerOrders = func(_ context.Context, _ string, _, _ int) ([]orders.Order, error) {
			return nil, errors.New("something went wrong")
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/orders", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestController_CancelMyOrder(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T, userID string) (*mockOrderService, *gin.Engine) {
		mockSvc := &mockOrderService{}
		router := setupTestRouter(mockSvc, userID)
		return mockSvc, router
	}

	t.Run("successful order cancellation", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "cust_123")

		mockSvc.getOrder = func(_ context.Context, id string) (*orders.Order, error) {
			assert.Equal(t, "ord_1", id)
			return &orders.Order{
				ID:         "ord_1",
				CustomerID: "cust_123",
				Status:     orders.StatusPaid,
			}, nil
		}

		mockSvc.transitionOrder = func(_ context.Context, id string, action string) (*orders.Order, error) {
			assert.Equal(t, "ord_1", id)
			assert.Equal(t, "cancel", action)
			return &orders.Order{
				ID:         "ord_1",
				CustomerID: "cust_123",
				Status:     orders.StatusCanceled,
			}, nil
		}

		req, _ := http.NewRequest(http.MethodPost, "/api/orders/ord_1/cancel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "ord_1", data["id"])
		assert.Equal(t, "canceled", data["status"])
	})

	t.Run("fails - unauthorized", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t, "")

		req, _ := http.NewRequest(http.MethodPost, "/api/orders/ord_1/cancel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("fails - forbidden ownership", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "cust_another")

		mockSvc.getOrder = func(_ context.Context, _ string) (*orders.Order, error) {
			return &orders.Order{
				ID:         "ord_1",
				CustomerID: "cust_123",
			}, nil
		}

		req, _ := http.NewRequest(http.MethodPost, "/api/orders/ord_1/cancel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("fails - transition error", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "cust_123")

		mockSvc.getOrder = func(_ context.Context, _ string) (*orders.Order, error) {
			return &orders.Order{
				ID:         "ord_1",
				CustomerID: "cust_123",
				Status:     orders.StatusShipped, // Cannot cancel shipped order
			}, nil
		}

		mockSvc.transitionOrder = func(_ context.Context, _, _ string) (*orders.Order, error) {
			return nil, apperrors.NewBadRequest("invalid state transition action", domain.ErrInvalidTransition)
		}

		req, _ := http.NewRequest(http.MethodPost, "/api/orders/ord_1/cancel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
