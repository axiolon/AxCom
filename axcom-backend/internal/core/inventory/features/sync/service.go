// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"context"

	"ecom-engine/internal/core/inventory/domain"
	"ecom-engine/internal/events"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
)

// Service defines the interface for syncing inventory stock levels.
type Service interface {
	// SyncStock updates the stock level for a specific variant at a given location.
	SyncStock(ctx context.Context, variantID string, locationID string, qty int) error
}

// syncService implements the Service interface.
type syncService struct {
	repo     Repository
	eventBus events.EventBus
	outbox   events.OutboxRepository
}

// NewService creates and returns a new instance of the stock sync Service.
func NewService(repo Repository, bus events.EventBus, outbox events.OutboxRepository) Service {
	return &syncService{
		repo:     repo,
		eventBus: bus,
		outbox:   outbox,
	}
}

// SyncStock handles updating the stock level for a given variant and location.
// If the location ID is empty, it defaults to "default".
// If the stock record does not exist, a new one is initialized.
// It also publishes a StockChanged event upon successful synchronization.
func (s *syncService) SyncStock(ctx context.Context, variantID string, locationID string, qty int) error {
	// Fallback to default location if none is provided
	if locationID == "" {
		locationID = "default"
	}

	logger.InfoCtx(ctx, "Syncing stock level for variant %s at location %s to %d", variantID, locationID, qty)

	// Retrieve existing stock; if not found, initialize a new stock item record
	stock, err := s.repo.GetStock(ctx, variantID, locationID)
	if err != nil {
		if isNotFoundError(err) {
			stock = &domain.StockItem{
				VariantID:         variantID,
				LocationID:        locationID,
				Quantity:          0,
				LowStockThreshold: domain.DefaultLowStockThreshold,
				AllowBackorders:   false,
				BackorderLimit:    0,
			}
		} else {
			logger.ErrorCtx(ctx, "Failed to retrieve stock level for variant %s: %v", variantID, err)
			return apperrors.NewInternal("failed to retrieve stock level during sync", err)
		}
	}

	oldQty := stock.Quantity

	// Use domain validation/update logic to ensure quantity rules (e.g. non-negative) are satisfied
	if err := stock.Update(qty); err != nil {
		logger.ErrorCtx(ctx, "Validation failed for stock level sync of variant %s: %v", variantID, err)
		return apperrors.NewBadRequest("invalid sync quantity", err)
	}

	// Save the updated stock item record
	if err := s.repo.SaveStock(ctx, stock); err != nil {
		logger.ErrorCtx(ctx, "Failed to save stock sync: %v", err)
		return apperrors.NewInternal("failed to save stock sync level", err)
	}

	evt := events.NewEventFromCtx(ctx, events.InventoryStockChangedTopic, "inventory",
		&events.StockChangedPayload{
			VariantID:    variantID,
			LocationID:   locationID,
			OldQuantity:  oldQty,
			NewQuantity:  qty,
			ChangeReason: "sync:external",
			ChangedBy:    "sync_service",
		})
	if s.outbox != nil {
		if err := s.outbox.Store(ctx, evt); err != nil {
			logger.ErrorCtx(ctx, "Failed to store sync event in outbox: %v", err)
		}
	} else if s.eventBus != nil {
		s.eventBus.Publish(evt)
	}

	logger.InfoCtx(ctx, "Successfully synced stock level for variant %s at location %s to %d", variantID, locationID, qty)
	return nil
}

// isNotFoundError returns true if the error indicates that the stock record does not exist.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return errStr == "mongo: no documents in result" ||
		errStr == "not found" ||
		errStr == "sql: no rows in result set" ||
		errStr == "stock item not found"
}
