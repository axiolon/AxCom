// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reviews

import (
	"context"
	"errors"
	"sync"
)

// Repository defines the persistence contract for reviews.
type Repository interface {
	CreateReview(_ context.Context, r *Review) error
	GetReviewsByProductID(_ context.Context, productID string) ([]Review, error)
	GetReviewByID(_ context.Context, id string) (*Review, error)
	UpdateReview(_ context.Context, r *Review) error
	DeleteReview(_ context.Context, id string) error
}

type memReviewRepo struct {
	mu      sync.RWMutex
	reviews map[string]*Review
}

// NewMemoryRepository creates a new in-memory repository for reviews.
func NewMemoryRepository() Repository {
	return &memReviewRepo{
		reviews: make(map[string]*Review),
	}
}

func (r *memReviewRepo) CreateReview(_ context.Context, review *Review) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.reviews[review.ID]; exists {
		return errors.New("review already exists")
	}

	r.reviews[review.ID] = review
	return nil
}

func (r *memReviewRepo) GetReviewsByProductID(_ context.Context, productID string) ([]Review, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Review
	for _, rev := range r.reviews {
		if rev.ProductID == productID {
			result = append(result, *rev)
		}
	}
	return result, nil
}

func (r *memReviewRepo) GetReviewByID(_ context.Context, id string) (*Review, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rev, exists := r.reviews[id]
	if !exists {
		return nil, ErrReviewNotFound
	}
	return rev, nil
}

func (r *memReviewRepo) DeleteReview(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.reviews[id]; !exists {
		return ErrReviewNotFound
	}
	delete(r.reviews, id)
	return nil
}

func (r *memReviewRepo) UpdateReview(_ context.Context, review *Review) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.reviews[review.ID]; !exists {
		return ErrReviewNotFound
	}
	r.reviews[review.ID] = review
	return nil
}
