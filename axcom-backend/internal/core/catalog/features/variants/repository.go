// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package variants

import (
	"context"
	"ecom-engine/internal/core/catalog/domain"
)

// Repository defines the storage contract for product variants.
type Repository interface {
	GetProductByID(ctx context.Context, id string) (*domain.Product, error)
	UpdateProductVariants(ctx context.Context, id string, variants []domain.Variant) error
}
