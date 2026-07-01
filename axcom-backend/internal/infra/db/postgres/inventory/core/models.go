// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

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

type alertDoc struct {
	ID        string    `db:"id"`
	Type      string    `db:"type"`
	Message   string    `db:"message"`
	VariantID string    `db:"variant_id"`
	CreatedAt time.Time `db:"created_at"`
	IsRead    bool      `db:"is_read"`
}

func toAlertDoc(a *domain.Alert) *alertDoc {
	if a == nil {
		return nil
	}
	return &alertDoc{
		ID:        a.ID,
		Type:      a.Type,
		Message:   a.Message,
		VariantID: a.VariantID,
		CreatedAt: a.CreatedAt,
		IsRead:    a.IsRead,
	}
}

func toDomainAlert(doc *alertDoc) *domain.Alert {
	if doc == nil {
		return nil
	}
	return &domain.Alert{
		ID:        doc.ID,
		Type:      doc.Type,
		Message:   doc.Message,
		VariantID: doc.VariantID,
		CreatedAt: doc.CreatedAt,
		IsRead:    doc.IsRead,
	}
}
