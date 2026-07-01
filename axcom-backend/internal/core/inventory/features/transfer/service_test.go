// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ecom-engine/internal/core/inventory/domain"
	"ecom-engine/internal/events"
)

type mockTransferRepo struct {
	stocks  map[string]*domain.StockItem
	err     error
	saveErr string
}

func (m *mockTransferRepo) GetStock(_ context.Context, variantID string, locationID string) (*domain.StockItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := variantID + ":" + locationID
	if s, ok := m.stocks[key]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}

func (m *mockTransferRepo) SaveStock(_ context.Context, stock *domain.StockItem) error {
	if m.saveErr != "" && stock.LocationID == m.saveErr {
		return errors.New("simulate save failure")
	}
	key := stock.VariantID + ":" + stock.LocationID
	m.stocks[key] = stock
	return nil
}

func (m *mockTransferRepo) AdjustQuantity(_ context.Context, _, _ string, _ int) error {
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

func TestService_TransferStock(t *testing.T) {
	t.Parallel()

	t.Run("successful transfer stock", func(t *testing.T) {
		repo := &mockTransferRepo{stocks: make(map[string]*domain.StockItem)}
		bus := &mockEventBus{}
		svc := NewService(repo, bus, nil)

		_ = repo.SaveStock(context.Background(), &domain.StockItem{
			VariantID:  "v-1",
			LocationID: "loc-a",
			Quantity:   10,
		})
		_ = repo.SaveStock(context.Background(), &domain.StockItem{
			VariantID:  "v-1",
			LocationID: "loc-b",
			Quantity:   2,
		})

		err := svc.TransferStock(context.Background(), "v-1", "loc-a", "loc-b", 5)
		assert.NoError(t, err)

		sA, _ := repo.GetStock(context.Background(), "v-1", "loc-a")
		assert.Equal(t, 5, sA.Quantity)

		sB, _ := repo.GetStock(context.Background(), "v-1", "loc-b")
		assert.Equal(t, 7, sB.Quantity)

		require.Len(t, bus.publishedEvents, 2)
		assert.Equal(t, events.InventoryStockChangedTopic, bus.publishedEvents[0].Topic)
	})

	t.Run("fails - same locations", func(t *testing.T) {
		repo := &mockTransferRepo{stocks: make(map[string]*domain.StockItem)}
		bus := &mockEventBus{}
		svc := NewService(repo, bus, nil)

		err := svc.TransferStock(context.Background(), "v-1", "loc-a", "loc-a", 5)
		assert.Error(t, err)
	})

	t.Run("fails - insufficient stock at source", func(t *testing.T) {
		repo := &mockTransferRepo{stocks: make(map[string]*domain.StockItem)}
		bus := &mockEventBus{}
		svc := NewService(repo, bus, nil)

		_ = repo.SaveStock(context.Background(), &domain.StockItem{
			VariantID:  "v-1",
			LocationID: "loc-a",
			Quantity:   3,
		})

		err := svc.TransferStock(context.Background(), "v-1", "loc-a", "loc-b", 5)
		assert.Error(t, err)
	})

	t.Run("fails - rollback trigger on destination save fail", func(t *testing.T) {
		repo := &mockTransferRepo{
			stocks:  make(map[string]*domain.StockItem),
			saveErr: "loc-b",
		}
		bus := &mockEventBus{}
		svc := NewService(repo, bus, nil)

		_ = repo.SaveStock(context.Background(), &domain.StockItem{
			VariantID:  "v-1",
			LocationID: "loc-a",
			Quantity:   10,
		})

		err := svc.TransferStock(context.Background(), "v-1", "loc-a", "loc-b", 5)
		assert.Error(t, err)

		// Assert source quantity rolled back to 10
		sA, _ := repo.GetStock(context.Background(), "v-1", "loc-a")
		assert.Equal(t, 10, sA.Quantity)
	})
}
