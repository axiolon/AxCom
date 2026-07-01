// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"context"
	"errors"
	"testing"

	"ecom-engine/internal/core/inventory/domain"
)

type mockInventoryRepo struct {
	stocks       map[string]*domain.StockItem
	alerts       []*domain.Alert
	deleteCalled bool
	listFilter   ListStockFilter
}

func (m *mockInventoryRepo) GetStock(_ context.Context, variantID string, locationID string) (*domain.StockItem, error) {
	if locationID == "" {
		locationID = "default"
	}
	key := variantID + ":" + locationID
	if s, ok := m.stocks[key]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}

func (m *mockInventoryRepo) SaveStock(_ context.Context, stock *domain.StockItem) error {
	if stock.LocationID == "" {
		stock.LocationID = "default"
	}
	key := stock.VariantID + ":" + stock.LocationID
	m.stocks[key] = stock
	return nil
}

func (m *mockInventoryRepo) DeleteStock(_ context.Context, variantID string, locationID string) error {
	m.deleteCalled = true
	if locationID == "" {
		locationID = "default"
	}
	key := variantID + ":" + locationID
	delete(m.stocks, key)
	return nil
}

func (m *mockInventoryRepo) ListStock(_ context.Context, filter ListStockFilter) ([]*domain.StockItem, error) {
	m.listFilter = filter
	var result []*domain.StockItem
	for _, s := range m.stocks {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockInventoryRepo) SaveAlert(_ context.Context, alert *domain.Alert) error {
	m.alerts = append(m.alerts, alert)
	return nil
}

func (m *mockInventoryRepo) ListAlerts(_ context.Context, _, _ int) ([]*domain.Alert, error) {
	return m.alerts, nil
}

func (m *mockInventoryRepo) AdjustQuantity(_ context.Context, _, _ string, _ int) error {
	return nil
}

type mockAlertDispatcher struct {
	dispatched []domain.Alert
}

func (d *mockAlertDispatcher) Dispatch(_ context.Context, alert domain.Alert) error {
	d.dispatched = append(d.dispatched, alert)
	return nil
}

func TestServiceAlerting(t *testing.T) {
	repo := &mockInventoryRepo{
		stocks: make(map[string]*domain.StockItem),
	}
	dispatcher := &mockAlertDispatcher{}
	svc := NewService(repo, dispatcher)

	ctx := context.Background()
	variantID := "v-123"
	locationID := "default"

	// 1. Initial Update - Stock set to 10 (above threshold 5)
	err := svc.UpdateStock(ctx, variantID, locationID, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dispatcher.dispatched) != 0 {
		t.Fatalf("expected 0 alerts, got %d", len(dispatcher.dispatched))
	}

	// 2. Stock set to 3 (below default threshold 5)
	err = svc.UpdateStock(ctx, variantID, locationID, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dispatcher.dispatched) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(dispatcher.dispatched))
	}
	if dispatcher.dispatched[0].VariantID != variantID {
		t.Fatalf("expected alert for %s, got %s", variantID, dispatcher.dispatched[0].VariantID)
	}
}

func TestServiceDeleteAndList(t *testing.T) {
	repo := &mockInventoryRepo{
		stocks: make(map[string]*domain.StockItem),
	}
	dispatcher := &mockAlertDispatcher{}
	svc := NewService(repo, dispatcher)

	ctx := context.Background()
	variantID := "v-999"
	locationID := "default"

	_ = svc.UpdateStock(ctx, variantID, locationID, 10)

	// List
	items, err := svc.ListStock(ctx, ListStockFilter{Status: "low_stock"})
	if err != nil {
		t.Fatalf("unexpected error listing: %v", err)
	}
	if len(items) != 1 || items[0].VariantID != variantID {
		t.Fatalf("expected listed item, got %v", items)
	}

	// Delete
	err = svc.DeleteStock(ctx, variantID, locationID)
	if err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}
	if !repo.deleteCalled {
		t.Fatalf("expected delete repository method to be called")
	}
}

func TestServiceConfigure(t *testing.T) {
	repo := &mockInventoryRepo{
		stocks: make(map[string]*domain.StockItem),
	}
	dispatcher := &mockAlertDispatcher{}
	svc := NewService(repo, dispatcher)

	ctx := context.Background()
	variantID := "v-777"
	locationID := "loc-a"

	lowStock := 3
	allowBO := true
	boLimit := 5
	err := svc.ConfigureStock(ctx, ConfigureStockSettings{
		VariantID:         variantID,
		LocationID:        locationID,
		LowStockThreshold: &lowStock,
		AllowBackorders:   &allowBO,
		BackorderLimit:    &boLimit,
	})
	if err != nil {
		t.Fatalf("unexpected error configuring: %v", err)
	}

	stock, err := repo.GetStock(ctx, variantID, locationID)
	if err != nil {
		t.Fatalf("failed to get stock: %v", err)
	}
	if stock.LowStockThreshold != 3 || stock.AllowBackorders != true || stock.BackorderLimit != 5 {
		t.Fatalf("configured settings do not match: %+v", stock)
	}
}
