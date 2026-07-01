// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dto

// SyncRequest represents the request body for syncing stock levels.
type SyncRequest struct {
	VariantID  string `json:"variant_id" binding:"required"`
	LocationID string `json:"location_id"`
	Quantity   *int   `json:"quantity" binding:"required,min=0"`
}
