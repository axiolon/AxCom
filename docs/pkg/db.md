---
title: db
sidebar_label: db
sidebar_position: 10
---

# db

<DocBadge status="under-review" version="v0.1.0-alpha" />

The `db` package defines a **database connection abstraction** used across the application. It provides a common `Connection` interface with concrete adapters for SQL databases (PostgreSQL via `database/sql`), MongoDB, and an in-memory no-op connection for testing.

**Import path:** `ecom-engine/pkg/db`

> This package only defines the connection contract and lightweight wrappers. The actual database driver setup (connection strings, pooling, migrations) lives in `internal/infra/db`.

---

## Connection interface

```go
type Connection interface {
    Ping(ctx context.Context) error
    Close() error
}
```

All adapters satisfy this interface. Use it as the parameter/field type anywhere you want to be agnostic about the underlying database.

| Method | Description |
|---|---|
| `Ping(ctx)` | Verifies the connection is alive. Used by health check endpoints. |
| `Close()` | Cleanly terminates the connection pool. Call on application shutdown. |

---

## Adapters

### SQLConnection

Wraps a standard `*database/sql.DB` connection pool. Suitable for any SQL driver (PostgreSQL via pgx, MySQL, SQLite, etc.).

```go
type SQLConnection struct {
    DB *sql.DB
}
```

`DB` is exported so callers can access the full `*sql.DB` API (query execution, transactions, etc.) when needed.

```go
conn := &db.SQLConnection{DB: sqlDB}

if err := conn.Ping(ctx); err != nil {
    log.Fatal("database unreachable:", err)
}
defer conn.Close()
```

### MongoConnection

Wraps a `*mongo.Client` from the official MongoDB Go driver v2.

```go
type MongoConnection struct {
    Client *mongo.Client
}
```

`Client` is exported for direct access to MongoDB databases and collections.

```go
conn := &db.MongoConnection{Client: mongoClient}

if err := conn.Ping(ctx); err != nil {
    log.Fatal("mongo unreachable:", err)
}
defer conn.Close()
```

`Close()` calls `Client.Disconnect` with a fresh `context.Background()` to avoid using a cancelled context during shutdown.

### MemoryConnection

A no-op adapter for tests and local simulation. `Ping` always returns `nil`, `Close` is a no-op.

```go
conn := &db.MemoryConnection{}
// Safe to pass anywhere a Connection is expected in tests
```

---

## Usage pattern

The connection is typically constructed in the application engine/startup layer and passed down via dependency injection:

```go
// in engine startup
sqlDB, err := sql.Open("pgx", dsn)
if err != nil { ... }

conn := &db.SQLConnection{DB: sqlDB}

// health check route
router.GET("/healthz", func(c *gin.Context) {
    if err := conn.Ping(c.Request.Context()); err != nil {
        response.GinError(c, http.StatusServiceUnavailable, "database unavailable")
        return
    }
    response.GinOK(c, gin.H{"status": "ok"})
})
```
