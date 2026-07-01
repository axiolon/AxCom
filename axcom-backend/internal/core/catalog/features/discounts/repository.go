// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package discounts

import (
	"context"
	"ecom-engine/internal/core/catalog/domain"
)

// Repository defines the storage contract for managing product discounts.
type Repository interface {
	GetProductByID(ctx context.Context, id string) (*domain.Product, error)
	UpdateProductDiscount(ctx context.Context, id string, discount *domain.ProductDiscount) error
}
