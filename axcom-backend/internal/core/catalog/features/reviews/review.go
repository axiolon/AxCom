// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reviews

import (
	"errors"
	"time"
)

var (
	ErrInvalidRating   = errors.New("rating must be between 1 and 5")
	ErrEmptyComment    = errors.New("comment cannot be empty")
	ErrProductNotFound = errors.New("product not found")
	ErrReviewNotFound  = errors.New("review not found")
)

// ReviewReply represents a reply to a product review.
type ReviewReply struct {
	UserID    string    `json:"user_id"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

// Review represents a customer review for a catalog product.
type Review struct {
	ID        string       `json:"id"`
	ProductID string       `json:"product_id"`
	UserID    string       `json:"user_id"`
	Rating    int          `json:"rating"`
	Comment   string       `json:"comment"`
	Reply     *ReviewReply `json:"reply,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

// ValidateReview performs validation checks on a review.
func ValidateReview(r Review) error {
	if r.Rating < 1 || r.Rating > 5 {
		return ErrInvalidRating
	}
	if len(r.Comment) == 0 {
		return ErrEmptyComment
	}
	return nil
}
