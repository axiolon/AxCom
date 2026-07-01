// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package bulk

import (
	"context"
	"ecom-engine/internal/core/catalog/domain"
)

// Repository defines the storage contract for product bulk operations.
type Repository interface {
	BulkCreate(ctx context.Context, products []*domain.Product) error
	BulkUpdate(ctx context.Context, products []*domain.Product) error
	BulkDelete(ctx context.Context, ids []string) error
	GetCategoryByID(ctx context.Context, id string) (*domain.Category, error)
}
