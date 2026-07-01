// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package variants

import (
	"context"
	"encoding/json"

	"ecom-engine/internal/core/catalog/domain"
	featureCore "ecom-engine/internal/core/catalog/features/core"
	featureVariants "ecom-engine/internal/core/catalog/features/variants"
	"ecom-engine/internal/infra/db"
	pgCatalogCore "ecom-engine/internal/infra/db/postgres/catalog/core"
	"ecom-engine/pkg/logger"

	"go.opentelemetry.io/otel"
)

type PostgresVariantRepository struct {
	db       db.Database
	coreRepo featureCore.Repository
}

func NewPostgresRepository(database db.Database) featureVariants.Repository {
	return &PostgresVariantRepository{
		db:       database,
		coreRepo: pgCatalogCore.NewPostgresCatalogRepository(database),
	}
}

func (r *PostgresVariantRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	return r.coreRepo.GetProductByID(ctx, id)
}

func (r *PostgresVariantRepository) UpdateProductVariants(ctx context.Context, id string, variants []domain.Variant) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresVariantRepository.UpdateProductVariants")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Updating product variants for ID: %s", id)

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		span.RecordError(err)
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	delQuery := "DELETE FROM variants WHERE product_id = $1"
	err = tx.Exec(ctx, delQuery, id)
	if err != nil {
		span.RecordError(err)
		return err
	}

	for _, v := range variants {
		attrsBytes, _ := json.Marshal(v.Attributes)
		insQuery := `INSERT INTO variants (id, product_id, sku, name, price, stock, attributes) 
                     VALUES ($1, $2, $3, $4, $5, $6, $7)`
		err = tx.Exec(ctx, insQuery, v.ID, id, v.SKU, v.Name, v.Price, v.Stock, string(attrsBytes))
		if err != nil {
			span.RecordError(err)
			return err
		}
	}
	return tx.Commit(ctx)
}
