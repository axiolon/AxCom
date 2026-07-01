# Stock Adjustment Feature (`internal/core/inventory/features/adjustment`)

This submodule implements manual stock adjustments (increments/decrements) with an audit reason field, updating stock records and dispatching events.

## Features

- **Manual Stock Adjustments**: Adjust quantity of stock up or down for a variant at a location, attaching a change reason.
- **Audit Trails**: Triggers events on the event bus (`InventoryStockChangedTopic`) containing details about quantities changes.

## Folder Structure

- [controller.go](controller.go): Exposes HTTP handler to adjust stock levels.
- [service.go](service.go): Contains core business logic rules, handles increments/decrements, checks negative constraints, and publishes events.
- [repository.go](repository.go): Declares the storage port interface (`Repository`).
- [routes.go](routes.go): Connects endpoints to the controller.
