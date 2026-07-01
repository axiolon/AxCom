// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package bulk

import (
	"context"
	"fmt"

	"ecom-engine/internal/core/catalog/domain"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/idgen"
	"ecom-engine/pkg/logger"
)

// Service defines the business contract for product bulk operations.
type Service interface {
	BulkCreate(ctx context.Context, products []*domain.Product) error
	BulkUpdate(ctx context.Context, products []*domain.Product) error
	BulkDelete(ctx context.Context, ids []string) error
}

type bulkService struct {
	repo Repository
}

// NewService creates a new bulkService.
func NewService(repo Repository) Service {
	return &bulkService{repo: repo}
}

func (s *bulkService) BulkCreate(ctx context.Context, products []*domain.Product) error {
	logger.InfoCtx(ctx, "Executing BulkCreate for %d products", len(products))

	for idx, p := range products {
		if p.ID == "" {
			id, err := idgen.Generate("prod_")
			if err != nil {
				return apperrors.NewInternal("failed to generate product ID", err)
			}
			p.ID = id
		}
		for i := range p.Variants {
			if p.Variants[i].ID == "" {
				id, err := idgen.Generate("var_")
				if err != nil {
					return apperrors.NewInternal("failed to generate variant ID", err)
				}
				p.Variants[i].ID = id
			}
		}

		if err := domain.ValidateProduct(*p); err != nil {
			return apperrors.NewBadRequest(fmt.Sprintf("validation failed for product at index %d: %v", idx, err), err)
		}

		// Verify category exists
		_, err := s.repo.GetCategoryByID(ctx, p.CategoryID)
		if err != nil {
			return apperrors.NewNotFound(fmt.Sprintf("category with ID %s not found for product at index %d", p.CategoryID, idx), err)
		}
	}

	if err := s.repo.BulkCreate(ctx, products); err != nil {
		return apperrors.NewInternal("bulk creation failed", err)
	}

	return nil
}

func (s *bulkService) BulkUpdate(ctx context.Context, products []*domain.Product) error {
	logger.InfoCtx(ctx, "Executing BulkUpdate for %d products", len(products))

	for idx, p := range products {
		if p.ID == "" {
			return apperrors.NewBadRequest(fmt.Sprintf("product ID is required for update at index %d", idx), nil)
		}

		if err := domain.ValidateProduct(*p); err != nil {
			return apperrors.NewBadRequest(fmt.Sprintf("validation failed for product at index %d: %v", idx, err), err)
		}

		// Verify category exists
		_, err := s.repo.GetCategoryByID(ctx, p.CategoryID)
		if err != nil {
			return apperrors.NewNotFound(fmt.Sprintf("category with ID %s not found for product at index %d", p.CategoryID, idx), err)
		}
	}

	if err := s.repo.BulkUpdate(ctx, products); err != nil {
		return apperrors.NewInternal("bulk update failed", err)
	}

	return nil
}

func (s *bulkService) BulkDelete(ctx context.Context, ids []string) error {
	logger.InfoCtx(ctx, "Executing BulkDelete for %d product IDs", len(ids))
	if len(ids) == 0 {
		return apperrors.NewBadRequest("at least one product ID must be provided", nil)
	}

	if err := s.repo.BulkDelete(ctx, ids); err != nil {
		return apperrors.NewInternal("bulk deletion failed", err)
	}

	return nil
}
