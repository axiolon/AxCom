// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package guest defines response contracts for guest checkout endpoints.
package guest

import "time"

// GuestOrderResponse is the JSON contract returned after a successful guest checkout.
type GuestOrderResponse struct { //nolint:revive // Name is intentionally explicit for the public API.
	OrderID   string            `json:"order_id"`
	Status    string            `json:"status"`
	Total     float64           `json:"total"`
	CreatedAt time.Time         `json:"created_at"`
	GuestInfo GuestInfoResponse `json:"guest_info"`
}

// GuestInfoResponse is the response shape for guest details.
type GuestInfoResponse struct { //nolint:revive // Name is intentionally explicit for the public API.
	Name          string `json:"name"`
	Email         string `json:"email"`
	ContactNumber string `json:"contact_number"`
}
