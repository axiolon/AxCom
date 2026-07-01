// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package merge

// Request represents the request body for merging guest cart with account cart.
type Request struct {
	GuestCartID string `json:"guest_cart_id" binding:"required"`
}

// Response represents the response after merging carts.
type Response struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}
