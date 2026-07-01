// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"context"

	"ecom-engine/internal/core/inventory/domain"
)

type ListStockFilter struct {
	VariantID  string
	LocationID string
	Status     string
	Limit      int64
	Offset     int64
}

type Repository interface {
	GetStock(ctx context.Context, variantID string, locationID string) (*domain.StockItem, error)
	SaveStock(ctx context.Context, stock *domain.StockItem) error
	DeleteStock(ctx context.Context, variantID string, locationID string) error
	ListStock(ctx context.Context, filter ListStockFilter) ([]*domain.StockItem, error)
	SaveAlert(ctx context.Context, alert *domain.Alert) error
	ListAlerts(ctx context.Context, limit, offset int) ([]*domain.Alert, error)
	AdjustQuantity(ctx context.Context, variantID, locationID string, delta int) error
}
