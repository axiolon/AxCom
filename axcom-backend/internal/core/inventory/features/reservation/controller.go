// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package reservation provides http controllers and services to handle stock reservation logic.
package reservation

import (
	"ecom-engine/internal/core/inventory/dto"
	resdto "ecom-engine/internal/core/inventory/features/reservation/dto"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

// Controller handles HTTP requests for inventory reservation operations.
type Controller struct {
	service Service
}

// NewController creates and returns a new reservation Controller instance.
func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// Reserve handles HTTP POST requests to reserve stock for a specific product variant.
// Expects variantID as a URI path parameter and a JSON body containing quantity and location.
func (ctrl *Controller) Reserve(c *gin.Context) {
	variantID := c.Param("variantID")
	logger.InfoCtx(c.Request.Context(), "Received Reserve request for variant: %s", variantID)

	if variantID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Variant ID is required", nil))
		return
	}

	var req resdto.ReserveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid reservation payload", err))
		return
	}

	res, err := ctrl.service.ReserveStock(c.Request.Context(), variantID, req.LocationID, req.Quantity)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, resdto.ReserveResponse{
		ReservationID: res.ID,
		VariantID:     res.VariantID,
		LocationID:    res.LocationID,
		Quantity:      res.Quantity,
		ExpiresAt:     res.ExpiresAt,
	})
}

// ReleaseReservation handles HTTP DELETE requests to release an existing stock reservation manually.
// Expects reservationID as a URI path parameter.
func (ctrl *Controller) ReleaseReservation(c *gin.Context) {
	reservationID := c.Param("reservationID")
	logger.InfoCtx(c.Request.Context(), "Received ReleaseReservation request for reservation: %s", reservationID)

	if reservationID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Reservation ID is required", nil))
		return
	}

	if err := ctrl.service.ReleaseReservation(c.Request.Context(), reservationID); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, dto.MessageResponse{
		Message: "reservation released",
	})
}
