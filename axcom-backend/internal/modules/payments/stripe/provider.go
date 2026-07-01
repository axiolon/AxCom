// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package stripe

import (
	"context"
	"ecom-engine/internal/modules/payments"
	"fmt"
	"time"
)

type StripeProvider struct { //nolint:revive // Name is intentionally explicit for the public API.
	apiKey string
}

func NewStripeProvider(apiKey string) *StripeProvider {
	return &StripeProvider{apiKey: apiKey}
}

func (p *StripeProvider) CreateIntent(_ context.Context, amount float64, currency string) (*payments.PaymentIntent, error) {
	fmt.Printf("Stripe: Creating payment intent for %.2f %s\n", amount, currency)
	return &payments.PaymentIntent{
		ID:       "pi_stripe_" + time.Now().Format("20060102150405"),
		Amount:   amount,
		Currency: currency,
		Status:   "succeeded",
	}, nil
}

func (p *StripeProvider) ConfirmIntent(_ context.Context, intentID string) error {
	fmt.Printf("Stripe: Confirming intent %s\n", intentID)
	return nil
}

func (p *StripeProvider) RefundIntent(_ context.Context, intentID string, amount float64) error {
	fmt.Printf("Stripe: Refunding %.2f for intent %s\n", amount, intentID)
	return nil
}
