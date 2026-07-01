// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package paymentswiring

import (
	"context"
	"fmt"

	"ecom-engine/internal/core/payments"
	paymentsAdmin "ecom-engine/internal/core/payments/admin"
	"ecom-engine/internal/engine"
	modulespayments "ecom-engine/internal/modules/payments"
	"ecom-engine/internal/modules/payments/payhere"
	"ecom-engine/internal/modules/payments/paypal"
	"ecom-engine/internal/modules/payments/stripe"

	"github.com/gin-gonic/gin"
)

// Module wires the payments domain. Only the configured provider is instantiated.
// Depends on orders for amount/status fetching.
type Module struct {
	cfg    engine.PaymentsModuleConfig
	authMW gin.HandlerFunc
	svc    payments.Service
}

func New(cfg engine.Config) engine.Module {
	return &Module{cfg: cfg.Modules.Payments}
}

func (m *Module) Name() string        { return "payments" }
func (m *Module) Requires() []string  { return []string{"orders"} }
func (m *Module) BasePaths() []string { return []string{"/payments"} }

func (m *Module) Init(c *engine.Container) error {
	m.authMW = c.AuthMiddleware

	orderSvc := engine.ResolveOrders(c)

	provider, err := buildProvider(m.cfg.Provider, m.cfg.APIKey)
	if err != nil {
		return fmt.Errorf("payments module: %w", err)
	}
	providers := map[string]modulespayments.PaymentProvider{
		m.cfg.Provider: provider,
	}

	svc, err := payments.NewPaymentService(
		c.Repos.PaymentRepo(),
		&orderAdapter{svc: orderSvc},
		providers,
		m.cfg.Provider,
		c.EventBus,
		c.Outbox,
		c.TxManager,
	)
	if err != nil {
		return fmt.Errorf("payments module: %w", err)
	}
	m.svc = svc
	c.Provide(engine.ServicePayments, m.svc)
	return nil
}

func (m *Module) RegisterRoutes(public, _, admin *gin.RouterGroup) {
	payments.RegisterRoutes(public, payments.NewController(m.svc), m.authMW)
	paymentsAdmin.RegisterAdminRoutes(admin, paymentsAdmin.NewController(m.svc))
}

func (m *Module) Shutdown(_ context.Context) error { return nil }

// buildProvider instantiates only the configured payment provider.
func buildProvider(name, apiKey string) (modulespayments.PaymentProvider, error) {
	switch name {
	case "stripe":
		return stripe.NewStripeProvider(apiKey), nil
	case "paypal":
		return paypal.NewPayPalProvider(apiKey), nil
	case "payhere":
		return payhere.NewPayHereProvider(apiKey), nil
	default:
		return nil, fmt.Errorf("unknown payment provider %q; supported: stripe, paypal, payhere", name)
	}
}
