// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dto

import (
	"ecom-engine/internal/core/inventory/domain"
	sharedDTO "ecom-engine/internal/core/inventory/dto"
)

// CheckStockResponse represents the response details for checking stock levels.
type CheckStockResponse struct {
	VariantID  string `json:"variant_id"`
	LocationID string `json:"location_id"`
	Quantity   int    `json:"quantity"`
}

// ListStockResponse represents the response containing multiple stock items.
type ListStockResponse struct {
	Items  []sharedDTO.StockResponse `json:"items"`
	Limit  int64                     `json:"limit"`
	Offset int64                     `json:"offset"`
}

// AlertsResponse represents the response containing active low-stock alerts.
type AlertsResponse struct {
	Alerts []*domain.Alert `json:"alerts"`
}
