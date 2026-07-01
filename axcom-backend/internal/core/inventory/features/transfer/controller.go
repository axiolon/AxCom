// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"ecom-engine/internal/core/inventory/dto"
	transferdto "ecom-engine/internal/core/inventory/features/transfer/dto"
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

// Transfer godoc
// @Summary Transfer stock
// @Description Transfer stock from one location to another
// @Tags Inventory Transfer
// @Accept json
// @Produce json
// @Param request body transferdto.TransferRequest true "Transfer request"
// @Success 200 {object} dto.MessageResponse
// @Failure 400 {object} apperrors.AppError
// @Failure 500 {object} apperrors.AppError
// @Router /inventory/transfer [post]
func (ctrl *Controller) Transfer(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received Transfer stock request")

	var req transferdto.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid transfer payload", err))
		return
	}

	err := ctrl.service.TransferStock(c.Request.Context(), req.VariantID, req.FromLocation, req.ToLocation, *req.Quantity)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, dto.MessageResponse{
		Message: "stock transfer completed successfully",
	})
}
