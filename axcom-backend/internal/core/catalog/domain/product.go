// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrProductNotFound indicates a product does not exist.
	ErrProductNotFound = errors.New("product not found")

	// ErrCategoryNotFound indicates a category does not exist.
	ErrCategoryNotFound = errors.New("category not found")

	// ErrProductRequiresVariant is returned when a product does not have at least one variant.
	ErrProductRequiresVariant = errors.New("product must have at least one variant")

	// ErrInvalidPrice is returned when a variant price is negative.
	ErrInvalidPrice = errors.New("price cannot be negative")

	// ErrDuplicateSKU is returned when a variant SKU is duplicated within a product.
	ErrDuplicateSKU = errors.New("duplicate SKU found in variants")

	// ErrSKURequired is returned when a variant SKU is empty.
	ErrSKURequired = errors.New("SKU is required")

	// ErrInvalidProductName is returned when a product name is empty.
	ErrInvalidProductName = errors.New("product name is required")

	// ErrInvalidProductCategoryID is returned when a product category ID is empty.
	ErrInvalidProductCategoryID = errors.New("product category ID is required")
)

// ProductImage represents a product media item.
type ProductImage struct {
	ID        string `json:"id" bson:"id"`
	URL       string `json:"url" bson:"url"`
	Key       string `json:"key" bson:"key"`
	IsPrimary bool   `json:"is_primary" bson:"is_primary"`
}

// ProductDiscount represents a price discount applied to a product.
type ProductDiscount struct {
	Type  string  `json:"type" bson:"type"` // "percentage" or "fixed"
	Value float64 `json:"value" bson:"value"`
}

// Variant represents a specific version of a product (e.g. size, color).
type Variant struct {
	ID         string            `json:"id"`
	SKU        string            `json:"sku"`
	Name       string            `json:"name"`
	Price      float64           `json:"price"`
	Stock      int               `json:"stock" bson:"stock"`
	Attributes map[string]string `json:"attributes"`
}

// Product represents a catalog item containing one or more variants.
type Product struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	CategoryID  string           `json:"category_id"`
	Variants    []Variant        `json:"variants"`
	Images      []ProductImage   `json:"images,omitempty" bson:"images,omitempty"`
	Discount    *ProductDiscount `json:"discount,omitempty" bson:"discount,omitempty"`
	Version     int              `json:"version" bson:"version"`
	CreatedAt   time.Time        `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at" bson:"updated_at"`
}

// ValidateProduct checks if the product is valid.
func ValidateProduct(p Product) error {
	if strings.TrimSpace(p.Name) == "" {
		return ErrInvalidProductName
	}
	if strings.TrimSpace(p.CategoryID) == "" {
		return ErrInvalidProductCategoryID
	}
	if len(p.Variants) == 0 {
		return ErrProductRequiresVariant
	}
	return ValidateVariants(p.Variants)
}

// ValidateVariants checks that all variants have a non-negative price
// and that their SKUs are unique within the product.
func ValidateVariants(variants []Variant) error {
	seenSKUs := make(map[string]bool)
	for _, v := range variants {
		if v.Price < 0 {
			return ErrInvalidPrice
		}

		sku := strings.TrimSpace(v.SKU)
		if sku == "" {
			return ErrSKURequired
		}
		if seenSKUs[sku] {
			return ErrDuplicateSKU
		}
		seenSKUs[sku] = true
	}
	return nil
}
