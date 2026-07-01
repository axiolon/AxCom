// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package engine

import (
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
)

// Well-known service keys used with Container.Provide / Container.Resolve.
// Each module that exports a service must use one of these constants so
// dependent modules can locate the service without string literals.
const (
	// Catalog
	ServiceCatalogQuery     = "catalog.query"
	ServiceCatalogCommand   = "catalog.command"
	ServiceCatalogImages    = "catalog.images"
	ServiceCatalogVariants  = "catalog.variants"
	ServiceCatalogDiscounts = "catalog.discounts"
	ServiceCatalogBulk      = "catalog.bulk"
	ServiceCatalogReviews   = "catalog.reviews"

	// Inventory
	ServiceInventoryCore        = "inventory.core"
	ServiceInventoryBulk        = "inventory.bulk"
	ServiceInventoryHistory     = "inventory.history"
	ServiceInventoryReservation = "inventory.reservation"
	ServiceInventoryReports     = "inventory.reports"
	ServiceInventoryTransfer    = "inventory.transfer"
	ServiceInventoryAdjustment  = "inventory.adjustment"
	ServiceInventorySync        = "inventory.sync"

	// Cart
	ServiceCart      = "cart"
	ServiceCartMerge = "cart.merge"

	// Orders
	ServiceOrders = "orders"

	// Payments
	ServicePayments = "payments"

	// Shipping
	ServiceShipping = "shipping"

	// Dashboard
	ServiceDashboard = "dashboard"
)

// --- Typed resolve helpers ---
// Using these avoids raw type assertions scattered across module code.
// Each helper panics if the service is not registered, which is safe when
// the calling module has declared the dependency in Requires().

func ResolveCatalogQuery(c *Container) catalogCore.QueryService {
	return c.MustResolve(ServiceCatalogQuery).(catalogCore.QueryService)
}

func ResolveCatalogCommand(c *Container) catalogCore.CommandService {
	return c.MustResolve(ServiceCatalogCommand).(catalogCore.CommandService)
}

func ResolveCatalogImages(c *Container) catalogImages.Service {
	return c.MustResolve(ServiceCatalogImages).(catalogImages.Service)
}

func ResolveCatalogVariants(c *Container) catalogVariants.Service {
	return c.MustResolve(ServiceCatalogVariants).(catalogVariants.Service)
}

func ResolveCatalogDiscounts(c *Container) catalogDiscounts.Service {
	return c.MustResolve(ServiceCatalogDiscounts).(catalogDiscounts.Service)
}

func ResolveCatalogBulk(c *Container) catalogBulk.Service {
	return c.MustResolve(ServiceCatalogBulk).(catalogBulk.Service)
}

func ResolveCatalogReviews(c *Container) catalogReviews.Service {
	return c.MustResolve(ServiceCatalogReviews).(catalogReviews.Service)
}

func ResolveInventoryCore(c *Container) inventoryCore.Service {
	return c.MustResolve(ServiceInventoryCore).(inventoryCore.Service)
}

func ResolveInventoryBulk(c *Container) inventoryBulk.Service {
	return c.MustResolve(ServiceInventoryBulk).(inventoryBulk.Service)
}

func ResolveInventoryHistory(c *Container) inventoryHistory.Service {
	return c.MustResolve(ServiceInventoryHistory).(inventoryHistory.Service)
}

func ResolveInventoryReservation(c *Container) inventoryReservation.Service {
	return c.MustResolve(ServiceInventoryReservation).(inventoryReservation.Service)
}

func ResolveInventoryReports(c *Container) inventoryReports.Service {
	return c.MustResolve(ServiceInventoryReports).(inventoryReports.Service)
}

func ResolveInventoryTransfer(c *Container) inventoryTransfer.Service {
	return c.MustResolve(ServiceInventoryTransfer).(inventoryTransfer.Service)
}

func ResolveInventoryAdjustment(c *Container) inventoryAdjustment.Service {
	return c.MustResolve(ServiceInventoryAdjustment).(inventoryAdjustment.Service)
}

func ResolveInventorySync(c *Container) inventorySync.Service {
	return c.MustResolve(ServiceInventorySync).(inventorySync.Service)
}

func ResolveCart(c *Container) cart.Service {
	return c.MustResolve(ServiceCart).(cart.Service)
}

func ResolveCartMerge(c *Container) cartMerge.Service {
	return c.MustResolve(ServiceCartMerge).(cartMerge.Service)
}

func ResolveOrders(c *Container) orders.Service {
	return c.MustResolve(ServiceOrders).(orders.Service)
}

func ResolvePayments(c *Container) payments.Service {
	return c.MustResolve(ServicePayments).(payments.Service)
}

func ResolveShipping(c *Container) shipping.Service {
	return c.MustResolve(ServiceShipping).(shipping.Service)
}
