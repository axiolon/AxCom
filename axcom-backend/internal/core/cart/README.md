# Cart Module

Manages the authenticated user's shopping cart. Stores minimal cart state and enriches items using product catalog data. Includes a guest cart merge submodule.

## Quick Links

- [Full Documentation](../../../../../docs/modules/cart.md)
- [Tests](./tests.md)

## Directory Layout

| File/Dir | Role |
| :--- | :--- |
| `controller.go` | HTTP handlers and request validation |
| `routes.go` | Route registration |
| `service.go` | Business logic, cart enrichment, count calculations |
| `repository.go` | Cart persistence contract |
| `model.go` | Domain models |
| `errors.go` | Sentinel error definitions |
| `dto/` | Request and response DTOs |
| `merge/` | Guest-cart merge submodule |