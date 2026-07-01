// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package domain

import "errors"

const DefaultLowStockThreshold = 5

var (
	// ErrInsufficientStock is returned when there is not enough stock to fulfill a reservation.
	ErrInsufficientStock = errors.New("insufficient stock")

	// ErrInvalidQuantity is returned when a stock quantity is negative.
	ErrInvalidQuantity = errors.New("quantity cannot be negative")

	// ErrInvalidReservationQuantity is returned when the quantity to reserve is less than or equal to zero.
	ErrInvalidReservationQuantity = errors.New("reservation quantity must be greater than zero")

	// ErrNotFound is returned when a stock item is not found.
	ErrNotFound = errors.New("stock item not found")

	// ErrReservationNotFound is returned when a reservation is not found.
	ErrReservationNotFound = errors.New("reservation not found")

	// ErrDuplicateReservation is returned when a reservation with the same ID already exists.
	ErrDuplicateReservation = errors.New("reservation already exists")
)

// StockItem represents a stock record for a product variant.
type StockItem struct {
	VariantID         string `json:"variant_id"`
	LocationID        string `json:"location_id"`
	Quantity          int    `json:"quantity"`
	LowStockThreshold int    `json:"low_stock_threshold"` // Default should be 5 in application layer
	AllowBackorders   bool   `json:"allow_backorders"`
	BackorderLimit    int    `json:"backorder_limit"`
}

// IsLowStock returns true if the quantity is at or below the threshold.
func (s *StockItem) IsLowStock() bool {
	return s.Quantity <= s.LowStockThreshold
}

// Reserve validates the quantity and attempts to deduct it from stock.
// Enforces that reservation quantity must be > 0 and stock is sufficient (accounting for backorder allowances).
func (s *StockItem) Reserve(qty int) error {
	if qty <= 0 {
		return ErrInvalidReservationQuantity
	}
	if s.AllowBackorders {
		if s.Quantity-qty < -s.BackorderLimit {
			return ErrInsufficientStock
		}
	} else {
		if s.Quantity < qty {
			return ErrInsufficientStock
		}
	}
	s.Quantity -= qty
	return nil
}

// Update validates that the new quantity is non-negative and updates the stock item.
func (s *StockItem) Update(qty int) error {
	if qty < 0 {
		return ErrInvalidQuantity
	}
	s.Quantity = qty
	return nil
}
