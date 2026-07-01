// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package inventory manages inventory operations.
package inventory

import (
	"ecom-engine/internal/core/inventory/domain"
)

// StockItem represents the stock quantity of a specific product variant.
// It is type-aliased to domain.StockItem to expose the domain model at the package boundary.
type StockItem = domain.StockItem

// Reservation represents a temporary hold on stock for checkout process.
// It is type-aliased to domain.Reservation to expose the domain model at the package boundary.
type Reservation = domain.Reservation

// Alert represents an inventory alert (e.g. low stock).
// It is type-aliased to domain.Alert to expose the domain model at the package boundary.
type Alert = domain.Alert

// StockHistory represents a change in stock quantity.
// It is type-aliased to domain.StockHistory to expose the domain model at the package boundary.
type StockHistory = domain.StockHistory
