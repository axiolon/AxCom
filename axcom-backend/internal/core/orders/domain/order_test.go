// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"testing"
)

func TestValidateItems(t *testing.T) {
	tests := []struct {
		name    string
		items   []OrderItem
		wantErr error
	}{
		{
			name:    "empty items list",
			items:   []OrderItem{},
			wantErr: ErrEmptyOrder,
		},
		{
			name: "missing variant id",
			items: []OrderItem{
				{VariantID: "", Quantity: 1, Price: 10.0},
			},
			wantErr: ErrVariantIDRequired,
		},
		{
			name: "zero quantity",
			items: []OrderItem{
				{VariantID: "var_1", Quantity: 0, Price: 10.0},
			},
			wantErr: ErrInvalidQuantity,
		},
		{
			name: "negative quantity",
			items: []OrderItem{
				{VariantID: "var_1", Quantity: -5, Price: 10.0},
			},
			wantErr: ErrInvalidQuantity,
		},
		{
			name: "negative price",
			items: []OrderItem{
				{VariantID: "var_1", Quantity: 1, Price: -2.0},
			},
			wantErr: ErrInvalidPrice,
		},
		{
			name: "valid items",
			items: []OrderItem{
				{VariantID: "var_1", Quantity: 2, Price: 10.0},
				{VariantID: "var_2", Quantity: 1, Price: 5.5},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateItems(tt.items)
			if err != tt.wantErr {
				t.Errorf("ValidateItems() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCalculateTotal(t *testing.T) {
	items := []OrderItem{
		{VariantID: "var_1", Quantity: 2, Price: 10.0},
		{VariantID: "var_2", Quantity: 3, Price: 5.5},
	}
	expected := 20.0 + 16.5 // 36.5
	got := CalculateTotal(items)
	if got != expected {
		t.Errorf("CalculateTotal() = %v, want %v", got, expected)
	}
}

func TestOrderStateMachine_Transition(t *testing.T) {
	sm := NewOrderStateMachine()

	tests := []struct {
		name       string
		current    OrderStatus
		action     string
		wantStatus OrderStatus
		wantErr    error
	}{
		{
			name:       "pending pay -> paid",
			current:    StatusPending,
			action:     "pay",
			wantStatus: StatusPaid,
			wantErr:    nil,
		},
		{
			name:       "pending cancel -> canceled",
			current:    StatusPending,
			action:     "cancel",
			wantStatus: StatusCanceled,
			wantErr:    nil,
		},
		{
			name:       "paid ship -> shipped",
			current:    StatusPaid,
			action:     "ship",
			wantStatus: StatusShipped,
			wantErr:    nil,
		},
		{
			name:       "shipped complete -> done",
			current:    StatusShipped,
			action:     "complete",
			wantStatus: StatusDone,
			wantErr:    nil,
		},
		{
			name:       "pending ship -> invalid",
			current:    StatusPending,
			action:     "ship",
			wantStatus: StatusPending,
			wantErr:    ErrInvalidTransition,
		},
		{
			name:       "canceled complete -> invalid",
			current:    StatusCanceled,
			action:     "complete",
			wantStatus: StatusCanceled,
			wantErr:    ErrInvalidTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sm.Transition(tt.current, tt.action)
			if err != tt.wantErr {
				t.Errorf("Transition() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.wantStatus {
				t.Errorf("Transition() got = %v, wantStatus %v", got, tt.wantStatus)
			}
		})
	}
}
