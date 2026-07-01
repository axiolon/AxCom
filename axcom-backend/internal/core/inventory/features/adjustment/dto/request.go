// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dto

// AdjustRequest represents the request body for adjusting stock.
type AdjustRequest struct {
	LocationID string `json:"location_id"`
	Adjustment *int   `json:"adjustment" binding:"required"`
	Reason     string `json:"reason" binding:"required"`
}
