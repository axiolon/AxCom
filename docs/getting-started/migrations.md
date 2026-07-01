---
title: "Migrations"
description: "How database schema migrations work, how to run them, and how to add migrations for a new module."
sidebar_position: 5
---

# Migrations

<DocBadge status="under-review" version="v0.1.0-alpha" />

Migrations are module-scoped. Each module owns its own migration directory. Only modules enabled in `config.yaml` are migrated — disabled modules are skipped automatically. The `core` module is always migrated regardless of config.

The migrate tool lives at `cmd/migrate/main.go` and is run independently of the server.

---

## Commands

```bash
# Apply all pending migrations for enabled modules (Postgres)
go run ./cmd/migrate up

# Roll back the latest migration for a single module (Postgres)
go run ./cmd/migrate down --module catalog

# Show per-module applied version vs latest available (Postgres)
go run ./cmd/migrate status

# Verify tables and indexes exist for all enabled modules
go run ./cmd/migrate verify

# Create MongoDB collections and indexes for enabled modules
go run ./cmd/migrate seed
```

All commands accept optional flags:

| Flag | Description |
|---|---|
| `--config <path>` | Path to config YAML (overrides `APP_CONFIG` env var) |
| `--root <path>` | Root directory of migration files (default: `migrations`) |
| `--module <name>` | Module name — required for `down` |

---

## Directory Layout

```
migrations/
├── postgres/
│   ├── core/
│   │   ├── 001_users.up.sql
│   │   ├── 001_users.down.sql
│   │   └── 002_tokens.up.sql
│   ├── catalog/
│   │   ├── 001_catalog.up.sql
│   │   ├── 001_catalog.down.sql
│   │   ├── 002_reviews.up.sql
│   │   └── 002_reviews.down.sql
│   └── <module>/
│       ├── 001_<name>.up.sql
│       └── 001_<name>.down.sql
└── mongodb/
    ├── core/
    │   └── indexes.json
    ├── catalog/
    │   └── indexes.json
    └── <module>/
        └── indexes.json
```

---

## Postgres: File Naming

Each migration is a pair of files with a zero-padded numeric prefix:

```
001_catalog.up.sql      # apply this migration
001_catalog.down.sql    # revert this migration
```

The engine reads the numeric prefix to determine version order. Versions are applied in ascending order; `down` rolls back the highest applied version only. Applied versions are tracked in the `schema_migrations` table (created automatically on first `up`):

```sql
schema_migrations (
    module      TEXT,
    version     INT,
    applied_at  TIMESTAMPTZ,
    PRIMARY KEY (module, version)
)
```

Each migration runs in its own transaction. If the SQL fails, the transaction is rolled back and no version is recorded.

---

## MongoDB: `indexes.json`

MongoDB does not use SQL migrations. Instead, each module has an `indexes.json` file describing the collections and indexes that must exist. Running `seed` creates missing collections and indexes idempotently — already-existing indexes are skipped by name.

```json
{
  "collections": [
    {
      "name": "products",
      "indexes": [
        {
          "keys": { "sku": 1 },
          "options": { "unique": true, "name": "sku_unique" }
        },
        {
          "keys": { "category_id": 1, "created_at": -1 },
          "options": { "name": "category_date" }
        }
      ]
    }
  ]
}
```

---

## Adding Migrations for a New Module

### Postgres

Create a subdirectory under `migrations/postgres/<module>/` and add numbered file pairs:

```
migrations/postgres/wishlist/
├── 001_wishlist.up.sql
└── 001_wishlist.down.sql
```

The migrator discovers directories automatically. No registration needed — the module just needs to be enabled in config and present in `Plan()` in `migrator.go`.

:::note
If you add a new module, also add it to the `Plan()` function in `internal/migrate/migrator.go` and add the corresponding `<Module>Enabled` field to `migrate.Config` and `cmd/migrate/main.go:toMigrateConfig()`.
:::

### MongoDB

Create `migrations/mongodb/<module>/indexes.json` and define your collections. Run `seed` to apply.

---

## Startup Check

When the server starts with Postgres configured, the engine runs a `QuickCheck` before accepting requests. This verifies:

1. The `schema_migrations` table exists
2. The `core` module has at least version 1 applied

If either check fails, the server exits immediately with an actionable error:

```
core schema not applied — run 'go run ./cmd/migrate up'
```

This prevents the server from booting against an unmigrated database.
