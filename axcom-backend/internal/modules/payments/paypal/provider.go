// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package paypal

import (
	"context"
	"ecom-engine/internal/modules/payments"
	"fmt"
	"time"
)

type PayPalProvider struct { //nolint:revive // Name is intentionally explicit for the public API.
	clientID string
}

func NewPayPalProvider(clientID string) *PayPalProvider {
	return &PayPalProvider{clientID: clientID}
}

func (p *PayPalProvider) CreateIntent(_ context.Context, amount float64, currency string) (*payments.PaymentIntent, error) {
	fmt.Printf("PayPal: Creating order for %.2f %s\n", amount, currency)
	return &payments.PaymentIntent{
		ID:          "pp_order_" + time.Now().Format("20060102150405"),
		Amount:      amount,
		Currency:    currency,
		Status:      "pending",
		RedirectURL: "https://www.paypal.com/checkoutnow?token=pp_order_" + time.Now().Format("20060102150405"),
	}, nil
}

func (p *PayPalProvider) ConfirmIntent(_ context.Context, intentID string) error {
	fmt.Printf("PayPal: Capturing order %s\n", intentID)
	return nil
}

func (p *PayPalProvider) RefundIntent(_ context.Context, intentID string, amount float64) error {
	fmt.Printf("PayPal: Refunding transaction %.2f for order %s\n", amount, intentID)
	return nil
}
