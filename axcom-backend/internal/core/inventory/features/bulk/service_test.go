// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package bulk

import (
	"context"
	"errors"
	"testing"

	"ecom-engine/internal/core/inventory/domain"

	"github.com/stretchr/testify/assert"
)

type mockBulkRepo struct {
	stocks map[string]*domain.StockItem
	err    error
}

func (m *mockBulkRepo) GetStock(_ context.Context, variantID string, locationID string) (*domain.StockItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := variantID + ":" + locationID
	if s, ok := m.stocks[key]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}

func (m *mockBulkRepo) AdjustQuantity(_ context.Context, variantID, locationID string, delta int) error {
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

func (m *mockBulkRepo) SaveStock(_ context.Context, stock *domain.StockItem) error {
	if m.err != nil {
		return m.err
	}
	key := stock.VariantID + ":" + stock.LocationID
	m.stocks[key] = stock
	return nil
}

func TestService_BulkUpdate(t *testing.T) {
	t.Parallel()

	t.Run("successful bulk updates", func(t *testing.T) {
		repo := &mockBulkRepo{stocks: make(map[string]*domain.StockItem)}
		svc := NewService(repo)

		updates := []UpdateItem{
			{VariantID: "v-1", LocationID: "loc-a", Quantity: 10},
			{VariantID: "v-2", LocationID: "loc-b", Quantity: 20},
		}

		err := svc.BulkUpdate(context.Background(), updates)
		assert.NoError(t, err)

		s1, err := repo.GetStock(context.Background(), "v-1", "loc-a")
		assert.NoError(t, err)
		assert.Equal(t, 10, s1.Quantity)

		s2, err := repo.GetStock(context.Background(), "v-2", "loc-b")
		assert.NoError(t, err)
		assert.Equal(t, 20, s2.Quantity)
	})

	t.Run("fails - negative quantity validation", func(t *testing.T) {
		repo := &mockBulkRepo{stocks: make(map[string]*domain.StockItem)}
		svc := NewService(repo)

		updates := []UpdateItem{
			{VariantID: "v-1", LocationID: "loc-a", Quantity: -5},
		}

		err := svc.BulkUpdate(context.Background(), updates)
		assert.Error(t, err)
	})
}
