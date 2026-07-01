// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dto

// ReserveRequest represents the request body for reserving stock.
type ReserveRequest struct {
	LocationID string `json:"location_id"`
	Quantity   int    `json:"quantity" binding:"required,min=1"`
}
