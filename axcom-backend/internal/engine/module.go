// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package engine provides core business logic abstractions, system initialization, and configuration parsing.
package engine

import (
	"context"

	"github.com/gin-gonic/gin"
)

// Module is the contract every engine module must satisfy.
// Each module is self-contained: it declares its dependencies, wires its own
// repos and services during Init, registers its own HTTP routes, and cleans
// up its own resources on Shutdown.
//
// To add a new module:
//  1. Create internal/modules/<name>/module.go implementing this interface.
//  2. Add its config struct to ModulesConfig in config.go.
//  3. Add one line to defaultRegistry in registry.go.
//  4. Enable it in config.yaml under modules.<name>.enabled: true.
type Module interface {
	// Name returns a unique identifier used in dependency declarations and
	// error messages (e.g. "catalog", "inventory", "payments").
	Name() string

	// Requires returns the Names of modules that must be initialized before
	// this one. The engine validates the dependency graph at startup and
	// returns a clear error if a required module is disabled.
	// Example: []string{"catalog"} for the cart module.
	Requires() []string

	// BasePaths returns the URL path prefixes owned by this module.
	// Used by the router to return helpful "module not enabled" JSON errors
	// when a disabled module's endpoints are called.
	// Example: []string{"/products", "/categories"} for catalog.
	BasePaths() []string

	// Init receives the shared Container and wires the module's internal
	// repos and services. It is called after all declared dependencies have
	// been initialized (topological order). Modules export services to other
	// modules by calling container.Provide(ServiceKey, svc).
	Init(c *Container) error

	// RegisterRoutes mounts the module's HTTP handlers onto the router groups.
	//   public  — /api (rate-limited, no authentication required)
	//   secured — /api + auth middleware
	//   admin   — /api + auth middleware + admin role check
	RegisterRoutes(public, secured, admin *gin.RouterGroup)

	// Shutdown gracefully releases module-owned resources (e.g. background
	// workers, open connections). Called in reverse-init order during engine
	// shutdown so that dependents are torn down before their dependencies.
	Shutdown(ctx context.Context) error
}

// DisabledModuleInfo holds the identity and route prefixes of a module that
// was present in the registry but disabled via config. The router uses this
// to return helpful errors instead of generic 404s.
type DisabledModuleInfo struct {
	Name      string
	BasePaths []string
}
