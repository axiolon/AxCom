// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"context"

	"ecom-engine/internal/core/inventory/domain"
)

type Repository interface {
	GetLowStockItems(ctx context.Context) ([]*domain.StockItem, error)
	GetAllStockItems(ctx context.Context) ([]*domain.StockItem, error)
}
