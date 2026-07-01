// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package images

import (
	"context"

	"ecom-engine/internal/core/catalog/domain"
	featureCore "ecom-engine/internal/core/catalog/features/core"
	featureImages "ecom-engine/internal/core/catalog/features/images"
	"ecom-engine/internal/infra/db"
	pgCatalogCore "ecom-engine/internal/infra/db/postgres/catalog/core"
	"ecom-engine/pkg/logger"

	"go.opentelemetry.io/otel"
)

type PostgresImageRepository struct {
	db       db.Database
	coreRepo featureCore.Repository
}

func NewPostgresRepository(database db.Database) featureImages.Repository {
	return &PostgresImageRepository{
		db:       database,
		coreRepo: pgCatalogCore.NewPostgresCatalogRepository(database),
	}
}

func (r *PostgresImageRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	return r.coreRepo.GetProductByID(ctx, id)
}

func (r *PostgresImageRepository) UpdateProductImages(ctx context.Context, id string, images []domain.ProductImage) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresImageRepository.UpdateProductImages")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Updating product images for ID: %s", id)

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		span.RecordError(err)
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	delQuery := "DELETE FROM product_images WHERE product_id = $1"
	err = tx.Exec(ctx, delQuery, id)
	if err != nil {
		span.RecordError(err)
		return err
	}

	for _, img := range images {
		insQuery := "INSERT INTO product_images (id, product_id, url, key, is_primary) VALUES ($1, $2, $3, $4, $5)"
		err = tx.Exec(ctx, insQuery, img.ID, id, img.URL, img.Key, img.IsPrimary)
		if err != nil {
			span.RecordError(err)
			return err
		}
	}
	return tx.Commit(ctx)
}
