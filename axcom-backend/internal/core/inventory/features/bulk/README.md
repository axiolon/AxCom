# Bulk Operations Feature (`internal/core/inventory/features/bulk`)

This submodule implements capabilities to adjust stock quantities for multiple product variants at multiple locations in a single batch request.

## Features

- **Batch Quantity Updates**: Bulk-updates stock levels for multiple variant-location mappings at once.

## Folder Structure

- [controller.go](controller.go): Exposes HTTP POST `/api/inventory/bulk-update` to bind JSON payloads and execute batch updates.
- [service.go](service.go): Iterates over item updates, checks domain validation rules, and saves items.
- [repository.go](repository.go): Storage interface port definition.
- [routes.go](routes.go): Route registration setup.
