# Inter-Warehouse Stock Transfer Feature (`internal/core/inventory/features/transfer`)

This submodule implements inter-location stock transfer capabilities, moving stock between locations safely, rolling back on failure, and dispatching event notifications.

## Features

- **Inter-Warehouse Transfers**: Move product variant quantities from a source location to a destination location safely.
- **Rollback on Failure**: Reverts source stock levels if destination save failures occur.
- **Detailed Events Logs**: Dispatches dual event reports (decrement and increment) to the event bus.

## Folder Structure

- [controller.go](controller.go): Exposes HTTP POST `/api/inventory/transfer` mapping.
- [service.go](service.go): Contains transactional transfer checking, rollback structures, and event notifications.
- [repository.go](repository.go): Declares the storage port (`Repository`).
- [routes.go](routes.go): Maps API routes.
