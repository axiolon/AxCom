# Channel Stock Sync Feature (`internal/core/inventory/features/sync`)

This submodule implements capabilities to sync stock values from third-party/external channels, updating internal stock values and publishing events.

## Features

- **Sync Stock Level**: Updates the quantity directly to match an external system and dispatches a stock changed event with a sync reason.

## Folder Structure

- [controller.go](controller.go): Exposes HTTP handler to update stock values directly.
- [service.go](service.go): Houses sync logic, overrides current stock, and notifies other modules via the event bus.
- [repository.go](repository.go): Declares the storage port (`Repository`).
- [routes.go](routes.go): Maps endpoints to controller functions.
