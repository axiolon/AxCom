// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reviews

import (
	"context"
	"fmt"
	"time"

	"ecom-engine/internal/core/catalog/domain"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/idgen"
)

// ProductVerifier defines the dependency required to verify product existence.
type ProductVerifier interface {
	GetProduct(ctx context.Context, id string) (*domain.Product, error)
}

// Service defines the business logic contract for reviews.
type Service interface {
	AddReview(ctx context.Context, r *Review) error
	GetReviewsByProductID(ctx context.Context, productID string) ([]Review, error)
	GetAverageRating(ctx context.Context, productID string) (float64, int, error)
	SubmitReply(ctx context.Context, reviewID string, reply *ReviewReply) error
	DeleteReview(ctx context.Context, reviewID string, userID string, isAdmin bool) error
	UpdateReview(ctx context.Context, reviewID string, userID string, rating int, comment string) (*Review, error)
}

type reviewService struct {
	repo       Repository
	productVer ProductVerifier
}

// NewReviewService creates and returns a new instance of the review Service.
func NewReviewService(repo Repository, productVer ProductVerifier) Service {
	return &reviewService{
		repo:       repo,
		productVer: productVer,
	}
}

func (s *reviewService) AddReview(ctx context.Context, r *Review) error {
	if r.ID == "" {
		id, err := idgen.Generate("rev_")
		if err != nil {
			return apperrors.NewInternal("failed to generate review ID", err)
		}
		r.ID = id
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}

	// Domain logic validation
	if err := ValidateReview(*r); err != nil {
		return apperrors.NewBadRequest(err.Error(), err)
	}

	// Verify target product exists
	_, err := s.productVer.GetProduct(ctx, r.ProductID)
	if err != nil {
		return apperrors.NewNotFound(fmt.Sprintf("product with ID %s not found", r.ProductID), err)
	}

	// Verify user hasn't already reviewed this product (C4-1)
	existing, err := s.repo.GetReviewsByProductID(ctx, r.ProductID)
	if err == nil {
		for _, ex := range existing {
			if ex.UserID == r.UserID {
				return apperrors.NewConflict("user has already submitted a review for this product", nil)
			}
		}
	}

	// Persist
	if err := s.repo.CreateReview(ctx, r); err != nil {
		return apperrors.NewInternal("failed to save review", err)
	}

	return nil
}

func (s *reviewService) GetReviewsByProductID(ctx context.Context, productID string) ([]Review, error) {
	// Verify target product exists
	_, err := s.productVer.GetProduct(ctx, productID)
	if err != nil {
		return nil, apperrors.NewNotFound(fmt.Sprintf("product with ID %s not found", productID), err)
	}

	return s.repo.GetReviewsByProductID(ctx, productID)
}

func (s *reviewService) GetAverageRating(ctx context.Context, productID string) (float64, int, error) {
	// Verify target product exists
	_, err := s.productVer.GetProduct(ctx, productID)
	if err != nil {
		return 0, 0, apperrors.NewNotFound(fmt.Sprintf("product with ID %s not found", productID), err)
	}

	reviews, err := s.repo.GetReviewsByProductID(ctx, productID)
	if err != nil {
		return 0, 0, apperrors.NewInternal("failed to retrieve reviews", err)
	}

	if len(reviews) == 0 {
		return 0, 0, nil
	}

	var sum int
	for _, r := range reviews {
		sum += r.Rating
	}

	avg := float64(sum) / float64(len(reviews))
	return avg, len(reviews), nil
}

func (s *reviewService) DeleteReview(ctx context.Context, reviewID string, userID string, isAdmin bool) error {
	rev, err := s.repo.GetReviewByID(ctx, reviewID)
	if err != nil {
		if err == ErrReviewNotFound {
			return apperrors.NewNotFound("review not found", err)
		}
		return apperrors.NewInternal("failed to retrieve review", err)
	}

	if !isAdmin && rev.UserID != userID {
		return apperrors.NewForbidden("you are not authorized to delete this review", nil)
	}

	if err := s.repo.DeleteReview(ctx, reviewID); err != nil {
		return apperrors.NewInternal("failed to delete review", err)
	}

	return nil
}

func (s *reviewService) SubmitReply(ctx context.Context, reviewID string, reply *ReviewReply) error {
	if reply.Comment == "" {
		return apperrors.NewBadRequest("reply comment cannot be empty", ErrEmptyComment)
	}
	if reply.CreatedAt.IsZero() {
		reply.CreatedAt = time.Now()
	}

	rev, err := s.repo.GetReviewByID(ctx, reviewID)
	if err != nil {
		if err == ErrReviewNotFound {
			return apperrors.NewNotFound("review not found", err)
		}
		return apperrors.NewInternal("failed to retrieve review", err)
	}

	rev.Reply = reply

	if err := s.repo.UpdateReview(ctx, rev); err != nil {
		return apperrors.NewInternal("failed to save reply", err)
	}

	return nil
}

func (s *reviewService) UpdateReview(ctx context.Context, reviewID string, userID string, rating int, comment string) (*Review, error) {
	rev, err := s.repo.GetReviewByID(ctx, reviewID)
	if err != nil {
		if err == ErrReviewNotFound {
			return nil, apperrors.NewNotFound("review not found", err)
		}
		return nil, apperrors.NewInternal("failed to retrieve review", err)
	}

	if rev.UserID != userID {
		return nil, apperrors.NewForbidden("you are not authorized to edit this review", nil)
	}

	rev.Rating = rating
	rev.Comment = comment

	if err := ValidateReview(*rev); err != nil {
		return nil, apperrors.NewBadRequest(err.Error(), err)
	}

	if err := s.repo.UpdateReview(ctx, rev); err != nil {
		return nil, apperrors.NewInternal("failed to update review", err)
	}

	return rev, nil
}
