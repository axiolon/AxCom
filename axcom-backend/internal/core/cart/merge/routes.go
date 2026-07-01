// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package merge

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all cart merge endpoints onto the provided RouterGroup.
func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller) {
	rg.POST("/cart/merge", ctrl.Merge)
}
