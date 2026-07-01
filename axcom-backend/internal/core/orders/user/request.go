// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package user handles HTTP routing and handler implementation for authenticated customer orders.
package user

// CreateOrderRequest is the JSON contract for POST /orders (authenticated)
type CreateOrderRequest struct {
	Items []OrderItemRequest `json:"items" binding:"required,dive"`
}

// OrderItemRequest is a single line-item in the order.
type OrderItemRequest struct {
	VariantID string  `json:"variant_id" binding:"required"`
	Quantity  int     `json:"quantity"   binding:"required,min=1"`
	Price     float64 `json:"price"      binding:"required,min=0"`
}
