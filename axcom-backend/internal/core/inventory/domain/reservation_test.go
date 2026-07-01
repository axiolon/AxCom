// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package domain defines the domain models for inventory operations.
package domain

import (
	"errors"
	"testing"
	"time"
)

func TestValidateReservation(t *testing.T) {
	tests := []struct {
		name        string
		reservation Reservation
		expectError error
	}{
		{
			name: "valid reservation",
			reservation: Reservation{
				ID:        "res_1",
				VariantID: "var_1",
				Quantity:  5,
				ExpiresAt: time.Now().Add(5 * time.Minute),
			},
			expectError: nil,
		},
		{
			name: "missing variant ID",
			reservation: Reservation{
				ID:        "res_1",
				VariantID: "   ",
				Quantity:  5,
				ExpiresAt: time.Now().Add(5 * time.Minute),
			},
			expectError: ErrInvalidVariantID,
		},
		{
			name: "invalid quantity zero",
			reservation: Reservation{
				ID:        "res_1",
				VariantID: "var_1",
				Quantity:  0,
				ExpiresAt: time.Now().Add(5 * time.Minute),
			},
			expectError: ErrInvalidReservationQuantity,
		},
		{
			name: "invalid quantity negative",
			reservation: Reservation{
				ID:        "res_1",
				VariantID: "var_1",
				Quantity:  -1,
				ExpiresAt: time.Now().Add(5 * time.Minute),
			},
			expectError: ErrInvalidReservationQuantity,
		},
		{
			name: "expired expiration time",
			reservation: Reservation{
				ID:        "res_1",
				VariantID: "var_1",
				Quantity:  5,
				ExpiresAt: time.Now().Add(-1 * time.Minute),
			},
			expectError: ErrReservationExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReservation(tt.reservation)
			if !errors.Is(err, tt.expectError) {
				t.Fatalf("expected error: %v, got: %v", tt.expectError, err)
			}
		})
	}
}
