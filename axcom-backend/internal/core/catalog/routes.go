// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package catalog manages the e-commerce product catalog features, including
// core products, images, variants, discounts, bulk operations, and reviews.
// It exposes aggregated controllers and route registration helpers.
package catalog

import (
	"github.com/gin-gonic/gin"

	"ecom-engine/internal/core/catalog/features/bulk"
	"ecom-engine/internal/core/catalog/features/core"
	"ecom-engine/internal/core/catalog/features/discounts"
	"ecom-engine/internal/core/catalog/features/images"
	"ecom-engine/internal/core/catalog/features/reviews"
	"ecom-engine/internal/core/catalog/features/variants"
)

// ModuleControllers aggregates controllers for all modular catalog features.
type ModuleControllers struct {
	Core      *core.Controller
	Images    *images.Controller
	Variants  *variants.Controller
	Discounts *discounts.Controller
	Bulk      *bulk.Controller
	Reviews   *reviews.Controller
}

// RegisterRoutes registers HTTP routes of all catalog features onto the RouterGroup.
func RegisterRoutes(rg *gin.RouterGroup, controllers *ModuleControllers, authMiddleware, adminOnlyMiddleware gin.HandlerFunc) {
	if controllers.Core != nil {
		core.RegisterRoutes(rg, controllers.Core, authMiddleware, adminOnlyMiddleware)
	}
	if controllers.Images != nil {
		images.RegisterRoutes(rg, controllers.Images, authMiddleware, adminOnlyMiddleware)
	}
	if controllers.Variants != nil {
		variants.RegisterRoutes(rg, controllers.Variants, authMiddleware, adminOnlyMiddleware)
	}
	if controllers.Discounts != nil {
		discounts.RegisterRoutes(rg, controllers.Discounts, authMiddleware, adminOnlyMiddleware)
	}
	if controllers.Bulk != nil {
		bulk.RegisterRoutes(rg, controllers.Bulk, authMiddleware, adminOnlyMiddleware)
	}
	if controllers.Reviews != nil {
		reviews.RegisterRoutes(rg, controllers.Reviews, authMiddleware, adminOnlyMiddleware)
	}
}
