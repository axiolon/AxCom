// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reservation

import (
	"time"

	"ecom-engine/internal/core/inventory/domain"
)

type stockItemDoc struct {
	VariantID         string `db:"variant_id"`
	LocationID        string `db:"location_id"`
	Quantity          int    `db:"quantity"`
	LowStockThreshold int    `db:"low_stock_threshold"`
	AllowBackorders   bool   `db:"allow_backorders"`
	BackorderLimit    int    `db:"backorder_limit"`
}

func toStockItemDoc(s *domain.StockItem) *stockItemDoc {
	if s == nil {
		return nil
	}
	return &stockItemDoc{
		VariantID:         s.VariantID,
		LocationID:        s.LocationID,
		Quantity:          s.Quantity,
		LowStockThreshold: s.LowStockThreshold,
		AllowBackorders:   s.AllowBackorders,
		BackorderLimit:    s.BackorderLimit,
	}
}

func toDomainStockItem(doc *stockItemDoc) *domain.StockItem {
	if doc == nil {
		return nil
	}
	return &domain.StockItem{
		VariantID:         doc.VariantID,
		LocationID:        doc.LocationID,
		Quantity:          doc.Quantity,
		LowStockThreshold: doc.LowStockThreshold,
		AllowBackorders:   doc.AllowBackorders,
		BackorderLimit:    doc.BackorderLimit,
	}
}

type reservationDoc struct {
	ID         string    `db:"id"`
	VariantID  string    `db:"variant_id"`
	LocationID string    `db:"location_id"`
	Quantity   int       `db:"quantity"`
	ExpiresAt  time.Time `db:"expires_at"`
}

func toReservationDoc(r *domain.Reservation) *reservationDoc {
	if r == nil {
		return nil
	}
	return &reservationDoc{
		ID:         r.ID,
		VariantID:  r.VariantID,
		LocationID: r.LocationID,
		Quantity:   r.Quantity,
		ExpiresAt:  r.ExpiresAt,
	}
}

func toDomainReservation(doc *reservationDoc) *domain.Reservation {
	if doc == nil {
		return nil
	}
	return &domain.Reservation{
		ID:         doc.ID,
		VariantID:  doc.VariantID,
		LocationID: doc.LocationID,
		Quantity:   doc.Quantity,
		ExpiresAt:  doc.ExpiresAt,
	}
}
