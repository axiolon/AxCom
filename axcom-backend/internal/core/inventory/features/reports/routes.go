// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package reports handles the inventory reports feature.
// Used for exporting inventory reports
package reports

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware gin.HandlerFunc) {
	rg.GET("/inventory/low-stock", authMiddleware, ctrl.LowStock)
	rg.GET("/inventory/export", authMiddleware, ctrl.Export)
}
