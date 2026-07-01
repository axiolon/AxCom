# ADR-011: Unified Database Migrations and Seeding CLI

**Date:** 2026-06-27  
**Status:** accepted

## Context
The ecom-engine supports both relational (PostgreSQL) and document (MongoDB) databases. Each database paradigm handles schema structure and maintenance differently:
- **PostgreSQL** uses versioned, sequential schema files (DDL) to transition schemas between defined versions step-by-step.
- **MongoDB** is schema-less but requires collections, validation schemas, and indexes to be programmatically provisioned (seeded) for performance and consistency.

Managing these using ad-hoc scripts, separate binaries, or manual shell commands increases developer friction and complicates automated CI/CD deployments. We need a unified tool to run, inspect, and verify the health of the target database schemas.

## Decision
1. **Unified Migration CLI:** Consolidate all schema administration into a single Go command-line tool (`cmd/migrate/main.go`) driven by dedicated subcommands:
   - `up`: Apply pending PostgreSQL migrations.
   - `down`: Roll back PostgreSQL migrations for a specific module.
   - `status`: Show migration history and pending items for PostgreSQL.
   - `verify`: Perform quick checks to ensure SQL schema matches expectations or MongoDB has the correct collection states.
   - `seed`: Execute index and collection provisioning for MongoDB.
2. **Subcommand Flag Isolation:** Build independent, command-specific flag sets (`flag.NewFlagSet`) rather than a single global namespace. This enforces argument boundary constraints (e.g. the `--module` parameter is strictly required and isolated to the `down` subcommand).
3. **Module-Scoped Execution:** Load the system configuration and dynamically scope schema changes using `toMigrateConfig()`. This ensures that migrations are only applied or verified for modules explicitly enabled in the configuration file.

## Alternatives Considered

| Option | Reason Rejected |
|--------|-----------------|
| Separate CLI tools per DB | Doubles the deployment scripting effort. CI/CD pipelines would have to dynamically select different tools depending on variables, increasing the risk of environment configuration drift. |
| Global Flag Namespace | Leads to namespace pollution and confusing CLI help options, where flags unique to one operation (e.g. rollback target module) are allowed to be passed into completely unrelated actions. |

## Why This Choice
Creating a single, cohesive command-line interface provides developers and infrastructure operators a consistent way to manage database state across all supported backends. It respects the unique characteristics of SQL and NoSQL databases under the hood while offering a unified, clean interface for deployments and verification.

## Tradeoffs
**Gains:**
* Simplified CI/CD deployment pipelines (one tool to rule all database prepare stages).
* Safer executions due to subcommand argument isolation and configuration-scoped executions.
* Explicit verification checks (`verify`) that can serve as health checks.

**Accepts:**
* Routing overhead and manually parsing positional CLI commands within the tool entry point.

## Consequences
* All database schema alterations, rollbacks, and indexes must be managed via the `go run ./cmd/migrate` utility.
* SQL migration files go under `migrations/postgres/`, while MongoDB index configurations go under `migrations/mongodb/`.
