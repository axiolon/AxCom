// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"context"
	"time"

	"ecom-engine/internal/core/catalog/domain"
	"ecom-engine/internal/infra/cache"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
)

// QueryService defines the business logic contract for reading catalog data.
type QueryService interface {
	// GetProduct retrieves a mapped ProductResponse detail by product ID.
	GetProduct(ctx context.Context, id string) (*ProductResponse, error)
	// GetProductEntity retrieves the raw product domain model for validation/direct checks.
	GetProductEntity(ctx context.Context, id string) (*domain.Product, error)
	// GetProductByVariantID retrieves the parent product of a specific variant ID.
	GetProductByVariantID(ctx context.Context, variantID string) (*domain.Product, error)
	// GetProducts returns products matching query, sorting, price, and category filters.
	GetProducts(ctx context.Context, query *ListProductsQuery) ([]ProductResponse, error)
	// GetCategory retrieves category details by ID.
	GetCategory(ctx context.Context, id string) (*domain.Category, error)
	// GetCategories lists all categories in the system.
	GetCategories(ctx context.Context) ([]domain.Category, error)
}

type catalogQueryService struct {
	repo         Repository
	cacheManager cache.Manager
}

// NewCatalogQueryService creates a new modular Catalog Query Service.
func NewCatalogQueryService(repo Repository, cacheManager cache.Manager) QueryService {
	return &catalogQueryService{
		repo:         repo,
		cacheManager: cacheManager,
	}
}

// GetProduct retrieves product details, maps them to a Response and applies active discount rates.
func (s *catalogQueryService) GetProduct(ctx context.Context, id string) (*ProductResponse, error) {
	logger.InfoCtx(ctx, "Retrieving product: %s", id)

	var p domain.Product
	key := "catalog:product:" + id
	err := s.cacheManager.GetOrFetch(ctx, key, &p, 15*time.Minute, func() (interface{}, error) {
		return s.repo.GetProductByID(ctx, id)
	})
	if err != nil {
		logger.ErrorCtx(ctx, "Product %s not found: %v", id, err)
		return nil, apperrors.NewNotFound("product not found", ErrProductNotFound)
	}

	res := s.mapProductToResponse(ctx, &p, true)
	return &res, nil
}

// GetProductEntity returns the database domain object directly.
func (s *catalogQueryService) GetProductEntity(ctx context.Context, id string) (*domain.Product, error) {
	var p domain.Product
	key := "catalog:product:" + id
	err := s.cacheManager.GetOrFetch(ctx, key, &p, 15*time.Minute, func() (interface{}, error) {
		return s.repo.GetProductByID(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetProductByVariantID retrieves the parent product mapping matching the input SKU/variant ID.
func (s *catalogQueryService) GetProductByVariantID(ctx context.Context, variantID string) (*domain.Product, error) {
	logger.InfoCtx(ctx, "Retrieving product by variant ID: %s", variantID)

	var p domain.Product
	key := "catalog:product:variant:" + variantID
	err := s.cacheManager.GetOrFetch(ctx, key, &p, 15*time.Minute, func() (interface{}, error) {
		return s.repo.GetProductByVariantID(ctx, variantID)
	})
	if err != nil {
		logger.ErrorCtx(ctx, "Product for variant %s not found: %v", variantID, err)
		return nil, apperrors.NewNotFound("product not found for variant", ErrProductNotFound)
	}
	return &p, nil
}

// GetProducts queries the database and applies dynamic filter criteria (category, price limits, tags, stock).
func (s *catalogQueryService) GetProducts(ctx context.Context, query *ListProductsQuery) ([]ProductResponse, error) {
	logger.InfoCtx(ctx, "Retrieving products list")

	filter := &ProductFilter{}
	if query != nil {
		catID := query.CategoryID
		if catID == "" {
			catID = query.Category
		}
		if catID != "" {
			categories, err := s.repo.ListCategories(ctx)
			if err != nil {
				logger.ErrorCtx(ctx, "Failed to retrieve categories for filtering: %v", err)
				return nil, apperrors.NewInternal("failed to retrieve categories for filtering", err)
			}
			allowedCategories := getCategoryDescendants(catID, categories)
			var catIDs []string
			for k := range allowedCategories {
				catIDs = append(catIDs, k)
			}
			filter.CategoryIDs = catIDs
		}

		minPrice := query.PriceMin
		if minPrice == nil {
			minPrice = query.MinPrice
		}
		filter.MinPrice = minPrice

		maxPrice := query.PriceMax
		if maxPrice == nil {
			maxPrice = query.MaxPrice
		}
		filter.MaxPrice = maxPrice

		filter.InStock = query.InStock
		filter.Attributes = parseAttributes(query.Attributes)
		filter.Q = query.Q

		limit := int64(20)
		if query.Limit != nil && *query.Limit > 0 {
			limit = int64(*query.Limit)
			if limit > 100 {
				limit = 100
			}
		}
		filter.Limit = limit

		offset := int64(0)
		if query.Page != nil && *query.Page > 1 {
			offset = int64(*query.Page-1) * limit
		}
		filter.Offset = offset
	} else {
		filter.Limit = 20
	}

	products, err := s.repo.ListProducts(ctx, filter)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to retrieve products: %v", err)
		return nil, apperrors.NewInternal("failed to retrieve products", err)
	}

	var resList []ProductResponse
	for _, p := range products {
		res := s.mapProductToResponse(ctx, &p, true)
		resList = append(resList, res)
	}

	return resList, nil
}

// GetCategories lists all categories registered.
func (s *catalogQueryService) GetCategories(ctx context.Context) ([]domain.Category, error) {
	logger.InfoCtx(ctx, "Retrieving all categories")

	categories, err := s.repo.ListCategories(ctx)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to retrieve categories: %v", err)
		return nil, apperrors.NewInternal("failed to retrieve categories", err)
	}

	return categories, nil
}

// GetCategory retrieves category data details.
func (s *catalogQueryService) GetCategory(ctx context.Context, id string) (*domain.Category, error) {
	logger.InfoCtx(ctx, "Retrieving category: %s", id)

	var c domain.Category
	key := "catalog:category:" + id
	err := s.cacheManager.GetOrFetch(ctx, key, &c, 15*time.Minute, func() (interface{}, error) {
		return s.repo.GetCategoryByID(ctx, id)
	})
	if err != nil {
		logger.ErrorCtx(ctx, "Category %s not found: %v", id, err)
		return nil, apperrors.NewNotFound("category not found", ErrCategoryNotFound)
	}

	return &c, nil
}

// mapProductToResponse transforms internal product entities into API Responses, applying active discounts.
func (s *catalogQueryService) mapProductToResponse(_ context.Context, p *domain.Product, includeStock bool) ProductResponse {
	var variants []VariantResponse
	for _, v := range p.Variants {
		discPrice := v.Price
		if p.Discount != nil {
			switch p.Discount.Type {
			case "percentage":
				discPrice = v.Price * (1.0 - p.Discount.Value/100.0)
			case "fixed":
				discPrice = v.Price - p.Discount.Value
				if discPrice < 0 {
					discPrice = 0
				}
			}
		}

		var stockVal *int
		if includeStock {
			stock := v.Stock
			stockVal = &stock
		}

		variants = append(variants, VariantResponse{
			ID:              v.ID,
			SKU:             v.SKU,
			Name:            v.Name,
			Price:           v.Price,
			DiscountedPrice: discPrice,
			Stock:           stockVal,
			Attributes:      v.Attributes,
		})
	}

	return ProductResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		CategoryID:  p.CategoryID,
		Variants:    variants,
		Images:      p.Images,
		Discount:    p.Discount,
	}
}
