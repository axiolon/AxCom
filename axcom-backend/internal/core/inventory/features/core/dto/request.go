// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dto

// UpdateStockRequest defines the request payload for updating stock levels.
type UpdateStockRequest struct {
	VariantID  string `json:"variant_id" binding:"required"`
	LocationID string `json:"location_id"`
	Quantity   *int   `json:"quantity" binding:"required,min=0"`
}

// ListStockRequest defines the query parameters for filtering and listing stock.
type ListStockRequest struct {
	VariantID  string `form:"variant_id"`
	LocationID string `form:"location_id"`
	Status     string `form:"status"`
	Limit      *int64 `form:"limit"`
	Offset     *int64 `form:"offset"`
}

// ConfigureStockRequest defines the request payload for setting up stock configurations.
type ConfigureStockRequest struct {
	VariantID         string `json:"variant_id" binding:"required"`
	LocationID        string `json:"location_id"`
	Quantity          *int   `json:"quantity"`
	LowStockThreshold *int   `json:"low_stock_threshold"`
	AllowBackorders   *bool  `json:"allow_backorders"`
	BackorderLimit    *int   `json:"backorder_limit"`
}
