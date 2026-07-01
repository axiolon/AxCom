// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reviews

import "time"

// CreateReviewRequest is the payload for submitting a product review.
type CreateReviewRequest struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5"`
	Comment string `json:"comment" binding:"required"`
}

// SubmitReplyRequest is the payload for submitting a reply to a review.
type SubmitReplyRequest struct {
	Comment string `json:"comment" binding:"required"`
}

// ReviewReplyResponse defines the HTTP response payload for a review reply.
type ReviewReplyResponse struct {
	UserID    string    `json:"user_id"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

// ReviewResponse defines the HTTP response payload for a review.
type ReviewResponse struct {
	ID        string               `json:"id"`
	ProductID string               `json:"product_id"`
	UserID    string               `json:"user_id"`
	Rating    int                  `json:"rating"`
	Comment   string               `json:"comment"`
	Reply     *ReviewReplyResponse `json:"reply,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
}

// ProductRatingResponse contains average rating stats for a product.
type ProductRatingResponse struct {
	ProductID     string  `json:"product_id"`
	AverageRating float64 `json:"average_rating"`
	ReviewCount   int     `json:"review_count"`
}

// UpdateReviewRequest is the payload for editing a product review.
type UpdateReviewRequest struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5"`
	Comment string `json:"comment" binding:"required"`
}
