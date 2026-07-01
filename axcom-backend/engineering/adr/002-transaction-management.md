# ADR-002: Transaction Management

**Status:** accepted

## Context

Several inventory operations involve multiple writes that must be atomic:

- **Transfer:** decrement source location, increment destination location
- **Reservation:** create reservation record, decrement available stock
- **Bulk update:** update N stock records

The `TransactionManager` interface already exists in `internal/infra/db/port.go`:

```go
type TransactionManager interface {
    RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
```

The Postgres adapter implements it (`postgres/adapter.go`), but no inventory code uses it. A failure between writes leaves the database in an inconsistent state.

## Decision

1. **Inject `TransactionManager` into services** that perform multi-write operations (transfer, reservation, bulk).

2. **Wrap multi-write operations in `RunInTx`:**

   ```go
   func (s *TransferService) Transfer(ctx context.Context, req TransferRequest) error {
       return s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
           src, err := s.repo.GetStock(txCtx, req.SourceVariantID, req.SourceLocationID)
           if err != nil { return err }
           // ... decrement source, increment dest ...
           return nil
       })
   }
   ```

3. **Transaction-scoped context** — `RunInTx` places the transaction on the context. The `Database` adapter checks for an active transaction on the context and uses it if present, otherwise uses the pool directly. This means repository methods work both inside and outside transactions without any code changes.

4. **MongoDB equivalent** — For MongoDB, `RunInTx` wraps a `mongo.Session` with `WithTransaction`. This requires a replica set (standalone MongoDB does not support transactions). Document this as a deployment requirement.

## Consequences

- Transfer and reservation operations are atomic — partial writes are rolled back on error.
- Repos remain transaction-unaware; the transaction is carried via context.
- MongoDB deployments must use replica sets (already required for change streams).
- Single-write operations (e.g., `SaveStock`) don't use transactions — no unnecessary overhead.
