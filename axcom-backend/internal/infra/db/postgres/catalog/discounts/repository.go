// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package discounts

import (
	"context"

	"ecom-engine/internal/core/catalog/domain"
	featureCore "ecom-engine/internal/core/catalog/features/core"
	featureDiscounts "ecom-engine/internal/core/catalog/features/discounts"
	"ecom-engine/internal/infra/db"
	pgCatalogCore "ecom-engine/internal/infra/db/postgres/catalog/core"
	"ecom-engine/pkg/logger"

	"go.opentelemetry.io/otel"
)

type PostgresDiscountRepository struct {
	db       db.Database
	coreRepo featureCore.Repository
}

func NewPostgresRepository(database db.Database) featureDiscounts.Repository {
	return &PostgresDiscountRepository{
		db:       database,
		coreRepo: pgCatalogCore.NewPostgresCatalogRepository(database),
	}
}

func (r *PostgresDiscountRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	return r.coreRepo.GetProductByID(ctx, id)
}

func (r *PostgresDiscountRepository) UpdateProductDiscount(ctx context.Context, id string, discount *domain.ProductDiscount) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresDiscountRepository.UpdateProductDiscount")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Updating product discount for ID: %s", id)

	var discType interface{}
	var discValue interface{}
	if discount != nil {
		discType = discount.Type
		discValue = discount.Value
	}

	query := "UPDATE products SET discount_type = $1, discount_value = $2 WHERE id = $3"
	err := r.db.Exec(ctx, query, discType, discValue, id)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to update product discount: %v", err)
		return err
	}
	return nil
}
