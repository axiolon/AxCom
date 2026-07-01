// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package admin

import (
	"github.com/gin-gonic/gin"
)

// RegisterAdminRoutes registers administrative payments HTTP routes.
func RegisterAdminRoutes(rg *gin.RouterGroup, ctrl *Controller) {
	rg.GET("/admin/payments", ctrl.ListPayments)
	rg.POST("/admin/payments/refund", ctrl.RefundPayment)
	rg.GET("/admin/payments/:id", ctrl.GetPaymentByID)
}
