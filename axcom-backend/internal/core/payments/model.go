// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package payments

import "time"

// PaymentStatus represents the status of a payment transaction.
type PaymentStatus string

const (
	StatusPending      PaymentStatus = "pending"
	StatusSucceeded    PaymentStatus = "succeeded"
	StatusFailed       PaymentStatus = "failed"
	StatusRefunded     PaymentStatus = "refunded"
	StatusRefundFailed PaymentStatus = "refund_failed"
)

// Payment represents a payment record in the system.
type Payment struct {
	ID               string        `json:"id"`
	OrderID          string        `json:"order_id"`
	CustomerID       string        `json:"customer_id"`
	Amount           float64       `json:"amount"`
	Currency         string        `json:"currency"`
	Provider         string        `json:"provider"`
	ProviderIntentID string        `json:"provider_intent_id"`
	Status           PaymentStatus `json:"status"`
	IdempotencyKey   string        `json:"idempotency_key"`
	FailureReason    string        `json:"failure_reason,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
	RefundedAt       *time.Time    `json:"refunded_at,omitempty"`
}
