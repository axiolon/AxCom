// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"errors"
	"testing"
	"time"
)

func TestValidateHistory(t *testing.T) {
	tests := []struct {
		name        string
		history     StockHistory
		expectError error
	}{
		{
			name: "valid history record",
			history: StockHistory{
				ID:           "hist_1",
				VariantID:    "var_1",
				LocationID:   "loc_1",
				OldQuantity:  10,
				NewQuantity:  15,
				ChangeReason: "Restock",
				ChangedBy:    "admin",
				ChangedAt:    time.Now(),
			},
			expectError: nil,
		},
		{
			name: "missing ID",
			history: StockHistory{
				ID:           "",
				VariantID:    "var_1",
				LocationID:   "loc_1",
				ChangeReason: "Restock",
			},
			expectError: ErrHistoryMissingID,
		},
		{
			name: "missing VariantID",
			history: StockHistory{
				ID:           "hist_1",
				VariantID:    "",
				LocationID:   "loc_1",
				ChangeReason: "Restock",
			},
			expectError: ErrHistoryMissingVariantID,
		},
		{
			name: "missing LocationID",
			history: StockHistory{
				ID:           "hist_1",
				VariantID:    "var_1",
				LocationID:   "",
				ChangeReason: "Restock",
			},
			expectError: ErrHistoryMissingLocationID,
		},
		{
			name: "missing ChangeReason",
			history: StockHistory{
				ID:           "hist_1",
				VariantID:    "var_1",
				LocationID:   "loc_1",
				ChangeReason: "",
			},
			expectError: ErrHistoryMissingReason,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHistory(tt.history)
			if tt.expectError == nil {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			} else {
				if !errors.Is(err, tt.expectError) {
					t.Fatalf("expected error: %v, got: %v", tt.expectError, err)
				}
			}
		})
	}
}
