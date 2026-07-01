// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package guest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ecom-engine/internal/core/orders"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Set Gin to test mode to suppress unnecessary debug logging output during tests.
	gin.SetMode(gin.TestMode)
}

// mockOrderService is a mock implementation of the orders.Service interface.
// It allows dynamic overriding of individual service methods within test cases.
type mockOrderService struct {
	createOrder       func(ctx context.Context, customerID string, customerSnapshot orders.OrderCustomerSnapshot, items []orders.OrderItem) (*orders.Order, error)
	transitionOrder   func(ctx context.Context, id string, action string) (*orders.Order, error)
	getOrder          func(ctx context.Context, id string) (*orders.Order, error)
	getCustomerOrders func(ctx context.Context, customerID string, limit, offset int) ([]orders.Order, error)
	getAllOrders      func(ctx context.Context, limit, offset int) ([]orders.Order, error)
}

// CreateOrder delegates the call to the dynamically configured createOrder function.
func (m *mockOrderService) CreateOrder(ctx context.Context, customerID string, customerSnapshot orders.OrderCustomerSnapshot, items []orders.OrderItem) (*orders.Order, error) {
	if m.createOrder != nil {
		return m.createOrder(ctx, customerID, customerSnapshot, items)
	}
	return nil, nil
}

// TransitionOrder delegates the call to the dynamically configured transitionOrder function.
func (m *mockOrderService) TransitionOrder(ctx context.Context, id string, action string) (*orders.Order, error) {
	if m.transitionOrder != nil {
		return m.transitionOrder(ctx, id, action)
	}
	return nil, nil
}

// GetOrder delegates the call to the dynamically configured getOrder function.
func (m *mockOrderService) GetOrder(ctx context.Context, id string) (*orders.Order, error) {
	if m.getOrder != nil {
		return m.getOrder(ctx, id)
	}
	return nil, nil
}

// GetCustomerOrders delegates the call to the dynamically configured getCustomerOrders function.
func (m *mockOrderService) GetCustomerOrders(ctx context.Context, customerID string, limit, offset int) ([]orders.Order, error) {
	if m.getCustomerOrders != nil {
		return m.getCustomerOrders(ctx, customerID, limit, offset)
	}
	return nil, nil
}

// GetAllOrders delegates the call to the dynamically configured getAllOrders function.
func (m *mockOrderService) GetAllOrders(ctx context.Context, limit, offset int) ([]orders.Order, error) {
	if m.getAllOrders != nil {
		return m.getAllOrders(ctx, limit, offset)
	}
	return nil, nil
}

// setupTestRouter initializes a Gin engine, registers the guest-specific order routes,
// and binds the given orders.Service implementation.
func setupTestRouter(svc orders.Service) *gin.Engine {
	router := gin.New()
	rg := router.Group("/api")
	RegisterGuestRoutes(rg, svc)
	return router
}

// TestController_CreateGuestOrder runs suite tests verifying guest checkout flows,
// covering successful orders and validation failure scenarios (bad body, missing/invalid fields).
func TestController_CreateGuestOrder(t *testing.T) {
	t.Parallel()

	// setup is a helper to instantiate a mock service and configure the test router.
	setup := func(_ *testing.T) (*mockOrderService, *gin.Engine) {
		mockSvc := &mockOrderService{}
		router := setupTestRouter(mockSvc)
		return mockSvc, router
	}

	t.Run("successful guest checkout", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		// Define mock behavior for successful creation
		mockSvc.createOrder = func(_ context.Context, customerID string, customerSnapshot orders.OrderCustomerSnapshot, items []orders.OrderItem) (*orders.Order, error) {
			assert.Empty(t, customerID) // Guest orders must not have a registered CustomerID
			assert.Len(t, items, 1)
			assert.Equal(t, "v_1", items[0].VariantID)
			assert.Equal(t, "John Doe", customerSnapshot.Name)
			assert.Equal(t, "john@example.com", customerSnapshot.Email)
			assert.Equal(t, "12345678", customerSnapshot.ContactNumber)
			return &orders.Order{
				ID:         "ord_gst_1",
				CustomerID: "",
				CustomerSnapshot: orders.OrderCustomerSnapshot{
					Name:          customerSnapshot.Name,
					Email:         customerSnapshot.Email,
					ContactNumber: customerSnapshot.ContactNumber,
				},
				Items:     items,
				Total:     50.00,
				Status:    orders.StatusPending,
				CreatedAt: time.Now(),
			}, nil
		}

		// Prepare a valid guest checkout request
		reqBody := CreateGuestOrderRequest{
			GuestInfo: GuestInfoRequest{
				Name:          "John Doe",
				Email:         "john@example.com",
				ContactNumber: "12345678",
			},
			Items: []OrderItemRequest{
				{VariantID: "v_1", Quantity: 2, Price: 25.00},
			},
		}
		jsonBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		// Perform the HTTP request
		req, _ := http.NewRequest(http.MethodPost, "/api/orders/guest", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Validate response status and payload
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "ord_gst_1", data["order_id"])
		assert.Equal(t, float64(50), data["total"])
	})

	t.Run("fails - invalid request body", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t)

		// Post malformed JSON content
		req, _ := http.NewRequest(http.MethodPost, "/api/orders/guest", bytes.NewBufferString("{bad_json}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("fails - missing guest name", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t)

		// Omit the name field in the guest information
		reqBody := CreateGuestOrderRequest{
			GuestInfo: GuestInfoRequest{
				Name:          "", // missing
				Email:         "john@example.com",
				ContactNumber: "12345678",
			},
			Items: []OrderItemRequest{
				{VariantID: "v_1", Quantity: 2, Price: 25.00},
			},
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/orders/guest", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("fails - invalid email format", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t)

		// Provide a malformed email address
		reqBody := CreateGuestOrderRequest{
			GuestInfo: GuestInfoRequest{
				Name:          "John Doe",
				Email:         "not-an-email", // invalid email
				ContactNumber: "12345678",
			},
			Items: []OrderItemRequest{
				{VariantID: "v_1", Quantity: 2, Price: 25.00},
			},
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/orders/guest", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("fails - invalid contact format", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t)

		// Provide an invalid/too-short phone/contact number
		reqBody := CreateGuestOrderRequest{
			GuestInfo: GuestInfoRequest{
				Name:          "John Doe",
				Email:         "john@example.com",
				ContactNumber: "too-short", // invalid contact format
			},
			Items: []OrderItemRequest{
				{VariantID: "v_1", Quantity: 2, Price: 25.00},
			},
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/orders/guest", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
