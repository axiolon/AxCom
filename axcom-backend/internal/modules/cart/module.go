// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cart

import (
	"context"

	cartCore "ecom-engine/internal/core/cart"
	cartMerge "ecom-engine/internal/core/cart/merge"
	"ecom-engine/internal/engine"

	"github.com/gin-gonic/gin"
)

// Module wires the cart domain. Depends on catalog for product validation.
type Module struct {
	svc      cartCore.Service
	mergeSvc cartMerge.Service
}

func New(_ engine.Config) engine.Module {
	return &Module{}
}

func (m *Module) Name() string        { return "cart" }
func (m *Module) Requires() []string  { return []string{"catalog"} }
func (m *Module) BasePaths() []string { return []string{"/cart"} }

func (m *Module) Init(c *engine.Container) error {
	catalogQuery := engine.ResolveCatalogQuery(c)

	cartRepo := c.Repos.CartRepo()
	m.svc = cartCore.NewCartService(cartRepo, catalogQuery)
	m.mergeSvc = cartMerge.NewMergeService(m.svc, cartRepo)

	c.Provide(engine.ServiceCart, m.svc)
	c.Provide(engine.ServiceCartMerge, m.mergeSvc)
	return nil
}

func (m *Module) RegisterRoutes(_, secured, _ *gin.RouterGroup) {
	cartCore.RegisterRoutes(secured, cartCore.NewController(m.svc))
	cartMerge.RegisterRoutes(secured, cartMerge.NewController(m.mergeSvc))
}

func (m *Module) Shutdown(_ context.Context) error { return nil }
