// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package inventory manages inventory operations.
package inventory

import (
	"ecom-engine/internal/core/inventory/features/adjustment"
	"ecom-engine/internal/core/inventory/features/bulk"
	"ecom-engine/internal/core/inventory/features/core"
	"ecom-engine/internal/core/inventory/features/history"
	"ecom-engine/internal/core/inventory/features/reports"
	"ecom-engine/internal/core/inventory/features/reservation"
	"ecom-engine/internal/core/inventory/features/sync"
	"ecom-engine/internal/core/inventory/features/transfer"
)

// ModuleServices holds the services for all inventory features.
type ModuleServices struct {
	Core        core.Service
	Bulk        bulk.Service
	History     history.Service
	Reservation reservation.Service
	Reports     reports.Service
	Transfer    transfer.Service
	Adjustment  adjustment.Service
	Sync        sync.Service
}
