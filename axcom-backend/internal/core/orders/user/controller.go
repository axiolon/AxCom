// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package user handles HTTP routing and handler implementation for authenticated customer orders.
package user

import (
	"strconv"

	"ecom-engine/internal/core/orders"
	"ecom-engine/internal/core/orders/domain"
	"ecom-engine/pkg/ctxkeys"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

// controller manages HTTP handler methods for authenticated customer order operations.
type controller struct {
	service orders.Service
}

// newController instantiates a new controller reference.
func newController(service orders.Service) *controller {
	return &controller{service: service}
}

// Create handles POST /orders and submits a checkout request for the authenticated user.
func (ctrl *controller) Create(c *gin.Context) {
	customerID := c.GetString(string(ctxkeys.UserIDKey))
	if customerID == "" {
		response.GinWriteError(c, apperrors.NewUnauthorized("unauthorized", nil))
		return
	}

	logger.InfoCtx(c.Request.Context(), "Received CreateOrder request for customer: %s", customerID)

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("invalid request payload", err))
		return
	}

	// Map DTO to internal orders.OrderItem model
	items := make([]orders.OrderItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = orders.OrderItem{
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}

	// Authenticated checkout
	o, err := ctrl.service.CreateOrder(c.Request.Context(), customerID, orders.OrderCustomerSnapshot{}, items)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, mapToOrderResponse(o))
}

// GetMyOrder handles GET /orders/:id, retrieving a specific order if it belongs to the authenticated user.
func (ctrl *controller) GetMyOrder(c *gin.Context) {
	customerID := c.GetString(string(ctxkeys.UserIDKey))
	if customerID == "" {
		response.GinWriteError(c, apperrors.NewUnauthorized("unauthorized", nil))
		return
	}

	id := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received GetMyOrder request for ID: %s, customer: %s", id, customerID)

	if id == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("missing order id", nil))
		return
	}

	o, err := ctrl.service.GetOrder(c.Request.Context(), id)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	// Validate ownership
	if o.CustomerID != customerID {
		logger.ErrorCtx(c.Request.Context(), "Customer %s attempted to access order %s belonging to customer %s", customerID, id, o.CustomerID)
		response.GinWriteError(c, apperrors.NewForbidden("you do not have access to this order", domain.ErrForbidden))
		return
	}

	response.GinOK(c, mapToOrderResponse(o))
}

// ListMyOrders handles GET /orders, listing all orders placed by the authenticated user.
func (ctrl *controller) ListMyOrders(c *gin.Context) {
	customerID := c.GetString(string(ctxkeys.UserIDKey))
	if customerID == "" {
		response.GinWriteError(c, apperrors.NewUnauthorized("unauthorized", nil))
		return
	}

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

	logger.InfoCtx(c.Request.Context(), "Received ListMyOrders request for customer: %s, limit: %d, offset: %d", customerID, limit, offset)

	ordersList, err := ctrl.service.GetCustomerOrders(c.Request.Context(), customerID, limit, offset)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	ordersReponseList := make([]OrderResponse, len(ordersList))
	for i, o := range ordersList {
		ordersReponseList[i] = mapToOrderResponse(&o)
	}

	response.GinOK(c, OrderListResponse{
		Orders: ordersReponseList,
		Count:  len(ordersReponseList),
	})
}

// CancelMyOrder handles POST /orders/:id/cancel, canceling an order if it belongs to the authenticated user.
func (ctrl *controller) CancelMyOrder(c *gin.Context) {
	customerID := c.GetString(string(ctxkeys.UserIDKey))
	if customerID == "" {
		response.GinWriteError(c, apperrors.NewUnauthorized("unauthorized", nil))
		return
	}

	id := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received CancelMyOrder request for ID: %s, customer: %s", id, customerID)

	if id == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("missing order id", nil))
		return
	}

	o, err := ctrl.service.GetOrder(c.Request.Context(), id)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	// Validate ownership
	if o.CustomerID != customerID {
		logger.ErrorCtx(c.Request.Context(), "Customer %s attempted to cancel order %s belonging to customer %s", customerID, id, o.CustomerID)
		response.GinWriteError(c, apperrors.NewForbidden("you do not have access to this order", domain.ErrForbidden))
		return
	}

	updatedOrder, err := ctrl.service.TransitionOrder(c.Request.Context(), id, "cancel")
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, mapToOrderResponse(updatedOrder))
}

// mapToOrderResponse is a helper that maps the internal Order model to the HTTP OrderResponse DTO.
func mapToOrderResponse(o *orders.Order) OrderResponse {
	items := make([]OrderItemResponse, len(o.Items))
	for i, item := range o.Items {
		items[i] = OrderItemResponse{
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}

	return OrderResponse{
		ID:        o.ID,
		Total:     o.Total,
		Status:    string(o.Status),
		CreatedAt: o.CreatedAt,
		Items:     items,
	}
}
