# Inventory Reservation Feature Tests

This document tracks and describes the testing strategy and validation suite for the `reservation` inventory feature.

---

## Overview

The `reservation` inventory feature contains unit tests and HTTP handler integration/E2E tests to validate stock reservations.
- **Unit Tests (`service_test.go`)**: Validates reservation locking, backorder constraints, and release procedures.
- **Integration & E2E Tests (`controller_test.go`)**: Validates HTTP reservation request parsing and E2E reservation cycles.

---

## Reservation Feature Test Suites

### 1. Service Layer Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-RES-SRV-001** | Reserve stock success (Available) | Quantity: 2, Current stock: 10 | Stock decremented by 2, returns Reservation ID | Positive |
| **INV-RES-SRV-002** | Reserve stock success (Backorder allowed) | Quantity: 2, Current stock: 0, Backorders permitted | Stock becomes -2, returns Reservation ID | Positive |
| **INV-RES-SRV-003** | Reserve stock fails (Backorder limit exceeded) | Quantity: 10, Current stock: 0, Backorder Limit: 5 | Error: "insufficient stock" | Negative |
| **INV-RES-SRV-004** | Release reservation success | Existing Reservation ID | Stock quantity restored, reservation deleted | Positive |

### 2. HTTP Handler Integration & E2E Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-RES-API-001** | Reserve stock HTTP success | POST `/api/inventory/:variantID/reserve` with valid body | HTTP 200, return reservation ID and expires_at | Positive |
| **INV-RES-API-002** | Release reservation HTTP success | DELETE `/api/inventory/:variantID/reserve/:reservationID` | HTTP 200, returns "reservation released" | Positive |
| **INV-RES-E2E-001** | Complete E2E reservation cycle | POST reserve -> DELETE release | Stock levels decrement and restore correctly across database | Positive |
