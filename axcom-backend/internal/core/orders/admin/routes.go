// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package admin handles HTTP routing and handler implementation for administrative order operations.
package admin

import (
	"ecom-engine/internal/core/orders"

	"github.com/gin-gonic/gin"
)

// RegisterAdminRoutes registers the administrative orders HTTP routes.
func RegisterAdminRoutes(rg *gin.RouterGroup, svc orders.Service) {
	ctrl := newController(svc)
	rg.GET("/admin/orders", ctrl.ListAllOrders)
	rg.GET("/admin/orders/:id", ctrl.GetOrder)
	rg.POST("/admin/orders/:id/transition", ctrl.TransitionOrder)
}
