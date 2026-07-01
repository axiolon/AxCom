// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cart

import (
	"encoding/json"
	"time"

	"ecom-engine/internal/core/cart"
)

type dbCart struct {
	CustomerID string    `db:"customer_id"`
	Items      []byte    `db:"items"` // JSONB column
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

func toDBCart(c *cart.Cart) (*dbCart, error) {
	if c == nil {
		return nil, nil
	}
	itemsBytes, err := json.Marshal(c.Items)
	if err != nil {
		return nil, err
	}
	return &dbCart{
		CustomerID: c.CustomerID,
		Items:      itemsBytes,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}, nil
}

func toDomainCart(dbC *dbCart) (*cart.Cart, error) {
	if dbC == nil {
		return nil, nil
	}
	var items []cart.CartItem
	if err := json.Unmarshal(dbC.Items, &items); err != nil {
		return nil, err
	}
	return &cart.Cart{
		CustomerID: dbC.CustomerID,
		Items:      items,
		CreatedAt:  dbC.CreatedAt,
		UpdatedAt:  dbC.UpdatedAt,
	}, nil
}
