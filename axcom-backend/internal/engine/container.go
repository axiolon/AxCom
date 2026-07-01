// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"ecom-engine/internal/core/auth"
	"ecom-engine/internal/events"
	"ecom-engine/internal/infra/cache"
	infradb "ecom-engine/internal/infra/db"
	"ecom-engine/internal/infra/storage"
	pkgdb "ecom-engine/pkg/db"
	"ecom-engine/pkg/token"

	"github.com/gin-gonic/gin"
)

// Container holds all shared infrastructure and cross-module services.
// It is built once during engine bootstrap and passed to every module's Init().
//
// Infrastructure fields (DB, cache, events, storage) are always populated.
// Auth fields are always populated — auth is infrastructure, not a module.
// Module services are populated lazily as modules initialize, via Provide().
type Container struct {
	// --- Infrastructure (always available) ---

	// Config is the fully-loaded application configuration.
	Config Config

	// EventBus is the application-wide event pub/sub system.
	EventBus events.EventBus

	// Cache is the L2 backend (Redis or memory) used for direct cache ops.
	Cache cache.Cache

	// CacheManager provides the two-layer (L1 memory + L2 Redis) cache facade.
	CacheManager cache.Manager

	// DBConn is the raw database connection, used for health checks.
	DBConn pkgdb.Connection

	// TxManager coordinates database transactions across repo calls.
	TxManager infradb.TransactionManager

	// FileStorage is the pluggable file storage backend (local, S3, R2).
	FileStorage storage.FileStorage

	// Repos is the DB-agnostic repository factory. Modules call its typed
	// methods (e.g. Repos.CatalogCoreRepo()) to receive the correct
	// implementation for the configured database without importing mongo or
	// postgres packages directly.
	Repos *RepoProvider

	// --- Auth (always available — auth is infrastructure, not a module) ---

	// AuthService handles user registration, login, and token management.
	AuthService auth.Service

	// JWTManager signs and validates JWT tokens.
	JWTManager *token.JWTManager

	// OIDCValidator validates external OIDC/JWT tokens (nil when mode is "local").
	OIDCValidator *token.OIDCValidator

	// AuthMiddleware is the Gin handler that enforces authentication on
	// "secured" route groups. Set by the engine during bootstrap.
	AuthMiddleware gin.HandlerFunc

	// AdminMiddleware is the Gin handler that enforces admin-only access on
	// "admin" route groups. Set by the engine during bootstrap.
	AdminMiddleware gin.HandlerFunc

	// --- Outbox (nil when outbox is disabled) ---

	// Outbox is the transactional outbox repository for persisting events alongside business data.
	Outbox events.OutboxRepository

	// DedupStore provides consumer-side idempotency checks.
	DedupStore events.DedupStore

	// --- Module services (populated during module Init via Provide) ---
	services map[string]any
}

// Provide registers a service by key so other modules can consume it.
// Called by modules inside their Init() implementation.
// Panics if the same key is registered twice (indicates a wiring bug).
func (c *Container) Provide(key string, svc any) {
	if c.services == nil {
		c.services = make(map[string]any)
	}
	if _, exists := c.services[key]; exists {
		panic("engine: duplicate service registration for key: " + key)
	}
	c.services[key] = svc
}

// Resolve retrieves a service by key. Returns (nil, false) if not registered.
// Use this when the dependency is optional.
func (c *Container) Resolve(key string) (any, bool) {
	svc, ok := c.services[key]
	return svc, ok
}

// MustResolve retrieves a service by key and panics if not found.
// Safe to call when Requires() guarantees the dependency is initialized.
func (c *Container) MustResolve(key string) any {
	svc, ok := c.services[key]
	if !ok {
		panic("engine: required service not registered: " + key)
	}
	return svc
}
