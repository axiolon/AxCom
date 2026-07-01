# ADR-010: Database-Agnostic RepoProvider Factory Pattern

**Date:** 2026-06-27  
**Status:** accepted

## Context
The ecom-engine is designed to be highly pluggable, supporting both **MongoDB** and **PostgreSQL** backends interchangeably based on configuration (`db.type`). 

If business modules (e.g. Catalog, Cart, Inventory) import database driver packages (like `go.mongodb.org/mongo-driver/v2` or `gorm.io/gorm`) directly, or if they depend on physical database schemas/queries, the codebase becomes coupled to a specific storage backend. This makes testing difficult and violates clean architecture boundaries.

## Decision
1. **Declare Interfaces in the Domain/Module:** Each module defines database-agnostic repository interfaces in its domain layer. These interfaces must only deal with domain entities and standard Go types (no BSON, SQL, or driver types).
2. **Centralize Implementation Wiring:** Implement a unified `RepoProvider` factory (`repoprovider.go`) within the dependency container (`c.Repos`). It acts as a single point of mapping:
   - It reads the configured `db.type`.
   - It instantiates and returns the corresponding driver-specific implementation (e.g., returns `pgCatalogCore.NewPostgresRepository` or `mongoCatalogCore.NewMongoCatalogRepository`).
3. **Strict Isolation of Driver Code:** Keep all database-specific packages, driver adapters, and database-specific structures isolated under `internal/infra/db/mongodb/` and `internal/infra/db/postgres/`.

## Alternatives Considered

| Option | Reason Rejected |
|--------|-----------------|
| Database-Specific Modules | Creating separate packages or modules for MongoDB vs Postgres engines. This results in massive duplication of business rules, controllers, and tests. |
| Ad-hoc Driver Imports in Modules | Allowing modules to initialize their own databases. This couples the core domain packages to database drivers, preventing database swapping via configuration. |

## Why This Choice
By isolating repository factory logic in `RepoProvider`, business modules remain completely unaware of the underlying database storage details. Swapping the entire database backend from MongoDB to PostgreSQL becomes a simple configuration flag change, with zero changes required to the domain or application logic layers.

## Tradeoffs
**Gains:**
* Total decoupling of business rules from infrastructure, conforming to Clean Architecture.
* Easier testing: unit tests can mock repository interfaces without mocking database driver internals.
* Clean database migrations and schema maintenance isolated from core modules.

**Accepts:**
* Duplication of repository query implementations: each database interface must have separate PostgreSQL and MongoDB query logic written and maintained.

## Consequences
* Every new repository interface must be implemented for both MongoDB and Postgres under `internal/infra/db/`.
* A corresponding getter method must be added to `RepoProvider` (`repoprovider.go`) to return the interface.
* Modules are strictly prohibited from importing code from `internal/infra/db/`.
