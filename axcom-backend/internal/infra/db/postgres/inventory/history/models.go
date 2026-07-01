// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"time"

	"ecom-engine/internal/core/inventory/domain"
)

type stockHistoryDoc struct {
	ID           string    `db:"id"`
	VariantID    string    `db:"variant_id"`
	LocationID   string    `db:"location_id"`
	OldQuantity  int       `db:"old_quantity"`
	NewQuantity  int       `db:"new_quantity"`
	ChangeReason string    `db:"change_reason"`
	ChangedBy    string    `db:"changed_by"`
	ChangedAt    time.Time `db:"changed_at"`
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
