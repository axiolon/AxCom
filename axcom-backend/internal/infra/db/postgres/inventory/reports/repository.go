// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reports

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

func (r *PostgresRepository) GetLowStockItems(ctx context.Context) ([]*domain.StockItem, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresInventoryReportsRepository.GetLowStockItems")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Querying low stock items")

	query := "SELECT variant_id, location_id, quantity, low_stock_threshold, allow_backorders, backorder_limit FROM stock_items WHERE quantity <= low_stock_threshold"
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to query low stock items: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []*domain.StockItem
	for rows.Next() {
		var doc stockItemDoc
		if err := rows.Scan(&doc.VariantID, &doc.LocationID, &doc.Quantity, &doc.LowStockThreshold, &doc.AllowBackorders, &doc.BackorderLimit); err != nil {
			span.RecordError(err)
			return nil, err
		}
		items = append(items, toDomainStockItem(&doc))
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		return nil, err
	}

	return items, nil
}

func (r *PostgresRepository) GetAllStockItems(ctx context.Context) ([]*domain.StockItem, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresInventoryReportsRepository.GetAllStockItems")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Querying all stock items")

	query := "SELECT variant_id, location_id, quantity, low_stock_threshold, allow_backorders, backorder_limit FROM stock_items"
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to query all stock items: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []*domain.StockItem
	for rows.Next() {
		var doc stockItemDoc
		if err := rows.Scan(&doc.VariantID, &doc.LocationID, &doc.Quantity, &doc.LowStockThreshold, &doc.AllowBackorders, &doc.BackorderLimit); err != nil {
			span.RecordError(err)
			return nil, err
		}
		items = append(items, toDomainStockItem(&doc))
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		return nil, err
	}

	return items, nil
}
