// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cart

import (
	"bytes"
	"context"
	"ecom-engine/internal/core/cart/dto"
	"ecom-engine/pkg/ctxkeys"
	"encoding/json"
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

type MockCartService struct {
	GetCartFunc           func(ctx context.Context, customerID string) (*dto.CartResponse, error)
	AddItemFunc           func(ctx context.Context, customerID string, item CartItem) (*dto.CartResponse, error)
	UpdateItemFunc        func(ctx context.Context, customerID string, variantID string, quantity int) (*dto.CartResponse, error)
	RemoveItemFunc        func(ctx context.Context, customerID string, variantID string) (*dto.CartResponse, error)
	ClearCartFunc         func(ctx context.Context, customerID string) error
	CartCountFunc         func(ctx context.Context, customerID string) (int, error)
	CartCountDetailedFunc func(ctx context.Context, customerID string) (int, int, error)
}

func (m *MockCartService) GetCart(ctx context.Context, customerID string) (*dto.CartResponse, error) {
	if m.GetCartFunc != nil {
		return m.GetCartFunc(ctx, customerID)
	}
	return nil, nil
}

func (m *MockCartService) AddItem(ctx context.Context, customerID string, item CartItem) (*dto.CartResponse, error) {
	if m.AddItemFunc != nil {
		return m.AddItemFunc(ctx, customerID, item)
	}
	return nil, nil
}

func (m *MockCartService) UpdateItem(ctx context.Context, customerID string, variantID string, quantity int) (*dto.CartResponse, error) {
	if m.UpdateItemFunc != nil {
		return m.UpdateItemFunc(ctx, customerID, variantID, quantity)
	}
	return nil, nil
}

func (m *MockCartService) RemoveItem(ctx context.Context, customerID string, variantID string) (*dto.CartResponse, error) {
	if m.RemoveItemFunc != nil {
		return m.RemoveItemFunc(ctx, customerID, variantID)
	}
	return nil, nil
}

func (m *MockCartService) ClearCart(ctx context.Context, customerID string) error {
	if m.ClearCartFunc != nil {
		return m.ClearCartFunc(ctx, customerID)
	}
	return nil
}

func (m *MockCartService) CartCount(ctx context.Context, customerID string) (int, error) {
	if m.CartCountFunc != nil {
		return m.CartCountFunc(ctx, customerID)
	}
	return 0, nil
}

func (m *MockCartService) CartCountDetailed(ctx context.Context, customerID string) (int, int, error) {
	if m.CartCountDetailedFunc != nil {
		return m.CartCountDetailedFunc(ctx, customerID)
	}
	return 0, 0, nil
}

func setupTestRouter(ctrl *Controller, userID string) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if userID != "" {
			c.Set(string(ctxkeys.UserIDKey), userID)
		}
		c.Next()
	})
	rg := router.Group("/api")
	RegisterRoutes(rg, ctrl)
	return router
}

func TestController_GetCart(t *testing.T) {
	t.Parallel()

	t.Run("unauthorized request", func(t *testing.T) {
		t.Parallel()
		svc := &MockCartService{}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl, "") // No user ID

		req, _ := http.NewRequest("GET", "/api/cart", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("successful get cart", func(t *testing.T) {
		t.Parallel()
		svc := &MockCartService{
			GetCartFunc: func(_ context.Context, customerID string) (*dto.CartResponse, error) {
				return &dto.CartResponse{
					CustomerID: customerID,
					Items:      []dto.CartItemResponse{},
				}, nil
			},
		}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl, "cust_123")

		req, _ := http.NewRequest("GET", "/api/cart", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
	})
}

func TestController_AddItem(t *testing.T) {
	t.Parallel()

	t.Run("successful add item", func(t *testing.T) {
		t.Parallel()
		svc := &MockCartService{
			AddItemFunc: func(_ context.Context, customerID string, item CartItem) (*dto.CartResponse, error) {
				return &dto.CartResponse{
					CustomerID: customerID,
					Items: []dto.CartItemResponse{
						{VariantID: item.VariantID, Quantity: item.Quantity},
					},
				}, nil
			},
		}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl, "cust_123")

		reqBody := dto.AddItemRequest{
			VariantID: "var_abc",
			Quantity:  2,
		}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/cart", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("validation error - missing variant ID", func(t *testing.T) {
		t.Parallel()
		svc := &MockCartService{}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl, "cust_123")

		reqBody := dto.AddItemRequest{
			VariantID: "",
			Quantity:  2,
		}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/cart", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error - quantity <= 0", func(t *testing.T) {
		t.Parallel()
		svc := &MockCartService{}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl, "cust_123")

		reqBody := dto.AddItemRequest{
			VariantID: "var_abc",
			Quantity:  -1,
		}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/cart", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestController_UpdateItem(t *testing.T) {
	t.Parallel()

	t.Run("successful update item", func(t *testing.T) {
		t.Parallel()
		svc := &MockCartService{
			UpdateItemFunc: func(_ context.Context, customerID string, variantID string, quantity int) (*dto.CartResponse, error) {
				return &dto.CartResponse{
					CustomerID: customerID,
					Items: []dto.CartItemResponse{
						{VariantID: variantID, Quantity: quantity},
					},
				}, nil
			},
		}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl, "cust_123")

		reqBody := dto.UpdateItemRequest{
			Quantity: 5,
		}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/cart/items/var_abc", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestController_RemoveItem(t *testing.T) {
	t.Parallel()

	t.Run("successful remove item", func(t *testing.T) {
		t.Parallel()
		svc := &MockCartService{
			RemoveItemFunc: func(_ context.Context, customerID string, _ string) (*dto.CartResponse, error) {
				return &dto.CartResponse{
					CustomerID: customerID,
					Items:      []dto.CartItemResponse{},
				}, nil
			},
		}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl, "cust_123")

		req, _ := http.NewRequest("DELETE", "/api/cart/items/var_abc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestController_Clear(t *testing.T) {
	t.Parallel()

	t.Run("successful clear cart", func(t *testing.T) {
		t.Parallel()
		svc := &MockCartService{
			ClearCartFunc: func(_ context.Context, _ string) error {
				return nil
			},
		}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl, "cust_123")

		req, _ := http.NewRequest("DELETE", "/api/cart", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestController_GetCartCount(t *testing.T) {
	t.Parallel()

	t.Run("successful get count", func(t *testing.T) {
		t.Parallel()
		svc := &MockCartService{
			CartCountDetailedFunc: func(_ context.Context, _ string) (int, int, error) {
				return 42, 10, nil
			},
		}
		ctrl := NewController(svc)
		router := setupTestRouter(ctrl, "cust_123")

		req, _ := http.NewRequest("GET", "/api/cart/count", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		dataVal := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(42), dataVal["count"])
		assert.Equal(t, float64(10), dataVal["distinct_count"])
	})
}
