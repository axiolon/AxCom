// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package adjustment

import (
	"context"

	"ecom-engine/internal/core/inventory/domain"
)

type Repository interface {
	GetStock(ctx context.Context, variantID string, locationID string) (*domain.StockItem, error)
	SaveStock(ctx context.Context, stock *domain.StockItem) error
	AdjustQuantity(ctx context.Context, variantID, locationID string, delta int) error
}
