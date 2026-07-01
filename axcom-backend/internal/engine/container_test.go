// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"testing"

	"ecom-engine/internal/core/cart"
	cartMerge "ecom-engine/internal/core/cart/merge"
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

	"github.com/stretchr/testify/assert"
)

// Minimal mock wrappers using interface embedding.
// This allows them to compile as satisfying the corresponding interfaces
// without having to implement dozens of dummy methods.
type mockCatalogQuery struct{ catalogCore.QueryService }
type mockCatalogCommand struct{ catalogCore.CommandService }
type mockCatalogImages struct{ catalogImages.Service }
type mockCatalogVariants struct{ catalogVariants.Service }
type mockCatalogDiscounts struct{ catalogDiscounts.Service }
type mockCatalogBulk struct{ catalogBulk.Service }
type mockCatalogReviews struct{ catalogReviews.Service }
type mockInventoryCore struct{ inventoryCore.Service }
type mockInventoryBulk struct{ inventoryBulk.Service }
type mockInventoryHistory struct{ inventoryHistory.Service }
type mockInventoryReservation struct{ inventoryReservation.Service }
type mockInventoryReports struct{ inventoryReports.Service }
type mockInventoryTransfer struct{ inventoryTransfer.Service }
type mockInventoryAdjustment struct{ inventoryAdjustment.Service }
type mockInventorySync struct{ inventorySync.Service }
type mockCart struct{ cart.Service }
type mockCartMerge struct{ cartMerge.Service }
type mockOrders struct{ orders.Service }
type mockPayments struct{ payments.Service }
type mockShipping struct{ shipping.Service }

func TestContainer_ProvideAndResolve(t *testing.T) {
	t.Parallel()

	t.Run("resolve registered generic service", func(t *testing.T) {
		t.Parallel()
		c := &Container{}

		val := "my-service-instance"
		c.Provide("custom.service", val)

		resolved, ok := c.Resolve("custom.service")
		assert.True(t, ok)
		assert.Equal(t, val, resolved)

		mustResolved := c.MustResolve("custom.service")
		assert.Equal(t, val, mustResolved)
	})

	t.Run("resolve unregistered service returns false", func(t *testing.T) {
		t.Parallel()
		c := &Container{}

		resolved, ok := c.Resolve("non-existent")
		assert.False(t, ok)
		assert.Nil(t, resolved)
	})

	t.Run("must resolve unregistered service panics", func(t *testing.T) {
		t.Parallel()
		c := &Container{}

		assert.PanicsWithValue(t, "engine: required service not registered: non-existent", func() {
			c.MustResolve("non-existent")
		})
	})

	t.Run("duplicate service registration panics", func(t *testing.T) {
		t.Parallel()
		c := &Container{}

		c.Provide("duplicate.key", "first")
		assert.PanicsWithValue(t, "engine: duplicate service registration for key: duplicate.key", func() {
			c.Provide("duplicate.key", "second")
		})
	})
}

func TestTypedResolveHelpers(t *testing.T) {
	t.Parallel()

	c := &Container{}

	// Register mock wrappers for all service constants
	c.Provide(ServiceCatalogQuery, mockCatalogQuery{})
	c.Provide(ServiceCatalogCommand, mockCatalogCommand{})
	c.Provide(ServiceCatalogImages, mockCatalogImages{})
	c.Provide(ServiceCatalogVariants, mockCatalogVariants{})
	c.Provide(ServiceCatalogDiscounts, mockCatalogDiscounts{})
	c.Provide(ServiceCatalogBulk, mockCatalogBulk{})
	c.Provide(ServiceCatalogReviews, mockCatalogReviews{})

	c.Provide(ServiceInventoryCore, mockInventoryCore{})
	c.Provide(ServiceInventoryBulk, mockInventoryBulk{})
	c.Provide(ServiceInventoryHistory, mockInventoryHistory{})
	c.Provide(ServiceInventoryReservation, mockInventoryReservation{})
	c.Provide(ServiceInventoryReports, mockInventoryReports{})
	c.Provide(ServiceInventoryTransfer, mockInventoryTransfer{})
	c.Provide(ServiceInventoryAdjustment, mockInventoryAdjustment{})
	c.Provide(ServiceInventorySync, mockInventorySync{})

	c.Provide(ServiceCart, mockCart{})
	c.Provide(ServiceCartMerge, mockCartMerge{})
	c.Provide(ServiceOrders, mockOrders{})
	c.Provide(ServicePayments, mockPayments{})
	c.Provide(ServiceShipping, mockShipping{})

	// Validate helper resolvers do not panic and return the correct types
	assert.NotNil(t, ResolveCatalogQuery(c))
	assert.NotNil(t, ResolveCatalogCommand(c))
	assert.NotNil(t, ResolveCatalogImages(c))
	assert.NotNil(t, ResolveCatalogVariants(c))
	assert.NotNil(t, ResolveCatalogDiscounts(c))
	assert.NotNil(t, ResolveCatalogBulk(c))
	assert.NotNil(t, ResolveCatalogReviews(c))

	assert.NotNil(t, ResolveInventoryCore(c))
	assert.NotNil(t, ResolveInventoryBulk(c))
	assert.NotNil(t, ResolveInventoryHistory(c))
	assert.NotNil(t, ResolveInventoryReservation(c))
	assert.NotNil(t, ResolveInventoryReports(c))
	assert.NotNil(t, ResolveInventoryTransfer(c))
	assert.NotNil(t, ResolveInventoryAdjustment(c))
	assert.NotNil(t, ResolveInventorySync(c))

	assert.NotNil(t, ResolveCart(c))
	assert.NotNil(t, ResolveCartMerge(c))
	assert.NotNil(t, ResolveOrders(c))
	assert.NotNil(t, ResolvePayments(c))
	assert.NotNil(t, ResolveShipping(c))
}
