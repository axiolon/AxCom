// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"context"
	"errors"

	"ecom-engine/internal/core/catalog/domain"
)

var (
	ErrProductNotFound  = domain.ErrProductNotFound
	ErrCategoryNotFound = domain.ErrCategoryNotFound
	ErrVersionConflict  = errors.New("product update conflict: document was modified by another process")
)

// Repository defines the persistence contract for core products and categories.
type Repository interface {
	CreateProduct(ctx context.Context, p *domain.Product) error
	GetProductByID(ctx context.Context, id string) (*domain.Product, error)
	ListProducts(ctx context.Context, filter *ProductFilter) ([]domain.Product, error)
	UpdateProduct(ctx context.Context, p *domain.Product) error
	DeleteProduct(ctx context.Context, id string) error
	GetProductByVariantID(ctx context.Context, variantID string) (*domain.Product, error)
	UpdateVariantStock(ctx context.Context, variantID string, stock int) error

	CreateCategory(ctx context.Context, c *domain.Category) error
	GetCategoryByID(ctx context.Context, id string) (*domain.Category, error)
	ListCategories(ctx context.Context) ([]domain.Category, error)
	UpdateCategory(ctx context.Context, c *domain.Category) error
	DeleteCategory(ctx context.Context, id string) error
}
