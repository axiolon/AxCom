---
title: "Repository Layer"
description: "How the RepoProvider factory keeps business modules decoupled from database drivers."
sidebar_position: 5
---

# Repository Layer

<DocBadge status="under-review" version="v0.1.0-alpha" />

Business modules never import database driver packages (like `pgx` or the MongoDB driver). Instead, they define repository interfaces in their domain layer and receive concrete implementations at runtime through the `RepoProvider`. Switching databases is a config change, not a code change.

---

## The Pattern

```
                 ┌──────────────────────┐
                 │     RepoProvider     │
                 └──────────┬───────────┘
                            │
           Based on:        │
           "db.type"        ├── "mongodb"  ──► mongoWishlist.NewRepository()
                            │
                            └── "postgres" ──► pgWishlist.NewRepository()
```

The `RepoProvider` is a struct held in the container (`c.Repos`). It reads the configured `db.type` and routes each repository request to the correct driver implementation. All database-specific code lives under `internal/infra/db/`.

---

## Module Side: Define the Interface

In your module's domain layer, define a repository interface with no imports from any DB package:

```go
// internal/modules/wishlist/domain.go
package wishlist

import "context"

type Repository interface {
    Add(ctx context.Context, item *WishlistItem) error
    Get(ctx context.Context, userID string) ([]*WishlistItem, error)
}
```

The module only knows this interface. It never knows which database backs it.

---

## Infra Side: Implement per Driver

Implement the interface for each supported database:

```
internal/infra/db/
├── mongodb/
│   └── wishlist/
│       └── repo.go     # implements wishlist.Repository using MongoDB
└── postgres/
    └── wishlist/
        └── repo.go     # implements wishlist.Repository using pgx / GORM
```

---

## Wire in RepoProvider

Add a method to `repoprovider.go` that switches on `db.type` and returns the correct implementation:

```go
func (rp *RepoProvider) WishlistRepo() wishlist.Repository {
    switch rp.dbType {
    case "mongodb":
        return mongoWishlist.NewRepository(rp.mongoDB)
    case "postgres":
        return pgWishlist.NewRepository(rp.pgAdapter)
    }
    return nil
}
```

---

## Module Init: Fetch from Container

In your module's `Init()`, call the provider method - you get back the interface, typed correctly:

```go
func (m *Module) Init(c *engine.Container) error {
    repo := c.Repos.WishlistRepo()
    m.service = NewWishlistService(repo)
    return nil
}
```

The module code is identical regardless of which database is configured. Only `repoprovider.go` and the `internal/infra/db/` implementations differ between drivers.
