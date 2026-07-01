// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"context"
	"errors"
	"fmt"

	"ecom-engine/internal/core/inventory/domain"
	featcore "ecom-engine/internal/core/inventory/features/core"
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
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresInventoryCoreRepository.GetStock")
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
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresInventoryCoreRepository.SaveStock")
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

func (r *PostgresRepository) DeleteStock(ctx context.Context, variantID string, locationID string) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresInventoryCoreRepository.DeleteStock")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Deleting stock for variant: %s, location: %s", variantID, locationID)

	query := "DELETE FROM stock_items WHERE variant_id = $1 AND location_id = $2"
	result, err := r.db.ExecResult(ctx, query, variantID, locationID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to delete stock: %v", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	logger.DebugCtx(ctx, "Postgres: Successfully deleted stock")
	return nil
}

func (r *PostgresRepository) ListStock(ctx context.Context, filter featcore.ListStockFilter) ([]*domain.StockItem, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresInventoryCoreRepository.ListStock")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Listing stock with filters: %+v", filter)

	query := "SELECT variant_id, location_id, quantity, low_stock_threshold, allow_backorders, backorder_limit FROM stock_items WHERE 1=1"
	var args []interface{}
	argCount := 1

	if filter.VariantID != "" {
		query += fmt.Sprintf(" AND variant_id = $%d", argCount)
		args = append(args, filter.VariantID)
		argCount++
	}
	if filter.LocationID != "" {
		query += fmt.Sprintf(" AND location_id = $%d", argCount)
		args = append(args, filter.LocationID)
		argCount++
	}
	if filter.Status == "LOW_STOCK" {
		query += " AND quantity <= low_stock_threshold"
	}

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to list stock: %v", err)
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

func (r *PostgresRepository) SaveAlert(ctx context.Context, alert *domain.Alert) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresInventoryCoreRepository.SaveAlert")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Saving alert for ID: %s", alert.ID)

	doc := toAlertDoc(alert)
	query := `INSERT INTO alerts (id, type, message, variant_id, created_at, is_read) 
              VALUES ($1, $2, $3, $4, $5, $6) 
              ON CONFLICT (id) 
              DO UPDATE SET type = EXCLUDED.type, message = EXCLUDED.message, variant_id = EXCLUDED.variant_id, created_at = EXCLUDED.created_at`
	err := r.db.Exec(ctx, query, doc.ID, doc.Type, doc.Message, doc.VariantID, doc.CreatedAt, doc.IsRead)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to save alert: %v", err)
	} else {
		logger.DebugCtx(ctx, "Postgres: Successfully saved alert")
	}
	return err
}

func (r *PostgresRepository) ListAlerts(ctx context.Context, limit, offset int) ([]*domain.Alert, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresInventoryCoreRepository.ListAlerts")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Listing alerts with limit: %d, offset: %d", limit, offset)

	query := "SELECT id, type, message, variant_id, created_at, is_read FROM alerts ORDER BY created_at DESC LIMIT $1 OFFSET $2"
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to query alerts: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var alerts []*domain.Alert
	for rows.Next() {
		var doc alertDoc
		if err := rows.Scan(&doc.ID, &doc.Type, &doc.Message, &doc.VariantID, &doc.CreatedAt, &doc.IsRead); err != nil {
			span.RecordError(err)
			return nil, err
		}
		alerts = append(alerts, toDomainAlert(&doc))
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		return nil, err
	}

	return alerts, nil
}

func (r *PostgresRepository) AdjustQuantity(ctx context.Context, variantID, locationID string, delta int) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresInventoryCoreRepository.AdjustQuantity")
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
