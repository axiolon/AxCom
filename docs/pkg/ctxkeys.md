---
title: ctxkeys
sidebar_label: ctxkeys
sidebar_position: 2
---

# ctxkeys

<DocBadge status="under-review" version="v0.1.0-alpha" />

The `ctxkeys` package defines a custom `ContextKey` string type to prevent collisions when storing values in Go's `context.Context`. Using a typed key (instead of a plain `string`) means two packages that both use the string `"user_id"` cannot accidentally read each other's values.

**Import path:** `ecom-engine/pkg/ctxkeys`

---

## Type

```go
type ContextKey string
```

A distinct named type wrapping `string`. Pass constants of this type as `context.WithValue` keys.

---

## Constants

| Constant | Value | Set by | Used by |
|---|---|---|---|
| `UserIDKey` | `"user_id"` | Auth middleware | Handlers, service layer |
| `UserRoleKey` | `"user_role"` | Auth middleware | Authorization checks |
| `CorrelationIDKey` | `"correlation_id"` | Request middleware | Logging, event publishing |

---

## Usage

### Reading from context

```go
import "ecom-engine/pkg/ctxkeys"

func myHandler(ctx context.Context) {
    userID, ok := ctx.Value(ctxkeys.UserIDKey).(string)
    if !ok || userID == "" {
        // not authenticated
    }

    role := ctx.Value(ctxkeys.UserRoleKey).(string)
    corrID := ctx.Value(ctxkeys.CorrelationIDKey).(string)
}
```

### Writing to context (middleware)

```go
ctx = context.WithValue(ctx, ctxkeys.UserIDKey, "usr_018f2fbb-d000-7000-8000-000000000001")
ctx = context.WithValue(ctx, ctxkeys.UserRoleKey, "admin")
ctx = context.WithValue(ctx, ctxkeys.CorrelationIDKey, requestID)
```

---

## Why a custom type?

Go's `context.WithValue` accepts `any` as the key. If two packages both use the bare string `"user_id"`, they share the same slot and can overwrite each other's values. By defining `ContextKey` as a separate type, the Go compiler treats `ctxkeys.UserIDKey` and `someotherpkg.UserIDKey` (a plain `string`) as different keys even if their underlying string values are the same.
