// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// This submodule implements capabilities to sync stock values from third-party/external channels, updating internal stock values and publishing events.
package sync

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware gin.HandlerFunc) {
	rg.POST("/inventory/sync", authMiddleware, ctrl.Sync)
}
