// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package domain defines the domain models for inventory operations.
package domain

import (
	"context"
	"time"
)

// Alert represents an inventory alert (e.g. low stock).
type Alert struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	VariantID string    `json:"variant_id"`
	CreatedAt time.Time `json:"created_at"`
	IsRead    bool      `json:"is_read"`
}

// AlertDispatcher defines the interface for dispatching alerts.
type AlertDispatcher interface {
	Dispatch(ctx context.Context, alert Alert) error
}
