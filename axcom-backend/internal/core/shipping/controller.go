// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package shipping

import (
	"context"
	"ecom-engine/internal/core/orders"
	"ecom-engine/pkg/ctxkeys"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

type OrderService interface {
	GetOrder(ctx context.Context, id string) (*orders.Order, error)
}

type Controller struct {
	service      Service
	orderService OrderService
}

func NewController(service Service, orderService OrderService) *Controller {
	return &Controller{
		service:      service,
		orderService: orderService,
	}
}

// CalculateRates handles POST /api/shipping/rates
func (ctrl *Controller) CalculateRates(c *gin.Context) {
	var req RateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("invalid request payload", err))
		return
	}

	logger.InfoCtx(c.Request.Context(), "Received CalculateRates request")

	rates, err := ctrl.service.CalculateRates(c.Request.Context(), req)
	if err != nil {
		response.GinWriteError(c, apperrors.NewInternal("failed to calculate rates", err))
		return
	}

	response.GinOK(c, rates)
}

// GetMyOrderShipment handles GET /api/shipping/order/:order_id
func (ctrl *Controller) GetMyOrderShipment(c *gin.Context) {
	customerID := c.GetString(string(ctxkeys.UserIDKey))
	if customerID == "" {
		response.GinWriteError(c, apperrors.NewUnauthorized("unauthorized", nil))
		return
	}

	orderID := c.Param("order_id")
	if orderID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("missing order_id", nil))
		return
	}

	logger.InfoCtx(c.Request.Context(), "Received GetMyOrderShipment for order %s", orderID)

	shipment, err := ctrl.service.GetShipmentByOrderID(c.Request.Context(), orderID)
	if err != nil {
		response.GinWriteError(c, apperrors.NewNotFound("shipment not found", err))
		return
	}

	// S3-3: Verify the order belongs to the customer
	order, err := ctrl.orderService.GetOrder(c.Request.Context(), shipment.OrderID)
	if err != nil {
		response.GinWriteError(c, apperrors.NewNotFound("associated order not found", err))
		return
	}

	if order.CustomerID != customerID {
		response.GinWriteError(c, apperrors.NewForbidden("you do not have access to this shipment", nil))
		return
	}

	response.GinOK(c, shipment)
}

// TrackShipment handles GET /api/shipping/track/:tracking_number
func (ctrl *Controller) TrackShipment(c *gin.Context) {
	trackingNumber := c.Param("tracking_number")
	if trackingNumber == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("missing tracking_number", nil))
		return
	}

	logger.InfoCtx(c.Request.Context(), "Received TrackShipment for tracking number %s", trackingNumber)

	shipment, err := ctrl.service.TrackShipment(c.Request.Context(), trackingNumber)
	if err != nil {
		if err == ErrTrackingNumberNotFound {
			response.GinWriteError(c, apperrors.NewNotFound("shipment not found for the given tracking number", err))
		} else {
			response.GinWriteError(c, apperrors.NewInternal("failed to track shipment", err))
		}
		return
	}

	resp := TrackingResponse{
		TrackingNumber:      shipment.TrackingNumber,
		Carrier:             shipment.Carrier,
		Status:              shipment.Status,
		EstimatedDeliveryAt: shipment.EstimatedDeliveryAt,
		CreatedAt:           shipment.CreatedAt,
		UpdatedAt:           shipment.UpdatedAt,
	}

	response.GinOK(c, resp)
}
