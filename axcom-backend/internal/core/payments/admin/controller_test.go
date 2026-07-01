// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package admin

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

	"ecom-engine/internal/core/payments"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter(ctrl *Controller) *gin.Engine {
	router := gin.New()
	rg := router.Group("/api")
	RegisterAdminRoutes(rg, ctrl)
	return router
}

func TestController_ListPayments(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*payments.MockPaymentsService, *gin.Engine) {
		mockSvc := &payments.MockPaymentsService{}
		ctrl := NewController(mockSvc)
		router := setupTestRouter(ctrl)
		return mockSvc, router
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		mockSvc.ListAllPaymentsFunc = func(_ context.Context, _, _ int) ([]payments.Payment, error) {
			return []payments.Payment{
				{ID: "pmt_1", OrderID: "order_1", Amount: 50.0},
				{ID: "pmt_2", OrderID: "order_2", Amount: 100.0},
			}, nil
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/admin/payments", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(2), data["count"])
		paymentsSlice := data["payments"].([]interface{})
		assert.Equal(t, 2, len(paymentsSlice))
	})

	t.Run("service failure", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		mockSvc.ListAllPaymentsFunc = func(_ context.Context, _, _ int) ([]payments.Payment, error) {
			return nil, errors.New("db error")
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/admin/payments", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestController_RefundPayment(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*payments.MockPaymentsService, *gin.Engine) {
		mockSvc := &payments.MockPaymentsService{}
		ctrl := NewController(mockSvc)
		router := setupTestRouter(ctrl)
		return mockSvc, router
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		mockSvc.RefundPaymentFunc = func(_ context.Context, orderID string, _ *float64) (*payments.Payment, error) {
			assert.Equal(t, "order_123", orderID)
			return &payments.Payment{
				ID:      "pmt_123",
				OrderID: orderID,
				Status:  payments.StatusRefunded,
			}, nil
		}

		reqBody := RefundRequest{
			OrderID: "order_123",
		}
		jsonBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodPost, "/api/admin/payments/refund", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		paymentData := data["payment"].(map[string]interface{})
		assert.Equal(t, "pmt_123", paymentData["id"])
		assert.Equal(t, "refunded", paymentData["status"])
	})

	t.Run("validation failure", func(t *testing.T) {
		t.Parallel()
		_, router := setup(t)

		reqBody := RefundRequest{
			OrderID: "", // Missing OrderID
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/admin/payments/refund", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("payment not found", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		mockSvc.RefundPaymentFunc = func(_ context.Context, _ string, _ *float64) (*payments.Payment, error) {
			return nil, payments.ErrPaymentNotFound
		}

		reqBody := RefundRequest{
			OrderID: "order_missing",
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/admin/payments/refund", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		mockSvc.RefundPaymentFunc = func(_ context.Context, _ string, _ *float64) (*payments.Payment, error) {
			return nil, errors.New("cannot refund non-succeeded payment")
		}

		reqBody := RefundRequest{
			OrderID: "order_failed_refund",
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/admin/payments/refund", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestController_GetPaymentByID(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*payments.MockPaymentsService, *gin.Engine) {
		mockSvc := &payments.MockPaymentsService{}
		ctrl := NewController(mockSvc)
		router := setupTestRouter(ctrl)
		return mockSvc, router
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		mockSvc.GetPaymentByIDFunc = func(_ context.Context, id string) (*payments.Payment, error) {
			assert.Equal(t, "pmt_123", id)
			return &payments.Payment{
				ID:      id,
				OrderID: "order_123",
				Status:  payments.StatusPending,
			}, nil
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/admin/payments/pmt_123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "pmt_123", data["id"])
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		mockSvc, router := setup(t)

		mockSvc.GetPaymentByIDFunc = func(_ context.Context, _ string) (*payments.Payment, error) {
			return nil, payments.ErrPaymentNotFound
		}

		req, _ := http.NewRequest(http.MethodGet, "/api/admin/payments/pmt_nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
