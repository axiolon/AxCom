// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package adjustment

import (
	"ecom-engine/internal/core/inventory/dto"
	adjdto "ecom-engine/internal/core/inventory/features/adjustment/dto"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

// Controller handles inventory adjustment requests.
type Controller struct {
	service Service
}

func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// Adjust handles stock adjustment requests.
// This is a critical operation. It is used to adjust the stock for a specific variant.
// It should include reason for compliance.
func (ctrl *Controller) Adjust(c *gin.Context) {
	variantID := c.Param("variantID")
	logger.InfoCtx(c.Request.Context(), "Received Adjust stock request for variant: %s", variantID)

	if variantID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Variant ID is required", nil))
		return
	}

	var req adjdto.AdjustRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid adjustment payload", err))
		return
	}

	err := ctrl.service.AdjustStock(c.Request.Context(), variantID, req.LocationID, *req.Adjustment, req.Reason)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, dto.MessageResponse{
		Message: "stock adjusted successfully",
	})
}
