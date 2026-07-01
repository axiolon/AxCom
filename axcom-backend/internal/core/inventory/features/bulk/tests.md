# Inventory Bulk Feature Tests

This document tracks and describes the testing strategy and validation suite for the `bulk` inventory feature.

---

## Overview

The `bulk` inventory feature contains unit tests and HTTP handler integration/E2E tests to validate batch stock modifications.
- **Unit Tests (`service_test.go`)**: Validates validation rules across multiple elements and transactional database persistence.
- **Integration & E2E Tests (`controller_test.go`)**: Validates batch JSON array deserialization, field requirements, and E2E batch updates.

---

## Bulk Feature Test Suites

### 1. Service Layer Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-BLK-SRV-UPD-001** | Bulk update success | Valid batch payload containing variants and quantities | Saves all variants to repository | Positive |
| **INV-BLK-SRV-UPD-002** | Bulk update negative quantity fails | Batch payload with quantity < 0 | Error 400: "bulk update item validation failed" | Negative |

### 2. HTTP Handler Integration & E2E Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-BLK-API-UPD-001** | Bulk update HTTP success | POST `/api/inventory/bulk-update` with valid items | HTTP 200, bulk update completed | Positive |
| **INV-BLK-API-UPD-002** | Bulk update HTTP bad request | POST `/api/inventory/bulk-update` with empty array or missing fields | HTTP 400 Bad Request | Negative |
| **INV-BLK-E2E-FLOW-001** | Complete E2E bulk flow | Run POST bulk update -> Verifies stock levels in repository | All values saved correctly in repository | Positive |
