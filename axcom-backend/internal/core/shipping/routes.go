// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package shipping

import "github.com/gin-gonic/gin"

// RegisterRoutes registers shipping HTTP routes.
func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware gin.HandlerFunc) {
	// Calculating rates can be public or user-scoped
	rg.POST("/shipping/rates", ctrl.CalculateRates)

	// Public tracking lookup
	rg.GET("/shipping/track/:tracking_number", ctrl.TrackShipment)

	// Fetching order tracking requires user authentication
	secured := rg.Group("")
	secured.Use(authMiddleware)
	{
		secured.GET("/shipping/order/:order_id", ctrl.GetMyOrderShipment)
	}
}
