// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package adjustment provides the facility for adjustment-related inventory operations.
package adjustment

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware gin.HandlerFunc) {
	rg.POST("/inventory/:variantID/adjust", authMiddleware, ctrl.Adjust)
}
