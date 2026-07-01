// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsModuleEnabled(t *testing.T) {
	t.Parallel()

	cfg := Config{}
	cfg.Modules.Catalog.Enabled = true
	cfg.Modules.Inventory.Enabled = false
	cfg.Modules.Cart.Enabled = true
	cfg.Modules.Orders.Enabled = false
	cfg.Modules.Payments.Enabled = true
	cfg.Modules.Shipping.Enabled = false
	cfg.Modules.Notifications.Enabled = true

	t.Run("catalog enabled", func(t *testing.T) {
		assert.True(t, IsModuleEnabled(cfg, "catalog"))
	})

	t.Run("inventory disabled", func(t *testing.T) {
		assert.False(t, IsModuleEnabled(cfg, "inventory"))
	})

	t.Run("cart enabled", func(t *testing.T) {
		assert.True(t, IsModuleEnabled(cfg, "cart"))
	})

	t.Run("orders disabled", func(t *testing.T) {
		assert.False(t, IsModuleEnabled(cfg, "orders"))
	})

	t.Run("payments enabled", func(t *testing.T) {
		assert.True(t, IsModuleEnabled(cfg, "payments"))
	})

	t.Run("shipping disabled", func(t *testing.T) {
		assert.False(t, IsModuleEnabled(cfg, "shipping"))
	})

	t.Run("notifications enabled", func(t *testing.T) {
		assert.True(t, IsModuleEnabled(cfg, "notifications"))
	})

	t.Run("unknown module returns false", func(t *testing.T) {
		assert.False(t, IsModuleEnabled(cfg, "unknown_module"))
	})
}
