// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cart

import "time"

// CartItem represents an item added to a customer's shopping cart.
type CartItem struct { //nolint:revive // Name is intentionally explicit for the public API.
	// VariantID uniquely identifies the product variant.
	VariantID string `json:"variant_id"`
	// Quantity specifies the number of items.
	Quantity int `json:"quantity"`
}

// Cart represents a collection of CartItems for a specific customer.
type Cart struct {
	// CustomerID uniquely identifies the customer who owns this cart.
	CustomerID string `json:"customer_id"`
	// Items lists all the items currently in the cart.
	Items []CartItem `json:"items"`
	// CreatedAt specifies when the cart was created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt specifies when the cart was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}
