# Inventory Module

Manages stock availability, reservations, audit history, transfers, adjustments, bulk imports/exports, reporting, and external system sync.

## Quick Links

- [Full Documentation](../../../../../docs/modules/inventory.md)
- [Tests](./tests.md)

## Directory Layout

| File/Dir | Role |
| :--- | :--- |
| `service.go` | `ModuleServices` — bundles all sub-feature service instances |
| `routes.go` | `ModuleControllers` and `RegisterRoutes` |
| `model.go` | Shared DTOs and models |
| `errors.go` | Package-level error definitions |
| `domain/` | Core domain concepts: `StockItem`, `Reservation`, `Alert`, `StockHistory` |
| `features/core/` | Stock queries, configuration, updates, low-stock alerts |
| `features/bulk/` | Bulk stock import, export, and updates |
| `features/history/` | Audit trail of stock mutations |
| `features/reservation/` | Temporary stock holds during checkout |
| `features/reports/` | Aggregations and analytical reports |
| `features/transfer/` | Inter-warehouse stock movements |
| `features/adjustment/` | Manual stock corrections |
| `features/sync/` | External channel / ERP synchronization |