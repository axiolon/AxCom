// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"

	"ecom-engine/internal/core/inventory/domain"
)

type Repository interface {
	// CreateHistory creates a new stock history record.
	CreateHistory(ctx context.Context, h *domain.StockHistory) error
	// GetHistory returns the stock history for a given variant ID.
	GetHistory(ctx context.Context, variantID string, limit, offset int) ([]*domain.StockHistory, error)
}
