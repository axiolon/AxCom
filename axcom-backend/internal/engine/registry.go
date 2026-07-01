// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package engine

// IsModuleEnabled reports whether the named module is enabled in cfg.
// Used by internal/modules/registry to build the active/disabled module slices.
func IsModuleEnabled(cfg Config, name string) bool {
	switch name {
	case "catalog":
		return cfg.Modules.Catalog.Enabled
	case "inventory":
		return cfg.Modules.Inventory.Enabled
	case "cart":
		return cfg.Modules.Cart.Enabled
	case "orders":
		return cfg.Modules.Orders.Enabled
	case "payments":
		return cfg.Modules.Payments.Enabled
	case "shipping":
		return cfg.Modules.Shipping.Enabled
	case "notifications":
		return cfg.Modules.Notifications.Enabled
	case "dashboard":
		return cfg.Modules.Dashboard.Enabled
	}
	return false
}
