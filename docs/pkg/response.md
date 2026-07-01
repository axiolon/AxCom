---
title: response
sidebar_label: response
sidebar_position: 6
---

# response

<DocBadge status="under-review" version="v0.1.0-alpha" />

The `response` package defines two response contracts used by all handlers:

- **`APIResponse`** - a success envelope for `200 OK` responses (`application/json`)
- **`ProblemDetail`** - an error envelope conforming to [RFC 7807 Problem Details for HTTP APIs](https://datatracker.ietf.org/doc/html/rfc7807) (`application/problem+json`)

Both automatically inject the OpenTelemetry `trace_id` from the active span in the request context. Helpers exist for both `net/http` and **Gin** handlers.

**Import path:** `ecom-engine/pkg/response`

---

## Success: APIResponse

Used for all successful responses.

```go
type APIResponse struct {
    Success bool   `json:"success"`
    Data    any    `json:"data,omitempty"`
    Error   string `json:"error,omitempty"`
    TraceID string `json:"trace_id,omitempty"`
}
```

**Example wire format:**

```json
{
  "success": true,
  "data": { "id": "prd_...", "name": "Widget" },
  "trace_id": "abc123..."
}
```

---

## Error: ProblemDetail (RFC 7807)

Used for all error responses. The `Content-Type` header is set to `application/problem+json`.

```go
type ProblemDetail struct {
    Type     string `json:"type"`               // URI identifying the problem type
    Title    string `json:"title"`               // HTTP status text
    Status   int    `json:"status"`              // HTTP status code
    Detail   string `json:"detail,omitempty"`    // Human-readable explanation
    Instance string `json:"instance,omitempty"`  // Request path (/api/v1/products/...)
    TraceID  string `json:"trace_id,omitempty"`  // OTel trace ID
}
```

**Example wire format:**

```json
{
  "type": "about:blank",
  "title": "Not Found",
  "status": 404,
  "detail": "product not found",
  "instance": "/api/v1/products/prd_unknown",
  "trace_id": "abc123..."
}
```

`instance` is populated automatically from `r.URL.Path` / `c.Request.URL.Path`. `TraceID` is injected if an OTel span is active.

---

## Gin helpers

These are used in the majority of handlers since the gateway layer uses Gin.

### GinOK

```go
func GinOK(c *gin.Context, data any)
```

Writes a `200 OK` `APIResponse` with the trace ID injected.

```go
response.GinOK(c, product)
```

### GinError

```go
func GinError(c *gin.Context, status int, errMsg string)
```

Writes an RFC 7807 problem detail with `type: "about:blank"` and the given status code.

```go
response.GinError(c, http.StatusBadRequest, "page must be a positive integer")
```

### GinWriteError

```go
func GinWriteError(c *gin.Context, err error)
```

The main error-writing helper. Inspects `err`:

- If `err` is an `*AppError` - writes RFC 7807 with the correct status code and message. Also logs the internal error via `logger.ErrorCtx`.
- Otherwise - logs the unhandled error and writes `500 Internal Server Error`.

```go
product, err := s.GetByID(ctx, id)
if err != nil {
    response.GinWriteError(c, err)
    return
}
response.GinOK(c, product)
```

---

## net/http helpers

Use these in plain `http.Handler` contexts (e.g. health check routes, middleware).

### JSON

```go
func JSON(w http.ResponseWriter, r *http.Request, status int, success bool, data any, errMsg string)
```

Full-control response writer. Sets `Content-Type: application/json`.

### OK

```go
func OK(w http.ResponseWriter, r *http.Request, data any)
```

Shorthand for a `200 OK` success response.

```go
response.OK(w, r, healthStatus)
```

### Error

```go
func Error(w http.ResponseWriter, r *http.Request, status int, errMsg string)
```

Writes an RFC 7807 problem detail with `type: "about:blank"`.

### WriteError

```go
func WriteError(w http.ResponseWriter, r *http.Request, err error)
```

Same logic as `GinWriteError` - inspects `*AppError` and writes the appropriate RFC 7807 response, or falls back to 500.

---

## Content-Type behaviour

| Scenario                                                   | Content-Type               |
| ---------------------------------------------------------- | -------------------------- |
| Success (`GinOK`, `OK`, `JSON`)                            | `application/json`         |
| Error (`GinWriteError`, `WriteError`, `GinError`, `Error`) | `application/problem+json` |

Clients can detect errors by checking the `Content-Type` header without parsing the body first.
