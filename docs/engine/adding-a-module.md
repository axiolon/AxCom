---
title: "Adding a Module"
description: "Step-by-step checklist for creating, wiring, and registering a new engine module."
sidebar_position: 6
---

# Adding a Module

<DocBadge status="under-review" version="v0.1.0-alpha" />

This is a practical wiring checklist. For the concepts behind each step, refer to the [Engine Overview](./overview.md).

The example below adds a `wishlist` module.

---

## Checklist

### 1. Create the module directory

```text
internal/modules/wishlist/
├── module.go       # engine.Module implementation
├── controller.go   # HTTP handlers
├── service.go      # business logic
└── domain.go       # domain models + repository interface
```

See [Module Lifecycle](./module-lifecycle.md) for what each file should contain and a full `module.go` example.

---

### 2. Implement `engine.Module`

In `module.go`, implement all six methods of the `engine.Module` interface: `Name`, `Requires`, `BasePaths`, `Init`, `RegisterRoutes`, and `Shutdown`.

---

### 3. Add configuration

Three files need to change to make your module toggle-able via `config.yaml`.

**In `internal/engine/config.go`** — add a config type and include it in `ModulesConfig`:

```go
type WishlistModuleConfig struct {
    Enabled bool `yaml:"enabled"`
}

type ModulesConfig struct {
    // ... existing modules
    Wishlist WishlistModuleConfig `yaml:"wishlist"`
}
```

Then set a default in `defaultModulesConfig()`:

```go
func defaultModulesConfig() ModulesConfig {
    return ModulesConfig{
        // ... existing modules
        Wishlist: WishlistModuleConfig{Enabled: true},
    }
}
```

**In `internal/engine/registry.go`** — add a case to the `IsModuleEnabled` switch:

```go
case "wishlist":
    return cfg.Modules.Wishlist.Enabled
```

**In `config.yaml`** — add the toggle:

```yaml
modules:
  wishlist:
    enabled: true
```

Setting `enabled: false` causes the engine to skip the module's `Init()` and `RegisterRoutes()` and mount a `503` catch-all on its `BasePaths()` instead.

---

### 4. Declare service keys (if other modules will depend on yours)

In `internal/engine/services.go`, add a constant and a typed resolver:

```go
const (
    // ... existing keys
    ServiceWishlist = "wishlist"
)

func ResolveWishlist(c *Container) wishlist.Service {
    return c.MustResolve(ServiceWishlist).(wishlist.Service)
}
```

Then in your module's `Init()`, publish the service:

```go
c.Provide(engine.ServiceWishlist, m.service)
```

See [Dependency Injection](./dependency-injection.md) for how `Provide` and `MustResolve` work.

---

### 5. Register in the master registry

In `internal/modules/registry/registry.go`, import your package and add it to the `factories` map:

```go
import (
    // ... existing imports
    moduleswishlist "ecom-engine/internal/modules/wishlist"
)

var factories = map[string]func(engine.Config) engine.Module{
    // ... existing entries
    "wishlist": moduleswishlist.New,
}
```

---

### 6. Implement repositories

Define your repository interface in `domain.go` (no DB imports). Implement it for each driver and wire it in `repoprovider.go`.

See [Repository Layer](./repository-layer.md) for the full pattern.

---

## Verification

After completing the checklist, start the server:

```bash
go run ./cmd/server/main.go
```

The engine will log module boot order. Confirm your module appears, initialized after its dependencies. If a dependency is missing or misspelled, the engine will exit at startup with a clear error before any module boots.
