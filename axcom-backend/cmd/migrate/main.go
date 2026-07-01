// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Command migrate manages module-scoped database schema migrations and integrity
// checks for both Postgres and MongoDB.
//
// Usage:
//
//	go run ./cmd/migrate <command> [flags]
//
// Commands:
//
//	up                    Apply all pending migrations for enabled modules
//	down --module <name>  Roll back latest migration for a specific module
//	status                Show per-module migration version (Postgres only)
//	verify                Run integrity checks for enabled modules
//	seed                  Ensure MongoDB collections and indexes exist
//
// Flags:
//
//	--module   Target a specific module (used with down)
//	--config   Path to config YAML file (defaults to engine.LoadConfig discovery)
//	--root     Root directory of migrations (default: migrations)
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ecom-engine/internal/engine"
	"ecom-engine/internal/migrate"
	"ecom-engine/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		logger.Error("%v", err)
		os.Exit(1)
	}
}

func run() error {
	// Initialize separate flag sets for each migration subcommand to isolate their arguments.
	upCmd := flag.NewFlagSet("up", flag.ExitOnError)
	downCmd := flag.NewFlagSet("down", flag.ExitOnError)
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	verifyCmd := flag.NewFlagSet("verify", flag.ExitOnError)
	seedCmd := flag.NewFlagSet("seed", flag.ExitOnError)

	// Register shared/common flags across all standard subcommands.
	var rootFlag string
	for _, fs := range []*flag.FlagSet{upCmd, downCmd, statusCmd, verifyCmd, seedCmd} {
		fs.String("config", "", "Path to config YAML file (overrides APP_CONFIG env var)")
		fs.StringVar(&rootFlag, "root", "migrations", "Root directory of migration files")
	}

	// Register the module-specific flag, which is mandatory and unique to the 'down' command.
	downModule := downCmd.String("module", "", "Module to roll back (required)")

	// Enforce that at least one subcommand argument is provided.
	if len(os.Args) < 2 {
		usage()
		return fmt.Errorf("missing command")
	}

	// Load configuration files depending on the active environment (e.g. dev, prod).
	// Fall back to default .env if environment-specific dotenv is not found.
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}
	_ = godotenv.Load(".env." + env)
	_ = godotenv.Load()

	// Route execution based on the first positional argument.
	switch os.Args[1] {
	case "up":
		// Parse flags, load application config, and run migrations up for Postgres.
		_ = upCmd.Parse(os.Args[2:])
		cfg, err := loadConfig(upCmd)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		pgRoot := filepath.Join(rootFlag, "postgres")
		switch cfg.DB.Type {
		case "postgres":
			if err := migrate.RunPostgresUp(ctx, toMigrateConfig(cfg), pgRoot); err != nil {
				return fmt.Errorf("migrate up failed: %w", err)
			}
			logger.Info("All pending migrations applied successfully.")
		case "mongodb":
			// MongoDB uses collections/indexes seeding instead of SQL schema migrations.
			logger.Info("MongoDB does not use SQL migrations. Use 'seed' instead.")
			return fmt.Errorf("cannot run up migrations on MongoDB")
		}

	case "down":
		// Roll back the latest migration for a single module.
		_ = downCmd.Parse(os.Args[2:])
		if *downModule == "" {
			return fmt.Errorf("--module is required for 'down'")
		}
		cfg, err := loadConfig(downCmd)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		pgRoot := filepath.Join(rootFlag, "postgres")
		if err := migrate.RunPostgresDown(ctx, toMigrateConfig(cfg), pgRoot, *downModule); err != nil {
			return fmt.Errorf("migrate down failed: %w", err)
		}

	case "status":
		// Retrieve and print the migration status (current version, pending runs) for Postgres modules.
		_ = statusCmd.Parse(os.Args[2:])
		cfg, err := loadConfig(statusCmd)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pgRoot := filepath.Join(rootFlag, "postgres")
		if err := migrate.RunPostgresStatus(ctx, toMigrateConfig(cfg), pgRoot); err != nil {
			return fmt.Errorf("status failed: %w", err)
		}

	case "verify":
		// Perform integrity/schema consistency verification checks against the target database.
		_ = verifyCmd.Parse(os.Args[2:])
		cfg, err := loadConfig(verifyCmd)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		var verifyErr error
		switch cfg.DB.Type {
		case "postgres":
			verifyErr = migrate.VerifyPostgres(ctx, toMigrateConfig(cfg))
		case "mongodb":
			verifyErr = migrate.VerifyMongo(ctx, toMigrateConfig(cfg))
		}
		if verifyErr != nil {
			return verifyErr
		}

	case "seed":
		// Ensure all MongoDB collections, schemas, and indexes are created.
		_ = seedCmd.Parse(os.Args[2:])
		cfg, err := loadConfig(seedCmd)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if cfg.DB.Type != "mongodb" {
			return fmt.Errorf("'seed' is only for MongoDB — use 'up' for Postgres")
		}
		mgRoot := filepath.Join(rootFlag, "mongodb")
		if err := migrate.RunMongoSeed(ctx, toMigrateConfig(cfg), mgRoot); err != nil {
			return fmt.Errorf("seed failed: %w", err)
		}
		logger.Info("MongoDB seed completed successfully.")

	default:
		usage()
		return fmt.Errorf("unknown command %q", os.Args[1])
	}
	return nil
}

// toMigrateConfig maps the global application config to the migration-specific config layout.
func toMigrateConfig(cfg engine.Config) migrate.Config {
	return migrate.Config{
		ConnectionString: cfg.DB.ConnectionString,
		Database:         cfg.DB.Database,
		CatalogEnabled:   cfg.Modules.Catalog.Enabled,
		OrdersEnabled:    cfg.Modules.Orders.Enabled,
		InventoryEnabled: cfg.Modules.Inventory.Enabled,
		PaymentsEnabled:  cfg.Modules.Payments.Enabled,
		ShippingEnabled:  cfg.Modules.Shipping.Enabled,
		CartEnabled:      cfg.Modules.Cart.Enabled,
		OutboxEnabled:    cfg.Outbox.Enabled,
	}
}

// loadConfig loads the configuration setting the APP_CONFIG override if defined in flags.
func loadConfig(fs *flag.FlagSet) (engine.Config, error) {
	if cf := fs.Lookup("config"); cf != nil && cf.Value.String() != "" {
		_ = os.Setenv("APP_CONFIG", cf.Value.String())
	}
	cfg, err := engine.LoadConfig()
	if err != nil {
		return engine.Config{}, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}

// usage prints the help menu showing available subcommands, flags, and usage examples.
func usage() {
	fmt.Print(`Usage: go run ./cmd/migrate <command> [flags]

Commands:
  up                    Apply all pending migrations for enabled modules (Postgres)
  down --module <name>  Roll back latest migration for a specific module (Postgres)
  status                Show per-module migration version (Postgres)
  verify                Run integrity checks for enabled modules (Postgres or MongoDB)
  seed                  Ensure collections and indexes exist (MongoDB)

Flags:
  --config <path>       Path to config YAML file
  --root   <path>       Root directory of migration files (default: migrations)
  --module <name>       Module name (required for 'down')

Examples:
  go run ./cmd/migrate up
  go run ./cmd/migrate down --module catalog
  go run ./cmd/migrate status
  go run ./cmd/migrate verify
  go run ./cmd/migrate seed
`)
}
