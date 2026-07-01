// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"context"
	"errors"

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

func (r *PostgresRepository) GetStock(ctx context.Context, variantID string, locationID string) (*domain.StockItem, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresInventoryTransferRepository.GetStock")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Fetching stock for variant: %s, location: %s", variantID, locationID)

	query := "SELECT variant_id, location_id, quantity, low_stock_threshold, allow_backorders, backorder_limit FROM stock_items WHERE variant_id = $1 AND location_id = $2 LIMIT 1"
	rows, err := r.db.Query(ctx, query, variantID, locationID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to fetch stock: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			span.RecordError(err)
			logger.ErrorCtx(ctx, "Postgres: Rows iteration error: %v", err)
			return nil, err
		}
		logger.DebugCtx(ctx, "Postgres: Stock not found for variant: %s, location: %s", variantID, locationID)
		return nil, domain.ErrNotFound
	}

	var doc stockItemDoc
	if err := rows.Scan(&doc.VariantID, &doc.LocationID, &doc.Quantity, &doc.LowStockThreshold, &doc.AllowBackorders, &doc.BackorderLimit); err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to scan stock: %v", err)
		return nil, err
	}

	return toDomainStockItem(&doc), nil
}

func (r *PostgresRepository) SaveStock(ctx context.Context, stock *domain.StockItem) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresInventoryTransferRepository.SaveStock")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Saving stock for variant: %s, location: %s", stock.VariantID, stock.LocationID)

	doc := toStockItemDoc(stock)
	query := `INSERT INTO stock_items (variant_id, location_id, quantity, low_stock_threshold, allow_backorders, backorder_limit) 
              VALUES ($1, $2, $3, $4, $5, $6) 
              ON CONFLICT (variant_id, location_id) 
              DO UPDATE SET quantity = EXCLUDED.quantity, low_stock_threshold = EXCLUDED.low_stock_threshold, allow_backorders = EXCLUDED.allow_backorders, backorder_limit = EXCLUDED.backorder_limit`
	err := r.db.Exec(ctx, query, doc.VariantID, doc.LocationID, doc.Quantity, doc.LowStockThreshold, doc.AllowBackorders, doc.BackorderLimit)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to save stock: %v", err)
	} else {
		logger.DebugCtx(ctx, "Postgres: Successfully saved stock")
	}
	return err
}

func (r *PostgresRepository) AdjustQuantity(ctx context.Context, variantID, locationID string, delta int) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresInventoryTransferRepository.AdjustQuantity")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Adjusting quantity atomically for variant: %s, location: %s, delta: %d", variantID, locationID, delta)

	query := `
		UPDATE stock_items
		SET quantity = quantity + $3
		WHERE variant_id = $1 AND location_id = $2
		  AND ((allow_backorders = false AND quantity + $3 >= 0)
		    OR (allow_backorders = true AND quantity + $3 >= -backorder_limit))`

	result, err := r.db.ExecResult(ctx, query, variantID, locationID, delta)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to adjust stock quantity: %v", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return err
	}

	if rowsAffected == 0 {
		_, err := r.GetStock(ctx, variantID, locationID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		return domain.ErrInsufficientStock
	}

	return nil
}
