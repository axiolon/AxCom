// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reservation

import (
	"context"

	"ecom-engine/internal/core/inventory/domain"
)

type Repository interface {
	GetStock(ctx context.Context, variantID string, locationID string) (*domain.StockItem, error)
	SaveStock(ctx context.Context, stock *domain.StockItem) error
	CreateReservation(ctx context.Context, res *domain.Reservation) error
	GetReservation(ctx context.Context, resID string) (*domain.Reservation, error)
	DeleteReservation(ctx context.Context, resID string) error
	AdjustQuantity(ctx context.Context, variantID, locationID string, delta int) error
}
