// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"
	"time"

	"ecom-engine/internal/core/inventory/domain"
	"ecom-engine/internal/events"
	"ecom-engine/pkg/idgen"
	"ecom-engine/pkg/logger"
)

type Service interface {
	GetHistory(ctx context.Context, variantID string, limit, offset int) ([]*domain.StockHistory, error)
	RecordHistory(ctx context.Context, h *domain.StockHistory) error
}

type historyService struct {
	repo Repository
}

func NewService(repo Repository, bus events.EventBus) Service {
	s := &historyService{
		repo: repo,
	}
	if bus != nil {
		bus.Subscribe(events.InventoryStockChangedTopic, s.handleStockChanged)
	}
	return s
}

// GetHistory returns the stock history for a given variant ID.
// Return domain.StockHistory list
// Return empty list if no history found
func (s *historyService) GetHistory(ctx context.Context, variantID string, limit, offset int) ([]*domain.StockHistory, error) {
	return s.repo.GetHistory(ctx, variantID, limit, offset)
}

func (s *historyService) RecordHistory(ctx context.Context, h *domain.StockHistory) error {
	if h.ID == "" {
		id, err := idgen.Generate("hist_")
		if err != nil {
			return err
		}
		h.ID = id
	}
	if h.ChangedAt.IsZero() {
		h.ChangedAt = time.Now()
	}
	return s.repo.CreateHistory(ctx, h)
}

// handleStockChanged handles the stock changed event via events
// It creates a new stock history record and saves it to the repository.
func (s *historyService) handleStockChanged(event events.Event) error {
	payload, ok := event.Payload.(*events.StockChangedPayload)
	if !ok {
		logger.Error("Received invalid stock changed payload type")
		return nil
	}

	histID, err := idgen.Generate("hist_")
	if err != nil {
		logger.Error("Failed to generate history ID: %v", err)
		return err
	}

	h := &domain.StockHistory{
		ID:           histID,
		VariantID:    payload.VariantID,
		LocationID:   payload.LocationID,
		OldQuantity:  payload.OldQuantity,
		NewQuantity:  payload.NewQuantity,
		ChangeReason: payload.ChangeReason,
		ChangedBy:    payload.ChangedBy,
		ChangedAt:    event.Timestamp,
	}

	ctx := context.Background()
	if err := s.RecordHistory(ctx, h); err != nil {
		logger.Error("Failed to record stock history: %v", err)
		return err
	}
	return nil
}
