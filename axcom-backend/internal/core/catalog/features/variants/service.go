// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package variants

import (
	"context"
	"fmt"

	"ecom-engine/internal/core/catalog/domain"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/idgen"
)

// Service defines the business contract for managing product variants.
type Service interface {
	GetVariants(ctx context.Context, productID string) ([]domain.Variant, error)
	AddVariant(ctx context.Context, productID string, v *domain.Variant) error
	UpdateVariant(ctx context.Context, productID string, v *domain.Variant) error
	DeleteVariant(ctx context.Context, productID string, variantID string) error
}

type variantService struct {
	repo Repository
}

// NewService creates a new variantService.
func NewService(repo Repository) Service {
	return &variantService{repo: repo}
}

func (s *variantService) GetVariants(ctx context.Context, productID string) ([]domain.Variant, error) {
	p, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return nil, apperrors.NewNotFound(fmt.Sprintf("product %s not found", productID), err)
	}
	return p.Variants, nil
}

func (s *variantService) AddVariant(ctx context.Context, productID string, v *domain.Variant) error {
	p, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return apperrors.NewNotFound(fmt.Sprintf("product %s not found", productID), err)
	}

	if v.ID == "" {
		id, err := idgen.Generate("var_")
		if err != nil {
			return apperrors.NewInternal("failed to generate variant ID", err)
		}
		v.ID = id
	}

	// Add to temporary slice to validate uniqueness (copy first to prevent in-place capacity mutation)
	candidateVariants := make([]domain.Variant, len(p.Variants))
	copy(candidateVariants, p.Variants)
	candidateVariants = append(candidateVariants, *v)
	if err := domain.ValidateVariants(candidateVariants); err != nil {
		return apperrors.NewBadRequest(err.Error(), err)
	}

	p.Variants = candidateVariants
	if err := s.repo.UpdateProductVariants(ctx, productID, p.Variants); err != nil {
		return apperrors.NewInternal("failed to add variant", err)
	}

	return nil
}

func (s *variantService) UpdateVariant(ctx context.Context, productID string, v *domain.Variant) error {
	p, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return apperrors.NewNotFound(fmt.Sprintf("product %s not found", productID), err)
	}

	if v.ID == "" {
		return apperrors.NewBadRequest("Variant ID is required for update", nil)
	}

	foundIdx := -1
	for i, existing := range p.Variants {
		if existing.ID == v.ID {
			foundIdx = i
			break
		}
	}

	if foundIdx == -1 {
		return apperrors.NewNotFound(fmt.Sprintf("variant %s not found on product", v.ID), nil)
	}

	// Update fields in a copy slice for validation
	candidateVariants := make([]domain.Variant, len(p.Variants))
	copy(candidateVariants, p.Variants)
	candidateVariants[foundIdx] = *v

	if err := domain.ValidateVariants(candidateVariants); err != nil {
		return apperrors.NewBadRequest(err.Error(), err)
	}

	p.Variants = candidateVariants
	if err := s.repo.UpdateProductVariants(ctx, productID, p.Variants); err != nil {
		return apperrors.NewInternal("failed to update variant", err)
	}

	return nil
}

func (s *variantService) DeleteVariant(ctx context.Context, productID string, variantID string) error {
	p, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return apperrors.NewNotFound(fmt.Sprintf("product %s not found", productID), err)
	}

	if len(p.Variants) <= 1 {
		return apperrors.NewBadRequest("cannot delete variant: products must have at least one variant", nil)
	}

	foundIdx := -1
	for i, v := range p.Variants {
		if v.ID == variantID {
			foundIdx = i
			break
		}
	}

	if foundIdx == -1 {
		return apperrors.NewNotFound(fmt.Sprintf("variant %s not found on product", variantID), nil)
	}

	// Remove element from slice
	p.Variants = append(p.Variants[:foundIdx], p.Variants[foundIdx+1:]...)

	if err := s.repo.UpdateProductVariants(ctx, productID, p.Variants); err != nil {
		return apperrors.NewInternal("failed to delete variant", err)
	}

	return nil
}
