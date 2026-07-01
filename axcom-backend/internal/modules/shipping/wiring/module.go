// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package shippingwiring

import (
	"context"
	"fmt"

	"ecom-engine/internal/core/orders"
	"ecom-engine/internal/core/shipping"
	shippingAdmin "ecom-engine/internal/core/shipping/admin"
	"ecom-engine/internal/engine"
	modulesshipping "ecom-engine/internal/modules/shipping"
	"ecom-engine/internal/modules/shipping/flatrate"
	"ecom-engine/internal/modules/shipping/freeabove"
	"ecom-engine/internal/modules/shipping/weightbased"

	"github.com/gin-gonic/gin"
)

// Module wires the shipping domain. Depends on orders.
// Each provider in config is instantiated and registered.
type Module struct {
	cfg      engine.ShippingModuleConfig
	authMW   gin.HandlerFunc
	svc      shipping.Service
	orderSvc orders.Service
}

func New(cfg engine.Config) engine.Module {
	return &Module{cfg: cfg.Modules.Shipping}
}

func (m *Module) Name() string        { return "shipping" }
func (m *Module) Requires() []string  { return []string{"orders"} }
func (m *Module) BasePaths() []string { return []string{"/shipments", "/shipping"} }

func (m *Module) Init(c *engine.Container) error {
	m.authMW = c.AuthMiddleware
	m.orderSvc = engine.ResolveOrders(c)

	providers, err := buildProviders(m.cfg.Providers)
	if err != nil {
		return fmt.Errorf("shipping module: %w", err)
	}

	m.svc = shipping.NewShipmentService(
		c.Repos.ShipmentRepo(),
		providers,
		c.EventBus,
		c.TxManager,
		c.Outbox,
	)
	c.Provide(engine.ServiceShipping, m.svc)
	return nil
}

func (m *Module) RegisterRoutes(public, _, admin *gin.RouterGroup) {
	shipping.RegisterRoutes(public, shipping.NewController(m.svc, &orderAdapter{svc: m.orderSvc}), m.authMW)
	shippingAdmin.RegisterAdminRoutes(admin, shippingAdmin.NewController(m.svc))
}

func (m *Module) Shutdown(_ context.Context) error { return nil }

// buildProviders creates provider instances from the config slice.
func buildProviders(cfgs []engine.ShippingProviderConfig) ([]modulesshipping.ShippingProvider, error) {
	providers := make([]modulesshipping.ShippingProvider, 0, len(cfgs))
	for _, p := range cfgs {
		switch p.Type {
		case "flatrate":
			providers = append(providers, flatrate.NewFlatRateProvider(p.Rate))
		case "freeabove":
			providers = append(providers, freeabove.NewFreeAboveProvider(p.Threshold, p.BaseRate))
		case "weightbased":
			providers = append(providers, weightbased.NewWeightBasedProvider(p.BaseRate, p.PerKg))
		default:
			return nil, fmt.Errorf("unknown shipping provider type %q; supported: flatrate, freeabove, weightbased", p.Type)
		}
	}
	return providers, nil
}
