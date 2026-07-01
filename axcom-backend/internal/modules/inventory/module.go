// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"context"

	inventoryAdjustment "ecom-engine/internal/core/inventory/features/adjustment"
	inventoryBulk "ecom-engine/internal/core/inventory/features/bulk"
	inventoryCore "ecom-engine/internal/core/inventory/features/core"
	inventoryHistory "ecom-engine/internal/core/inventory/features/history"
	inventoryReports "ecom-engine/internal/core/inventory/features/reports"
	inventoryReservation "ecom-engine/internal/core/inventory/features/reservation"
	inventorySync "ecom-engine/internal/core/inventory/features/sync"
	inventoryTransfer "ecom-engine/internal/core/inventory/features/transfer"
	"ecom-engine/internal/engine"

	"github.com/gin-gonic/gin"
)

// Module wires the inventory domain. Core (stock CRUD + alerts) always runs.
// Bulk, history, reservation, reports, transfer, adjustment, and sync are optional.
type Module struct {
	cfg     engine.InventoryModuleConfig
	authMW  gin.HandlerFunc
	adminMW gin.HandlerFunc

	coreSvc inventoryCore.Service

	bulkSvc        inventoryBulk.Service
	historySvc     inventoryHistory.Service
	reservationSvc inventoryReservation.Service
	reportsSvc     inventoryReports.Service
	transferSvc    inventoryTransfer.Service
	adjustmentSvc  inventoryAdjustment.Service
	syncSvc        inventorySync.Service
}

func New(cfg engine.Config) engine.Module {
	return &Module{cfg: cfg.Modules.Inventory}
}

func (m *Module) Name() string       { return "inventory" }
func (m *Module) Requires() []string { return nil }
func (m *Module) BasePaths() []string {
	return []string{"/inventory", "/stock"}
}

func (m *Module) Init(c *engine.Container) error {
	m.authMW = c.AuthMiddleware
	m.adminMW = c.AdminMiddleware

	// Core — always initialized
	coreRepo := c.Repos.InventoryCoreRepo()
	alertDispatcher := inventoryCore.NewDashboardAlertDispatcher(coreRepo)
	m.coreSvc = inventoryCore.NewService(coreRepo, alertDispatcher)
	c.Provide(engine.ServiceInventoryCore, m.coreSvc)

	// Optional features
	if m.cfg.Features.Bulk {
		m.bulkSvc = inventoryBulk.NewService(c.Repos.InventoryBulkRepo())
		c.Provide(engine.ServiceInventoryBulk, m.bulkSvc)
	}
	if m.cfg.Features.History {
		m.historySvc = inventoryHistory.NewService(c.Repos.InventoryHistoryRepo(), c.EventBus)
		c.Provide(engine.ServiceInventoryHistory, m.historySvc)
	}
	if m.cfg.Features.Reservation {
		m.reservationSvc = inventoryReservation.NewService(c.Repos.InventoryReservationRepo(), c.EventBus, c.Outbox)
		c.Provide(engine.ServiceInventoryReservation, m.reservationSvc)
	}
	if m.cfg.Features.Reports {
		m.reportsSvc = inventoryReports.NewService(c.Repos.InventoryReportsRepo())
		c.Provide(engine.ServiceInventoryReports, m.reportsSvc)
	}
	if m.cfg.Features.Transfer {
		m.transferSvc = inventoryTransfer.NewService(c.Repos.InventoryTransferRepo(), c.EventBus, c.Outbox)
		c.Provide(engine.ServiceInventoryTransfer, m.transferSvc)
	}
	if m.cfg.Features.Adjustment {
		m.adjustmentSvc = inventoryAdjustment.NewService(c.Repos.InventoryAdjustmentRepo(), c.EventBus, c.Outbox)
		c.Provide(engine.ServiceInventoryAdjustment, m.adjustmentSvc)
	}
	if m.cfg.Features.Sync {
		m.syncSvc = inventorySync.NewService(c.Repos.InventorySyncRepo(), c.EventBus, c.Outbox)
		c.Provide(engine.ServiceInventorySync, m.syncSvc)
	}

	return nil
}

func (m *Module) RegisterRoutes(public, secured, _ *gin.RouterGroup) {
	inventoryCore.RegisterRoutes(public, inventoryCore.NewController(m.coreSvc), m.authMW, m.adminMW)

	if m.bulkSvc != nil {
		inventoryBulk.RegisterRoutes(public, inventoryBulk.NewController(m.bulkSvc), m.authMW)
	}
	if m.historySvc != nil {
		inventoryHistory.RegisterRoutes(public, inventoryHistory.NewController(m.historySvc), m.authMW)
	}
	if m.reservationSvc != nil {
		inventoryReservation.RegisterRoutes(public, inventoryReservation.NewController(m.reservationSvc), m.authMW)
	}
	if m.reportsSvc != nil {
		inventoryReports.RegisterRoutes(public, inventoryReports.NewController(m.reportsSvc), m.authMW)
	}
	if m.transferSvc != nil {
		inventoryTransfer.RegisterRoutes(public, inventoryTransfer.NewController(m.transferSvc), m.authMW)
	}
	if m.adjustmentSvc != nil {
		inventoryAdjustment.RegisterRoutes(public, inventoryAdjustment.NewController(m.adjustmentSvc), m.authMW)
	}
	if m.syncSvc != nil {
		inventorySync.RegisterRoutes(public, inventorySync.NewController(m.syncSvc), m.authMW)
	}

	_ = secured // inventory routes use inline auth, not the secured group directly
}

func (m *Module) Shutdown(_ context.Context) error { return nil }
