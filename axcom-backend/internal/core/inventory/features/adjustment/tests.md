# Inventory Adjustment Feature Tests

This document tracks and describes the testing strategy and validation suite for the `adjustment` inventory feature.

---

## Overview

The `adjustment` inventory feature contains unit tests and HTTP handler integration/E2E tests to validate stock adjustments.
- **Unit Tests (`service_test.go`)**: Validates adjustments calculations, check constraints (preventing negative quantities), and event bus publishing on updates.
- **Integration & E2E Tests (`controller_test.go`)**: Validates HTTP POST routing, request body validations, status codes, and E2E flows checking adjustments and event emissions.

---

## Adjustment Feature Test Suites

### 1. Service Layer Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-ADJ-SRV-001** | Adjust stock success (Increment) | Quantity: +10, Current stock: 5 | New stock: 15, event dispatched | Positive |
| **INV-ADJ-SRV-002** | Adjust stock success (Decrement) | Quantity: -3, Current stock: 5 | New stock: 2, event dispatched | Positive |
| **INV-ADJ-SRV-003** | Adjust stock fails - negative quantity | Quantity: -10, Current stock: 5 | Error: "adjusted stock quantity cannot be negative" | Negative |
| **INV-ADJ-SRV-004** | Adjust stock fails - missing reason | Empty reason string | Error: "adjustment reason is required" | Negative |

### 2. HTTP Handler Integration & E2E Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-ADJ-API-001** | Adjust stock HTTP success | POST `/api/inventory/:variantID/adjust` with valid body | HTTP 200, adjusted successfully | Positive |
| **INV-ADJ-API-002** | Adjust stock HTTP bad request | POST `/api/inventory/:variantID/adjust` with missing reason/adjustment | HTTP 400 Bad Request | Negative |
| **INV-ADJ-E2E-001** | Complete E2E adjustment flow | GET stock -> POST adjust -> GET stock again | Validates value matches mathematically and event is registered | Positive |
