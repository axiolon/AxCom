// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cart

import "context"

type Repository interface {
	// GetByCustomerID returns the cart for the given customer.
	// Returns ErrCartNotFound if no cart exists.
	GetByCustomerID(ctx context.Context, customerID string) (*Cart, error)
	Save(ctx context.Context, cart *Cart) error
	Delete(ctx context.Context, customerID string) error
}
