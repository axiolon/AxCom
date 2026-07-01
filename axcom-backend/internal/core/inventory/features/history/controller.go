// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"strconv"

	histdto "ecom-engine/internal/core/inventory/features/history/dto"
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

// @Summary Get stock history for a given variant ID
// @Description Get stock history for a given variant ID
// @Tags inventory
// @Accept json
// @Produce json
// @Param variantID path string true "Variant ID"
// @Success 200 {object} histdto.HistoryResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /inventory/{variantID}/history [get]
func (ctrl *Controller) History(c *gin.Context) {
	variantID := c.Param("variantID")
	logger.InfoCtx(c.Request.Context(), "Received stock history request for variant: %s", variantID)

	if variantID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Variant ID is required", nil))
		return
	}

	limit := 20
	offset := 0
	if limitQuery := c.Query("limit"); limitQuery != "" {
		if val, err := strconv.Atoi(limitQuery); err == nil {
			limit = val
		}
	}
	if offsetQuery := c.Query("offset"); offsetQuery != "" {
		if val, err := strconv.Atoi(offsetQuery); err == nil {
			offset = val
		}
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	hist, err := ctrl.service.GetHistory(c.Request.Context(), variantID, limit, offset)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, histdto.HistoryResponse{
		History: hist,
	})
}
