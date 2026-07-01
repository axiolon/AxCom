// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cart

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all cart HTTP endpoints onto the provided RouterGroup.
func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller) {
	rg.GET("/cart", ctrl.GetCart)
	rg.GET("/cart/count", ctrl.GetCartCount)
	rg.POST("/cart", ctrl.AddItem)
	rg.PUT("/cart/items/:variantId", ctrl.UpdateItem)
	rg.DELETE("/cart/items/:variantId", ctrl.RemoveItem)
	rg.DELETE("/cart", ctrl.Clear)
}
