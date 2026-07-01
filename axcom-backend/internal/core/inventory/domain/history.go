// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"errors"
	"time"
)

// ErrInvalidHistoryRecord is returned when the stock history record is invalid.
var (
	ErrInvalidHistoryRecord     = errors.New("invalid stock history record")
	ErrHistoryMissingID         = errors.New("history record must have a unique ID")
	ErrHistoryMissingVariantID  = errors.New("history record must have a Variant ID")
	ErrHistoryMissingLocationID = errors.New("history record must have a Location ID")
	ErrHistoryMissingReason     = errors.New("history record must specify a change reason")
)

// StockHistory represents a change in stock quantity.
type StockHistory struct {
	ID           string    `json:"id"`
	VariantID    string    `json:"variant_id"`
	LocationID   string    `json:"location_id"`
	OldQuantity  int       `json:"old_quantity"`
	NewQuantity  int       `json:"new_quantity"`
	ChangeReason string    `json:"change_reason"`
	ChangedBy    string    `json:"changed_by"`
	ChangedAt    time.Time `json:"changed_at"`
}

// ValidateHistory validates the stock history record.
func ValidateHistory(h StockHistory) error {
	if h.ID == "" {
		return ErrHistoryMissingID
	}
	if h.VariantID == "" {
		return ErrHistoryMissingVariantID
	}
	if h.LocationID == "" {
		return ErrHistoryMissingLocationID
	}
	if h.ChangeReason == "" {
		return ErrHistoryMissingReason
	}
	return nil
}
