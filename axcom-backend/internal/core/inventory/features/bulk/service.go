// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package bulk

import (
	"context"

	"ecom-engine/internal/core/inventory/domain"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
)

// UpdateItem is a struct that represents an item that needs to be updated.
type UpdateItem struct {
	VariantID  string
	LocationID string
	Quantity   int
}

type Service interface {
	BulkUpdate(ctx context.Context, updates []UpdateItem) error
}

type bulkService struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &bulkService{
		repo: repo,
	}
}

// BulkUpdate is used to update the stock of multiple items at once.
func (s *bulkService) BulkUpdate(ctx context.Context, updates []UpdateItem) error {
	logger.InfoCtx(ctx, "Starting bulk update of %d items", len(updates))

	for _, item := range updates {
		loc := item.LocationID
		if loc == "" {
			loc = "default"
		}

		stock, err := s.repo.GetStock(ctx, item.VariantID, loc)
		if err != nil {
			stock = &domain.StockItem{
				VariantID:         item.VariantID,
				LocationID:        loc,
				Quantity:          0,
				LowStockThreshold: domain.DefaultLowStockThreshold,
				AllowBackorders:   false,
				BackorderLimit:    0,
			}
		}

		if err := stock.Update(item.Quantity); err != nil {
			logger.ErrorCtx(ctx, "Bulk update failed validation for variant %s at location %s: %v", item.VariantID, loc, err)
			return apperrors.NewBadRequest("bulk update item validation failed", err)
		}

		if err := s.repo.SaveStock(ctx, stock); err != nil {
			logger.ErrorCtx(ctx, "Bulk update failed saving for variant %s at location %s: %v", item.VariantID, loc, err)
			return apperrors.NewInternal("bulk update database error", err)
		}
	}

	logger.InfoCtx(ctx, "Successfully completed bulk update of %d items", len(updates))
	return nil
}
