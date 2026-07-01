---
title: "Dependency Injection"
description: "How the engine's DI Container wires shared infrastructure and module services together."
sidebar_position: 3
---

# Dependency Injection

<DocBadge status="under-review" version="v0.1.0-alpha" />

The engine uses a hand-rolled, map-backed DI container rather than a framework. It is created once at startup and passed to every module's `Init()` call. This keeps the dependency graph explicit and avoids reflection-heavy magic.

---

## The Container

The `Container` struct is the single shared repository of all runtime resources. It holds two categories of things:

```
┌──────────────────────────────────────────────────────────────┐
│                      DI CONTAINER                            │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │ SHARED INFRASTRUCTURE (always available at Init time)  │  │
│  │ - DBConn        (raw database connection)              │  │
│  │ - Repos         (RepoProvider factory)                 │  │
│  │ - Cache         (L1 in-memory + L2 Redis)              │  │
│  │ - EventBus      (pub/sub message bus)                  │  │
│  │ - FileStorage   (Local / S3 / R2 adapter)              │  │
│  │ - AuthService   (authentication)                       │  │
│  │ - JWTManager    (token signing & verification)         │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │ MODULE SERVICES (lazily populated as modules boot)     │  │
│  │ - "catalog.query"   --> catalogCore.QueryService       │  │
│  │ - "catalog.command" --> catalogCore.CommandService     │  │
│  │ - "cart"            --> cart.Service                   │  │
│  │ - "orders"          --> orders.Service                 │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

Shared infrastructure is initialized by the engine before any module boots. Module services are added to the container as each module's `Init()` runs, in topological order - so a module's dependencies are always present by the time it calls `Resolve`.

---

## Provide, Resolve, MustResolve

Modules interact with the container through three methods:

**`Provide(key string, svc any)`**
Registers a module's public service under a string key. If the same key is registered twice, the engine panics immediately - fail-fast prevents silent overwrites.

**`Resolve(key string) (any, bool)`**
Looks up a service by key. Returns the value and a boolean indicating whether it was found. Used for optional dependencies.

**`MustResolve(key string) any`**
Looks up a service and panics if not found. Safe to use when the dependency graph guarantees the service has already been provided.

---

## Service Keys & Type-Safe Helpers

Raw string keys and type assertions at call sites are fragile. The engine centralizes both in `services.go`.

**Well-known key constants** eliminate typos:

```go
const (
    ServiceCatalogQuery   = "catalog.query"
    ServiceCatalogCommand = "catalog.command"
    ServiceCart           = "cart"
    ServiceOrders         = "orders"
)
```

**Type-safe resolver functions** wrap `MustResolve` with the correct cast, so callers never touch the raw `any`:

```go
func ResolveCatalogQuery(c *Container) catalogcore.QueryService {
    return c.MustResolve(ServiceCatalogQuery).(catalogcore.QueryService)
}

func ResolveCart(c *Container) cart.Service {
    return c.MustResolve(ServiceCart).(cart.Service)
}
```

A module that needs the catalog service calls `engine.ResolveCatalogQuery(c)` - one line, fully typed, no casting at the call site.

---

## Adding a New Service

When you build a new module that other modules will depend on:

1. Add a constant to `services.go`:

   ```go
   ServiceWishlist = "wishlist"
   ```

2. Add a resolver function to `services.go`:

   ```go
   func ResolveWishlist(c *Container) wishlist.Service {
       return c.MustResolve(ServiceWishlist).(wishlist.Service)
   }
   ```

3. In your module's `Init()`, call `c.Provide`:
   ```go
   c.Provide(engine.ServiceWishlist, m.service)
   ```

See [Adding a Module](./adding-a-module.md) for the full wiring checklist.
