// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package migrate orchestrates module-scoped database migrations and integrity
// checks for both Postgres and MongoDB. Each module owns its own migration
// directory; only enabled modules are migrated.
package migrate

import (
	"context"
	"fmt"
	"path/filepath"

	"ecom-engine/pkg/logger"
)

// Config holds the configuration fields required to plan and execute migrations.
type Config struct {
	ConnectionString string
	Database         string
	CatalogEnabled   bool
	OrdersEnabled    bool
	InventoryEnabled bool
	PaymentsEnabled  bool
	ShippingEnabled  bool
	CartEnabled      bool
	OutboxEnabled    bool
}

// ModuleEntry describes a single migratable module: its key name and whether
// it is currently enabled in the loaded config.
type ModuleEntry struct {
	Key     string
	Enabled bool
}

// Plan resolves which module migration directories to run based on the config.
// Always includes "core" and conditionally includes "events" (outbox).
// Returns entries in the order migrations should be applied.
func Plan(cfg Config) []ModuleEntry {
	entries := []ModuleEntry{
		{Key: "core", Enabled: true},
	}

	entries = append(entries, ModuleEntry{Key: "catalog", Enabled: cfg.CatalogEnabled})
	entries = append(entries, ModuleEntry{Key: "orders", Enabled: cfg.OrdersEnabled})
	entries = append(entries, ModuleEntry{Key: "inventory", Enabled: cfg.InventoryEnabled})
	entries = append(entries, ModuleEntry{Key: "payments", Enabled: cfg.PaymentsEnabled})
	entries = append(entries, ModuleEntry{Key: "shipping", Enabled: cfg.ShippingEnabled})
	entries = append(entries, ModuleEntry{Key: "cart", Enabled: cfg.CartEnabled})
	entries = append(entries, ModuleEntry{Key: "events", Enabled: cfg.OutboxEnabled})

	return entries
}

// RunPostgresUp runs all pending up migrations for enabled modules.
// migrationsRoot should be the path to migrations/postgres.
func RunPostgresUp(ctx context.Context, cfg Config, migrationsRoot string) error {
	pg, err := newPostgresMigrator(ctx, cfg.ConnectionString)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pg.close()

	if err := pg.ensureTrackingTable(ctx); err != nil {
		return fmt.Errorf("tracking table: %w", err)
	}

	for _, entry := range Plan(cfg) {
		if !entry.Enabled {
			logger.Info("[migrate] skip  module=%s (disabled)", entry.Key)
			continue
		}
		dir := filepath.Join(migrationsRoot, entry.Key)
		if err := pg.migrateUp(ctx, entry.Key, dir); err != nil {
			return fmt.Errorf("module %q up: %w", entry.Key, err)
		}
	}
	return nil
}

// RunPostgresDown rolls back the latest migration version for the given module.
func RunPostgresDown(ctx context.Context, cfg Config, migrationsRoot, module string) error {
	pg, err := newPostgresMigrator(ctx, cfg.ConnectionString)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pg.close()

	dir := filepath.Join(migrationsRoot, module)
	return pg.migrateDown(ctx, module, dir)
}

// RunPostgresStatus prints the applied version for each module.
func RunPostgresStatus(ctx context.Context, cfg Config, migrationsRoot string) error {
	pg, err := newPostgresMigrator(ctx, cfg.ConnectionString)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pg.close()

	return pg.status(ctx, Plan(cfg), migrationsRoot)
}

// RunMongoSeed ensures all collections and indexes exist for enabled modules.
// migrationsRoot should be the path to migrations/mongodb.
func RunMongoSeed(ctx context.Context, cfg Config, migrationsRoot string) error {
	mg, err := newMongoMigrator(ctx, cfg.ConnectionString, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer mg.close()

	for _, entry := range Plan(cfg) {
		if !entry.Enabled {
			logger.Info("[migrate] skip  module=%s (disabled)", entry.Key)
			continue
		}
		dir := filepath.Join(migrationsRoot, entry.Key)
		if err := mg.seed(ctx, entry.Key, dir); err != nil {
			return fmt.Errorf("module %q seed: %w", entry.Key, err)
		}
	}
	return nil
}

// VerifyPostgres checks that all expected tables and indexes exist for enabled modules.
func VerifyPostgres(ctx context.Context, cfg Config) error {
	pg, err := newPostgresMigrator(ctx, cfg.ConnectionString)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pg.close()

	return verifyPostgres(ctx, pg, Plan(cfg))
}

// VerifyMongo checks that all expected collections and indexes exist for enabled modules.
func VerifyMongo(ctx context.Context, cfg Config) error {
	mg, err := newMongoMigrator(ctx, cfg.ConnectionString, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer mg.close()

	return verifyMongo(ctx, mg, Plan(cfg))
}

// QuickCheck is a fast startup check — verifies the schema_migrations tracking
// table exists and that core has been applied. Used by engine.go.
func QuickCheck(ctx context.Context, connStr string) error {
	pg, err := newPostgresMigrator(ctx, connStr)
	if err != nil {
		return err
	}
	defer pg.close()
	return pg.quickCheck(ctx)
}
