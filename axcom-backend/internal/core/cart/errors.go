// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cart

import "errors"

var (
	// ErrCartNotFound is returned when no cart exists for the given customer.
	ErrCartNotFound = errors.New("cart not found")

	// ErrInvalidQuantity is returned when a cart item has a quantity less than or equal to zero.
	ErrInvalidQuantity = errors.New("quantity must be greater than zero")

	// ErrVariantIDRequired is returned when the variant ID is empty.
	ErrVariantIDRequired = errors.New("variant ID is required")
)
