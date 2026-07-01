# Shipping Module

Handles package rate calculation across multiple providers, shipment creation, status tracking, and publishes `order.shipped` events when packages enter transit.

## Quick Links

- [Full Documentation](../../../../../docs/modules/shipping.md)
- [Tests](./tests.md)

## Directory Layout

| File/Dir | Role |
| :--- | :--- |
| `controller.go` | Customer-facing HTTP handlers |
| `admin/` | Admin shipping handlers (create, update) |
| `service.go` | Business logic — rates, shipment lifecycle, event dispatch |
| `repository.go` | Shipment persistence contract |
| `model.go` | Shipment and status models |
| `errors.go` | Shipping-specific error definitions |
| `providers/` | Shipping provider implementations (Flat Rate, Free Above X, Weight-Based) |