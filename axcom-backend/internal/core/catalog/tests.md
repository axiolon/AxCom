# Catalog Module Tests

This document tracks and describes the testing strategy, validation suite, and feature-specific tests for the `catalog` module.

---

## Overview

The `catalog` module is organized into self-contained features. Each feature contains:
- **Unit Tests (`service_test.go` in each feature)**: Validates business logic, domain constraints, validation rules, and stock updates.
- **Integration Tests (`controller_test.go` in each feature)**: Validates Gin API route registrations, request parsing, binding validation, and HTTP responses.

To read the detailed test specifications and scenario lists for each feature, please refer to the respective test documents below.

---

## Feature-Specific Test Suites

Select a feature below to view its detailed test documentation, including unit test scenarios and HTTP endpoints:

- 📑 **[Core Feature](features/core/tests.md)** - Basic product & category CRUD, stock synchronization events, and catalog listing queries.
- 📑 **[Product Variants Feature](features/variants/tests.md)** - Individual product variant configuration, SKUs, and constraint checks.
- 📑 **[Bulk Operations Feature](features/bulk/tests.md)** - Batch creations, updates, and deletions.
- 📑 **[Product Discounts Feature](features/discounts/tests.md)** - Percentage-based and fixed-value product discounts rules.
- 📑 **[Product Images Feature](features/images/tests.md)** - Storage integration, upload presigning, file registration, and primary image selection.
- 📑 **[Product Reviews Feature](features/reviews/tests.md)** - User ratings, comments, rating summaries, and merchant replies.

---

## Running the Tests

To run all catalog feature tests:

```bash
go test -v ./internal/core/catalog/...
```

To run with coverage calculation:

```bash
go test -coverprofile=coverage.out ./internal/core/catalog/...
go tool cover -func=coverage.out
```
