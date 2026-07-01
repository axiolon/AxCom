// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dto

// AddItemRequest represents the request body for adding an item to the cart.
type AddItemRequest struct {
	VariantID string `json:"variant_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,gt=0"`
}

// UpdateItemRequest represents the request body for updating item quantity.
type UpdateItemRequest struct {
	Quantity int `json:"quantity" binding:"required,gt=0"`
}
