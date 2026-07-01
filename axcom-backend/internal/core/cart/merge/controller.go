// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package merge

import (
	"ecom-engine/pkg/ctxkeys"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

// Controller handles HTTP requests related to cart merging.
type Controller struct {
	service Service
}

// NewController creates a new instance of Controller with the provided Service.
func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// Merge handles POST requests to merge a guest cart with the authenticated user's account cart.
func (ctrl *Controller) Merge(ctx *gin.Context) {
	logger.InfoCtx(ctx.Request.Context(), "Received Merge cart request")

	accountCustomerID := ctx.GetString(string(ctxkeys.UserIDKey))
	if accountCustomerID == "" {
		response.GinWriteError(ctx, apperrors.NewUnauthorized("unauthorized", nil))
		return
	}

	var req Request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(ctx, apperrors.NewBadRequest("Invalid request payload", err))
		return
	}

	if req.GuestCartID == "" {
		response.GinWriteError(ctx, apperrors.NewBadRequest("Guest cart ID is required", nil))
		return
	}

	cartData, err := ctrl.service.MergeGuestCartWithAccount(ctx.Request.Context(), accountCustomerID, req.GuestCartID)
	if err != nil {
		response.GinWriteError(ctx, err)
		return
	}

	response.GinOK(ctx, cartData)
}
