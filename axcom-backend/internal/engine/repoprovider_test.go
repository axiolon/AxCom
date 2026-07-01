// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestRepoProvider_Routing(t *testing.T) {
	t.Parallel()

	t.Run("mongodb provider returns non-nil repositories", func(t *testing.T) {
		t.Parallel()
		client, _ := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
		mongoDB := client.Database("test_db")
		rp := newRepoProvider("mongodb", mongoDB, nil, nil)

		assert.NotNil(t, rp.AuthUserRepo())
		assert.NotNil(t, rp.AuthTokenRepo())
		assert.NotNil(t, rp.CartRepo())
		assert.NotNil(t, rp.CatalogCoreRepo())
		assert.NotNil(t, rp.CatalogImagesRepo())
		assert.NotNil(t, rp.CatalogVariantsRepo())
		assert.NotNil(t, rp.CatalogDiscountsRepo())
		assert.NotNil(t, rp.CatalogBulkRepo())
		assert.NotNil(t, rp.CatalogReviewsRepo())
		assert.NotNil(t, rp.InventoryCoreRepo())
		assert.NotNil(t, rp.InventoryBulkRepo())
		assert.NotNil(t, rp.InventoryHistoryRepo())
		assert.NotNil(t, rp.InventoryReservationRepo())
		assert.NotNil(t, rp.InventoryReportsRepo())
		assert.NotNil(t, rp.InventoryTransferRepo())
		assert.NotNil(t, rp.InventoryAdjustmentRepo())
		assert.NotNil(t, rp.InventorySyncRepo())
		assert.NotNil(t, rp.OrderRepo())
		assert.NotNil(t, rp.PaymentRepo())
		assert.NotNil(t, rp.ShipmentRepo())
		assert.NotNil(t, rp.OutboxRepo())
		assert.NotNil(t, rp.DedupStore())
	})

	t.Run("postgres provider returns non-nil repositories", func(t *testing.T) {
		t.Parallel()
		rp := newRepoProvider("postgres", nil, nil, nil)

		assert.NotNil(t, rp.AuthUserRepo())
		assert.NotNil(t, rp.AuthTokenRepo())
		assert.NotNil(t, rp.CartRepo())
		assert.NotNil(t, rp.CatalogCoreRepo())
		assert.NotNil(t, rp.CatalogImagesRepo())
		assert.NotNil(t, rp.CatalogVariantsRepo())
		assert.NotNil(t, rp.CatalogDiscountsRepo())
		assert.NotNil(t, rp.CatalogBulkRepo())
		assert.NotNil(t, rp.CatalogReviewsRepo())
		assert.NotNil(t, rp.InventoryCoreRepo())
		assert.NotNil(t, rp.InventoryBulkRepo())
		assert.NotNil(t, rp.InventoryHistoryRepo())
		assert.NotNil(t, rp.InventoryReservationRepo())
		assert.NotNil(t, rp.InventoryReportsRepo())
		assert.NotNil(t, rp.InventoryTransferRepo())
		assert.NotNil(t, rp.InventoryAdjustmentRepo())
		assert.NotNil(t, rp.InventorySyncRepo())
		assert.NotNil(t, rp.OrderRepo())
		assert.NotNil(t, rp.PaymentRepo())
		assert.NotNil(t, rp.ShipmentRepo())
		assert.NotNil(t, rp.OutboxRepo())
		assert.NotNil(t, rp.DedupStore())
	})

	t.Run("unknown db type returns nil", func(t *testing.T) {
		t.Parallel()
		rp := newRepoProvider("sqlite", nil, nil, nil)

		assert.Nil(t, rp.AuthUserRepo())
		assert.Nil(t, rp.AuthTokenRepo())
		assert.Nil(t, rp.CartRepo())
		assert.Nil(t, rp.CatalogCoreRepo())
		assert.Nil(t, rp.OrderRepo())
		assert.Nil(t, rp.PaymentRepo())
		assert.Nil(t, rp.ShipmentRepo())
		assert.Nil(t, rp.OutboxRepo())
		assert.Nil(t, rp.DedupStore())
	})
}
