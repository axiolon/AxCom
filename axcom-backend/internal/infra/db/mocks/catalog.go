// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"errors"
	"sync"

	"ecom-engine/internal/core/catalog/domain"
	"ecom-engine/internal/core/catalog/features/core"
)

type MemCatalogRepo struct {
	mu         sync.RWMutex
	products   map[string]*domain.Product
	categories map[string]*domain.Category
}

func NewMemCatalogRepo() core.Repository {
	return &MemCatalogRepo{
		products:   make(map[string]*domain.Product),
		categories: make(map[string]*domain.Category),
	}
}

func (r *MemCatalogRepo) CreateProduct(_ context.Context, p *domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.products[p.ID] = p
	return nil
}

func (r *MemCatalogRepo) GetProductByID(_ context.Context, id string) (*domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.products[id]
	if !ok {
		return nil, errors.New("product not found")
	}
	return p, nil
}

func (r *MemCatalogRepo) GetProductByVariantID(_ context.Context, variantID string) (*domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.products {
		for _, v := range p.Variants {
			if v.ID == variantID {
				return p, nil
			}
		}
	}
	return nil, errors.New("product not found for variant")
}

func (r *MemCatalogRepo) UpdateVariantStock(_ context.Context, variantID string, stock int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.products {
		for i, v := range p.Variants {
			if v.ID == variantID {
				p.Variants[i].Stock = stock
				return nil
			}
		}
	}
	return core.ErrProductNotFound
}

func (r *MemCatalogRepo) ListProducts(_ context.Context, filter *core.ProductFilter) ([]domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := []domain.Product{}
	for _, p := range r.products {
		list = append(list, *p)
	}
	if filter != nil {
		if filter.Limit > 0 {
			end := filter.Offset + filter.Limit
			if end > int64(len(list)) {
				end = int64(len(list))
			}
			if filter.Offset < int64(len(list)) {
				list = list[filter.Offset:end]
			} else {
				list = []domain.Product{}
			}
		}
	}
	return list, nil
}

func (r *MemCatalogRepo) UpdateProduct(_ context.Context, p *domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.products[p.ID]
	if !ok {
		return core.ErrProductNotFound
	}
	if existing.Version != p.Version {
		return core.ErrVersionConflict
	}
	p.Version++
	r.products[p.ID] = p
	return nil
}

func (r *MemCatalogRepo) DeleteProduct(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.products[id]; !ok {
		return errors.New("product not found")
	}
	delete(r.products, id)
	return nil
}

func (r *MemCatalogRepo) CreateCategory(_ context.Context, c *domain.Category) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.categories[c.ID] = c
	return nil
}

func (r *MemCatalogRepo) GetCategoryByID(_ context.Context, id string) (*domain.Category, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.categories[id]
	if !ok {
		return nil, errors.New("category not found")
	}
	return c, nil
}

func (r *MemCatalogRepo) ListCategories(_ context.Context) ([]domain.Category, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := []domain.Category{}
	for _, c := range r.categories {
		list = append(list, *c)
	}
	return list, nil
}

func (r *MemCatalogRepo) UpdateCategory(_ context.Context, c *domain.Category) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.categories[c.ID]; !ok {
		return errors.New("category not found")
	}
	r.categories[c.ID] = c
	return nil
}

func (r *MemCatalogRepo) DeleteCategory(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.categories[id]; !ok {
		return errors.New("category not found")
	}
	delete(r.categories, id)
	return nil
}
