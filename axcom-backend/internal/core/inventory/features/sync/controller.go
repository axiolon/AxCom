// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"ecom-engine/internal/core/inventory/dto"
	syncdto "ecom-engine/internal/core/inventory/features/sync/dto"
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

func (ctrl *Controller) Sync(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received Sync stock request")

	var req syncdto.SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid sync payload", err))
		return
	}

	err := ctrl.service.SyncStock(c.Request.Context(), req.VariantID, req.LocationID, *req.Quantity)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, dto.MessageResponse{
		Message: "stock synced successfully",
	})
}
