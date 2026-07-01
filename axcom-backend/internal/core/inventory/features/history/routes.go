// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package history handles the inventory history feature.
// Used for audit service so that we can know inventory updates.
package history

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware gin.HandlerFunc) {
	rg.GET("/inventory/:variantID/history", authMiddleware, ctrl.History)
}
