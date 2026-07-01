// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package bulk

import (
	"context"

	"ecom-engine/internal/core/catalog/domain"
	featureBulk "ecom-engine/internal/core/catalog/features/bulk"
	featureCore "ecom-engine/internal/core/catalog/features/core"
	"ecom-engine/internal/infra/db"
	pgCatalogCore "ecom-engine/internal/infra/db/postgres/catalog/core"
	"ecom-engine/pkg/logger"

	"go.opentelemetry.io/otel"
)

type PostgresBulkRepository struct {
	db       db.Database
	txMgr    db.TransactionManager
	coreRepo featureCore.Repository
}

func NewPostgresRepository(database db.Database, txManager db.TransactionManager) featureBulk.Repository {
	return &PostgresBulkRepository{
		db:       database,
		txMgr:    txManager,
		coreRepo: pgCatalogCore.NewPostgresCatalogRepository(database),
	}
}

func (r *PostgresBulkRepository) GetCategoryByID(ctx context.Context, id string) (*domain.Category, error) {
	return r.coreRepo.GetCategoryByID(ctx, id)
}

func (r *PostgresBulkRepository) BulkCreate(ctx context.Context, products []*domain.Product) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresBulkRepository.BulkCreate")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Bulk creating %d products", len(products))

	return r.txMgr.RunInTx(ctx, func(txCtx context.Context) error {
		for _, p := range products {
			if err := r.coreRepo.CreateProduct(txCtx, p); err != nil {
				span.RecordError(err)
				return err
			}
		}
		return nil
	})
}

func (r *PostgresBulkRepository) BulkUpdate(ctx context.Context, products []*domain.Product) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresBulkRepository.BulkUpdate")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Bulk updating %d products", len(products))

	return r.txMgr.RunInTx(ctx, func(txCtx context.Context) error {
		for _, p := range products {
			if err := r.coreRepo.UpdateProduct(txCtx, p); err != nil {
				span.RecordError(err)
				return err
			}
		}
		return nil
	})
}

func (r *PostgresBulkRepository) BulkDelete(ctx context.Context, ids []string) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresBulkRepository.BulkDelete")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Bulk deleting %d products", len(ids))

	return r.txMgr.RunInTx(ctx, func(txCtx context.Context) error {
		for _, id := range ids {
			if err := r.coreRepo.DeleteProduct(txCtx, id); err != nil {
				span.RecordError(err)
				return err
			}
		}
		return nil
	})
}
