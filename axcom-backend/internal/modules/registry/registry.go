// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package registry maps module names to their factories and collects
// enabled/disabled module instances from the global config.
//
// It lives outside the engine package to avoid an import cycle:
//
//	engine → modules/* → engine
//
// Bootstrap flow:
//
//	active, disabled := registry.Collect(cfg)
//	eng, err := engine.NewEngine(cfg, active, disabled)
package registry

import (
	"ecom-engine/internal/engine"
	modulescart "ecom-engine/internal/modules/cart"
	modulescatalog "ecom-engine/internal/modules/catalog"
	modulesdashboard "ecom-engine/internal/modules/dashboard"
	modulesinventory "ecom-engine/internal/modules/inventory"
	modulesnotifications "ecom-engine/internal/modules/notifications"
	modulesorders "ecom-engine/internal/modules/orders"
	paymentswiring "ecom-engine/internal/modules/payments/wiring"
	shippingwiring "ecom-engine/internal/modules/shipping/wiring"
)

// factories maps every known module name to its constructor.
// To add a module: implement engine.Module + add one line here.
var factories = map[string]func(engine.Config) engine.Module{
	"catalog":       modulescatalog.New,
	"inventory":     modulesinventory.New,
	"cart":          modulescart.New,
	"orders":        modulesorders.New,
	"payments":      paymentswiring.New,
	"shipping":      shippingwiring.New,
	"notifications": modulesnotifications.New,
	"dashboard":     modulesdashboard.New,
}

// Collect partitions the registered modules into enabled (active) and disabled
// slices based on the supplied config. The caller passes these to engine.NewEngine.
func Collect(cfg engine.Config) (active []engine.Module, disabled []engine.DisabledModuleInfo) {
	for name, factory := range factories {
		if engine.IsModuleEnabled(cfg, name) {
			active = append(active, factory(cfg))
		} else {
			m := factory(cfg)
			disabled = append(disabled, engine.DisabledModuleInfo{
				Name:      m.Name(),
				BasePaths: m.BasePaths(),
			})
		}
	}
	return active, disabled
}
