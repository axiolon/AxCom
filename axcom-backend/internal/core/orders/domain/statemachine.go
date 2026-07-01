// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package domain

import "errors"

var (
	// ErrInvalidTransition is returned when an action is not allowed from the current state.
	ErrInvalidTransition = errors.New("invalid transition action")
)

// OrderStateMachine manages status transitions for an order.
type OrderStateMachine struct{}

// NewOrderStateMachine creates and returns a new OrderStateMachine.
func NewOrderStateMachine() *OrderStateMachine {
	return &OrderStateMachine{}
}

// Transition evaluates the action against the current status and returns the next status or an error.
func (sm *OrderStateMachine) Transition(current OrderStatus, action string) (OrderStatus, error) {
	switch current {
	case StatusPending:
		if action == "pay" {
			return StatusPaid, nil
		}
		if action == "cancel" {
			return StatusCanceled, nil
		}
	case StatusPaid:
		if action == "ship" {
			return StatusShipped, nil
		}
		if action == "cancel" {
			return StatusCanceled, nil
		}
	case StatusShipped:
		if action == "complete" {
			return StatusDone, nil
		}
	}
	return current, ErrInvalidTransition
}
