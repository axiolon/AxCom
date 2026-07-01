# ADR-001: Error Handling

**Status:** accepted

## Context

Our repository layer spans two database backends (MongoDB, Postgres). Each driver returns its own error types — e.g., `mongo.ErrNoDocuments`, Postgres driver-specific errors. If these leak into the service/core layer, business logic becomes coupled to infrastructure.

Additionally, the Postgres repos use `errors.New("stock not found")` as ad-hoc strings, which callers can't reliably match with `errors.Is`.

## Decision

1. **Define domain sentinel errors** in a shared package (e.g., `internal/core/inventory/errors.go`):

   ```go
   var (
       ErrStockNotFound       = errors.New("stock not found")
       ErrReservationNotFound = errors.New("reservation not found")
       ErrInsufficientStock   = errors.New("insufficient stock")
   )
   ```

2. **Every repository implementation wraps infrastructure errors** into domain errors at the boundary:

   ```go
   // MongoDB
   if errors.Is(err, mongo.ErrNoDocuments) {
       return nil, ErrStockNotFound
   }

   // Postgres
   if err == pgx.ErrNoRows {
       return nil, ErrStockNotFound
   }
   ```

3. **Unexpected errors are wrapped with context**, not replaced:

   ```go
   return nil, fmt.Errorf("query stock %s: %w", variantID, err)
   ```

4. **Service layer matches only domain errors** using `errors.Is`. It never imports database driver packages.

## Consequences

- Service tests can assert on `errors.Is(err, ErrStockNotFound)` without mocking driver internals.
- Switching databases doesn't change service-layer error handling.
- OpenTelemetry spans record the original error via `span.RecordError(err)` before wrapping, so observability is preserved.
