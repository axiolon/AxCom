# ADR-009: Hand-rolled Map-Backed Dependency Injection Container

**Date:** 2026-06-27  
**Status:** accepted

## Context
Modules must access shared infrastructure (such as the database connection pool, EventBus, Cache, and FileStorage) as well as public services provided by other modules (e.g. `catalog.query` and `catalog.command`).

Using global variables introduces race conditions in testing and tightly couples code. Conversely, importing complex reflection-heavy Dependency Injection (DI) frameworks (e.g. Uber fx, Dig, or wire) adds cognitive overhead, runtime magic, obscures stack traces during crashes, and complicates debugging.

## Decision
1. **Explicit, Hand-Rolled Container:** Implement a simple, thread-safe, map-backed `Container` struct (`container.go`) that holds shared infrastructure and a registry map of module services.
2. **Explicit Lifecycles:**
   - The engine initializes shared infrastructure upfront.
   - The container is passed to each module's `Init(*Container)` function in topological order.
   - Modules register their exported services to the container using `Provide(key, svc)`.
   - Dependent modules retrieve services via `Resolve(key)` or `MustResolve(key)`.
3. **Fail-Fast on Duplicates:** `Provide()` panics if a duplicate key is registered, immediately capturing wiring bugs on startup.
4. **Type-Safe Wrapper Resolvers:** To avoid fragile string keys and `any` type assertions throughout call sites, centralize service keys and type-safe resolver helpers in `services.go`:
   ```go
   func ResolveCatalogQuery(c *Container) catalogCore.QueryService {
       return c.MustResolve(ServiceCatalogQuery).(catalogCore.QueryService)
   }
   ```

## Alternatives Considered

| Option | Reason Rejected |
|--------|-----------------|
| Uber `dig` / `fx` | Slower startup, heavy reliance on reflection, complex setup boilerplate, and hard-to-read stack traces when dependency wiring fails. |
| Strict Constructor Injection | Difficult to manage when modules are enabled or disabled dynamically via YAML settings. Requiring modules to pass dependencies directly through constructors results in cyclic import issues in Go. |

## Why This Choice
This approach gives us the best of both worlds: the safety and ease of debugging of explicit manual wiring, and the flexibility needed to load modular components dynamically at runtime. Stack traces are clear, compilation remains extremely fast, and the code is plain, readable Go without magic.

## Tradeoffs
**Gains:**
* Simple, easily debuggable dependency resolution with zero runtime reflection overhead.
* Clear compile-time safety and explicit runtime tracking of provided services.
* Decoupled package dependencies (no cyclic imports).

**Accepts:**
* Small boilerplate of adding service keys and resolver helpers in `services.go` when adding new cross-module APIs.

## Consequences
* Developer must add a string key constant and a type-safe resolver function to `services.go` when exporting a new service.
* Any missing dependency or duplicate service registration results in a boot-time panic, protecting the production environment.
