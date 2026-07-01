// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cart

import (
	"ecom-engine/internal/core/cart/dto"
	"ecom-engine/pkg/ctxkeys"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

// Controller handles HTTP requests related to the customer's shopping cart.
type Controller struct {
	service Service
}

// NewController creates a new instance of Controller with the provided Service.
func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// GetCart handles GET requests to retrieve the active user's cart.
func (ctrl *Controller) GetCart(ctx *gin.Context) {
	logger.InfoCtx(ctx.Request.Context(), "Received GetCart request")

	customerID := ctx.GetString(string(ctxkeys.UserIDKey))
	if customerID == "" {
		response.GinWriteError(ctx, apperrors.NewUnauthorized("unauthorized", nil))
		return
	}

	cartData, err := ctrl.service.GetCart(ctx.Request.Context(), customerID)
	if err != nil {
		response.GinWriteError(ctx, err)
		return
	}

	response.GinOK(ctx, cartData)
}

// AddItem handles POST requests to add a product variant to the cart.
func (ctrl *Controller) AddItem(ctx *gin.Context) {
	logger.InfoCtx(ctx.Request.Context(), "Received AddItem request")

	customerID := ctx.GetString(string(ctxkeys.UserIDKey))
	if customerID == "" {
		response.GinWriteError(ctx, apperrors.NewUnauthorized("unauthorized", nil))
		return
	}

	var req dto.AddItemRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(ctx, apperrors.NewBadRequest("Invalid request payload", err))
		return
	}

	item := CartItem{
		VariantID: req.VariantID,
		Quantity:  req.Quantity,
	}

	cartData, err := ctrl.service.AddItem(ctx.Request.Context(), customerID, item)
	if err != nil {
		response.GinWriteError(ctx, err)
		return
	}

	response.GinOK(ctx, cartData)
}

// UpdateItem handles PUT requests to set a specific variant's quantity in the cart.
func (ctrl *Controller) UpdateItem(ctx *gin.Context) {
	logger.InfoCtx(ctx.Request.Context(), "Received UpdateItem request")

	customerID := ctx.GetString(string(ctxkeys.UserIDKey))
	if customerID == "" {
		response.GinWriteError(ctx, apperrors.NewUnauthorized("unauthorized", nil))
		return
	}

	variantID := ctx.Param("variantId")
	if variantID == "" {
		response.GinWriteError(ctx, apperrors.NewBadRequest("Variant ID path parameter is required", ErrVariantIDRequired))
		return
	}

	var req dto.UpdateItemRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(ctx, apperrors.NewBadRequest("Invalid request payload", err))
		return
	}

	if req.Quantity <= 0 {
		response.GinWriteError(ctx, apperrors.NewBadRequest("Quantity must be greater than zero", ErrInvalidQuantity))
		return
	}

	cartData, err := ctrl.service.UpdateItem(ctx.Request.Context(), customerID, variantID, req.Quantity)
	if err != nil {
		response.GinWriteError(ctx, err)
		return
	}

	response.GinOK(ctx, cartData)
}

// RemoveItem handles DELETE requests to remove a variant from the cart.
func (ctrl *Controller) RemoveItem(ctx *gin.Context) {
	logger.InfoCtx(ctx.Request.Context(), "Received RemoveItem request")

	customerID := ctx.GetString(string(ctxkeys.UserIDKey))
	if customerID == "" {
		response.GinWriteError(ctx, apperrors.NewUnauthorized("unauthorized", nil))
		return
	}

	variantID := ctx.Param("variantId")
	if variantID == "" {
		response.GinWriteError(ctx, apperrors.NewBadRequest("Variant ID path parameter is required", ErrVariantIDRequired))
		return
	}

	cartData, err := ctrl.service.RemoveItem(ctx.Request.Context(), customerID, variantID)
	if err != nil {
		response.GinWriteError(ctx, err)
		return
	}

	response.GinOK(ctx, cartData)
}

// Clear handles DELETE requests to empty the active user's cart.
func (ctrl *Controller) Clear(ctx *gin.Context) {
	logger.InfoCtx(ctx.Request.Context(), "Received Clear cart request")

	customerID := ctx.GetString(string(ctxkeys.UserIDKey))
	if customerID == "" {
		response.GinWriteError(ctx, apperrors.NewUnauthorized("unauthorized", nil))
		return
	}

	err := ctrl.service.ClearCart(ctx.Request.Context(), customerID)
	if err != nil {
		response.GinWriteError(ctx, err)
		return
	}

	response.GinOK(ctx, map[string]string{"message": "cart cleared"})
}

// GetCartCount handles GET requests to retrieve the count of items in the user's cart.
func (ctrl *Controller) GetCartCount(ctx *gin.Context) {
	logger.InfoCtx(ctx.Request.Context(), "Received GetCartCount request")

	customerID := ctx.GetString(string(ctxkeys.UserIDKey))
	if customerID == "" {
		response.GinWriteError(ctx, apperrors.NewUnauthorized("unauthorized", nil))
		return
	}

	total, distinct, err := ctrl.service.CartCountDetailed(ctx.Request.Context(), customerID)
	if err != nil {
		response.GinWriteError(ctx, err)
		return
	}

	response.GinOK(ctx, dto.CartCountResponse{
		Count:         total,
		DistinctCount: distinct,
	})
}
