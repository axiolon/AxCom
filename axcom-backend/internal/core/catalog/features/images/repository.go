// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package images

import (
	"context"
	"ecom-engine/internal/core/catalog/domain"
)

// Repository defines the storage contract for managing product images.
type Repository interface {
	GetProductByID(ctx context.Context, id string) (*domain.Product, error)
	UpdateProductImages(ctx context.Context, id string, images []domain.ProductImage) error
}
