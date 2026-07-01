// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package bulk provides the facility for bulk-related inventory operations.
// It is used for adding stock to multiple variants at once.
package bulk

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware gin.HandlerFunc) {
	rg.POST("/inventory/bulk-update", authMiddleware, ctrl.BulkUpdate)
}
