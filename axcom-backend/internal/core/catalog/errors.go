// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"ecom-engine/internal/core/catalog/domain"
)

var (
	// ErrProductNotFound is returned when a product with the requested ID is not found.
	ErrProductNotFound = domain.ErrProductNotFound

	// ErrCategoryNotFound is returned when a category with the requested ID is not found.
	ErrCategoryNotFound = domain.ErrCategoryNotFound

	// ErrDuplicateSKU is returned when a variant SKU is duplicated.
	ErrDuplicateSKU = domain.ErrDuplicateSKU

	// ErrInvalidPrice is returned when a variant price is negative.
	ErrInvalidPrice = domain.ErrInvalidPrice

	// ErrProductRequiresVariant is returned when a product does not have at least one variant.
	ErrProductRequiresVariant = domain.ErrProductRequiresVariant

	// ErrInvalidCategory is returned when category name or slug is invalid.
	ErrInvalidCategory = domain.ErrInvalidCategory
)
