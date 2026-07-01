// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dto

// BulkUpdateItem represents a single item in a bulk update request.
type BulkUpdateItem struct {
	VariantID  string `json:"variant_id" binding:"required"`
	LocationID string `json:"location_id"`
	Quantity   *int   `json:"quantity" binding:"required,min=0"`
}

// BulkUpdateRequest represents the request body for bulk stock updates.
type BulkUpdateRequest struct {
	Items []BulkUpdateItem `json:"items" binding:"required,dive"`
}
