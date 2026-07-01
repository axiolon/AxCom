// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package discounts

import (
	"ecom-engine/internal/core/catalog/domain"
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

type ApplyDiscountRequest struct {
	Type  string  `json:"type" binding:"required,oneof=percentage fixed"`
	Value float64 `json:"value" binding:"required,min=0"`
}

func (ctrl *Controller) Apply(c *gin.Context) {
	productID := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received ApplyDiscount request for product ID: %s", productID)

	if productID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Product ID is required", nil))
		return
	}

	var req ApplyDiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid discount payload", err))
		return
	}

	d := &domain.ProductDiscount{
		Type:  req.Type,
		Value: req.Value,
	}

	if err := ctrl.service.ApplyDiscount(c.Request.Context(), productID, d); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, d)
}

func (ctrl *Controller) Remove(c *gin.Context) {
	productID := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received RemoveDiscount request for product ID: %s", productID)

	if productID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Product ID is required", nil))
		return
	}

	if err := ctrl.service.RemoveDiscount(c.Request.Context(), productID); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, gin.H{"message": "discount removed"})
}
