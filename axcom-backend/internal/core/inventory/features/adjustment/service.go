// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package adjustment

import (
	"context"
	"errors"
	"fmt"

	"ecom-engine/internal/core/inventory/domain"
	"ecom-engine/internal/events"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
)

// Interface for Adjustment service
type Service interface {
	AdjustStock(ctx context.Context, variantID string, locationID string, qty int, reason string) error
}

type adjustmentService struct {
	repo     Repository
	eventBus events.EventBus
	outbox   events.OutboxRepository
}

func NewService(repo Repository, bus events.EventBus, outbox events.OutboxRepository) Service {
	return &adjustmentService{
		repo:     repo,
		eventBus: bus,
		outbox:   outbox,
	}
}

// AdjustStock adjusts the stock for a specific variant.
// Can Increase or decrease stock quantity.
// Should Include reason for compliance.
func (s *adjustmentService) AdjustStock(ctx context.Context, variantID string, locationID string, qty int, reason string) error {
	if locationID == "" {
		locationID = "default"
	}
	if reason == "" {
		return apperrors.NewBadRequest("adjustment reason is required", nil)
	}

	logger.InfoCtx(ctx, "Adjusting stock of variant %s at %s by %d (reason: %s)", variantID, locationID, qty, reason)

	stock, err := s.repo.GetStock(ctx, variantID, locationID)
	if err != nil {
		stock = &domain.StockItem{
			VariantID:         variantID,
			LocationID:        locationID,
			Quantity:          0,
			LowStockThreshold: domain.DefaultLowStockThreshold,
			AllowBackorders:   false,
			BackorderLimit:    0,
		}
	}

	oldQty := stock.Quantity
	newQty := oldQty + qty

	if err := stock.Update(newQty); err != nil {
		logger.ErrorCtx(ctx, "Adjustment validation failed: %v", err)
		if errors.Is(err, domain.ErrInvalidQuantity) {
			return apperrors.NewBadRequest("adjusted stock quantity cannot be negative", domain.ErrInvalidQuantity)
		}
		return apperrors.NewBadRequest("adjustment validation failed", err)
	}

	if err := s.repo.SaveStock(ctx, stock); err != nil {
		logger.ErrorCtx(ctx, "Failed to save adjusted stock: %v", err)
		return apperrors.NewInternal("failed to save adjusted stock", err)
	}

	evt := events.NewEventFromCtx(ctx, events.InventoryStockChangedTopic, "inventory",
		&events.StockChangedPayload{
			VariantID:    variantID,
			LocationID:   locationID,
			OldQuantity:  oldQty,
			NewQuantity:  stock.Quantity,
			ChangeReason: fmt.Sprintf("adjustment:%s", reason),
			ChangedBy:    "system",
		})
	if s.outbox != nil {
		if err := s.outbox.Store(ctx, evt); err != nil {
			logger.ErrorCtx(ctx, "Failed to store adjustment event in outbox: %v", err)
		}
	} else if s.eventBus != nil {
		s.eventBus.Publish(evt)
	}

	logger.InfoCtx(ctx, "Successfully adjusted stock of variant %s at %s by %d", variantID, locationID, qty)
	return nil
}
