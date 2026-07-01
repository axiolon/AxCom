// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package merge

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

type MockMergeService struct {
	MergeGuestCartWithAccountFunc func(ctx context.Context, accountCustomerID string, guestCartID string) (*dto.CartResponse, error)
}

func (m *MockMergeService) MergeGuestCartWithAccount(ctx context.Context, accountCustomerID string, guestCartID string) (*dto.CartResponse, error) {
	if m.MergeGuestCartWithAccountFunc != nil {
		return m.MergeGuestCartWithAccountFunc(ctx, accountCustomerID, guestCartID)
	}
	return nil, nil
}

func setupMergeRouter(ctrl *Controller, userID string) *gin.Engine {
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

func TestMergeController_Merge(t *testing.T) {
	t.Parallel()

	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()
		svc := &MockMergeService{}
		ctrl := NewController(svc)
		router := setupMergeRouter(ctrl, "")

		reqBody := Request{GuestCartID: "guest_123"}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/cart/merge", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("validation error - missing guest cart id", func(t *testing.T) {
		t.Parallel()
		svc := &MockMergeService{}
		ctrl := NewController(svc)
		router := setupMergeRouter(ctrl, "cust_123")

		reqBody := Request{GuestCartID: ""}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/cart/merge", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("successful merge", func(t *testing.T) {
		t.Parallel()
		svc := &MockMergeService{
			MergeGuestCartWithAccountFunc: func(_ context.Context, accountCustomerID string, _ string) (*dto.CartResponse, error) {
				return &dto.CartResponse{
					CustomerID: accountCustomerID,
					Items: []dto.CartItemResponse{
						{VariantID: "var_123", Quantity: 5},
					},
				}, nil
			},
		}
		ctrl := NewController(svc)
		router := setupMergeRouter(ctrl, "cust_123")

		reqBody := Request{GuestCartID: "guest_123"}
		jsonBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/cart/merge", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
	})
}
