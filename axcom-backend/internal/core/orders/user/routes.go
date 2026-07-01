// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package user handles HTTP routing and handler implementation for authenticated customer orders.
package user

import (
	"ecom-engine/internal/core/orders"

	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes registers the authenticated user orders HTTP routes.
func RegisterUserRoutes(rg *gin.RouterGroup, svc orders.Service) {
	ctrl := newController(svc)
	rg.POST("/orders", ctrl.Create)
	rg.GET("/orders", ctrl.ListMyOrders)
	rg.GET("/orders/:id", ctrl.GetMyOrder)
	rg.POST("/orders/:id/cancel", ctrl.CancelMyOrder)
}
