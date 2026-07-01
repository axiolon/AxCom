// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package orders contains the core service, domain models, validation, and state machine transitions for managing orders in the system.
package orders

import (
	"ecom-engine/internal/core/orders/domain"
)

// OrderStateMachine manages status transitions for an order.
// It is type-aliased to domain.OrderStateMachine to expose it at the package boundary.
type OrderStateMachine = domain.OrderStateMachine

// NewOrderStateMachine creates and returns a new OrderStateMachine.
func NewOrderStateMachine() *OrderStateMachine {
	return domain.NewOrderStateMachine()
}
