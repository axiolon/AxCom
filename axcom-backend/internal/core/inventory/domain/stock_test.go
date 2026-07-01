// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"errors"
	"testing"
)

func TestReserve(t *testing.T) {
	tests := []struct {
		name        string
		initialQty  int
		reserveQty  int
		allowBO     bool
		boLimit     int
		expectError error
		expectedQty int
	}{
		{
			name:        "successful reservation",
			initialQty:  10,
			reserveQty:  4,
			expectError: nil,
			expectedQty: 6,
		},
		{
			name:        "reserve exact stock quantity",
			initialQty:  5,
			reserveQty:  5,
			expectError: nil,
			expectedQty: 0,
		},
		{
			name:        "insufficient stock",
			initialQty:  5,
			reserveQty:  6,
			expectError: ErrInsufficientStock,
			expectedQty: 5,
		},
		{
			name:        "reserve zero quantity",
			initialQty:  5,
			reserveQty:  0,
			expectError: ErrInvalidReservationQuantity,
			expectedQty: 5,
		},
		{
			name:        "reserve negative quantity",
			initialQty:  5,
			reserveQty:  -2,
			expectError: ErrInvalidReservationQuantity,
			expectedQty: 5,
		},
		{
			name:        "backorder allowed and within limit",
			initialQty:  5,
			reserveQty:  8,
			allowBO:     true,
			boLimit:     5,
			expectError: nil,
			expectedQty: -3,
		},
		{
			name:        "backorder allowed exactly at limit",
			initialQty:  5,
			reserveQty:  10,
			allowBO:     true,
			boLimit:     5,
			expectError: nil,
			expectedQty: -5,
		},
		{
			name:        "backorder allowed exceeding limit",
			initialQty:  5,
			reserveQty:  11,
			allowBO:     true,
			boLimit:     5,
			expectError: ErrInsufficientStock,
			expectedQty: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stock := &StockItem{
				VariantID:       "test-var",
				Quantity:        tt.initialQty,
				AllowBackorders: tt.allowBO,
				BackorderLimit:  tt.boLimit,
			}
			err := stock.Reserve(tt.reserveQty)

			if !errors.Is(err, tt.expectError) {
				t.Fatalf("expected error: %v, got: %v", tt.expectError, err)
			}
			if stock.Quantity != tt.expectedQty {
				t.Fatalf("expected stock quantity: %d, got: %d", tt.expectedQty, stock.Quantity)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name        string
		initialQty  int
		updateQty   int
		expectError error
		expectedQty int
	}{
		{
			name:        "successful positive update",
			initialQty:  10,
			updateQty:   15,
			expectError: nil,
			expectedQty: 15,
		},
		{
			name:        "successful update to zero",
			initialQty:  10,
			updateQty:   0,
			expectError: nil,
			expectedQty: 0,
		},
		{
			name:        "negative update rejected",
			initialQty:  10,
			updateQty:   -5,
			expectError: ErrInvalidQuantity,
			expectedQty: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stock := &StockItem{VariantID: "test-var", Quantity: tt.initialQty}
			err := stock.Update(tt.updateQty)

			if !errors.Is(err, tt.expectError) {
				t.Fatalf("expected error: %v, got: %v", tt.expectError, err)
			}
			if stock.Quantity != tt.expectedQty {
				t.Fatalf("expected stock quantity: %d, got: %d", tt.expectedQty, stock.Quantity)
			}
		})
	}
}

func TestIsLowStock(t *testing.T) {
	tests := []struct {
		name      string
		qty       int
		threshold int
		expected  bool
	}{
		{
			name:      "quantity above threshold",
			qty:       10,
			threshold: 5,
			expected:  false,
		},
		{
			name:      "quantity equal to threshold",
			qty:       5,
			threshold: 5,
			expected:  true,
		},
		{
			name:      "quantity below threshold",
			qty:       3,
			threshold: 5,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stock := &StockItem{VariantID: "test-var", Quantity: tt.qty, LowStockThreshold: tt.threshold}
			if stock.IsLowStock() != tt.expected {
				t.Fatalf("expected IsLowStock to be %v, got %v", tt.expected, stock.IsLowStock())
			}
		})
	}
}
