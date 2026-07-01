// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dto

import "time"

// ReserveResponse represents the details of a successful stock reservation.
type ReserveResponse struct {
	ReservationID string    `json:"reservation_id"`
	VariantID     string    `json:"variant_id"`
	LocationID    string    `json:"location_id"`
	Quantity      int       `json:"quantity"`
	ExpiresAt     time.Time `json:"expires_at"`
}
