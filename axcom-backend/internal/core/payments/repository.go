// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package payments

import (
	"context"
	"errors"
)

var (
	ErrNotFound         = errors.New("repository: record not found")
	ErrDuplicatePayment = errors.New("repository: duplicate payment")
)

// Repository defines the contract for persisting payment records.
type Repository interface {
	Create(ctx context.Context, p *Payment) error
	GetByID(ctx context.Context, id string) (*Payment, error)
	GetByOrderID(ctx context.Context, orderID string) (*Payment, error)
	GetByProviderIntentID(ctx context.Context, provider string, intentID string) (*Payment, error)
	Update(ctx context.Context, p *Payment) error
	ListAll(ctx context.Context, limit, offset int) ([]Payment, error)
	ListByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]Payment, error)
}
