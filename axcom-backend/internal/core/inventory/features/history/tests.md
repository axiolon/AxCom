# Inventory History Feature Tests

This document tracks and describes the testing strategy and validation suite for the `history` inventory feature.

---

## Overview

The `history` inventory feature contains unit tests and HTTP handler integration/E2E tests to validate variant audit trails and events subscriptions.
- **Unit Tests (`service_test.go`)**: Validates query functions, default ID generations, timestamp updates, and handling of stock changed events on the bus.
- **Integration & E2E Tests (`controller_test.go`)**: Validates HTTP controllers, path parameter bindings, and E2E flows executing change events and asserting history updates.

---

## History Feature Test Suites

### 1. Service Layer Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-HIST-SRV-GET-001** | Get stock history success | Populated repository state | Slice of StockHistory logs | Positive |
| **INV-HIST-SRV-REC-001** | Record history creates ID and ChangedAt | StockHistory item with empty ID/Time | Generated `hist_` ID, non-zero ChangedAt timestamp | Positive |
| **INV-HIST-SRV-EVT-001** | Handle stock changed event | StockChanged event published to bus | Triggers `RecordHistory` automatically | Positive |

### 2. HTTP Handler Integration & E2E Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-HIST-API-GET-001** | Get stock history HTTP success | GET `/api/inventory/:variantID/history` | HTTP 200, history list envelope | Positive |
| **INV-HIST-API-GET-002** | Get stock history bad request | GET `/api/inventory//history` | HTTP 404/400 validation error | Negative |
| **INV-HIST-E2E-FLOW-001** | Complete E2E history flow | Publish event -> Call HTTP GET history | Assert HTTP response contains exactly the published changes | Positive |
