// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package transfer implements inventory stock transfer logic between different locations.
package transfer

import (
	"context"
	"fmt"

	"ecom-engine/internal/core/inventory/domain"
	"ecom-engine/internal/events"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
)

// Service defines the interface for executing stock transfers between locations.
type Service interface {
	// TransferStock transfers a specified quantity of a variant from one location to another.
	TransferStock(ctx context.Context, variantID string, fromLoc string, toLoc string, qty int) error
}

// transferService implements the Service interface using a repository and an event bus.
type transferService struct {
	repo     Repository
	eventBus events.EventBus
	outbox   events.OutboxRepository
}

// NewService constructs a new transfer Service instance.
func NewService(repo Repository, bus events.EventBus, outbox events.OutboxRepository) Service {
	return &transferService{
		repo:     repo,
		eventBus: bus,
		outbox:   outbox,
	}
}

// TransferStock moves stock quantities between the specified source and destination locations.
// It handles input validation, stock level checks, transactional-style rollback if updates fail,
// and publishes InventoryStockChanged events.
func (s *transferService) TransferStock(ctx context.Context, variantID string, fromLoc string, toLoc string, qty int) error {
	// Validate stock transfer quantity
	if qty <= 0 {
		return apperrors.NewBadRequest("transfer quantity must be positive", nil)
	}

	// Default empty locations to "default"
	if fromLoc == "" {
		fromLoc = "default"
	}
	if toLoc == "" {
		toLoc = "default"
	}

	// Prevent transferring to the same location
	if fromLoc == toLoc {
		return apperrors.NewBadRequest("source and destination locations must be different", nil)
	}

	logger.InfoCtx(ctx, "Attempting to transfer %d units of variant %s from %s to %s", qty, variantID, fromLoc, toLoc)

	// Fetch source stock details
	sourceStock, err := s.repo.GetStock(ctx, variantID, fromLoc)
	if err != nil {
		return apperrors.NewNotFound(fmt.Sprintf("source stock record not found for variant %s at location %s", variantID, fromLoc), err)
	}

	// Validate sufficient stock is available for transfer
	if qty > sourceStock.Quantity {
		return apperrors.NewConflict(fmt.Sprintf("insufficient stock for transfer. available: %d, requested: %d", sourceStock.Quantity, qty), nil)
	}

	// Fetch destination stock details; initialize a new record if it does not exist
	destStock, err := s.repo.GetStock(ctx, variantID, toLoc)
	if err != nil {
		destStock = &domain.StockItem{
			VariantID:         variantID,
			LocationID:        toLoc,
			Quantity:          0,
			LowStockThreshold: domain.DefaultLowStockThreshold,
			AllowBackorders:   false,
			BackorderLimit:    0,
		}
	}

	// Record original quantities for potential rollback
	sourceOldQty := sourceStock.Quantity
	destOldQty := destStock.Quantity

	// Perform stock adjustments
	sourceStock.Quantity -= qty
	destStock.Quantity += qty

	// Save source stock decrement
	if err := s.repo.SaveStock(ctx, sourceStock); err != nil {
		logger.ErrorCtx(ctx, "Failed to save source stock after decrement: %v", err)
		return apperrors.NewInternal("failed to update source stock level", err)
	}

	// Save destination stock increment
	if err := s.repo.SaveStock(ctx, destStock); err != nil {
		// Attempt rollback on source to preserve data consistency
		sourceStock.Quantity = sourceOldQty
		_ = s.repo.SaveStock(ctx, sourceStock)
		logger.ErrorCtx(ctx, "Failed to save destination stock after increment: %v", err)
		return apperrors.NewInternal("failed to update destination stock level", err)
	}

	srcEvt := events.NewEventFromCtx(ctx, events.InventoryStockChangedTopic, "inventory",
		&events.StockChangedPayload{
			VariantID:    variantID,
			LocationID:   fromLoc,
			OldQuantity:  sourceOldQty,
			NewQuantity:  sourceStock.Quantity,
			ChangeReason: fmt.Sprintf("transfer:out:to:%s", toLoc),
			ChangedBy:    "system",
		})
	destEvt := events.NewEventFromCtx(ctx, events.InventoryStockChangedTopic, "inventory",
		&events.StockChangedPayload{
			VariantID:    variantID,
			LocationID:   toLoc,
			OldQuantity:  destOldQty,
			NewQuantity:  destStock.Quantity,
			ChangeReason: fmt.Sprintf("transfer:in:from:%s", fromLoc),
			ChangedBy:    "system",
		})

	if s.outbox != nil {
		if err := s.outbox.Store(ctx, srcEvt, destEvt); err != nil {
			logger.ErrorCtx(ctx, "Failed to store transfer events in outbox: %v", err)
		}
	} else if s.eventBus != nil {
		s.eventBus.Publish(srcEvt)
		s.eventBus.Publish(destEvt)
	}

	logger.InfoCtx(ctx, "Successfully transferred %d units of variant %s from %s to %s", qty, variantID, fromLoc, toLoc)
	return nil
}
