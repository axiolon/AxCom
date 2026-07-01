// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package payments

import "context"

type PaymentIntent struct {
	ID          string  `json:"id"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Status      string  `json:"status"` // succeeded, pending, failed
	RedirectURL string  `json:"redirect_url,omitempty"`
}

type PaymentProvider interface {
	CreateIntent(ctx context.Context, amount float64, currency string) (*PaymentIntent, error)
	ConfirmIntent(ctx context.Context, intentID string) error
	RefundIntent(ctx context.Context, intentID string, amount float64) error
}
