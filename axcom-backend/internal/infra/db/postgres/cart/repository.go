// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cart

import (
	"context"

	"ecom-engine/internal/core/cart"
	"ecom-engine/internal/infra/db"
	"ecom-engine/pkg/logger"

	"go.opentelemetry.io/otel"
)

type PostgresCartRepository struct {
	db db.Database
}

func NewPostgresCartRepository(database db.Database) *PostgresCartRepository {
	return &PostgresCartRepository{
		db: database,
	}
}

func (r *PostgresCartRepository) GetByCustomerID(ctx context.Context, customerID string) (*cart.Cart, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresCartRepository.GetByCustomerID")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Fetching cart for customer ID: %s", customerID)

	query := "SELECT customer_id, items, created_at, updated_at FROM carts WHERE customer_id = $1 LIMIT 1"
	rows, err := r.db.Query(ctx, query, customerID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to fetch cart: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			span.RecordError(err)
			logger.ErrorCtx(ctx, "Postgres: Error iterating cart rows: %v", err)
			return nil, err
		}
		logger.DebugCtx(ctx, "Postgres: Cart not found for customer ID: %s", customerID)
		return nil, cart.ErrCartNotFound
	}

	var dbC dbCart
	if err := rows.Scan(&dbC.CustomerID, &dbC.Items, &dbC.CreatedAt, &dbC.UpdatedAt); err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to scan cart: %v", err)
		return nil, err
	}

	return toDomainCart(&dbC)
}

func (r *PostgresCartRepository) Save(ctx context.Context, c *cart.Cart) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresCartRepository.Save")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Saving cart for customer ID: %s", c.CustomerID)

	dbC, err := toDBCart(c)
	if err != nil {
		span.RecordError(err)
		return err
	}

	query := `INSERT INTO carts (customer_id, items, created_at, updated_at) 
              VALUES ($1, $2, $3, $4) 
              ON CONFLICT (customer_id) 
              DO UPDATE SET items = EXCLUDED.items, updated_at = EXCLUDED.updated_at`
	err = r.db.Exec(ctx, query, dbC.CustomerID, dbC.Items, dbC.CreatedAt, dbC.UpdatedAt)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to save cart: %v", err)
	} else {
		logger.DebugCtx(ctx, "Postgres: Successfully saved cart")
	}
	return err
}

func (r *PostgresCartRepository) Delete(ctx context.Context, customerID string) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresCartRepository.Delete")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Deleting cart for customer ID: %s", customerID)

	query := "DELETE FROM carts WHERE customer_id = $1"
	err := r.db.Exec(ctx, query, customerID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to delete cart: %v", err)
	} else {
		logger.DebugCtx(ctx, "Postgres: Successfully deleted cart")
	}
	return err
}
