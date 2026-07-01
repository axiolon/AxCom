// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package payhere

import (
	"context"
	"ecom-engine/internal/modules/payments"
	"fmt"
	"time"
)

type PayHereProvider struct { //nolint:revive // Name is intentionally explicit for the public API.
	merchantID string
}

func NewPayHereProvider(merchantID string) *PayHereProvider {
	return &PayHereProvider{merchantID: merchantID}
}

func (p *PayHereProvider) CreateIntent(_ context.Context, amount float64, currency string) (*payments.PaymentIntent, error) {
	fmt.Printf("PayHere: Creating payment session for %.2f %s\n", amount, currency)
	return &payments.PaymentIntent{
		ID:          "ph_intent_" + time.Now().Format("20060102150405"),
		Amount:      amount,
		Currency:    currency,
		Status:      "pending",
		RedirectURL: "https://sandbox.payhere.lk/pay/checkout?merchant_id=" + p.merchantID,
	}, nil
}

func (p *PayHereProvider) ConfirmIntent(_ context.Context, intentID string) error {
	fmt.Printf("PayHere: Processing callback/notification for payment %s\n", intentID)
	return nil
}

func (p *PayHereProvider) RefundIntent(_ context.Context, intentID string, amount float64) error {
	fmt.Printf("PayHere: Refunding %.2f for intent %s\n", amount, intentID)
	return nil
}
