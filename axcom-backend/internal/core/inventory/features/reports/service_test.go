// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"context"
	"errors"
	"testing"

	"ecom-engine/internal/core/inventory/domain"

	"github.com/stretchr/testify/assert"
)

type mockReportsRepo struct {
	lowStockItems []*domain.StockItem
	allStockItems []*domain.StockItem
	err           error
}

func (m *mockReportsRepo) GetLowStockItems(_ context.Context) ([]*domain.StockItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.lowStockItems, nil
}

func (m *mockReportsRepo) GetAllStockItems(_ context.Context) ([]*domain.StockItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.allStockItems, nil
}

func TestService_GetLowStockReport(t *testing.T) {
	t.Parallel()

	t.Run("successful retrieval", func(t *testing.T) {
		expectedItems := []*domain.StockItem{
			{VariantID: "var-1", LocationID: "default", Quantity: 2},
		}
		repo := &mockReportsRepo{lowStockItems: expectedItems}
		svc := NewService(repo)

		result, err := svc.GetLowStockReport(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, expectedItems, result)
	})

	t.Run("fails - repo error", func(t *testing.T) {
		repo := &mockReportsRepo{err: errors.New("database issue")}
		svc := NewService(repo)

		result, err := svc.GetLowStockReport(context.Background())
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestService_ExportInventoryCSV(t *testing.T) {
	t.Parallel()

	t.Run("successful CSV export with items", func(t *testing.T) {
		items := []*domain.StockItem{
			{
				VariantID:         "var-1",
				LocationID:        "loc-a",
				Quantity:          10,
				LowStockThreshold: 5,
				AllowBackorders:   true,
				BackorderLimit:    20,
			},
		}
		repo := &mockReportsRepo{allStockItems: items}
		svc := NewService(repo)

		csvBytes, err := svc.ExportInventoryCSV(context.Background())
		assert.NoError(t, err)

		csvString := string(csvBytes)
		assert.Contains(t, csvString, "Variant ID,Location ID,Quantity,Low Stock Threshold,Allow Backorders,Backorder Limit")
		assert.Contains(t, csvString, "var-1,loc-a,10,5,true,20")
	})

	t.Run("successful CSV export with empty database", func(t *testing.T) {
		repo := &mockReportsRepo{allStockItems: []*domain.StockItem{}}
		svc := NewService(repo)

		csvBytes, err := svc.ExportInventoryCSV(context.Background())
		assert.NoError(t, err)

		csvString := string(csvBytes)
		assert.Equal(t, "Variant ID,Location ID,Quantity,Low Stock Threshold,Allow Backorders,Backorder Limit\n", csvString)
	})
}
