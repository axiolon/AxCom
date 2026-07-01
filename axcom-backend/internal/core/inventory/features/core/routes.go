// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package core contains the core inventory management logic.
package core

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers the inventory routes with the given router group.
func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware gin.HandlerFunc, adminOnlyMiddleware gin.HandlerFunc) {
	rg.GET("/inventory", authMiddleware, adminOnlyMiddleware, ctrl.List)
	rg.GET("/inventory/alerts", authMiddleware, adminOnlyMiddleware, ctrl.Alerts)
	rg.GET("/inventory/:variantID", ctrl.Check)
	rg.POST("/inventory/update", authMiddleware, adminOnlyMiddleware, ctrl.Update)
	rg.POST("/inventory/configure", authMiddleware, adminOnlyMiddleware, ctrl.Configure)
	rg.DELETE("/inventory/:variantID", authMiddleware, adminOnlyMiddleware, ctrl.Delete)
}
