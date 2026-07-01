// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reservation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ecom-engine/internal/core/inventory/domain"
	"ecom-engine/internal/events"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/idgen"
	"ecom-engine/pkg/logger"
)

// Service defines the business logic contract for reserving and releasing stock.
type Service interface {
	// ReserveStock attempts to temporarily deduct stock for a variant and create a reservation record.
	ReserveStock(ctx context.Context, variantID string, locationID string, qty int) (*domain.Reservation, error)
	// ReleaseReservation manually releases a stock reservation, returning the quantity back to available stock.
	ReleaseReservation(ctx context.Context, reservationID string) error
}

// reservationService implements the Service interface using a repository and an event bus.
type reservationService struct {
	repo     Repository
	eventBus events.EventBus
	outbox   events.OutboxRepository
}

// NewService constructs a new reservation Service instance.
func NewService(repo Repository, bus events.EventBus, outbox events.OutboxRepository) Service {
	return &reservationService{
		repo:     repo,
		eventBus: bus,
		outbox:   outbox,
	}
}

// ReserveStock handles the stock reservation process.
// It retrieves stock levels, adjusts the quantity in memory, saves the stock,
// creates a reservation record in the database, and publishes a stock change event.
//
// NOTE/CAVEAT: This operation is non-transactional at the database layer.
//  1. Concurrent updates to the same stock item may result in a Lost Update (race condition).
//  2. If reservation creation fails after saving the stock, the service attempts to rollback
//     the stock quantity. In a highly concurrent environment, this rollback save will overwrite
//     and lose any other successful stock modifications made by concurrent requests.
func (s *reservationService) ReserveStock(ctx context.Context, variantID string, locationID string, qty int) (*domain.Reservation, error) {
	if locationID == "" {
		locationID = "default"
	}
	logger.InfoCtx(ctx, "Attempting to reserve %d stock for variant %s at location %s", qty, variantID, locationID)

	// 1. Retrieve current stock levels
	stock, err := s.repo.GetStock(ctx, variantID, locationID)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to retrieve stock for variant %s at location %s: %v", variantID, locationID, err)
		return nil, apperrors.NewInternal("failed to check stock availability", err)
	}

	oldQty := stock.Quantity

	// 2. Apply domain reservation logic (checks availability and backorder thresholds)
	if err = stock.Reserve(qty); err != nil {
		logger.ErrorCtx(ctx, "Domain stock reservation failed for variant %s at location %s: %v", variantID, locationID, err)
		if errors.Is(err, domain.ErrInsufficientStock) {
			return nil, apperrors.NewConflict("insufficient stock available", domain.ErrInsufficientStock)
		}
		if errors.Is(err, domain.ErrInvalidReservationQuantity) {
			return nil, apperrors.NewBadRequest("invalid reservation quantity", domain.ErrInvalidReservationQuantity)
		}
		return nil, apperrors.NewBadRequest("reservation validation failed", err)
	}

	// 3. Save modified stock quantity to database
	if err = s.repo.SaveStock(ctx, stock); err != nil {
		logger.ErrorCtx(ctx, "Failed to update stock after reservation for variant %s at location %s: %v", variantID, locationID, err)
		return nil, apperrors.NewInternal("failed to update stock levels", err)
	}

	// 4. Generate unique reservation ID
	resID, err := idgen.Generate("res_")
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to generate reservation ID: %v", err)
		return nil, apperrors.NewInternal("failed to generate reservation ID", err)
	}

	res := &domain.Reservation{
		ID:         resID,
		VariantID:  variantID,
		LocationID: locationID,
		Quantity:   qty,
		ExpiresAt:  time.Now().Add(15 * time.Minute),
	}

	// 5. Validate the generated reservation domain entity
	if err := domain.ValidateReservation(*res); err != nil {
		// Rollback stock decrement on validation failure
		stock.Quantity = oldQty
		_ = s.repo.SaveStock(ctx, stock)
		logger.ErrorCtx(ctx, "Reservation domain validation failed: %v", err)
		return nil, apperrors.NewBadRequest("invalid reservation record", err)
	}

	// 6. Persist the reservation record in the database
	if err := s.repo.CreateReservation(ctx, res); err != nil {
		// Rollback stock decrement on persistence failure
		stock.Quantity = oldQty
		_ = s.repo.SaveStock(ctx, stock)
		logger.ErrorCtx(ctx, "Failed to create reservation record for variant %s at location %s: %v", variantID, locationID, err)
		return nil, apperrors.NewInternal("failed to save reservation", err)
	}

	// 7. Publish stock changed event
	evt := events.NewEventFromCtx(ctx, events.InventoryStockChangedTopic, "inventory",
		&events.StockChangedPayload{
			VariantID:    variantID,
			LocationID:   locationID,
			OldQuantity:  oldQty,
			NewQuantity:  stock.Quantity,
			ChangeReason: fmt.Sprintf("reservation:%s", res.ID),
			ChangedBy:    "system",
		})
	if s.outbox != nil {
		if err := s.outbox.Store(ctx, evt); err != nil {
			logger.ErrorCtx(ctx, "Failed to store reservation event in outbox: %v", err)
		}
	} else if s.eventBus != nil {
		s.eventBus.Publish(evt)
	}

	logger.InfoCtx(ctx, "Successfully reserved %d stock for variant %s at location %s (Reservation ID: %s)", qty, variantID, locationID, res.ID)
	return res, nil
}

// ReleaseReservation handles the manual release of a reservation.
// It retrieves the reservation details, increments stock, saves the stock,
// deletes the reservation record, and publishes a stock change event.
//
// NOTE/CAVEAT: This operation is non-transactional.
//  1. If DeleteReservation fails, the service attempts to rollback the stock level to oldQty.
//  2. If this rollback save also fails, the database remains in an inconsistent state where
//     stock is increased but the reservation still exists. A subsequent retry of this operation
//     would retrieve the reservation again and add the stock quantity a second time, creating a
//     double-release/inflation of stock levels.
func (s *reservationService) ReleaseReservation(ctx context.Context, reservationID string) error {
	logger.InfoCtx(ctx, "Attempting to release reservation %s", reservationID)

	// 1. Retrieve the existing reservation record
	res, err := s.repo.GetReservation(ctx, reservationID)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to retrieve reservation %s: %v", reservationID, err)
		return apperrors.NewNotFound("reservation not found", err)
	}

	// 2. Retrieve stock for the reserved variant and location
	stock, err := s.repo.GetStock(ctx, res.VariantID, res.LocationID)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to retrieve stock for reservation release: %v", err)
		return apperrors.NewInternal("failed to check stock levels", err)
	}

	oldQty := stock.Quantity
	stock.Quantity += res.Quantity

	// 3. Save the incremented stock level
	if err := s.repo.SaveStock(ctx, stock); err != nil {
		logger.ErrorCtx(ctx, "Failed to update stock after reservation release: %v", err)
		return apperrors.NewInternal("failed to update stock levels", err)
	}

	// 4. Delete the reservation record from the database
	if err := s.repo.DeleteReservation(ctx, reservationID); err != nil {
		// Rollback stock increment on deletion failure
		stock.Quantity = oldQty
		_ = s.repo.SaveStock(ctx, stock)
		logger.ErrorCtx(ctx, "Failed to delete reservation %s: %v", reservationID, err)
		return apperrors.NewInternal("failed to delete reservation", err)
	}

	// 5. Publish stock changed event
	releaseEvt := events.NewEventFromCtx(ctx, events.InventoryStockChangedTopic, "inventory",
		&events.StockChangedPayload{
			VariantID:    res.VariantID,
			LocationID:   res.LocationID,
			OldQuantity:  oldQty,
			NewQuantity:  stock.Quantity,
			ChangeReason: fmt.Sprintf("release:%s", res.ID),
			ChangedBy:    "system",
		})
	if s.outbox != nil {
		if err := s.outbox.Store(ctx, releaseEvt); err != nil {
			logger.ErrorCtx(ctx, "Failed to store release event in outbox: %v", err)
		}
	} else if s.eventBus != nil {
		s.eventBus.Publish(releaseEvt)
	}

	logger.InfoCtx(ctx, "Successfully released reservation: %s", reservationID)
	return nil
}
