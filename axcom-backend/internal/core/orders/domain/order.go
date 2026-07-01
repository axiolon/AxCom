// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"errors"
	"time"
)

// OrderStatus represents the current state of an order.
type OrderStatus string

const (
	StatusPending  OrderStatus = "pending"
	StatusPaid     OrderStatus = "paid"
	StatusShipped  OrderStatus = "shipped"
	StatusDone     OrderStatus = "done"
	StatusCanceled OrderStatus = "canceled"
)

var (
	// ErrEmptyOrder is returned when attempting to create an order without items.
	ErrEmptyOrder = errors.New("order must contain at least one item")

	// ErrVariantIDRequired is returned when an order item lacks a variant ID.
	ErrVariantIDRequired = errors.New("variant ID is required")

	// ErrInvalidQuantity is returned when an item quantity is zero or negative.
	ErrInvalidQuantity = errors.New("quantity must be greater than zero")

	// ErrInvalidPrice is returned when an item price is negative.
	ErrInvalidPrice = errors.New("price cannot be negative")

	// ErrOrderNotFound is returned when the requested order is not found.
	ErrOrderNotFound = errors.New("order not found")

	// ErrForbidden is returned when a user attempts to access an order that does not belong to them.
	ErrForbidden = errors.New("you do not have access to this order")
)

// OrderItem represents a single item in an order.
type OrderItem struct {
	VariantID string  `json:"variant_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// OrderCustomerSnapshot holds contact details for the customer associated with the order at the time of checkout.
type OrderCustomerSnapshot struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	ContactNumber string `json:"contact_number"`
}

// Order represents a customer order in the system.
type Order struct {
	ID               string                `json:"id"`
	CustomerID       string                `json:"customer_id"` // Empty if Guest
	CustomerSnapshot OrderCustomerSnapshot `json:"customer_snapshot"`
	Items            []OrderItem           `json:"items"`
	Total            float64               `json:"total"`
	Status           OrderStatus           `json:"status"`
	CreatedAt        time.Time             `json:"created_at"`
}

// ValidateItems validates order item entries.
func ValidateItems(items []OrderItem) error {
	if len(items) == 0 {
		return ErrEmptyOrder
	}
	for _, item := range items {
		if item.VariantID == "" {
			return ErrVariantIDRequired
		}
		if item.Quantity <= 0 {
			return ErrInvalidQuantity
		}
		if item.Price < 0 {
			return ErrInvalidPrice
		}
	}
	return nil
}

// CalculateTotal calculates the total cost of all order items.
func CalculateTotal(items []OrderItem) float64 {
	var total float64
	for _, item := range items {
		total += item.Price * float64(item.Quantity)
	}
	return total
}
