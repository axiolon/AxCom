// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package dashboard wires the admin dashboard module into the engine.
package dashboard

import (
	"context"

	"ecom-engine/internal/core/dashboard"
	"ecom-engine/internal/engine"

	"github.com/gin-gonic/gin"
)

// Module wires the dashboard domain into the engine.
type Module struct {
	handler *dashboard.Handler
}

// New constructs the dashboard module. cfg is unused here; tier selection
// happens in Init once the Container (and its repos/cache) is available.
func New(_ engine.Config) engine.Module {
	return &Module{}
}

func (m *Module) Name() string        { return "dashboard" }
func (m *Module) Requires() []string  { return []string{"orders", "inventory"} }
func (m *Module) BasePaths() []string { return []string{"/admin/dashboard"} }

func (m *Module) Init(c *engine.Container) error {
	cfg := c.Config.Modules.Dashboard
	orderRepo := c.Repos.OrderRepo()
	invRepo := c.Repos.InventoryCoreRepo()

	var svc dashboard.Service
	switch cfg.Tier {
	case "medium":
		svc = dashboard.NewMediumService(orderRepo, invRepo, c.Cache, cfg.CacheTTL)
	default:
		svc = dashboard.NewSmallService(orderRepo, invRepo)
	}

	m.handler = dashboard.NewHandler(svc)
	return nil
}

// RegisterRoutes mounts the dashboard on the admin group only.
func (m *Module) RegisterRoutes(_, _, adminGroup *gin.RouterGroup) {
	dashboard.RegisterRoutes(adminGroup, m.handler)
}

func (m *Module) Shutdown(_ context.Context) error { return nil }
