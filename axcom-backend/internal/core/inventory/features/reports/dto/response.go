// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dto

import sharedDTO "ecom-engine/internal/core/inventory/dto"

// LowStockReportResponse represents the report containing low stock items.
type LowStockReportResponse struct {
	Items []sharedDTO.StockResponse `json:"items"`
}
