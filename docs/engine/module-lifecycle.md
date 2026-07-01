---
title: "Module Lifecycle"
description: "The engine.Module interface - what each method does, when it is called, and how to structure a module."
sidebar_position: 2
---

# Module Lifecycle

<DocBadge status="under-review" version="v0.1.0-alpha" />

Every feature in the application is packaged as a module. A module is any Go type that satisfies the `engine.Module` interface. The engine drives the lifecycle of every module - it never calls methods out of order, and it always respects the dependency graph when deciding when to call them.

---

## The `engine.Module` Interface

```go
type Module interface {
    Name()           string
    Requires()       []string
    BasePaths()      []string
    Init(c *Container) error
    RegisterRoutes(public, secured, admin *gin.RouterGroup)
    Shutdown(ctx context.Context) error
}
```

### `Name() string`

Returns the module's unique identifier (e.g., `"catalog"`, `"cart"`). This string is the key used in `Requires()` declarations across all modules. It must be unique across the entire registry.

### `Requires() []string`

Declares which other modules must be fully initialized before this one. The engine validates these names at startup and uses them to determine boot order. See [Dependency Graph](./dependency-graph.md).

### `BasePaths() []string`

Returns the URL prefixes owned by this module (e.g., `[]string{"/api/catalog"}`). The gateway uses these paths to mount catch-all `503` handlers for disabled modules, preventing requests from silently routing nowhere.

### `Init(c *Container) error`

The module's initialization entry point. Called by the engine in dependency-first order after all shared infrastructure is ready. Typical work done here:

1. Fetch domain repositories from `c.Repos`
2. Resolve services from other modules via `c.MustResolve` or the typed helpers in `services.go`
3. Instantiate this module's own services
4. Publish this module's services to the container via `c.Provide`

If `Init` returns an error, the engine aborts startup.

### `RegisterRoutes(public, secured, admin *gin.RouterGroup)`

Mounts HTTP handlers onto the gateway's route groups. Called after all modules have been successfully initialized - so the full container is available if needed, but by this point wiring should already be complete.

- `public` - unauthenticated routes
- `secured` - routes behind JWT authentication middleware
- `admin` - routes behind admin-role middleware

### `Shutdown(ctx context.Context) error`

Called during graceful shutdown in **reverse dependency order** - dependents shut down before their dependencies. Use this to drain in-flight work, stop background goroutines, close cursors, or flush buffers. The context carries the shutdown deadline.

---

## Directory Layout

A module lives under `internal/modules/<name>/` and follows a flat clean-architecture layout:

```text
internal/modules/wishlist/
├── module.go       # Implements engine.Module - wiring and entry point
├── controller.go   # HTTP handlers
├── service.go      # Business logic
└── domain.go       # Domain models and repository interfaces
```

`module.go` is purely wiring. It should not contain business logic - that belongs in `service.go`. The domain layer (`domain.go`) defines repository interfaces without importing any database driver.

---

## Example: `module.go`

```go
package wishlist

import (
    "context"
    "ecom-engine/internal/engine"
    "github.com/gin-gonic/gin"
)

type Module struct {
    cfg     engine.Config
    service *WishlistService
}

func New(cfg engine.Config) engine.Module {
    return &Module{cfg: cfg}
}

func (m *Module) Name() string      { return "wishlist" }
func (m *Module) Requires() []string { return []string{"catalog"} }
func (m *Module) BasePaths() []string { return []string{"/api/wishlist"} }

func (m *Module) Init(c *engine.Container) error {
    repo       := c.Repos.WishlistRepo()
    catalogSvc := engine.ResolveCatalogQuery(c)
    m.service   = NewWishlistService(repo, catalogSvc)
    c.Provide(engine.ServiceWishlist, m.service)
    return nil
}

func (m *Module) RegisterRoutes(public, secured, admin *gin.RouterGroup) {
    ctrl := NewController(m.service)
    g := secured.Group("/wishlist")
    {
        g.POST("/items", ctrl.AddItem)
        g.GET("", ctrl.GetWishlist)
    }
}

func (m *Module) Shutdown(ctx context.Context) error {
    return nil
}
```
