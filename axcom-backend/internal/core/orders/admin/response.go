// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package admin

import "time"

// OrderItemResponse represents the response shape for an order item.
type OrderItemResponse struct {
	VariantID string  `json:"variant_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// GuestCustomerInfoResponse represents the response shape for guest contact info.
type GuestCustomerInfoResponse struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	ContactNumber string `json:"contact_number"`
}

// OrderResponse represents the admin view of a single order.
type OrderResponse struct {
	ID         string                     `json:"id"`
	CustomerID string                     `json:"customer_id,omitempty"`
	GuestInfo  *GuestCustomerInfoResponse `json:"guest_info,omitempty"`
	Items      []OrderItemResponse        `json:"items"`
	Total      float64                    `json:"total"`
	Status     string                     `json:"status"`
	CreatedAt  time.Time                  `json:"created_at"`
}

// OrderListResponse is the JSON contract for listing multiple orders.
type OrderListResponse struct {
	Orders []OrderResponse `json:"orders"`
	Count  int             `json:"count"`
}
