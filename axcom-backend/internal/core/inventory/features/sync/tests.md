# Inventory Sync Feature Tests

This document tracks and describes the testing strategy and validation suite for the `sync` inventory feature.

---

## Overview

The `sync` inventory feature contains unit tests and HTTP handler integration/E2E tests to validate stock synchronization.
- **Unit Tests (`service_test.go`)**: Validates absolute quantity overrides, mock database updates, and event bus emissions.
- **Integration & E2E Tests (`controller_test.go`)**: Validates HTTP POST payloads parsing, status codes, and E2E stock updates.

---

## Sync Feature Test Suites

### 1. Service Layer Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-SYNC-SRV-001** | Sync stock success (absolute update) | Quantity: 30, Current stock: 5 | Stock overridden to 30, event dispatched | Positive |
| **INV-SYNC-SRV-002** | Sync stock error (DB failure) | DB mock returns error | Returns internal server error | Negative |

### 2. HTTP Handler Integration & E2E Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-SYNC-API-001** | Sync stock HTTP success | POST `/api/inventory/sync` with valid body | HTTP 200, synced successfully | Positive |
| **INV-SYNC-API-002** | Sync stock HTTP bad request | POST `/api/inventory/sync` with missing fields | HTTP 400 Bad Request | Negative |
| **INV-SYNC-E2E-001** | Complete E2E stock sync flow | POST `/api/inventory/sync` -> database matches quantity | Quantity is updated absolutely | Positive |
