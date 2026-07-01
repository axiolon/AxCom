// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package guest provides HTTP routing, controllers, and database repository interfaces
// for managing guest (unauthenticated checkout) orders.
package guest

import (
	"ecom-engine/internal/core/orders"

	"github.com/gin-gonic/gin"
)

// RegisterGuestRoutes registers the public guest order routes.
func RegisterGuestRoutes(rg *gin.RouterGroup, svc orders.Service) {
	ctrl := newController(svc)
	rg.POST("/orders/guest", ctrl.CreateGuestOrder)
}
