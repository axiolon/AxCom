// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package orders contains the core service, domain models, validation, and state machine transitions for managing orders in the system.
package orders

import (
	"ecom-engine/internal/core/orders/domain"
)

// OrderStatus represents the status of an order.
// It is type-aliased to domain.OrderStatus to expose the domain model at the package boundary.
type OrderStatus = domain.OrderStatus

const (
	// StatusPending is the initial state of a created order.
	StatusPending OrderStatus = domain.StatusPending

	// StatusPaid represents an order that has been successfully paid.
	StatusPaid OrderStatus = domain.StatusPaid

	// StatusShipped represents an order that has been shipped.
	StatusShipped OrderStatus = domain.StatusShipped

	// StatusDone represents a completed order.
	StatusDone OrderStatus = domain.StatusDone

	// StatusCanceled represents a canceled order.
	StatusCanceled OrderStatus = domain.StatusCanceled
)

// OrderItem represents a single item inside an order.
// It is type-aliased to domain.OrderItem to expose the domain model at the package boundary.
type OrderItem = domain.OrderItem

// Order represents an order placed by a customer.
// It is type-aliased to domain.Order to expose the domain model at the package boundary.
type Order = domain.Order

// OrderCustomerSnapshot holds contact details for the customer associated with the order at the time of checkout.
// It is type-aliased to domain.OrderCustomerSnapshot to expose the domain model at the package boundary.
type OrderCustomerSnapshot = domain.OrderCustomerSnapshot
