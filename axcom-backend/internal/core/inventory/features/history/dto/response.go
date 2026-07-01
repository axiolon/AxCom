// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dto

import "ecom-engine/internal/core/inventory/domain"

// HistoryResponse represents the history log of stock changes for a variant.
type HistoryResponse struct {
	History []*domain.StockHistory `json:"history"`
}
