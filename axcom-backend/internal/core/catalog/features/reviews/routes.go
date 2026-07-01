// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reviews

import "github.com/gin-gonic/gin"

// RegisterRoutes registers reviews-specific API endpoints onto the provided RouterGroup.
func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware, adminOnlyMiddleware gin.HandlerFunc) {
	// Public endpoints
	rg.GET("/products/:id/reviews", ctrl.GetReviews)
	rg.GET("/products/:id/reviews/rating", ctrl.GetRatingSummary)

	// Protected endpoints
	rg.POST("/products/:id/reviews", authMiddleware, ctrl.SubmitReview)
	rg.PUT("/products/:id/reviews/:reviewId", authMiddleware, ctrl.UpdateReview)
	rg.POST("/products/:id/reviews/:reviewId/reply", authMiddleware, adminOnlyMiddleware, ctrl.SubmitReply)
	rg.DELETE("/products/:id/reviews/:reviewId", authMiddleware, ctrl.DeleteReview)
}
