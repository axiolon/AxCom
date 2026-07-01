---
title: errors
sidebar_label: errors
sidebar_position: 3
---

# errors

<DocBadge status="under-review" version="v0.1.0-alpha" />

The `errors` package provides `AppError`, a domain-specific error type that carries an HTTP status code, a user-safe message, an optional internal error for logging, and an RFC 7807 problem type URI. It implements the standard `error` interface and `Unwrap()` for compatibility with `errors.Is` / `errors.As`.

**Import path:** `ecom-engine/pkg/errors`

> Use an import alias to avoid shadowing the standard library `errors` package:
>
> ```go
> import apperrors "ecom-engine/pkg/errors"
> ```

---

## AppError struct

```go
type AppError struct {
    Code    int    // HTTP status code
    Message string // User-safe message sent to the client
    Err     error  // Internal wrapped error (logged, never sent to client)
    Type    string // RFC 7807 problem type URI (defaults to "about:blank")
}
```

- `Code` - written directly to the HTTP response status line.
- `Message` - included in the response body as `detail`. Keep it user-friendly.
- `Err` - the underlying technical error. Used only for server-side logging. Never reaches the client.
- `Type` - a URI that identifies the problem type per [RFC 7807](https://datatracker.ietf.org/doc/html/rfc7807). Defaults to `"about:blank"`.

---

## Constructors

| Constructor                    | HTTP Code | Typical use case                           |
| ------------------------------ | --------- | ------------------------------------------ |
| `NewBadRequest(msg, err)`      | 400       | Malformed input, failed validation         |
| `NewUnauthorized(msg, err)`    | 401       | Missing or invalid credentials             |
| `NewForbidden(msg, err)`       | 403       | Authenticated but insufficient permissions |
| `NewNotFound(msg, err)`        | 404       | Resource does not exist                    |
| `NewConflict(msg, err)`        | 409       | Duplicate record or state conflict         |
| `NewTooManyRequests(msg, err)` | 429       | Rate limit exceeded                        |
| `NewInternal(msg, err)`        | 500       | Unexpected server-side error               |
| `NewAppError(code, msg, err)`  | custom    | Any other status code                      |

All constructors set `Type` to `"about:blank"` by default. Use `.WithType()` to set a custom problem type URI.

---

## Methods

### WithType

```go
func (e *AppError) WithType(t string) *AppError
```

Sets the RFC 7807 problem type URI and returns the same `*AppError` for chaining.

```go
return apperrors.NewNotFound("product not found", err).
    WithType("https://errors.axiolon.io/product-not-found")
```

### Error

```go
func (e *AppError) Error() string
```

Returns a string representation including the code, message, and wrapped error (if any). Used by Go's logging and `fmt` machinery.

### Unwrap

```go
func (e *AppError) Unwrap() error
```

Returns the internal `Err` field, allowing `errors.Is` and `errors.As` to traverse the error chain.

---

## Usage

### Returning errors from service/handler code

```go
import apperrors "ecom-engine/pkg/errors"

func (s *ProductService) GetByID(ctx context.Context, id string) (*Product, error) {
    p, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, apperrors.NewNotFound("product not found", err)
    }
    return p, nil
}
```

### Inspecting errors with errors.As

```go
import (
    "errors"
    apperrors "ecom-engine/pkg/errors"
)

var appErr *apperrors.AppError
if errors.As(err, &appErr) {
    log.Printf("HTTP %d: %s", appErr.Code, appErr.Message)
}
```

### Handing errors to the response package

Pass `AppError` directly to `response.WriteError` or `response.GinWriteError` - they inspect the type automatically and write the correct HTTP status + RFC 7807 body.

```go
if err != nil {
    response.GinWriteError(c, err)
    return
}
```

See the [response package docs](./response) for details.
