# Engine Package Testing Documentation

This document outlines the test plan, scenarios, and test suites proposed/implemented for the core `internal/engine` package.

## Overview

The `engine` package is the core backbone of the application. It handles environment configuration loading, dependency injection (via `Container`), module lifecycle management (`Module` interface), repository routing (`RepoProvider`), and dependency resolution (`depgraph`). 

Writing robust unit and integration test cases for this package is **critical** to guarantee system bootstrap stability, correct dependency ordering (topological sorting), and graceful shutdown execution.

---

## Test Suites

### 1. Configuration Tests (`config_test.go`)

This suite validates YAML configuration file parsing, validation constraints, default fallbacks, and environment variable overrides.

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **ENG-CFG-VAL-001** | Validate config with valid params | Correct secret, DB connection, and cache type | Nil error (valid configuration) | Positive |
| **ENG-CFG-VAL-002** | Validate config missing secret | Empty secret key string | Return config validation error | Negative |
| **ENG-CFG-VAL-003** | Validate config invalid DB type | DB type set to "sqlite" or empty | Return validation error for database type | Negative |
| **ENG-CFG-VAL-004** | Validate config missing connection string | DB type valid, connection string empty | Return config validation error | Negative |
| **ENG-CFG-VAL-005** | Validate config invalid cache type | Cache type set to "invalid_cache" | Return config validation error | Negative |
| **ENG-CFG-VAL-006** | Validate config invalid auth mode | Auth mode set to "oauth" | Return config validation error | Negative |
| **ENG-CFG-LDF-001** | Load config from file successfully | Valid yaml file path | Returns fully parsed configuration struct | Positive |
| **ENG-CFG-LDF-002** | Load config missing yaml file | Non-existent file path | Returns default config with env overlay | Positive |

---

### 2. Dependency Graph & Sorting Tests (`depgraph_test.go`)

This suite verifies Kahn's topological sorting algorithm used to resolve order-of-initialization dependencies for registered modules.

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **ENG-DEP-SRT-001** | Topological sort of independent modules | Slices of modules with no cross-dependencies | Sorted list matching original order | Positive |
| **ENG-DEP-SRT-002** | Topological sort of sequential dependencies | Modules: A requires B, B requires C | Sorted order: C, B, A | Positive |
| **ENG-DEP-SRT-003** | Missing required dependency | Module A requires disabled Module B | Returns validation error detailing missing dependency | Negative |
| **ENG-DEP-SRT-004** | Direct circular dependency | Modules A requires B, B requires A | Returns circular dependency error | Negative |
| **ENG-DEP-SRT-005** | Multi-node circular dependency | Modules A requires B, B requires C, C requires A | Returns circular dependency error with cycle path | Negative |

---

### 3. Container & Services Injection Tests (`container_test.go` / `services_test.go`)

This suite validates dependency injection behavior, typed helper resolutions, and panic protections for double registrations.

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **ENG-CON-PRV-001** | Provide and resolve service successfully | Register service interface under key and query it | Service interface returned matching registered type | Positive |
| **ENG-CON-PRV-002** | Duplicate service registration | Provide same key twice | Panics with duplicate registration error | Negative |
| **ENG-CON-RES-001** | MustResolve service successfully | Registered service key | Service interface returned | Positive |
| **ENG-CON-RES-002** | MustResolve unregistered service | Unregistered service key | Panics with service not registered error | Negative |
| **ENG-CON-HLP-001** | Typed helper resolution | Resolved catalog, inventory, cart, etc. | Correctly casted concrete implementations | Positive |

---

### 4. Database-Agnostic Repository Provider Tests (`repoprovider_test.go`)

This suite validates that the `RepoProvider` dynamically returns the correct DB-specific implementations matching configuration settings.

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **ENG-REP-MGO-001** | Mongo Repository instantiation | RepoProvider initialized with mongodb dbType | Returns Mongo implementation of repositories (e.g. `mongoCart.CartRepository`) | Positive |
| **ENG-REP-PG-001** | Postgres Repository instantiation | RepoProvider initialized with postgres dbType | Returns Postgres implementation of repositories (e.g. `pgCart.CartRepository`) | Positive |
| **ENG-REP-UNK-001** | Unknown database type mapping | RepoProvider with empty/invalid dbType | Returns nil repositories safely | Negative |

---

### 5. Engine Bootstrap & Lifecycle Tests (`engine_test.go`)

This suite tests the integration flow of bootstrapping infrastructure, running module `Init` routines, and graceful resource teardown (`Shutdown`).

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **ENG-LFC-INI-001** | Successful Mongo Engine bootstrap | Valid config with mongo, mock modules | Engine instance initialized, routes registered, relay active | Positive |
| **ENG-LFC-INI-002** | Successful Postgres Engine bootstrap | Valid config with postgres, mock modules | Engine instance initialized | Positive |
| **ENG-LFC-INI-003** | Engine bootstrap database connection failure | Invalid database connection string | Returns connection timeout/connection error | Negative |
| **ENG-LFC-SHD-001** | Graceful Engine shutdown | active modules and running outbox relay | Shutdown runs in reverse topological order, relay stopped, DB closed | Positive |

---

### 6. Module Registry Verification Tests (`registry_test.go`)

This suite ensures the correctness of module state checks (enabled vs. disabled) extracted from configuration models.

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **ENG-REG-ENA-001** | Module enabled check | Enabled module name in config | Returns true | Positive |
| **ENG-REG-ENA-002** | Module disabled check | Disabled module name in config | Returns false | Positive |
| **ENG-REG-ENA-003** | Unknown module name check | Unregistered module name | Returns false | Positive |

---

## Running Tests

Run the full engine test suite with:

```bash
go test -v ./internal/engine/...
```

To run with coverage profiling:

```bash
go test -v -coverprofile=coverage.out ./internal/engine/...
```

## Structure of Tests

- **Isolation**: Test files should mock external network and storage endpoints (e.g., real Redis, Postgres, MongoDB) where possible, or use lightweight local/memory implementations (like `pkgdb.MemoryConnection`) to ensure unit tests are fast and hermetic.
- **Parallelism**: Every test suite and individual subtest leverages `t.Parallel()` to run concurrently.
- **Thread Safety**: All mock services and mock repositories use `sync.RWMutex` to guard state during concurrent assertions.
- **Assertions**: Handled using `github.com/stretchr/testify/assert` and `github.com/stretchr/testify/require`.
