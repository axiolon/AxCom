// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package reservation implements stock reservation capabilities, allowing stock to be locked during checkout sequences and released on cancellations or timeouts.
package reservation

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware gin.HandlerFunc) {
	rg.POST("/inventory/:variantID/reserve", authMiddleware, ctrl.Reserve)
	rg.DELETE("/inventory/:variantID/reserve/:reservationID", authMiddleware, ctrl.ReleaseReservation)
}
