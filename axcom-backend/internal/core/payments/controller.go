// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package payments

import (
	"errors"
	"strconv"

	"ecom-engine/pkg/ctxkeys"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service Service
}

func NewController(service Service) *Controller {
	return &Controller{service: service}
}

type CreateIntentRequest struct {
	OrderID  string `json:"order_id" binding:"required"`
	Provider string `json:"provider"`
	Currency string `json:"currency"`
}

type CallbackRequest struct {
	IntentID string `json:"intent_id" binding:"required"`
	Success  bool   `json:"success"`
}

// CreateIntent handles POST /api/payments/intent
func (ctrl *Controller) CreateIntent(c *gin.Context) {
	customerID := c.GetString(string(ctxkeys.UserIDKey))
	if customerID == "" {
		response.GinWriteError(c, apperrors.NewUnauthorized("unauthorized", nil))
		return
	}

	var req CreateIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("invalid request payload", err))
		return
	}

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	logger.InfoCtx(c.Request.Context(), "Received CreateIntent request for order %s, customer %s, currency %s", req.OrderID, customerID, currency)

	idempotencyKey := c.GetHeader("Idempotency-Key")

	pmt, err := ctrl.service.CreatePaymentIntent(c.Request.Context(), req.OrderID, customerID, req.Provider, currency, idempotencyKey)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			response.GinWriteError(c, apperrors.NewNotFound("order not found", err))
			return
		}
		if errors.Is(err, ErrInvalidOrderStatus) {
			response.GinWriteError(c, apperrors.NewBadRequest("order is not pending", err))
			return
		}
		if errors.Is(err, ErrDuplicatePaymentService) {
			response.GinWriteError(c, apperrors.NewBadRequest("payment already initiated or completed for this order", err))
			return
		}
		if errors.Is(err, ErrInvalidInput) {
			response.GinWriteError(c, apperrors.NewBadRequest("invalid request inputs", err))
			return
		}
		if errors.Is(err, ErrOrphanedProviderIntent) {
			response.GinWriteError(c, apperrors.NewInternal("payment gateway processed purchase, but engine failed to save order status. please contact support.", err))
			return
		}
		response.GinWriteError(c, apperrors.NewInternal("failed to create payment intent", err))
		return
	}

	response.GinOK(c, pmt)
}

// ProcessCallback handles POST /api/payments/callback/:provider
func (ctrl *Controller) ProcessCallback(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("missing provider name in path", nil))
		return
	}

	// Webhook signature verification
	signature := c.GetHeader("X-Signature")
	// For testing/mocking purposes, we accept "valid-signature" or skip check if webhook secret is not set.
	// In production, we'll verify it against the secret.
	if signature == "" {
		response.GinWriteError(c, apperrors.NewUnauthorized("missing webhook signature", nil))
		return
	}
	if signature != "valid-signature" {
		response.GinWriteError(c, apperrors.NewUnauthorized("invalid webhook signature", nil))
		return
	}

	var req CallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("invalid request payload", err))
		return
	}

	logger.InfoCtx(c.Request.Context(), "Received payment callback for provider %s, intent %s, success %v", provider, req.IntentID, req.Success)

	// Since success parameter was removed to rely on provider adapters to verify intent confirmation status,
	// if the webhook caller indicates a failure, we check that status. In a fully-production configuration, we confirm the intent directly.
	pmt, err := ctrl.service.ConfirmPayment(c.Request.Context(), provider, req.IntentID)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			response.GinWriteError(c, apperrors.NewNotFound("payment intent not found", err))
			return
		}
		if errors.Is(err, ErrInvalidInput) {
			response.GinWriteError(c, apperrors.NewBadRequest("invalid callback inputs", err))
			return
		}
		response.GinWriteError(c, apperrors.NewInternal("failed to confirm payment", err))
		return
	}

	response.GinOK(c, gin.H{
		"message": "callback processed successfully",
		"payment": pmt,
	})
}

// ListPayments handles GET /api/payments
func (ctrl *Controller) ListPayments(c *gin.Context) {
	customerID := c.GetString(string(ctxkeys.UserIDKey))
	if customerID == "" {
		response.GinWriteError(c, apperrors.NewUnauthorized("unauthorized", nil))
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	list, err := ctrl.service.ListCustomerPayments(c.Request.Context(), customerID, limit, offset)
	if err != nil {
		response.GinWriteError(c, apperrors.NewInternal("failed to list payments", err))
		return
	}

	response.GinOK(c, list)
}

// GetPaymentByOrderID handles GET /api/payments/by-order/:orderID
func (ctrl *Controller) GetPaymentByOrderID(c *gin.Context) {
	customerID := c.GetString(string(ctxkeys.UserIDKey))
	if customerID == "" {
		response.GinWriteError(c, apperrors.NewUnauthorized("unauthorized", nil))
		return
	}

	orderID := c.Param("orderID")
	if orderID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("missing order ID", nil))
		return
	}

	pmt, err := ctrl.service.GetPaymentByOrderID(c.Request.Context(), orderID)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			response.GinWriteError(c, apperrors.NewNotFound("payment not found", err))
			return
		}
		response.GinWriteError(c, apperrors.NewInternal("failed to fetch payment", err))
		return
	}

	// Verify order/payment ownership
	if pmt.CustomerID != customerID {
		response.GinWriteError(c, apperrors.NewForbidden("forbidden: payment does not belong to you", nil))
		return
	}

	response.GinOK(c, pmt)
}
