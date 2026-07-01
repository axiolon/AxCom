// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ecom-engine/internal/core/catalog/domain"
	"ecom-engine/internal/events"
	"ecom-engine/internal/infra/cache"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/idgen"
	"ecom-engine/pkg/logger"
)

// CommandService defines the business logic contract for writing catalog data.
type CommandService interface {
	// AddProduct validates and adds a new product to the catalog.
	AddProduct(ctx context.Context, p *domain.Product) error
	// UpdateProduct validates and updates fields on an existing product.
	UpdateProduct(ctx context.Context, p *domain.Product) error
	// DeleteProduct removes a product from the catalog registry.
	DeleteProduct(ctx context.Context, id string) error

	// AddCategory validates and creates a new category.
	AddCategory(ctx context.Context, c *domain.Category) error
	// UpdateCategory validates and updates properties of an existing category.
	UpdateCategory(ctx context.Context, c *domain.Category) error
	// DeleteCategory removes a category if it has no associated products or child categories.
	DeleteCategory(ctx context.Context, id string) error

	// SubscribeStockEvents registers listeners on event bus to keep product stock levels synchronized.
	SubscribeStockEvents(bus events.EventBus)
}

type catalogCommandService struct {
	repo         Repository
	idGenerator  func(prefix string) (string, error)
	cacheManager cache.Manager
}

// NewCatalogCommandService creates a new modular Catalog Command Service.
func NewCatalogCommandService(repo Repository, cacheManager cache.Manager) CommandService {
	return &catalogCommandService{
		repo:         repo,
		idGenerator:  idgen.Generate,
		cacheManager: cacheManager,
	}
}

// AddProduct generates IDs for products/variants, validates domain constraints, and persists the entity.
func (s *catalogCommandService) AddProduct(ctx context.Context, p *domain.Product) error {
	logger.InfoCtx(ctx, "Adding new product: %s", p.Name)

	if p.ID == "" {
		id, err := s.idGenerator("prod_")
		if err != nil {
			logger.ErrorCtx(ctx, "Failed to generate product ID: %v", err)
			return apperrors.NewInternal("failed to generate product ID", err)
		}
		p.ID = id
	}
	for i := range p.Variants {
		if p.Variants[i].ID == "" {
			id, err := s.idGenerator("var_")
			if err != nil {
				logger.ErrorCtx(ctx, "Failed to generate variant ID: %v", err)
				return apperrors.NewInternal("failed to generate variant ID", err)
			}
			p.Variants[i].ID = id
		}
	}

	if err := domain.ValidateProduct(*p); err != nil {
		logger.ErrorCtx(ctx, "Product domain validation failed for %s: %v", p.Name, err)
		return apperrors.NewBadRequest(err.Error(), err)
	}

	// Verify category exists
	_, err := s.repo.GetCategoryByID(ctx, p.CategoryID)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to add product: category ID %s does not exist: %v", p.CategoryID, err)
		return apperrors.NewNotFound(fmt.Sprintf("category with ID %s not found", p.CategoryID), ErrCategoryNotFound)
	}

	if err := s.repo.CreateProduct(ctx, p); err != nil {
		logger.ErrorCtx(ctx, "Failed to persist product %s: %v", p.Name, err)
		return apperrors.NewInternal("failed to save product", err)
	}

	logger.InfoCtx(ctx, "Successfully added product %s (ID: %s)", p.Name, p.ID)
	return nil
}

// UpdateProduct applies field updates to the product collection, verifying category constraints and values.
func (s *catalogCommandService) UpdateProduct(ctx context.Context, p *domain.Product) error {
	logger.InfoCtx(ctx, "Updating product: %s (ID: %s)", p.Name, p.ID)

	if p.ID == "" {
		return apperrors.NewBadRequest("Product ID is required for update", nil)
	}

	if err := domain.ValidateProduct(*p); err != nil {
		logger.ErrorCtx(ctx, "Product domain validation failed for %s: %v", p.Name, err)
		return apperrors.NewBadRequest("Invalid product data", err)
	}

	_, err := s.repo.GetCategoryByID(ctx, p.CategoryID)
	if err != nil {
		logger.ErrorCtx(ctx, "Category ID %s not found: %v", p.CategoryID, err)
		return apperrors.NewNotFound("category not found", ErrCategoryNotFound)
	}

	if err := s.repo.UpdateProduct(ctx, p); err != nil {
		logger.ErrorCtx(ctx, "Failed to update product %s: %v", p.Name, err)
		if errors.Is(err, ErrProductNotFound) {
			return apperrors.NewNotFound("product not found", err)
		}
		if errors.Is(err, ErrVersionConflict) {
			return apperrors.NewConflict("product update conflict: document was modified by another process", err)
		}
		return apperrors.NewInternal("failed to update product", err)
	}

	// Invalidate cache
	if err := s.cacheManager.Invalidate(ctx, "catalog:product:"+p.ID); err != nil {
		logger.WarnCtx(ctx, "failed to invalidate cache: %v", err)
	}
	for _, v := range p.Variants {
		if err := s.cacheManager.Invalidate(ctx, "catalog:product:variant:"+v.ID); err != nil {
			logger.WarnCtx(ctx, "failed to invalidate cache: %v", err)
		}
	}

	return nil
}

// DeleteProduct removes a product catalog mapping.
func (s *catalogCommandService) DeleteProduct(ctx context.Context, id string) error {
	logger.InfoCtx(ctx, "Deleting product: %s", id)

	if id == "" {
		return apperrors.NewBadRequest("Product ID is required for deletion", nil)
	}

	// Fetch product details first to invalidate variants
	p, err := s.repo.GetProductByID(ctx, id)
	if err == nil {
		defer func() {
			bgCtx := context.Background()
			if err := s.cacheManager.Invalidate(bgCtx, "catalog:product:"+p.ID); err != nil {
				logger.WarnCtx(bgCtx, "failed to invalidate cache: %v", err)
			}
			for _, v := range p.Variants {
				if err := s.cacheManager.Invalidate(bgCtx, "catalog:product:variant:"+v.ID); err != nil {
					logger.WarnCtx(bgCtx, "failed to invalidate cache: %v", err)
				}
			}
		}()
	}

	if err := s.repo.DeleteProduct(ctx, id); err != nil {
		logger.ErrorCtx(ctx, "Failed to delete product %s: %v", id, err)
		if errors.Is(err, ErrProductNotFound) {
			return apperrors.NewNotFound("product not found", err)
		}
		return apperrors.NewInternal("failed to delete product", err)
	}

	return nil
}

// AddCategory inserts a new product category, generating friendly URL slugs and validating parent references.
func (s *catalogCommandService) AddCategory(ctx context.Context, c *domain.Category) error {
	logger.InfoCtx(ctx, "Adding new category: %s", c.Name)

	if c.ID == "" {
		id, err := s.idGenerator("cat_")
		if err != nil {
			logger.ErrorCtx(ctx, "Failed to generate category ID: %v", err)
			return apperrors.NewInternal("failed to generate category ID", err)
		}
		c.ID = id
	}

	if c.Slug == "" {
		c.Slug = domain.GenerateSlug(c.Name)
	}

	if err := domain.ValidateCategory(*c); err != nil {
		logger.ErrorCtx(ctx, "Category validation failed for %s: %v", c.Name, err)
		return apperrors.NewBadRequest(err.Error(), err)
	}

	// Verify slug uniqueness (C4-3)
	existingCats, err := s.repo.ListCategories(ctx)
	if err == nil {
		for _, ex := range existingCats {
			if ex.Slug == c.Slug && ex.ID != c.ID {
				return apperrors.NewConflict(fmt.Sprintf("category slug %q is already taken", c.Slug), nil)
			}
		}
	}

	if c.ParentID != nil && *c.ParentID != "" {
		_, err := s.repo.GetCategoryByID(ctx, *c.ParentID)
		if err != nil {
			logger.ErrorCtx(ctx, "Parent category %s not found: %v", *c.ParentID, err)
			return apperrors.NewNotFound("parent category not found", err)
		}
	}

	if err := s.repo.CreateCategory(ctx, c); err != nil {
		logger.ErrorCtx(ctx, "Failed to persist category %s: %v", c.Name, err)
		return apperrors.NewInternal("failed to save category", err)
	}

	logger.InfoCtx(ctx, "Successfully added category %s (ID: %s, Slug: %s)", c.Name, c.ID, c.Slug)
	return nil
}

// UpdateCategory modifies category titles, slugs, and validates tree dependencies (no self-loops).
func (s *catalogCommandService) UpdateCategory(ctx context.Context, c *domain.Category) error {
	logger.InfoCtx(ctx, "Updating category: %s (ID: %s)", c.Name, c.ID)

	if c.ID == "" {
		return apperrors.NewBadRequest("Category ID is required for update", nil)
	}
	if c.Slug == "" {
		c.Slug = domain.GenerateSlug(c.Name)
	}

	if err := domain.ValidateCategory(*c); err != nil {
		logger.ErrorCtx(ctx, "Category validation failed: %v", err)
		return apperrors.NewBadRequest("Invalid category data", err)
	}

	// Verify slug uniqueness (C4-3)
	existingCats, err := s.repo.ListCategories(ctx)
	if err == nil {
		for _, ex := range existingCats {
			if ex.Slug == c.Slug && ex.ID != c.ID {
				return apperrors.NewConflict(fmt.Sprintf("category slug %q is already taken", c.Slug), nil)
			}
		}
	}

	if c.ParentID != nil && *c.ParentID == c.ID {
		return apperrors.NewBadRequest("Category cannot be its own parent", nil)
	}

	if c.ParentID != nil && *c.ParentID != "" {
		_, err := s.repo.GetCategoryByID(ctx, *c.ParentID)
		if err != nil {
			logger.ErrorCtx(ctx, "Parent category %s not found: %v", *c.ParentID, err)
			return apperrors.NewNotFound("parent category not found", err)
		}

		isCyclic, err := s.detectCategoryCycle(ctx, *c.ParentID, c.ID)
		if err != nil {
			return apperrors.NewInternal("failed to validate category hierarchy", err)
		}
		if isCyclic {
			return apperrors.NewBadRequest("circular category dependency detected", nil)
		}
	}

	if err := s.repo.UpdateCategory(ctx, c); err != nil {
		logger.ErrorCtx(ctx, "Failed to update category %s: %v", c.Name, err)
		if errors.Is(err, ErrCategoryNotFound) {
			return apperrors.NewNotFound("category not found", err)
		}
		return apperrors.NewInternal("failed to update category", err)
	}

	if err := s.cacheManager.Invalidate(ctx, "catalog:category:"+c.ID); err != nil {
		logger.WarnCtx(ctx, "failed to invalidate cache: %v", err)
	}

	return nil
}

// DeleteCategory removes a category after ensuring no products or subcategories reference it.
func (s *catalogCommandService) DeleteCategory(ctx context.Context, id string) error {
	logger.InfoCtx(ctx, "Deleting category: %s", id)

	if id == "" {
		return apperrors.NewBadRequest("Category ID is required for deletion", nil)
	}

	// In the real DB we should query with a filter to only get products in this category.
	// We pass a ProductFilter with CategoryIDs to avoid loading all products if supported.
	filter := &ProductFilter{CategoryIDs: []string{id}}
	products, err := s.repo.ListProducts(ctx, filter)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to list products for category %s check: %v", id, err)
		return apperrors.NewInternal("failed to verify category references", err)
	}
	for _, p := range products {
		if p.CategoryID == id {
			return apperrors.NewConflict("cannot delete category: products are assigned to it", nil)
		}
	}

	// Check if any category is a child of this category
	categories, err := s.repo.ListCategories(ctx)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to list categories for parent check: %v", err)
		return apperrors.NewInternal("failed to verify category references", err)
	}
	for _, c := range categories {
		if c.ParentID != nil && *c.ParentID == id {
			return apperrors.NewConflict("cannot delete category: has child subcategories", nil)
		}
	}

	if err := s.repo.DeleteCategory(ctx, id); err != nil {
		logger.ErrorCtx(ctx, "Failed to delete category %s: %v", id, err)
		if errors.Is(err, ErrCategoryNotFound) {
			return apperrors.NewNotFound("category not found", err)
		}
		return apperrors.NewInternal("failed to delete category", err)
	}

	if err := s.cacheManager.Invalidate(ctx, "catalog:category:"+id); err != nil {
		logger.WarnCtx(ctx, "failed to invalidate cache: %v", err)
	}

	return nil
}

// SubscribeStockEvents attaches a subscriber for stock inventory updates to sync database states in real-time.
func (s *catalogCommandService) SubscribeStockEvents(bus events.EventBus) {
	bus.Subscribe(events.InventoryStockChangedTopic, func(ev events.Event) error {
		var payload *events.StockChangedPayload
		if pPtr, ok := ev.Payload.(*events.StockChangedPayload); ok {
			payload = pPtr
		} else if pVal, ok := ev.Payload.(events.StockChangedPayload); ok {
			payload = &pVal
		}

		if payload == nil {
			logger.Error("Invalid stock changed payload type")
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		logger.InfoCtx(ctx, "Received stock changed event for variant %s: %d -> %d", payload.VariantID, payload.OldQuantity, payload.NewQuantity)

		if err := s.repo.UpdateVariantStock(ctx, payload.VariantID, payload.NewQuantity); err != nil {
			logger.ErrorCtx(ctx, "Failed to synchronize stock to %d for variant %s: %v", payload.NewQuantity, payload.VariantID, err)
			return err
		}

		// Invalidate product cache as variant stock changed
		p, err := s.repo.GetProductByVariantID(ctx, payload.VariantID)
		if err == nil {
			if err := s.cacheManager.Invalidate(ctx, "catalog:product:"+p.ID); err != nil {
				logger.WarnCtx(ctx, "failed to invalidate cache: %v", err)
			}
			if err := s.cacheManager.Invalidate(ctx, "catalog:product:variant:"+payload.VariantID); err != nil {
				logger.WarnCtx(ctx, "failed to invalidate cache: %v", err)
			}
		}

		logger.InfoCtx(ctx, "Successfully synchronized stock to %d for variant %s in catalog", payload.NewQuantity, payload.VariantID)
		return nil
	})
}

// detectCategoryCycle detects if setting startParentID as parent of targetCategoryID introduces a circular loop.
func (s *catalogCommandService) detectCategoryCycle(ctx context.Context, startParentID, targetCategoryID string) (bool, error) {
	visited := map[string]bool{targetCategoryID: true}
	currentID := startParentID
	for currentID != "" {
		if visited[currentID] {
			return true, nil
		}
		visited[currentID] = true

		parentCat, err := s.repo.GetCategoryByID(ctx, currentID)
		if err != nil {
			if errors.Is(err, ErrCategoryNotFound) {
				return false, nil
			}
			return false, err
		}

		if parentCat.ParentID == nil {
			break
		}
		currentID = *parentCat.ParentID
	}
	return false, nil
}
