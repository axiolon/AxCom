// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package variants

import "github.com/gin-gonic/gin"

// RegisterRoutes registers product variants routes on the router group.
func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware gin.HandlerFunc, adminOnlyMiddleware gin.HandlerFunc) {
	// Public routes
	rg.GET("/products/:id/variants", ctrl.GetVariants)

	// Protected routes (admin/authenticated updates)
	rg.POST("/products/:id/variants", authMiddleware, adminOnlyMiddleware, ctrl.AddVariant)
	rg.PUT("/products/:id/variants/:variantId", authMiddleware, adminOnlyMiddleware, ctrl.UpdateVariant)
	rg.DELETE("/products/:id/variants/:variantId", authMiddleware, adminOnlyMiddleware, ctrl.DeleteVariant)
}
