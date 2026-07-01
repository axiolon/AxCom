# ADR-004: Query Safety

**Status:** accepted

## Context

Several query-related safety issues exist in the inventory repositories:

1. **Missing `rows.Err()` check** — After `for rows.Next()` loops in Postgres repos, `rows.Err()` is not checked. If the connection drops mid-iteration, the loop exits silently and returns incomplete data with no error.

2. **Untyped filters** — `ListStock` accepts `map[string]interface{}` for filtering. Callers can pass wrong types, misspelled keys, or invalid values that are silently ignored.

3. **No default pagination** — `ListStock` and `GetAllStockItems` have no default `LIMIT`. On large datasets this returns unbounded results, risking OOM.

4. **`Exec` return value ignored** — The `Database.Exec` interface returns `(Result, error)`, but all callers assign only `error`, which won't compile.

## Decision

### 1. Always check `rows.Err()`

Every `for rows.Next()` loop must be followed by:

```go
if err := rows.Err(); err != nil {
    return nil, fmt.Errorf("iterating rows: %w", err)
}
```

This is non-negotiable — silent data truncation is worse than an error.

### 2. Typed filter structs

Replace `map[string]interface{}` with typed structs:

```go
type ListFilter struct {
    VariantID  string
    LocationID string
    Status     string
    Limit      int64
    Offset     int64
}
```

Build queries from struct fields. Zero values mean "no filter" for that field.

### 3. Default pagination

If `Limit` is 0 or exceeds a maximum, apply defaults:

```go
const (
    DefaultLimit = 50
    MaxLimit     = 500
)
```

### 4. Capture `Exec` result

All `Exec` calls must use `_, err := r.db.Exec(...)`. If we later need affected-row counts (e.g., to detect no-op updates), the `Result` is available without changing call sites.

## Consequences

- No silent data loss from broken connections mid-query.
- Compile errors are fixed.
- Callers get type safety and IDE autocompletion for filters.
- Large unbounded queries are prevented by default limits.
