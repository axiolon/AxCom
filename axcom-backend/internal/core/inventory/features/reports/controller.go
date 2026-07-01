// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"net/http"

	"ecom-engine/internal/core/inventory/dto"
	reportsdto "ecom-engine/internal/core/inventory/features/reports/dto"
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

func (ctrl *Controller) LowStock(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received LowStock report request")

	stocks, err := ctrl.service.GetLowStockReport(c.Request.Context())
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	items := make([]dto.StockResponse, len(stocks))
	for i, s := range stocks {
		items[i] = dto.StockResponse{
			VariantID:         s.VariantID,
			LocationID:        s.LocationID,
			Quantity:          s.Quantity,
			LowStockThreshold: s.LowStockThreshold,
			AllowBackorders:   s.AllowBackorders,
			BackorderLimit:    s.BackorderLimit,
			IsLowStock:        s.IsLowStock(),
		}
	}

	response.GinOK(c, reportsdto.LowStockReportResponse{
		Items: items,
	})
}

func (ctrl *Controller) Export(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received CSV Export request")

	data, err := ctrl.service.ExportInventoryCSV(c.Request.Context())
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	c.Header("Content-Disposition", "attachment; filename=inventory.csv")
	c.Data(http.StatusOK, "text/csv", data)
}
