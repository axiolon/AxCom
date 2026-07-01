// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package discounts

import "github.com/gin-gonic/gin"

// RegisterRoutes registers product discounts HTTP routes.
func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware, adminOnlyMiddleware gin.HandlerFunc) {
	// Protected endpoints (discounts are admin operations)
	rg.POST("/products/:id/discount", authMiddleware, adminOnlyMiddleware, ctrl.Apply)
	rg.DELETE("/products/:id/discount", authMiddleware, adminOnlyMiddleware, ctrl.Remove)
}
