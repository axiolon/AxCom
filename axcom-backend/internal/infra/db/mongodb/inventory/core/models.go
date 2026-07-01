// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"time"

	"ecom-engine/internal/core/inventory/domain"
)

type stockItemDoc struct {
	VariantID         string `bson:"variant_id"`
	LocationID        string `bson:"location_id"`
	Quantity          int    `bson:"quantity"`
	LowStockThreshold int    `bson:"low_stock_threshold"`
	AllowBackorders   bool   `bson:"allow_backorders"`
	BackorderLimit    int    `bson:"backorder_limit"`
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
	ID        string    `bson:"_id"`
	Type      string    `bson:"type"`
	Message   string    `bson:"message"`
	VariantID string    `bson:"variant_id"`
	CreatedAt time.Time `bson:"created_at"`
	IsRead    bool      `bson:"is_read"`
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
