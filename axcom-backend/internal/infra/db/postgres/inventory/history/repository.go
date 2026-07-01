// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"

	"ecom-engine/internal/core/inventory/domain"
	"ecom-engine/internal/infra/db"
	"ecom-engine/pkg/logger"

	"go.opentelemetry.io/otel"
)

type PostgresRepository struct {
	db db.Database
}

func NewPostgresRepository(database db.Database) *PostgresRepository {
	return &PostgresRepository{
		db: database,
	}
}

func (r *PostgresRepository) CreateHistory(ctx context.Context, h *domain.StockHistory) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresInventoryHistoryRepository.CreateHistory")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Creating stock history for variant ID: %s", h.VariantID)

	doc := toHistoryDoc(h)
	query := `INSERT INTO stock_history (id, variant_id, location_id, old_quantity, new_quantity, change_reason, changed_by, changed_at) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	err := r.db.Exec(ctx, query, doc.ID, doc.VariantID, doc.LocationID, doc.OldQuantity, doc.NewQuantity, doc.ChangeReason, doc.ChangedBy, doc.ChangedAt)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to create stock history: %v", err)
	} else {
		logger.DebugCtx(ctx, "Postgres: Successfully created stock history")
	}
	return err
}

func (r *PostgresRepository) GetHistory(ctx context.Context, variantID string, limit, offset int) ([]*domain.StockHistory, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresInventoryHistoryRepository.GetHistory")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding history for variant ID: %s, limit: %d, offset: %d", variantID, limit, offset)

	query := `SELECT id, variant_id, location_id, old_quantity, new_quantity, change_reason, changed_by, changed_at 
              FROM stock_history WHERE variant_id = $1 ORDER BY changed_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, variantID, limit, offset)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to fetch stock history: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var history []*domain.StockHistory
	for rows.Next() {
		var doc stockHistoryDoc
		if err := rows.Scan(&doc.ID, &doc.VariantID, &doc.LocationID, &doc.OldQuantity, &doc.NewQuantity, &doc.ChangeReason, &doc.ChangedBy, &doc.ChangedAt); err != nil {
			span.RecordError(err)
			return nil, err
		}
		history = append(history, toDomainHistory(&doc))
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		return nil, err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully fetched history")
	return history, nil
}
