# Inventory Module Tests

This document tracks and describes the testing strategy, validation suite, and feature-specific tests for the `inventory` module.

---

## Overview

The `inventory` module is organized into self-contained features. Each feature contains:

- **Unit Tests (`service_test.go` in each feature)**: Validates stock allocation, thresholds, backordering limits, transfer constraints, and adjustment entries.
- **Integration/E2E Tests (`controller_test.go` in each feature)**: Validates Gin API route registrations, route handlers, input bindings, middleware validations, and full lifecycle requests.

To read the detailed test specifications and scenario lists for each feature, please refer to the respective test documents below.

---

## Feature-Specific Test Suites

Select a feature below to view its detailed test documentation, including unit test scenarios and HTTP endpoints:

- **[Core Feature](./features/core/tests.md)** - Stock checking, configuration, basic adjustments, low-stock alerting, and stock item deletion.
- **[Adjustment Feature](./features/adjustment/tests.md)** - Manual stock adjustments with audit logging.
- **[Bulk Operations Feature](./features/bulk/tests.md)** - Batch stock updates.
- **[History Feature](./features/history/tests.md)** - Audit trails and historical stock levels.
- **[Reports Feature](./features/reports/tests.md)** - Valuation and low stock alerts summary reports.
- **[Reservation Feature](./features/reservation/tests.md)** - Reserving stock for carts/pending checkouts and timeout release.
- **[Sync Feature](./features/sync/tests.md)** - Real-time sync with third-party channels.
- **[Transfer Feature](./features/transfer/tests.md)** - Inter-warehouse stock transfers.

---

## Running the Tests

To run all inventory feature tests:

```bash
go test -v ./internal/core/inventory/...
```

To run with coverage calculation:

```bash
go test -coverprofile=coverage.out ./internal/core/inventory/...
go tool cover -func=coverage.out
```
