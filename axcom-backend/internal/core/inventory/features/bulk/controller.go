// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package bulk

import (
	"ecom-engine/internal/core/inventory/dto"
	bulkdto "ecom-engine/internal/core/inventory/features/bulk/dto"
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

// BulkUpdate handles bulk inventory update requests.
func (ctrl *Controller) BulkUpdate(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received BulkUpdate request")

	var req bulkdto.BulkUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid bulk update payload", err))
		return
	}

	updates := make([]UpdateItem, len(req.Items))
	for i, item := range req.Items {
		updates[i] = UpdateItem{
			VariantID:  item.VariantID,
			LocationID: item.LocationID,
			Quantity:   *item.Quantity,
		}
	}

	if err := ctrl.service.BulkUpdate(c.Request.Context(), updates); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, dto.MessageResponse{
		Message: "bulk update completed",
	})
}
