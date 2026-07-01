// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"ecom-engine/internal/core/catalog/domain"
)

type VariantDTO struct {
	ID         string            `json:"id"`
	SKU        string            `json:"sku" binding:"required"`
	Name       string            `json:"name" binding:"required"`
	Price      float64           `json:"price" binding:"required,min=0"`
	Attributes map[string]string `json:"attributes"`
}

type CreateProductRequest struct {
	Name        string       `json:"name" binding:"required"`
	Description string       `json:"description"`
	CategoryID  string       `json:"category_id" binding:"required"`
	Variants    []VariantDTO `json:"variants" binding:"required,dive"`
}

type UpdateProductRequest struct {
	Name        string       `json:"name" binding:"required"`
	Description string       `json:"description"`
	CategoryID  string       `json:"category_id" binding:"required"`
	Variants    []VariantDTO `json:"variants" binding:"required,dive"`
}

type CreateCategoryRequest struct {
	Name     string  `json:"name" binding:"required"`
	Slug     string  `json:"slug"`
	ParentID *string `json:"parent_id"`
}

type UpdateCategoryRequest struct {
	Name     string  `json:"name" binding:"required"`
	Slug     string  `json:"slug"`
	ParentID *string `json:"parent_id"`
}

type ListProductsQuery struct {
	CategoryID string   `form:"category_id"`
	Category   string   `form:"category"`
	PriceMin   *float64 `form:"price_min"`
	PriceMax   *float64 `form:"price_max"`
	MinPrice   *float64 `form:"minPrice"`
	MaxPrice   *float64 `form:"maxPrice"`
	Attributes string   `form:"attributes"` // e.g. "size:XL,color:blue"
	InStock    *bool    `form:"inStock"`
	Q          string   `form:"q"`
	Page       *int     `form:"page"`
	Limit      *int     `form:"limit"`
}

type ProductFilter struct {
	CategoryIDs []string
	MinPrice    *float64
	MaxPrice    *float64
	InStock     *bool
	Q           string
	Attributes  map[string]string
	Limit       int64
	Offset      int64
}

type VariantResponse struct {
	ID              string            `json:"id"`
	SKU             string            `json:"sku"`
	Name            string            `json:"name"`
	Price           float64           `json:"price"`
	DiscountedPrice float64           `json:"discounted_price,omitempty"`
	Stock           *int              `json:"stock,omitempty"`
	Attributes      map[string]string `json:"attributes"`
}

type ProductResponse struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	CategoryID  string                  `json:"category_id"`
	Variants    []VariantResponse       `json:"variants"`
	Images      []domain.ProductImage   `json:"images,omitempty"`
	Discount    *domain.ProductDiscount `json:"discount,omitempty"`
}
