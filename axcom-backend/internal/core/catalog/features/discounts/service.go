// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package discounts

import (
	"context"
	"fmt"

	"ecom-engine/internal/core/catalog/domain"
	apperrors "ecom-engine/pkg/errors"
)

// Service defines the business contract for product discounts.
type Service interface {
	ApplyDiscount(ctx context.Context, productID string, discount *domain.ProductDiscount) error
	RemoveDiscount(ctx context.Context, productID string) error
}

type discountService struct {
	repo Repository
}

// NewService creates a new discountService.
func NewService(repo Repository) Service {
	return &discountService{repo: repo}
}

func (s *discountService) ApplyDiscount(ctx context.Context, productID string, d *domain.ProductDiscount) error {
	if d == nil {
		return apperrors.NewBadRequest("Discount payload is required", nil)
	}

	if d.Type != "percentage" && d.Type != "fixed" {
		return apperrors.NewBadRequest("Discount type must be either 'percentage' or 'fixed'", nil)
	}

	if d.Value < 0 {
		return apperrors.NewBadRequest("Discount value cannot be negative", nil)
	}

	if d.Type == "percentage" && d.Value > 100 {
		return apperrors.NewBadRequest("Percentage discount cannot exceed 100%", nil)
	}

	// Verify product exists
	_, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return apperrors.NewNotFound(fmt.Sprintf("product %s not found", productID), err)
	}

	if err := s.repo.UpdateProductDiscount(ctx, productID, d); err != nil {
		return apperrors.NewInternal("failed to apply discount", err)
	}

	return nil
}

func (s *discountService) RemoveDiscount(ctx context.Context, productID string) error {
	// Verify product exists
	_, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return apperrors.NewNotFound(fmt.Sprintf("product %s not found", productID), err)
	}

	if err := s.repo.UpdateProductDiscount(ctx, productID, nil); err != nil {
		return apperrors.NewInternal("failed to remove discount", err)
	}

	return nil
}
