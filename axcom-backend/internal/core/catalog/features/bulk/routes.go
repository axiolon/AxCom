// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package bulk

import "github.com/gin-gonic/gin"

// RegisterRoutes registers bulk operations HTTP routes.
func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware gin.HandlerFunc, adminOnlyMiddleware gin.HandlerFunc) {
	// Secured endpoints (bulk operations are admin tasks)
	rg.POST("/products/bulk", authMiddleware, adminOnlyMiddleware, ctrl.BulkCreate)
	rg.PUT("/products/bulk", authMiddleware, adminOnlyMiddleware, ctrl.BulkUpdate)
	rg.DELETE("/products/bulk", authMiddleware, adminOnlyMiddleware, ctrl.BulkDelete)
}
