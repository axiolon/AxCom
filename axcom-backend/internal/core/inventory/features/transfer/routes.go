// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package transfer handles the inventory transfer feature.
// Used for transferring inventory between locations
package transfer

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware gin.HandlerFunc) {
	rg.POST("/inventory/transfer", authMiddleware, ctrl.Transfer)
}
