# Core Inventory Feature Tests

This document tracks and describes the testing strategy and validation suite for the `core` inventory feature.

---

## Overview

The `core` inventory feature contains unit tests and HTTP handler integration/E2E tests to validate stock checks, configurations, adjustments, alerts, and deletion operations.
- **Unit Tests (`service_test.go`)**: Validates core business logic inside `service`. It uses a mock repository and mock alert dispatcher to isolate and check alerting thresholds, configuration properties, and deletion behaviour.
- **Integration & E2E Tests (`controller_test.go`)**: Validates HTTP controllers, status codes, query bindings, error mapping, and authentication/role restrictions using a local Gin HTTP router in `TestMode` and `httptest.ResponseRecorder`. It also features a full lifecycle test executing a complete checkout check, update, alert, and configuration flow.

---

## Core Feature Test Suites

### 1. Service Layer Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-CORE-SRV-ALT-001** | Initial Update above threshold | Quantity: 10, Threshold: 5 | Stock updated, no alerts dispatched | Positive |
| **INV-CORE-SRV-ALT-002** | Update stock below threshold | Quantity: 3, Threshold: 5 | Stock updated, 1 alert dispatched to dispatcher | Positive |
| **INV-CORE-SRV-LST-001** | List stock with filtering | Filter by status/variant/location | List of matching StockItem entities | Positive |
| **INV-CORE-SRV-DEL-001** | Delete stock successfully | Variant ID and Location ID | Stock item is removed from repository | Positive |
| **INV-CORE-SRV-CFG-001** | Configure stock settings | Valid configuration settings | Configuration parameters stored in repository | Positive |

### 2. HTTP Handler Integration & E2E Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-CORE-API-CHK-001** | Check stock success (Public) | GET `/api/inventory/:variantID?location_id=default` | HTTP 200, return quantity count | Positive |
| **INV-CORE-API-LST-001** | List stock success | GET `/api/inventory?status=low_stock` | HTTP 200, return array of stock responses | Positive |
| **INV-CORE-API-ALT-001** | Get alerts success | GET `/api/inventory/alerts` | HTTP 200, return triggered alerts list | Positive |
| **INV-CORE-API-UPD-001** | Update stock success | POST `/api/inventory/update` with valid JSON body | HTTP 200, returns success message | Positive |
| **INV-CORE-API-CFG-001** | Configure stock success | POST `/api/inventory/configure` with valid settings | HTTP 200, returns success message | Positive |
| **INV-CORE-API-DEL-001** | Delete stock success | DELETE `/api/inventory/:variantID` | HTTP 200, returns success message | Positive |
| **INV-CORE-E2E-FLOW-001** | Complete inventory lifecycle flow | Multi-step request sequence: Check -> Configure -> Update -> Alerts -> Delete | Verifies state changes, alerts dispatching, and deletion across complete stack | Positive |
