// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"time"

	"ecom-engine/internal/core/inventory/domain"
)

type stockHistoryDoc struct {
	ID           string    `bson:"_id"`
	VariantID    string    `bson:"variant_id"`
	LocationID   string    `bson:"location_id"`
	OldQuantity  int       `bson:"old_quantity"`
	NewQuantity  int       `bson:"new_quantity"`
	ChangeReason string    `bson:"change_reason"`
	ChangedBy    string    `bson:"changed_by"`
	ChangedAt    time.Time `bson:"changed_at"`
}

func toHistoryDoc(h *domain.StockHistory) *stockHistoryDoc {
	if h == nil {
		return nil
	}
	return &stockHistoryDoc{
		ID:           h.ID,
		VariantID:    h.VariantID,
		LocationID:   h.LocationID,
		OldQuantity:  h.OldQuantity,
		NewQuantity:  h.NewQuantity,
		ChangeReason: h.ChangeReason,
		ChangedBy:    h.ChangedBy,
		ChangedAt:    h.ChangedAt,
	}
}

func toDomainHistory(doc *stockHistoryDoc) *domain.StockHistory {
	if doc == nil {
		return nil
	}
	return &domain.StockHistory{
		ID:           doc.ID,
		VariantID:    doc.VariantID,
		LocationID:   doc.LocationID,
		OldQuantity:  doc.OldQuantity,
		NewQuantity:  doc.NewQuantity,
		ChangeReason: doc.ChangeReason,
		ChangedBy:    doc.ChangedBy,
		ChangedAt:    doc.ChangedAt,
	}
}
