// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package guest handles guest-specific checkouts, routes, and controllers.
package guest

import (
	"ecom-engine/internal/core/orders"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

// controller coordinates HTTP request-response cycles for guest order endpoints.
type controller struct {
	service orders.Service
}

// newController constructs a new guest order controller instance.
func newController(service orders.Service) *controller {
	return &controller{
		service: service,
	}
}

// CreateGuestOrder handles POST /orders/guest
func (ctrl *controller) CreateGuestOrder(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received CreateGuestOrder request")

	var req CreateGuestOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("invalid request payload", err))
		return
	}

	// Validate Guest Details at the boundary level
	guestInfo := &GuestCustomerInfo{
		Name:          req.GuestInfo.Name,
		Email:         req.GuestInfo.Email,
		ContactNumber: req.GuestInfo.ContactNumber,
	}

	if err := ValidateGuestInfo(guestInfo); err != nil {
		logger.ErrorCtx(c.Request.Context(), "Guest validation failed: %v", err)
		response.GinWriteError(c, apperrors.NewBadRequest(err.Error(), err))
		return
	}

	// Map DTO to core orders.OrderItem
	items := make([]orders.OrderItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = orders.OrderItem{
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}

	// Build the order customer snapshot
	snapshot := orders.OrderCustomerSnapshot{
		Name:          guestInfo.Name,
		Email:         guestInfo.Email,
		ContactNumber: guestInfo.ContactNumber,
	}

	// Create anonymous core order with customer snapshot
	o, err := ctrl.service.CreateOrder(c.Request.Context(), "", snapshot, items)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	// Map to guest-specific contract
	res := GuestOrderResponse{
		OrderID:   o.ID,
		Status:    string(o.Status),
		Total:     o.Total,
		CreatedAt: o.CreatedAt,
		GuestInfo: GuestInfoResponse{
			Name:          o.CustomerSnapshot.Name,
			Email:         o.CustomerSnapshot.Email,
			ContactNumber: o.CustomerSnapshot.ContactNumber,
		},
	}

	response.GinOK(c, res)
}
