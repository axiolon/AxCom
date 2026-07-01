// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package orders

import (
	"context"

	"ecom-engine/internal/core/orders"
	ordersAdmin "ecom-engine/internal/core/orders/admin"
	ordersGuest "ecom-engine/internal/core/orders/guest"
	ordersUser "ecom-engine/internal/core/orders/user"
	"ecom-engine/internal/engine"

	"github.com/gin-gonic/gin"
)

// Module wires the orders domain. No optional features at this time.
type Module struct {
	svc orders.Service
}

func New(_ engine.Config) engine.Module {
	return &Module{}
}

func (m *Module) Name() string        { return "orders" }
func (m *Module) Requires() []string  { return []string{"catalog"} }
func (m *Module) BasePaths() []string { return []string{"/orders"} }

func (m *Module) Init(c *engine.Container) error {
	m.svc = orders.NewOrderService(c.Repos.OrderRepo(), c.EventBus, c.Outbox, c.TxManager)
	c.Provide(engine.ServiceOrders, m.svc)
	return nil
}

func (m *Module) RegisterRoutes(public, secured, admin *gin.RouterGroup) {
	ordersGuest.RegisterGuestRoutes(public, m.svc)
	ordersUser.RegisterUserRoutes(secured, m.svc)
	ordersAdmin.RegisterAdminRoutes(admin, m.svc)
}

func (m *Module) Shutdown(_ context.Context) error { return nil }
