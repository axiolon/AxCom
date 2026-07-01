// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package payments

import "github.com/gin-gonic/gin"

// RegisterRoutes registers the public / secure user payments HTTP routes.
func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware gin.HandlerFunc) {
	// Webhooks/callbacks are public
	rg.POST("/payments/callback/:provider", ctrl.ProcessCallback)

	// Intent creation / listing requires authenticated customer
	secured := rg.Group("")
	secured.Use(authMiddleware)
	{
		secured.POST("/payments/intent", ctrl.CreateIntent)
		secured.GET("/payments", ctrl.ListPayments)
		secured.GET("/payments/by-order/:orderID", ctrl.GetPaymentByOrderID)
	}
}
