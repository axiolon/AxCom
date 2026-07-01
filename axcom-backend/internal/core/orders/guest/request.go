// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package guest defines request contracts and input validation schemas for guest checkout endpoints.
package guest

// CreateGuestOrderRequest is the JSON contract for POST /orders/guest
type CreateGuestOrderRequest struct {
	GuestInfo GuestInfoRequest   `json:"guest_info" binding:"required"`
	Items     []OrderItemRequest `json:"items"      binding:"required,dive"`
}

// GuestInfoRequest carries the contact details for an unauthenticated buyer.
type GuestInfoRequest struct { //nolint:revive // Name is intentionally explicit for the public API.
	Name          string `json:"name"           binding:"required"`
	Email         string `json:"email"          binding:"required,email"`
	ContactNumber string `json:"contact_number" binding:"required"`
}

// OrderItemRequest is a single line-item in the order.
type OrderItemRequest struct {
	VariantID string  `json:"variant_id" binding:"required"`
	Quantity  int     `json:"quantity"   binding:"required,min=1"`
	Price     float64 `json:"price"      binding:"required,min=0"`
}
