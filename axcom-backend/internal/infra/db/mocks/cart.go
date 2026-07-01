// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"errors"
	"sync"

	"ecom-engine/internal/core/cart"
)

type MemCartRepo struct {
	mu    sync.RWMutex
	carts map[string]*cart.Cart
}

func NewMemCartRepo() *MemCartRepo {
	return &MemCartRepo{carts: make(map[string]*cart.Cart)}
}

func (r *MemCartRepo) GetByCustomerID(_ context.Context, customerID string) (*cart.Cart, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.carts[customerID]
	if !ok {
		return nil, errors.New("cart not found")
	}
	return c, nil
}

func (r *MemCartRepo) Save(_ context.Context, c *cart.Cart) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.carts[c.CustomerID] = c
	return nil
}

func (r *MemCartRepo) Delete(_ context.Context, customerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.carts, customerID)
	return nil
}
