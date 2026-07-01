// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package admin

import (
	"errors"
	"strconv"

	"ecom-engine/internal/core/shipping"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service shipping.Service
}

func NewController(service shipping.Service) *Controller {
	return &Controller{service: service}
}

type CreateShipmentRequest struct {
	OrderID        string  `json:"order_id" binding:"required"`
	Carrier        string  `json:"carrier" binding:"required"`
	TrackingNumber string  `json:"tracking_number"`
	Weight         float64 `json:"weight" binding:"required"`
	Value          float64 `json:"value"`
}

type UpdateShipmentRequest struct {
	Status         string `json:"status" binding:"required"`
	TrackingNumber string `json:"tracking_number"`
}

// ListShipments handles GET /api/admin/shipping
func (ctrl *Controller) ListShipments(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received ListShipments admin request")

	var limit int
	var offset int
	var err error

	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")

	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			response.GinWriteError(c, apperrors.NewBadRequest("invalid limit parameter", err))
			return
		}
	} else {
		limit = 20
	}

	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			response.GinWriteError(c, apperrors.NewBadRequest("invalid offset parameter", err))
			return
		}
	} else {
		offset = 0
	}

	list, err := ctrl.service.ListAllShipments(c.Request.Context(), limit, offset)
	if err != nil {
		response.GinWriteError(c, apperrors.NewInternal("failed to list shipments", err))
		return
	}

	response.GinOK(c, gin.H{
		"shipments": list,
		"count":     len(list),
	})
}

// CreateShipment handles POST /api/admin/shipping
func (ctrl *Controller) CreateShipment(c *gin.Context) {
	var req CreateShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("invalid request payload", err))
		return
	}

	logger.InfoCtx(c.Request.Context(), "Received CreateShipment admin request for order %s", req.OrderID)

	s, err := ctrl.service.CreateShipment(c.Request.Context(), req.OrderID, req.Carrier, req.TrackingNumber, req.Weight, req.Value)
	if err != nil {
		response.GinWriteError(c, apperrors.NewInternal("failed to create shipment", err))
		return
	}

	response.GinOK(c, s)
}

// UpdateShipmentStatus handles PUT /api/admin/shipping/:id
func (ctrl *Controller) UpdateShipmentStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("missing shipment id in path", nil))
		return
	}

	var req UpdateShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("invalid request payload", err))
		return
	}

	logger.InfoCtx(c.Request.Context(), "Received UpdateShipmentStatus admin request for %s, status %s", id, req.Status)

	status := shipping.ShipmentStatus(req.Status)
	if status != shipping.StatusPending && status != shipping.StatusInTransit && status != shipping.StatusDelivered && status != shipping.StatusReturned {
		response.GinWriteError(c, apperrors.NewBadRequest("invalid status value: "+req.Status, nil))
		return
	}

	s, err := ctrl.service.UpdateShipmentStatus(c.Request.Context(), id, status, req.TrackingNumber)
	if err != nil {
		if errors.Is(err, shipping.ErrShipmentNotFound) {
			response.GinWriteError(c, apperrors.NewNotFound("shipment not found", err))
			return
		}
		response.GinWriteError(c, apperrors.NewInternal("failed to update shipment", err))
		return
	}

	response.GinOK(c, s)
}

// DeleteShipment handles DELETE /api/admin/shipping/:id
func (ctrl *Controller) DeleteShipment(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("missing shipment id in path", nil))
		return
	}

	logger.InfoCtx(c.Request.Context(), "Received DeleteShipment admin request for %s", id)

	err := ctrl.service.DeleteShipment(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, shipping.ErrShipmentNotFound) || err.Error() == "shipment not found" {
			response.GinWriteError(c, apperrors.NewNotFound("shipment not found", err))
			return
		}
		response.GinWriteError(c, apperrors.NewInternal("failed to delete shipment", err))
		return
	}

	response.GinOK(c, map[string]string{"message": "shipment deleted successfully"})
}
