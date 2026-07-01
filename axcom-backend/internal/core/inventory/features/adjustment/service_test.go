// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package adjustment

import (
	"context"
	"errors"
	"testing"

	"ecom-engine/internal/core/inventory/domain"
	"ecom-engine/internal/events"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAdjustmentRepo struct {
	stocks map[string]*domain.StockItem
	err    error
}

func (m *mockAdjustmentRepo) GetStock(_ context.Context, variantID string, locationID string) (*domain.StockItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := variantID + ":" + locationID
	if s, ok := m.stocks[key]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}

func (m *mockAdjustmentRepo) AdjustQuantity(_ context.Context, variantID, locationID string, delta int) error {
	if m.err != nil {
		return m.err
	}
	key := variantID + ":" + locationID
	if s, ok := m.stocks[key]; ok {
		s.Quantity += delta
		return nil
	}
	return errors.New("not found")
}

func (m *mockAdjustmentRepo) SaveStock(_ context.Context, stock *domain.StockItem) error {
	if m.err != nil {
		return m.err
	}
	key := stock.VariantID + ":" + stock.LocationID
	m.stocks[key] = stock
	return nil
}

type mockEventBus struct {
	publishedEvents []events.Event
}

func (m *mockEventBus) Subscribe(_ string, _ events.EventHandler) {}
func (m *mockEventBus) Publish(event events.Event) {
	m.publishedEvents = append(m.publishedEvents, event)
}
func (m *mockEventBus) Close() error { return nil }

func TestService_AdjustStock(t *testing.T) {
	t.Parallel()

	t.Run("successful adjustment increment", func(t *testing.T) {
		repo := &mockAdjustmentRepo{stocks: make(map[string]*domain.StockItem)}
		bus := &mockEventBus{}
		svc := NewService(repo, bus, nil)

		// Create initial stock level
		_ = repo.SaveStock(context.Background(), &domain.StockItem{
			VariantID:  "v-1",
			LocationID: "loc-a",
			Quantity:   5,
		})

		err := svc.AdjustStock(context.Background(), "v-1", "loc-a", 10, "restock")
		assert.NoError(t, err)

		s, err := repo.GetStock(context.Background(), "v-1", "loc-a")
		assert.NoError(t, err)
		assert.Equal(t, 15, s.Quantity)

		require.Len(t, bus.publishedEvents, 1)
		payload, ok := bus.publishedEvents[0].Payload.(*events.StockChangedPayload)
		require.True(t, ok)
		assert.Equal(t, 5, payload.OldQuantity)
		assert.Equal(t, 15, payload.NewQuantity)
	})

	t.Run("fails - negative quantity limit", func(t *testing.T) {
		repo := &mockAdjustmentRepo{stocks: make(map[string]*domain.StockItem)}
		bus := &mockEventBus{}
		svc := NewService(repo, bus, nil)

		_ = repo.SaveStock(context.Background(), &domain.StockItem{
			VariantID:  "v-1",
			LocationID: "loc-a",
			Quantity:   5,
		})

		err := svc.AdjustStock(context.Background(), "v-1", "loc-a", -10, "shrinkage")
		assert.Error(t, err)
	})

	t.Run("fails - missing reason", func(t *testing.T) {
		repo := &mockAdjustmentRepo{stocks: make(map[string]*domain.StockItem)}
		bus := &mockEventBus{}
		svc := NewService(repo, bus, nil)

		err := svc.AdjustStock(context.Background(), "v-1", "loc-a", 10, "")
		assert.Error(t, err)
	})
}
