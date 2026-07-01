// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"ecom-engine/internal/core/auth"
	"ecom-engine/internal/core/cart"
	catalogBulk "ecom-engine/internal/core/catalog/features/bulk"
	catalogCore "ecom-engine/internal/core/catalog/features/core"
	catalogDiscounts "ecom-engine/internal/core/catalog/features/discounts"
	catalogImages "ecom-engine/internal/core/catalog/features/images"
	catalogReviews "ecom-engine/internal/core/catalog/features/reviews"
	catalogVariants "ecom-engine/internal/core/catalog/features/variants"
	inventoryAdjustment "ecom-engine/internal/core/inventory/features/adjustment"
	inventoryBulk "ecom-engine/internal/core/inventory/features/bulk"
	inventoryCore "ecom-engine/internal/core/inventory/features/core"
	inventoryHistory "ecom-engine/internal/core/inventory/features/history"
	inventoryReports "ecom-engine/internal/core/inventory/features/reports"
	inventoryReservation "ecom-engine/internal/core/inventory/features/reservation"
	inventorySync "ecom-engine/internal/core/inventory/features/sync"
	inventoryTransfer "ecom-engine/internal/core/inventory/features/transfer"
	"ecom-engine/internal/core/orders"
	"ecom-engine/internal/core/payments"
	"ecom-engine/internal/core/shipping"
	"ecom-engine/internal/events"
	infradb "ecom-engine/internal/infra/db"
	mongoAuth "ecom-engine/internal/infra/db/mongodb/auth"
	mongoCart "ecom-engine/internal/infra/db/mongodb/cart"
	mongoCatalogBulk "ecom-engine/internal/infra/db/mongodb/catalog/bulk"
	mongoCatalogCore "ecom-engine/internal/infra/db/mongodb/catalog/core"
	mongoCatalogDiscounts "ecom-engine/internal/infra/db/mongodb/catalog/discounts"
	mongoCatalogImages "ecom-engine/internal/infra/db/mongodb/catalog/images"
	mongoCatalogReviews "ecom-engine/internal/infra/db/mongodb/catalog/reviews"
	mongoCatalogVariants "ecom-engine/internal/infra/db/mongodb/catalog/variants"
	mongoInventoryAdjustment "ecom-engine/internal/infra/db/mongodb/inventory/adjustment"
	mongoInventoryBulk "ecom-engine/internal/infra/db/mongodb/inventory/bulk"
	mongoInventoryCore "ecom-engine/internal/infra/db/mongodb/inventory/core"
	mongoInventoryHistory "ecom-engine/internal/infra/db/mongodb/inventory/history"
	mongoInventoryReports "ecom-engine/internal/infra/db/mongodb/inventory/reports"
	mongoInventoryReservation "ecom-engine/internal/infra/db/mongodb/inventory/reservation"
	mongoInventorySync "ecom-engine/internal/infra/db/mongodb/inventory/sync"
	mongoInventoryTransfer "ecom-engine/internal/infra/db/mongodb/inventory/transfer"
	mongoOrders "ecom-engine/internal/infra/db/mongodb/orders"
	mongoOutbox "ecom-engine/internal/infra/db/mongodb/outbox"
	mongoPayments "ecom-engine/internal/infra/db/mongodb/payments"
	mongoShipping "ecom-engine/internal/infra/db/mongodb/shipping"
	postgres "ecom-engine/internal/infra/db/postgres"
	pgAuth "ecom-engine/internal/infra/db/postgres/auth"
	pgCart "ecom-engine/internal/infra/db/postgres/cart"
	pgCatalogBulk "ecom-engine/internal/infra/db/postgres/catalog/bulk"
	pgCatalogCore "ecom-engine/internal/infra/db/postgres/catalog/core"
	pgCatalogDiscounts "ecom-engine/internal/infra/db/postgres/catalog/discounts"
	pgCatalogImages "ecom-engine/internal/infra/db/postgres/catalog/images"
	pgCatalogReviews "ecom-engine/internal/infra/db/postgres/catalog/reviews"
	pgCatalogVariants "ecom-engine/internal/infra/db/postgres/catalog/variants"
	pgInvAdjustment "ecom-engine/internal/infra/db/postgres/inventory/adjustment"
	pgInvBulk "ecom-engine/internal/infra/db/postgres/inventory/bulk"
	pgInvCore "ecom-engine/internal/infra/db/postgres/inventory/core"
	pgInvHistory "ecom-engine/internal/infra/db/postgres/inventory/history"
	pgInvReports "ecom-engine/internal/infra/db/postgres/inventory/reports"
	pgInvReservation "ecom-engine/internal/infra/db/postgres/inventory/reservation"
	pgInvSync "ecom-engine/internal/infra/db/postgres/inventory/sync"
	pgInvTransfer "ecom-engine/internal/infra/db/postgres/inventory/transfer"
	pgOrders "ecom-engine/internal/infra/db/postgres/orders"
	pgOutbox "ecom-engine/internal/infra/db/postgres/outbox"
	pgPayments "ecom-engine/internal/infra/db/postgres/payments"
	pgShipping "ecom-engine/internal/infra/db/postgres/shipping"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RepoProvider is a DB-agnostic repository factory. It is initialized once
// during engine bootstrap based on the global DB config. Modules call its
// typed methods to receive the correct repository implementation without
// importing MongoDB or Postgres packages directly.
//
// Adding a new database backend: add a case to each method here.
// No other files need to change.
type RepoProvider struct {
	dbType    string
	mongoDB   *mongo.Database           // non-nil when dbType == "mongodb"
	pgAdapter *postgres.PostgresAdapter // non-nil when dbType == "postgres"
	txManager infradb.TransactionManager
}

// newRepoProvider constructs a RepoProvider for the given DB backend.
func newRepoProvider(dbType string, mongoDB *mongo.Database, pgAdapter *postgres.PostgresAdapter, txManager infradb.TransactionManager) *RepoProvider {
	return &RepoProvider{
		dbType:    dbType,
		mongoDB:   mongoDB,
		pgAdapter: pgAdapter,
		txManager: txManager,
	}
}

// ---- Auth ----

func (rp *RepoProvider) AuthUserRepo() auth.UserRepository {
	switch rp.dbType {
	case "mongodb":
		return mongoAuth.NewMongoUserRepository(rp.mongoDB)
	case "postgres":
		return pgAuth.NewPostgresUserRepository(rp.pgAdapter)
	}
	return nil
}

func (rp *RepoProvider) AuthTokenRepo() auth.TokenRepository {
	switch rp.dbType {
	case "mongodb":
		return mongoAuth.NewMongoTokenRepository(rp.mongoDB)
	case "postgres":
		return pgAuth.NewPostgresTokenRepository(rp.pgAdapter)
	}
	return nil
}

// ---- Cart ----

func (rp *RepoProvider) CartRepo() cart.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoCart.NewMongoCartRepository(rp.mongoDB)
	case "postgres":
		return pgCart.NewPostgresCartRepository(rp.pgAdapter)
	}
	return nil
}

// ---- Catalog ----

func (rp *RepoProvider) CatalogCoreRepo() catalogCore.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoCatalogCore.NewMongoCatalogRepository(rp.mongoDB)
	case "postgres":
		return pgCatalogCore.NewPostgresCatalogRepository(rp.pgAdapter)
	}
	return nil
}

func (rp *RepoProvider) CatalogImagesRepo() catalogImages.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoCatalogImages.NewMongoRepository(rp.mongoDB)
	case "postgres":
		return pgCatalogImages.NewPostgresRepository(rp.pgAdapter)
	}
	return nil
}

func (rp *RepoProvider) CatalogVariantsRepo() catalogVariants.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoCatalogVariants.NewMongoRepository(rp.mongoDB)
	case "postgres":
		return pgCatalogVariants.NewPostgresRepository(rp.pgAdapter)
	}
	return nil
}

func (rp *RepoProvider) CatalogDiscountsRepo() catalogDiscounts.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoCatalogDiscounts.NewMongoRepository(rp.mongoDB)
	case "postgres":
		return pgCatalogDiscounts.NewPostgresRepository(rp.pgAdapter)
	}
	return nil
}

func (rp *RepoProvider) CatalogBulkRepo() catalogBulk.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoCatalogBulk.NewMongoRepository(rp.mongoDB)
	case "postgres":
		// Postgres bulk repo needs the txManager for transactional batch ops.
		return pgCatalogBulk.NewPostgresRepository(rp.pgAdapter, rp.txManager)
	}
	return nil
}

func (rp *RepoProvider) CatalogReviewsRepo() catalogReviews.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoCatalogReviews.NewMongoRepository(rp.mongoDB)
	case "postgres":
		return pgCatalogReviews.NewPostgresRepository(rp.pgAdapter)
	}
	return nil
}

// ---- Inventory ----

func (rp *RepoProvider) InventoryCoreRepo() inventoryCore.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoInventoryCore.NewMongoRepository(rp.mongoDB)
	case "postgres":
		return pgInvCore.NewPostgresRepository(rp.pgAdapter)
	}
	return nil
}

func (rp *RepoProvider) InventoryBulkRepo() inventoryBulk.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoInventoryBulk.NewMongoRepository(rp.mongoDB)
	case "postgres":
		return pgInvBulk.NewPostgresRepository(rp.pgAdapter)
	}
	return nil
}

func (rp *RepoProvider) InventoryHistoryRepo() inventoryHistory.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoInventoryHistory.NewMongoRepository(rp.mongoDB)
	case "postgres":
		return pgInvHistory.NewPostgresRepository(rp.pgAdapter)
	}
	return nil
}

func (rp *RepoProvider) InventoryReservationRepo() inventoryReservation.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoInventoryReservation.NewMongoRepository(rp.mongoDB)
	case "postgres":
		return pgInvReservation.NewPostgresRepository(rp.pgAdapter)
	}
	return nil
}

func (rp *RepoProvider) InventoryReportsRepo() inventoryReports.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoInventoryReports.NewMongoRepository(rp.mongoDB)
	case "postgres":
		return pgInvReports.NewPostgresRepository(rp.pgAdapter)
	}
	return nil
}

func (rp *RepoProvider) InventoryTransferRepo() inventoryTransfer.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoInventoryTransfer.NewMongoRepository(rp.mongoDB)
	case "postgres":
		return pgInvTransfer.NewPostgresRepository(rp.pgAdapter)
	}
	return nil
}

func (rp *RepoProvider) InventoryAdjustmentRepo() inventoryAdjustment.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoInventoryAdjustment.NewMongoRepository(rp.mongoDB)
	case "postgres":
		return pgInvAdjustment.NewPostgresRepository(rp.pgAdapter)
	}
	return nil
}

func (rp *RepoProvider) InventorySyncRepo() inventorySync.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoInventorySync.NewMongoRepository(rp.mongoDB)
	case "postgres":
		return pgInvSync.NewPostgresRepository(rp.pgAdapter)
	}
	return nil
}

// ---- Orders ----

func (rp *RepoProvider) OrderRepo() orders.OrderRepository {
	switch rp.dbType {
	case "mongodb":
		return mongoOrders.NewMongoOrderRepository(rp.mongoDB)
	case "postgres":
		return pgOrders.NewPostgresOrderRepository(rp.pgAdapter)
	}
	return nil
}

// ---- Payments ----

func (rp *RepoProvider) PaymentRepo() payments.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoPayments.NewMongoPaymentRepository(rp.mongoDB)
	case "postgres":
		return pgPayments.NewPostgresPaymentRepository(rp.pgAdapter)
	}
	return nil
}

// ---- Shipping ----

func (rp *RepoProvider) ShipmentRepo() shipping.Repository {
	switch rp.dbType {
	case "mongodb":
		return mongoShipping.NewMongoShipmentRepository(rp.mongoDB)
	case "postgres":
		return pgShipping.NewPostgresShipmentRepository(rp.pgAdapter)
	}
	return nil
}

// ---- Outbox ----

func (rp *RepoProvider) OutboxRepo() events.OutboxRepository {
	switch rp.dbType {
	case "postgres":
		return pgOutbox.NewPostgresOutboxRepository(rp.pgAdapter)
	case "mongodb":
		return mongoOutbox.NewMongoOutboxRepository()
	}
	return nil
}

func (rp *RepoProvider) DedupStore() events.DedupStore {
	switch rp.dbType {
	case "postgres":
		return pgOutbox.NewPostgresDedupStore(rp.pgAdapter)
	case "mongodb":
		return mongoOutbox.NewMongoDedupStore()
	}
	return nil
}
