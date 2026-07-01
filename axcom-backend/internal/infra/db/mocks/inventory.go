// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"ecom-engine/internal/core/inventory"
)

type MemInventoryRepo struct {
	mu     sync.RWMutex
	stocks map[string]*inventory.StockItem
	res    map[string]*inventory.Reservation
	alerts []*inventory.Alert
}

func NewMemInventoryRepo() *MemInventoryRepo {
	return &MemInventoryRepo{
		stocks: make(map[string]*inventory.StockItem),
		res:    make(map[string]*inventory.Reservation),
		alerts: make([]*inventory.Alert, 0),
	}
}

func (r *MemInventoryRepo) GetStock(_ context.Context, variantID string, locationID string) (*inventory.StockItem, error) {
	if locationID == "" {
		locationID = "default"
	}
	key := variantID + ":" + locationID
	r.mu.RLock()
	defer r.mu.RUnlock()
	stock, ok := r.stocks[key]
	if !ok {
		return &inventory.StockItem{
			VariantID:         variantID,
			LocationID:        locationID,
			Quantity:          100,
			LowStockThreshold: 5,
			AllowBackorders:   false,
			BackorderLimit:    0,
		}, nil
	}
	return stock, nil
}

func (r *MemInventoryRepo) SaveStock(_ context.Context, stock *inventory.StockItem) error {
	if stock.LocationID == "" {
		stock.LocationID = "default"
	}
	key := stock.VariantID + ":" + stock.LocationID
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stocks[key] = stock
	return nil
}

func (r *MemInventoryRepo) CreateReservation(_ context.Context, res *inventory.Reservation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.res[res.ID] = res
	return nil
}

func (r *MemInventoryRepo) DeleteReservation(_ context.Context, resID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.res, resID)
	return nil
}

func (r *MemInventoryRepo) DeleteStock(_ context.Context, variantID string, locationID string) error {
	if locationID == "" {
		locationID = "default"
	}
	key := variantID + ":" + locationID
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.stocks, key)
	return nil
}

func (r *MemInventoryRepo) ListStock(_ context.Context, filter map[string]interface{}) ([]*inventory.StockItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*inventory.StockItem
	for _, stock := range r.stocks {
		matches := true
		if statusVal, ok := filter["status"]; ok {
			if statusVal == "low_stock" && !stock.IsLowStock() {
				matches = false
			}
		}
		if variantIDVal, ok := filter["variant_id"]; ok {
			if stock.VariantID != variantIDVal {
				matches = false
			}
		}
		if locationIDVal, ok := filter["location_id"]; ok {
			if stock.LocationID != locationIDVal {
				matches = false
			}
		}
		if matches {
			result = append(result, stock)
		}
	}
	return result, nil
}

func (r *MemInventoryRepo) SaveAlert(_ context.Context, alert *inventory.Alert) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.alerts = append(r.alerts, alert)
	return nil
}

func (r *MemInventoryRepo) ListAlerts(_ context.Context) ([]*inventory.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copied := make([]*inventory.Alert, len(r.alerts))
	copy(copied, r.alerts)
	return copied, nil
}
