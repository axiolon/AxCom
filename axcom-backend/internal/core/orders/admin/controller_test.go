// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecom-engine/internal/core/orders"
	"ecom-engine/internal/core/orders/domain"
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

func setupTestRouter(svc orders.Service) *gin.Engine {
	router := gin.New()
	rg := router.Group("/api")
	RegisterAdminRoutes(rg, svc)
	return router
}

func TestController_ListAllOrders(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockOrderService, *gin.Engine) {
		mockSvc := &mockOrderService{}
		router := setupTestRouter(mockSvc)
		return mockSvc, router
	}

	t.Run("successful list orders with guest mapping", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		mockSvc.getAllOrders = func(_ context.Context, limit, offset int) ([]orders.Order, error) {
			assert.Equal(t, 10, limit)
			assert.Equal(t, 5, offset)
			return []orders.Order{
				{ID: "ord_1", CustomerID: "cust_1", Status: orders.StatusPending},
				{
					ID:         "ord_gst_1",
					CustomerID: "",
					Status:     orders.StatusPending,
					CustomerSnapshot: orders.OrderCustomerSnapshot{
						Name:          "John Doe",
						Email:         "john@example.com",
						ContactNumber: "99999",
					},
				},
			}, nil
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/admin/orders?limit=10&offset=5", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		ordersList := data["orders"].([]interface{})
		assert.Len(t, ordersList, 2)

		// Check guest customer mapping
		order1 := ordersList[0].(map[string]interface{})
		assert.Equal(t, "ord_1", order1["id"])
		assert.Nil(t, order1["guest_info"])

		order2 := ordersList[1].(map[string]interface{})
		assert.Equal(t, "ord_gst_1", order2["id"])
		require.NotNil(t, order2["guest_info"])
		gInfo := order2["guest_info"].(map[string]interface{})
		assert.Equal(t, "John Doe", gInfo["name"])
	})
}

func TestController_GetOrder(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockOrderService, *gin.Engine) {
		mockSvc := &mockOrderService{}
		router := setupTestRouter(mockSvc)
		return mockSvc, router
	}

	t.Run("successful order retrieve by admin", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		mockSvc.getOrder = func(_ context.Context, id string) (*orders.Order, error) {
			assert.Equal(t, "ord_1", id)
			return &orders.Order{
				ID:     "ord_1",
				Status: orders.StatusPending,
			}, nil
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/admin/orders/ord_1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("fails - order not found", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		mockSvc.getOrder = func(_ context.Context, _ string) (*orders.Order, error) {
			return nil, apperrors.NewNotFound("order not found", domain.ErrOrderNotFound)
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/admin/orders/ord_missing", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestController_TransitionOrder(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockOrderService, *gin.Engine) {
		mockSvc := &mockOrderService{}
		router := setupTestRouter(mockSvc)
		return mockSvc, router
	}

	t.Run("successful state transition", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		mockSvc.transitionOrder = func(_ context.Context, id string, action string) (*orders.Order, error) {
			assert.Equal(t, "ord_1", id)
			assert.Equal(t, "pay", action)
			return &orders.Order{
				ID:     "ord_1",
				Status: orders.StatusPaid,
			}, nil
		}

		reqBody := TransitionRequest{
			Action: "pay",
		}
		jsonBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodPost, "/api/admin/orders/ord_1/transition", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "paid", data["status"])
	})

	t.Run("fails - invalid request payload", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t)

		req, _ := http.NewRequest(http.MethodPost, "/api/admin/orders/ord_1/transition", bytes.NewBufferString("{bad_json}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("fails - logic transition error", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		mockSvc.transitionOrder = func(_ context.Context, _, _ string) (*orders.Order, error) {
			return nil, apperrors.NewBadRequest("invalid state transition action", domain.ErrInvalidTransition)
		}

		reqBody := TransitionRequest{
			Action: "complete",
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/admin/orders/ord_1/transition", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
