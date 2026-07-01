// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package admin

import (
	"context"
	"strconv"

	"ecom-engine/internal/core/orders"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

// controller handles HTTP endpoints for administrative order operations.
type controller struct {
	service orders.Service
}

// newController creates and returns a new controller instance.
func newController(service orders.Service) *controller {
	return &controller{
		service: service,
	}
}

// ListAllOrders handles GET /admin/orders and lists all orders in the system with pagination.
func (ctrl *controller) ListAllOrders(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	logger.InfoCtx(c.Request.Context(), "Received ListAllOrders request, limit: %d, offset: %d", limit, offset)

	ordersList, err := ctrl.service.GetAllOrders(c.Request.Context(), limit, offset)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	ordersResponseList := make([]OrderResponse, len(ordersList))
	for i, o := range ordersList {
		ordersResponseList[i] = ctrl.mapToOrderResponse(c.Request.Context(), &o)
	}

	response.GinOK(c, OrderListResponse{
		Orders: ordersResponseList,
		Count:  len(ordersResponseList),
	})
}

// GetOrder handles GET /admin/orders/:id and retrieves a specific order by ID.
func (ctrl *controller) GetOrder(c *gin.Context) {
	id := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received GetOrder request for ID: %s", id)

	if id == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("missing order id", nil))
		return
	}

	o, err := ctrl.service.GetOrder(c.Request.Context(), id)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, ctrl.mapToOrderResponse(c.Request.Context(), o))
}

// TransitionOrder handles POST /admin/orders/:id/transition and transitions an order status based on the action input.
func (ctrl *controller) TransitionOrder(c *gin.Context) {
	id := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received TransitionOrder request for ID: %s", id)

	if id == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("missing order id", nil))
		return
	}

	var req TransitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("invalid request payload", err))
		return
	}

	o, err := ctrl.service.TransitionOrder(c.Request.Context(), id, req.Action)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, ctrl.mapToOrderResponse(c.Request.Context(), o))
}

// mapToOrderResponse maps the internal Order model to the admin OrderResponse DTO.
func (ctrl *controller) mapToOrderResponse(_ context.Context, o *orders.Order) OrderResponse {
	items := make([]OrderItemResponse, len(o.Items))
	for i, item := range o.Items {
		items[i] = OrderItemResponse{
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}

	var guestInfo *GuestCustomerInfoResponse
	if o.CustomerID == "" && (o.CustomerSnapshot.Name != "" || o.CustomerSnapshot.Email != "" || o.CustomerSnapshot.ContactNumber != "") {
		guestInfo = &GuestCustomerInfoResponse{
			Name:          o.CustomerSnapshot.Name,
			Email:         o.CustomerSnapshot.Email,
			ContactNumber: o.CustomerSnapshot.ContactNumber,
		}
	}

	return OrderResponse{
		ID:         o.ID,
		CustomerID: o.CustomerID,
		GuestInfo:  guestInfo,
		Items:      items,
		Total:      o.Total,
		Status:     string(o.Status),
		CreatedAt:  o.CreatedAt,
	}
}
