// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package admin

import (
	"errors"
	"strconv"

	"ecom-engine/internal/core/payments"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service payments.Service
}

func NewController(service payments.Service) *Controller {
	return &Controller{service: service}
}

type RefundRequest struct {
	OrderID string   `json:"order_id" binding:"required"`
	Amount  *float64 `json:"amount,omitempty"`
}

// ListPayments handles GET /api/admin/payments
func (ctrl *Controller) ListPayments(c *gin.Context) {
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

	logger.InfoCtx(c.Request.Context(), "Received ListPayments request by admin with limit %d, offset %d", limit, offset)

	list, err := ctrl.service.ListAllPayments(c.Request.Context(), limit, offset)
	if err != nil {
		response.GinWriteError(c, apperrors.NewInternal("failed to list payments", err))
		return
	}

	response.GinOK(c, gin.H{
		"payments": list,
		"count":    len(list),
	})
}

// RefundPayment handles POST /api/admin/payments/refund
func (ctrl *Controller) RefundPayment(c *gin.Context) {
	var req RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("invalid request payload", err))
		return
	}

	logger.InfoCtx(c.Request.Context(), "Received RefundPayment request by admin for order %s", req.OrderID)

	pmt, err := ctrl.service.RefundPayment(c.Request.Context(), req.OrderID, req.Amount)
	if err != nil {
		if errors.Is(err, payments.ErrPaymentNotFound) {
			response.GinWriteError(c, apperrors.NewNotFound("payment not found", err))
			return
		}
		response.GinWriteError(c, apperrors.NewInternal("failed to refund payment", err))
		return
	}

	response.GinOK(c, gin.H{
		"message": "payment refunded successfully",
		"payment": pmt,
	})
}

// GetPaymentByID handles GET /api/admin/payments/:id
func (ctrl *Controller) GetPaymentByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("missing payment ID in path", nil))
		return
	}

	pmt, err := ctrl.service.GetPaymentByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, payments.ErrPaymentNotFound) {
			response.GinWriteError(c, apperrors.NewNotFound("payment not found", err))
			return
		}
		response.GinWriteError(c, apperrors.NewInternal("failed to retrieve payment", err))
		return
	}

	response.GinOK(c, pmt)
}
