# Inventory Transfer Feature Tests

This document tracks and describes the testing strategy and validation suite for the `transfer` inventory feature.

---

## Overview

The `transfer` inventory feature contains unit tests and HTTP handler integration/E2E tests to validate stock transfers.
- **Unit Tests (`service_test.go`)**: Validates transfer validation, stock checks, transactional rollback logic on target store failures, and dual event publishing.
- **Integration & E2E Tests (`controller_test.go`)**: Validates HTTP POST payloads parsing, status codes, and E2E transfers.

---

## Transfer Feature Test Suites

### 1. Service Layer Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-TRSF-SRV-001** | Transfer stock success | Quantity: 5, Source: 10, Target: 0 | Decrements source to 5, increments target to 5, publishes 2 events | Positive |
| **INV-TRSF-SRV-002** | Transfer stock fails - same locations | Source: "default", Target: "default" | Error 400: "source and destination locations must be different" | Negative |
| **INV-TRSF-SRV-003** | Transfer stock fails - insufficient stock | Quantity: 15, Source: 10 | Error 409 Conflict: "insufficient stock for transfer" | Negative |
| **INV-TRSF-SRV-004** | Transfer stock fails - missing source record | Non-existent source location | Error 404: "source stock record not found" | Negative |

### 2. HTTP Handler Integration & E2E Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-TRSF-API-001** | Transfer stock HTTP success | POST `/api/inventory/transfer` with valid body | HTTP 200, stock transfer completed successfully | Positive |
| **INV-TRSF-API-002** | Transfer stock HTTP bad request | POST `/api/inventory/transfer` with missing values | HTTP 400 Bad Request | Negative |
| **INV-TRSF-E2E-001** | Complete E2E transfer cycle | POST transfer -> Verify source and target quantities in DB | Quantities are updated correctly across database | Positive |
