// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package inventory manages inventory operations.
package inventory

import (
	"github.com/gin-gonic/gin"

	"ecom-engine/internal/core/inventory/features/adjustment"
	"ecom-engine/internal/core/inventory/features/bulk"
	"ecom-engine/internal/core/inventory/features/core"
	"ecom-engine/internal/core/inventory/features/history"
	"ecom-engine/internal/core/inventory/features/reports"
	"ecom-engine/internal/core/inventory/features/reservation"
	"ecom-engine/internal/core/inventory/features/sync"
	"ecom-engine/internal/core/inventory/features/transfer"
)

// ModuleControllers holds the controllers for all inventory features.
type ModuleControllers struct {
	Core        *core.Controller
	Bulk        *bulk.Controller
	History     *history.Controller
	Reservation *reservation.Controller
	Reports     *reports.Controller
	Transfer    *transfer.Controller
	Adjustment  *adjustment.Controller
	Sync        *sync.Controller
}

// RegisterRoutes registers all inventory routes.
// @BasePath /api/v1/inventory
func RegisterRoutes(rg *gin.RouterGroup, controllers *ModuleControllers, authMiddleware, adminOnlyMiddleware gin.HandlerFunc) {
	if controllers.Core != nil {
		core.RegisterRoutes(rg, controllers.Core, authMiddleware, adminOnlyMiddleware)
	}
	if controllers.Bulk != nil {
		bulk.RegisterRoutes(rg, controllers.Bulk, authMiddleware)
	}
	if controllers.History != nil {
		history.RegisterRoutes(rg, controllers.History, authMiddleware)
	}
	if controllers.Reservation != nil {
		reservation.RegisterRoutes(rg, controllers.Reservation, authMiddleware)
	}
	if controllers.Reports != nil {
		reports.RegisterRoutes(rg, controllers.Reports, authMiddleware)
	}
	if controllers.Transfer != nil {
		transfer.RegisterRoutes(rg, controllers.Transfer, authMiddleware)
	}
	if controllers.Adjustment != nil {
		adjustment.RegisterRoutes(rg, controllers.Adjustment, authMiddleware)
	}
	if controllers.Sync != nil {
		sync.RegisterRoutes(rg, controllers.Sync, authMiddleware)
	}
}
