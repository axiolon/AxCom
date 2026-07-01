// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cart

import (
	"ecom-engine/internal/core/cart"
	"time"
)

type cartItemDoc struct {
	VariantID string `bson:"variant_id"`
	Quantity  int    `bson:"quantity"`
}

type cartDoc struct {
	CustomerID string        `bson:"_id"`
	Items      []cartItemDoc `bson:"items"`
	CreatedAt  time.Time     `bson:"created_at"`
	UpdatedAt  time.Time     `bson:"updated_at"`
}

func toCartDoc(c *cart.Cart) *cartDoc {
	if c == nil {
		return nil
	}
	items := make([]cartItemDoc, len(c.Items))
	for i, item := range c.Items {
		items[i] = cartItemDoc{
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
		}
	}
	return &cartDoc{
		CustomerID: c.CustomerID,
		Items:      items,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}

func toDomainCart(doc *cartDoc) *cart.Cart {
	if doc == nil {
		return nil
	}
	items := make([]cart.CartItem, len(doc.Items))
	for i, item := range doc.Items {
		items[i] = cart.CartItem{
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
		}
	}
	return &cart.Cart{
		CustomerID: doc.CustomerID,
		Items:      items,
		CreatedAt:  doc.CreatedAt,
		UpdatedAt:  doc.UpdatedAt,
	}
}
