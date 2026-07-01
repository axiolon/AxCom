// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dto

// TransferRequest represents the request body for transferring stock between locations.
type TransferRequest struct {
	VariantID    string `json:"variant_id" binding:"required"`
	FromLocation string `json:"from_location" binding:"required"`
	ToLocation   string `json:"to_location" binding:"required"`
	Quantity     *int   `json:"quantity" binding:"required,min=1"`
}
