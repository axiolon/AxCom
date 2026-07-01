// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reservation

import (
	"context"
	"errors"
	"testing"

	"ecom-engine/internal/core/inventory/domain"
	"ecom-engine/internal/events"
)

type mockReservationRepo struct {
	stocks map[string]*domain.StockItem
	res    map[string]*domain.Reservation
}

func (m *mockReservationRepo) GetStock(_ context.Context, variantID string, locationID string) (*domain.StockItem, error) {
	if locationID == "" {
		locationID = "default"
	}
	key := variantID + ":" + locationID
	if s, ok := m.stocks[key]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}

func (m *mockReservationRepo) AdjustQuantity(_ context.Context, variantID, locationID string, delta int) error {
	if locationID == "" {
		locationID = "default"
	}
	key := variantID + ":" + locationID
	if s, ok := m.stocks[key]; ok {
		s.Quantity += delta
		return nil
	}
	return errors.New("not found")
}

func (m *mockReservationRepo) SaveStock(_ context.Context, stock *domain.StockItem) error {
	if stock.LocationID == "" {
		stock.LocationID = "default"
	}
	key := stock.VariantID + ":" + stock.LocationID
	m.stocks[key] = stock
	return nil
}

func (m *mockReservationRepo) CreateReservation(_ context.Context, res *domain.Reservation) error {
	m.res[res.ID] = res
	return nil
}

func (m *mockReservationRepo) GetReservation(_ context.Context, resID string) (*domain.Reservation, error) {
	if r, ok := m.res[resID]; ok {
		return r, nil
	}
	return nil, errors.New("not found")
}

func (m *mockReservationRepo) DeleteReservation(_ context.Context, resID string) error {
	delete(m.res, resID)
	return nil
}

func TestReservationAndBackorders(t *testing.T) {
	repo := &mockReservationRepo{
		stocks: make(map[string]*domain.StockItem),
		res:    make(map[string]*domain.Reservation),
	}
	bus := events.NewLocalEventBus()
	svc := NewService(repo, bus, nil)

	ctx := context.Background()
	variantID := "v-777"
	locationID := "loc-a"

	// Initialize stock item in repo directly
	repo.stocks[variantID+":"+locationID] = &domain.StockItem{
		VariantID:         variantID,
		LocationID:        locationID,
		Quantity:          0,
		LowStockThreshold: 3,
		AllowBackorders:   true,
		BackorderLimit:    5,
	}

	// 1. Reserve stock - order 3 (within limit of 5 backorders, since quantity is 0, new quantity will be -3)
	res, err := svc.ReserveStock(ctx, variantID, locationID, 3)
	if err != nil {
		t.Fatalf("unexpected reservation failure: %v", err)
	}
	if res.Quantity != 3 {
		t.Fatalf("expected reservation qty 3, got %d", res.Quantity)
	}

	stock := repo.stocks[variantID+":"+locationID]
	if stock.Quantity != -3 {
		t.Fatalf("expected stock quantity -3, got %d", stock.Quantity)
	}

	// 2. Reserve stock - try to order 3 more (exceeds limit of 5 backorders since current is -3, needed -6 is below -5)
	_, err = svc.ReserveStock(ctx, variantID, locationID, 3)
	if err == nil {
		t.Fatalf("expected reservation to fail due to backorder limit")
	}

	// 3. Release first reservation - should restore stock level from -3 to 0 and delete reservation
	err = svc.ReleaseReservation(ctx, res.ID)
	if err != nil {
		t.Fatalf("unexpected failure releasing reservation: %v", err)
	}
	if stock.Quantity != 0 {
		t.Fatalf("expected stock quantity 0 after release, got %d", stock.Quantity)
	}
	if _, ok := repo.res[res.ID]; ok {
		t.Fatalf("expected reservation to be deleted from repo")
	}
}
