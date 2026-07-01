# Stock Reservation Feature (`internal/core/inventory/features/reservation`)

This submodule implements stock reservation capabilities, allowing stock to be locked during checkout sequences and released on cancellations or timeouts.

## Features

- **Reserve Stock**: Locks specific inventory quantities for a variant/location. Permits backorders if configured.
- **Release Reservation**: Restores stock counts and deletes active reservations.

## Folder Structure

- [controller.go](controller.go): Exposes endpoints to reserve and release stock.
- [service.go](service.go): Manages reservation timelines, expiry checks, and backorder constraints.
- [repository.go](repository.go): Declares the storage port (`Repository`).
- [routes.go](routes.go): Maps HTTP routes for reservations.
