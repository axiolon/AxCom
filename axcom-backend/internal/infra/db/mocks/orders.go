// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"errors"
	"sync"

	"ecom-engine/internal/core/orders"
)

type MemOrderRepo struct {
	mu     sync.RWMutex
	orders map[string]*orders.Order
}

func NewMemOrderRepo() *MemOrderRepo {
	return &MemOrderRepo{orders: make(map[string]*orders.Order)}
}

func (r *MemOrderRepo) Create(_ context.Context, o *orders.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[o.ID] = o
	return nil
}

func (r *MemOrderRepo) GetByID(_ context.Context, id string) (*orders.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.orders[id]
	if !ok {
		return nil, errors.New("order not found")
	}
	return o, nil
}

func (r *MemOrderRepo) Update(_ context.Context, o *orders.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[o.ID] = o
	return nil
}

func (r *MemOrderRepo) ListByCustomerID(_ context.Context, customerID string, limit, offset int) ([]orders.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []orders.Order
	for _, o := range r.orders {
		if o.CustomerID == customerID {
			list = append(list, *o)
		}
	}

	if offset > len(list) {
		return []orders.Order{}, nil
	}
	end := offset + limit
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end], nil
}

func (r *MemOrderRepo) ListAll(_ context.Context, limit, offset int) ([]orders.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]orders.Order, 0, len(r.orders))
	for _, o := range r.orders {
		list = append(list, *o)
	}

	if offset > len(list) {
		return []orders.Order{}, nil
	}
	end := offset + limit
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end], nil
}
