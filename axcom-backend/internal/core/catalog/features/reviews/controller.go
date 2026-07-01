// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reviews

import (
	"time"

	"ecom-engine/pkg/ctxkeys"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

// Controller handles HTTP requests for reviews.
type Controller struct {
	service Service
}

// NewController creates a new Controller instance.
func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// SubmitReview handles POST /products/:id/reviews requests.
func (ctrl *Controller) SubmitReview(c *gin.Context) {
	productID := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received SubmitReview request for product ID: %s", productID)

	userID := c.GetString(string(ctxkeys.UserIDKey))
	if userID == "" {
		response.GinWriteError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}

	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid review payload", err))
		return
	}

	r := &Review{
		ProductID: productID,
		UserID:    userID,
		Rating:    req.Rating,
		Comment:   req.Comment,
	}

	if err := ctrl.service.AddReview(c.Request.Context(), r); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, mapReviewToResponse(*r))
}

// GetReviews handles GET /products/:id/reviews requests.
func (ctrl *Controller) GetReviews(c *gin.Context) {
	productID := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received GetReviews request for product ID: %s", productID)

	reviews, err := ctrl.service.GetReviewsByProductID(c.Request.Context(), productID)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	res := make([]ReviewResponse, len(reviews))
	for i, r := range reviews {
		res[i] = mapReviewToResponse(r)
	}

	response.GinOK(c, res)
}

// GetRatingSummary handles GET /products/:id/reviews/rating requests.
func (ctrl *Controller) GetRatingSummary(c *gin.Context) {
	productID := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received GetRatingSummary request for product ID: %s", productID)

	avg, count, err := ctrl.service.GetAverageRating(c.Request.Context(), productID)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, ProductRatingResponse{
		ProductID:     productID,
		AverageRating: avg,
		ReviewCount:   count,
	})
}

// DeleteReview handles DELETE /products/:id/reviews/:reviewId requests.
func (ctrl *Controller) DeleteReview(c *gin.Context) {
	reviewID := c.Param("reviewId")
	userID := c.GetString(string(ctxkeys.UserIDKey))
	role := c.GetString(string(ctxkeys.UserRoleKey))

	logger.InfoCtx(c.Request.Context(), "Received DeleteReview request for review ID: %s, user ID: %s", reviewID, userID)

	if userID == "" {
		response.GinWriteError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}

	if reviewID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Review ID is required", nil))
		return
	}

	isAdmin := role == "admin"

	if err := ctrl.service.DeleteReview(c.Request.Context(), reviewID, userID, isAdmin); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, gin.H{"message": "review deleted"})
}

// SubmitReply handles POST /products/:id/reviews/:reviewId/reply requests.
func (ctrl *Controller) SubmitReply(c *gin.Context) {
	reviewID := c.Param("reviewId")
	userID := c.GetString(string(ctxkeys.UserIDKey))

	logger.InfoCtx(c.Request.Context(), "Received SubmitReply request for review ID: %s, user ID: %s", reviewID, userID)

	if userID == "" {
		response.GinWriteError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}

	if reviewID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Review ID is required", nil))
		return
	}

	var req SubmitReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid reply payload", err))
		return
	}

	reply := &ReviewReply{
		UserID:    userID,
		Comment:   req.Comment,
		CreatedAt: time.Now(),
	}

	if err := ctrl.service.SubmitReply(c.Request.Context(), reviewID, reply); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, ReviewReplyResponse{
		UserID:    reply.UserID,
		Comment:   reply.Comment,
		CreatedAt: reply.CreatedAt,
	})
}

func mapReviewToResponse(r Review) ReviewResponse {
	var replyRes *ReviewReplyResponse
	if r.Reply != nil {
		replyRes = &ReviewReplyResponse{
			UserID:    r.Reply.UserID,
			Comment:   r.Reply.Comment,
			CreatedAt: r.Reply.CreatedAt,
		}
	}

	return ReviewResponse{
		ID:        r.ID,
		ProductID: r.ProductID,
		UserID:    r.UserID,
		Rating:    r.Rating,
		Comment:   r.Comment,
		Reply:     replyRes,
		CreatedAt: r.CreatedAt,
	}
}

// UpdateReview handles PUT /products/:id/reviews/:reviewId requests.
func (ctrl *Controller) UpdateReview(c *gin.Context) {
	reviewID := c.Param("reviewId")
	userID := c.GetString(string(ctxkeys.UserIDKey))

	logger.InfoCtx(c.Request.Context(), "Received UpdateReview request for review ID: %s, user ID: %s", reviewID, userID)

	if userID == "" {
		response.GinWriteError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}

	if reviewID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Review ID is required", nil))
		return
	}

	var req UpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid review payload", err))
		return
	}

	updated, err := ctrl.service.UpdateReview(c.Request.Context(), reviewID, userID, req.Rating, req.Comment)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, mapReviewToResponse(*updated))
}
