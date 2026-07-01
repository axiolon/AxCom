// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package adjustment

import (
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
