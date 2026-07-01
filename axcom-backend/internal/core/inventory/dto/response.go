// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dto

// MessageResponse is a standard success message payload.
type MessageResponse struct {
	Message string `json:"message"`
}

// StockResponse represents standard stock item details.
type StockResponse struct {
	VariantID         string `json:"variant_id"`
	LocationID        string `json:"location_id"`
	Quantity          int    `json:"quantity"`
	LowStockThreshold int    `json:"low_stock_threshold"`
	AllowBackorders   bool   `json:"allow_backorders"`
	BackorderLimit    int    `json:"backorder_limit"`
	IsLowStock        bool   `json:"is_low_stock"`
}
