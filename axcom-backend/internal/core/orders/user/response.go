// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package user handles HTTP routing and handler implementation for authenticated customer orders.
package user

import "time"

// OrderItemResponse is the response shape for a single item.
type OrderItemResponse struct {
	VariantID string  `json:"variant_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// OrderResponse is the JSON contract returned for a single order.
type OrderResponse struct {
	ID        string              `json:"id"`
	Total     float64             `json:"total"`
	Status    string              `json:"status"`
	CreatedAt time.Time           `json:"created_at"`
	Items     []OrderItemResponse `json:"items"`
}

// OrderListResponse is the JSON contract returned for a list of orders.
type OrderListResponse struct {
	Orders []OrderResponse `json:"orders"`
	Count  int             `json:"count"`
}
