// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dto

// CartCountResponse represents the response containing the cart item count.
type CartCountResponse struct {
	Count         int `json:"count"`          // Total quantity of all items
	DistinctCount int `json:"distinct_count"` // Number of unique/distinct items
}
