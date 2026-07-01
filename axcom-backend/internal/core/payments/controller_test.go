// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ecom-engine/pkg/ctxkeys"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter(ctrl *Controller, userID string) *gin.Engine {
	router := gin.New()
	dummyAuth := func(c *gin.Context) {
		if userID != "" {
			c.Set(string(ctxkeys.UserIDKey), userID)
		}
		c.Next()
	}
	rg := router.Group("/api")
	RegisterRoutes(rg, ctrl, dummyAuth)
	return router
}

func TestController_CreateIntent(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T, userID string) (*MockPaymentsService, *gin.Engine) {
		mockSvc := &MockPaymentsService{}
		ctrl := NewController(mockSvc)
		router := setupTestRouter(ctrl, userID)
		return mockSvc, router
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "user_123")

		mockSvc.CreatePaymentIntentFunc = func(_ context.Context, orderID, customerID, provider, currency, _ string) (*Payment, error) {
			assert.Equal(t, "order_123", orderID)
			assert.Equal(t, "user_123", customerID)
			assert.Equal(t, "stripe", provider)
			assert.Equal(t, "USD", currency)
			return &Payment{
				ID:       "pmt_123",
				OrderID:  orderID,
				Provider: provider,
				Status:   StatusPending,
			}, nil
		}

		reqBody := CreateIntentRequest{
			OrderID:  "order_123",
			Provider: "stripe",
			Currency: "USD",
		}
		jsonBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodPost, "/api/payments/intent", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "pmt_123", data["id"])
		assert.Equal(t, "order_123", data["order_id"])
	})

	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t, "") // No user ID in context

		reqBody := CreateIntentRequest{
			OrderID:  "order_123",
			Provider: "stripe",
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/payments/intent", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("validation failure", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t, "user_123")

		reqBody := CreateIntentRequest{
			Provider: "stripe", // Missing order_id
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/payments/intent", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("order not found", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "user_123")

		mockSvc.CreatePaymentIntentFunc = func(_ context.Context, _, _, _, _, _ string) (*Payment, error) {
			return nil, ErrOrderNotFound
		}

		reqBody := CreateIntentRequest{
			OrderID: "order_missing",
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/payments/intent", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("order not pending", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "user_123")

		mockSvc.CreatePaymentIntentFunc = func(_ context.Context, _, _, _, _, _ string) (*Payment, error) {
			return nil, ErrInvalidOrderStatus
		}

		reqBody := CreateIntentRequest{
			OrderID: "order_not_pending",
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/payments/intent", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "user_123")

		mockSvc.CreatePaymentIntentFunc = func(_ context.Context, _, _, _, _, _ string) (*Payment, error) {
			return nil, errors.New("something went wrong")
		}

		reqBody := CreateIntentRequest{
			OrderID: "order_err",
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/payments/intent", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestController_ProcessCallback(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*MockPaymentsService, *gin.Engine) {
		mockSvc := &MockPaymentsService{}
		ctrl := NewController(mockSvc)
		router := setupTestRouter(ctrl, "")
		return mockSvc, router
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		mockSvc.ConfirmPaymentFunc = func(_ context.Context, providerName, intentID string) (*Payment, error) {
			assert.Equal(t, "stripe", providerName)
			assert.Equal(t, "intent_123", intentID)
			return &Payment{
				ID:               "pmt_123",
				Provider:         providerName,
				ProviderIntentID: intentID,
				Status:           StatusSucceeded,
			}, nil
		}

		reqBody := CallbackRequest{
			IntentID: "intent_123",
			Success:  true,
		}
		jsonBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodPost, "/api/payments/callback/stripe", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Signature", "valid-signature")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("missing signature", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t)

		reqBody := CallbackRequest{
			IntentID: "intent_123",
			Success:  true,
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/payments/callback/stripe", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid signature", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t)

		reqBody := CallbackRequest{
			IntentID: "intent_123",
			Success:  true,
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/payments/callback/stripe", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Signature", "invalid-sig")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("validation failure", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t)

		reqBody := CallbackRequest{
			Success: true, // Missing intent_id
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/payments/callback/stripe", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Signature", "valid-signature")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("payment not found", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		mockSvc.ConfirmPaymentFunc = func(_ context.Context, _, _ string) (*Payment, error) {
			return nil, ErrPaymentNotFound
		}

		reqBody := CallbackRequest{
			IntentID: "intent_missing",
			Success:  true,
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/payments/callback/stripe", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Signature", "valid-signature")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		mockSvc.ConfirmPaymentFunc = func(_ context.Context, _, _ string) (*Payment, error) {
			return nil, errors.New("database connection lost")
		}

		reqBody := CallbackRequest{
			IntentID: "intent_err",
			Success:  true,
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/payments/callback/stripe", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Signature", "valid-signature")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestController_ListPayments(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T, userID string) (*MockPaymentsService, *gin.Engine) {
		mockSvc := &MockPaymentsService{}
		ctrl := NewController(mockSvc)
		router := setupTestRouter(ctrl, userID)
		return mockSvc, router
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "user_123")

		mockSvc.ListCustomerPaymentsFunc = func(_ context.Context, customerID string, limit, offset int) ([]Payment, error) {
			assert.Equal(t, "user_123", customerID)
			assert.Equal(t, 20, limit)
			assert.Equal(t, 0, offset)
			return []Payment{
				{ID: "pmt_1", CustomerID: "user_123", Amount: 100},
			}, nil
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/payments?limit=20&offset=0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t, "")

		req, _ := http.NewRequest(http.MethodGet, "/api/payments", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestController_GetPaymentByOrderID(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T, userID string) (*MockPaymentsService, *gin.Engine) {
		mockSvc := &MockPaymentsService{}
		ctrl := NewController(mockSvc)
		router := setupTestRouter(ctrl, userID)
		return mockSvc, router
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "user_123")

		mockSvc.GetPaymentByOrderIDFunc = func(_ context.Context, orderID string) (*Payment, error) {
			assert.Equal(t, "order_999", orderID)
			return &Payment{
				ID:         "pmt_999",
				OrderID:    orderID,
				CustomerID: "user_123",
				Status:     StatusPending,
			}, nil
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/payments/by-order/order_999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "pmt_999", data["id"])
	})

	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t, "")

		req, _ := http.NewRequest(http.MethodGet, "/api/payments/by-order/order_999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("forbidden customer mismatch", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "user_123")

		mockSvc.GetPaymentByOrderIDFunc = func(_ context.Context, orderID string) (*Payment, error) {
			return &Payment{
				ID:         "pmt_999",
				OrderID:    orderID,
				CustomerID: "another_user",
				Status:     StatusPending,
			}, nil
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/payments/by-order/order_999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t, "user_123")

		mockSvc.GetPaymentByOrderIDFunc = func(_ context.Context, _ string) (*Payment, error) {
			return nil, ErrPaymentNotFound
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/payments/by-order/order_nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
