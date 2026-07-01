# Orders Module

Implements order creation, lifecycle state machine, repository contract, and guest checkout support for authenticated customers, guests, and admin operations.

## Quick Links

- [Full Documentation](../../../../../docs/modules/orders.md)
- [Tests](./tests.md)

## Directory Layout

| File/Dir | Role |
| :--- | :--- |
| `service.go` | Core business logic — creation, transitions, retrieval |
| `repository.go` | `OrderRepository` persistence contract |
| `model.go` | `Order`, `OrderItem`, `OrderStatus`, `GuestInfo` types |
| `statemachine.go` | Order state machine adapter |
| `errors.go` | Order-specific error definitions |
| `domain/` | Order domain model, validation, and state machine logic |
| `admin/` | Admin HTTP handlers and DTOs |
| `user/` | Authenticated customer order handlers and DTOs |
| `guest/` | Guest checkout handler, models, and repository |