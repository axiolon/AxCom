// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package shipping

import "context"

// Repository defines the contract for persisting shipment records.
type Repository interface {
	Create(ctx context.Context, s *Shipment) error
	GetByID(ctx context.Context, id string) (*Shipment, error)
	GetByOrderID(ctx context.Context, orderID string) (*Shipment, error)
	Update(ctx context.Context, s *Shipment) error
	ListAll(ctx context.Context, limit, offset int) ([]Shipment, error)
	GetByTrackingNumber(ctx context.Context, trackingNumber string) (*Shipment, error)
	Delete(ctx context.Context, id string) error
}
