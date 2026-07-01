// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrInvalidVariantID is returned when the variant ID is empty.
	ErrInvalidVariantID = errors.New("variant ID is required")

	// ErrReservationExpired is returned when the reservation expiration is in the past.
	ErrReservationExpired = errors.New("reservation expiration time must be in the future")
)

// Reservation represents a temporary lock on stock for a variant.
type Reservation struct {
	ID         string    `json:"id"`
	VariantID  string    `json:"variant_id"`
	LocationID string    `json:"location_id"`
	Quantity   int       `json:"quantity"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// ValidateReservation checks if all fields of a reservation are valid.
func ValidateReservation(r Reservation) error {
	if strings.TrimSpace(r.VariantID) == "" {
		return ErrInvalidVariantID
	}
	if r.Quantity <= 0 {
		return ErrInvalidReservationQuantity
	}
	if r.ExpiresAt.Before(time.Now()) {
		return ErrReservationExpired
	}
	return nil
}
