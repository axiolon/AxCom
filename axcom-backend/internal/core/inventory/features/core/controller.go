// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package core contains the core inventory management logic.
package core

import (
	"strconv"

	"ecom-engine/internal/core/inventory/dto"
	coredto "ecom-engine/internal/core/inventory/features/core/dto"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

// Controller handles HTTP requests for the core inventory management feature.
type Controller struct {
	service Service
}

// NewController creates a new instance of Controller.
func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// Check retrieves the current stock quantity for a variant at a specific location.
// @Summary Check stock level
// @Description Get available stock for a product variant at a location.
// @Tags Core Inventory
// @Param variantID path string true "Variant ID"
// @Param location_id query string false "Location ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} apperrors.AppError
// @Router /api/v1/inventory/{variantID} [get]
func (ctrl *Controller) Check(c *gin.Context) {
	variantID := c.Param("variantID")
	locationID := c.Query("location_id")
	if locationID == "" {
		locationID = "default"
	}
	logger.InfoCtx(c.Request.Context(), "Received CheckStock request for variant: %s, location: %s", variantID, locationID)

	if variantID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Variant ID is required", nil))
		return
	}

	qty, err := ctrl.service.CheckStock(c.Request.Context(), variantID, locationID)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, coredto.CheckStockResponse{
		VariantID:  variantID,
		LocationID: locationID,
		Quantity:   qty,
	})
}

// Update updates the stock quantity of a variant at a specific location.
// @Summary Update stock level
// @Description Modify the stock count for a variant at an inventory location.
// @Tags Core Inventory
// @Param body body coredto.UpdateStockRequest true "Stock Update Request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} apperrors.AppError
// @Router /api/v1/inventory/update [post]
func (ctrl *Controller) Update(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received UpdateStock request")

	var req coredto.UpdateStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid stock update payload", err))
		return
	}

	if req.Quantity == nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Quantity is required", nil))
		return
	}

	if err := ctrl.service.UpdateStock(c.Request.Context(), req.VariantID, req.LocationID, *req.Quantity); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, dto.MessageResponse{
		Message: "stock updated",
	})
}

// List retrieves stock items matching the filter parameters.
// @Summary List stock items
// @Description Get list of stock items filtered by variant, location, or status.
// @Tags Core Inventory
// @Param variant_id query string false "Variant ID"
// @Param location_id query string false "Location ID"
// @Param status query string false "Status"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} apperrors.AppError
// @Router /api/v1/inventory [get]
func (ctrl *Controller) List(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received ListStock request")

	var req coredto.ListStockRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid query parameters", err))
		return
	}

	limit := int64(20)
	if req.Limit != nil {
		if *req.Limit > 0 {
			limit = *req.Limit
		}
		if limit > 100 {
			limit = 100
		}
	}

	offset := int64(0)
	if req.Offset != nil && *req.Offset >= 0 {
		offset = *req.Offset
	}

	filter := ListStockFilter{
		VariantID:  req.VariantID,
		LocationID: req.LocationID,
		Status:     req.Status,
		Limit:      limit,
		Offset:     offset,
	}

	stocks, err := ctrl.service.ListStock(c.Request.Context(), filter)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	var items []dto.StockResponse
	for _, s := range stocks {
		items = append(items, dto.StockResponse{
			VariantID:         s.VariantID,
			LocationID:        s.LocationID,
			Quantity:          s.Quantity,
			LowStockThreshold: s.LowStockThreshold,
			AllowBackorders:   s.AllowBackorders,
			BackorderLimit:    s.BackorderLimit,
			IsLowStock:        s.IsLowStock(),
		})
	}

	response.GinOK(c, coredto.ListStockResponse{
		Items:  items,
		Limit:  limit,
		Offset: offset,
	})
}

// Delete removes the stock information for a variant and location.
// @Summary Delete stock record
// @Description Remove a variant's stock item mapping entirely from a location.
// @Tags Core Inventory
// @Param variantID path string true "Variant ID"
// @Param location_id query string false "Location ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} apperrors.AppError
// @Router /api/v1/inventory/{variantID} [delete]
func (ctrl *Controller) Delete(c *gin.Context) {
	variantID := c.Param("variantID")
	locationID := c.Query("location_id")
	if locationID == "" {
		locationID = "default"
	}
	logger.InfoCtx(c.Request.Context(), "Received DeleteStock request for variant: %s, location: %s", variantID, locationID)

	if variantID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Variant ID is required", nil))
		return
	}

	if err := ctrl.service.DeleteStock(c.Request.Context(), variantID, locationID); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, dto.MessageResponse{
		Message: "stock deleted",
	})
}

// Alerts retrieves all triggered low stock alerts.
// @Summary List stock alerts
// @Description Retrieve list of all triggered low-stock alerts.
// @Tags Core Inventory
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} apperrors.AppError
// @Router /api/v1/inventory/alerts [get]
func (ctrl *Controller) Alerts(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received GetAlerts request")

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

	alerts, err := ctrl.service.ListAlerts(c.Request.Context(), limit, offset)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, coredto.AlertsResponse{
		Alerts: alerts,
	})
}

// Configure sets configuration values like thresholds and backorder controls on a stock item.
// @Summary Configure stock settings
// @Description Configure threshold alerts, backorders, and thresholds for a variant.
// @Tags Core Inventory
// @Param body body coredto.ConfigureStockRequest true "Stock Configuration Request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} apperrors.AppError
// @Router /api/v1/inventory/configure [post]
func (ctrl *Controller) Configure(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received ConfigureStock request")

	var req coredto.ConfigureStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid configure request payload", err))
		return
	}

	settings := ConfigureStockSettings{
		VariantID:         req.VariantID,
		LocationID:        req.LocationID,
		Quantity:          req.Quantity,
		LowStockThreshold: req.LowStockThreshold,
		AllowBackorders:   req.AllowBackorders,
		BackorderLimit:    req.BackorderLimit,
	}

	if err := ctrl.service.ConfigureStock(c.Request.Context(), settings); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, dto.MessageResponse{
		Message: "stock configured",
	})
}
