// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package admin

import (
	"github.com/gin-gonic/gin"
)

// RegisterAdminRoutes registers administrative shipping HTTP routes.
func RegisterAdminRoutes(rg *gin.RouterGroup, ctrl *Controller) {
	rg.GET("/admin/shipping", ctrl.ListShipments)
	rg.POST("/admin/shipping", ctrl.CreateShipment)
	rg.PUT("/admin/shipping/:id", ctrl.UpdateShipmentStatus)
	rg.DELETE("/admin/shipping/:id", ctrl.DeleteShipment)
}
