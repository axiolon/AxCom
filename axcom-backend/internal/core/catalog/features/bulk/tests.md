# Bulk Operations Feature Tests

This document tracks and describes the testing strategy and validation suite for the `bulk` catalog feature.

---

## Overview

The `bulk` catalog feature enables batch operations (creating, updating, deleting) on multiple products simultaneously.
- **Unit Tests (`service_test.go`)**: Validates transaction-like batch creations, category verification, product name/price schema validations, and repository error mappings.
- **Integration Tests (`controller_test.go`)**: Validates batch HTTP request handlers, request array binding validation, and end-to-end integration flows.

---

## Bulk Feature Test Suites

### 1. Service Layer Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **CTL-BLK-SRV-CRT-001** | Bulk create successfully | Array of valid product specifications | Products created with generated IDs and variants, saved | Positive |
| **CTL-BLK-SRV-CRT-002** | Bulk create fails - empty name | Array containing a product with an empty name | Error 400: "validation failed" | Negative |
| **CTL-BLK-SRV-CRT-003** | Bulk create fails - category not found | Product referencing non-existent category ID | Error 404: Category not found | Negative |
| **CTL-BLK-SRV-CRT-004** | Bulk create fails - repo error | Valid inputs but DB write fails | Error 500: Bulk create DB failure | Negative |
| **CTL-BLK-SRV-UPD-001** | Bulk update successfully | Array of updated product objects with valid IDs | All specified products updated in DB | Positive |
| **CTL-BLK-SRV-UPD-002** | Bulk update fails - missing product ID | Product with empty ID string | Error 400: "product ID is required for update" | Negative |
| **CTL-BLK-SRV-UPD-003** | Bulk update fails - negative price | Variant with price < 0 | Error 400: "validation failed" | Negative |
| **CTL-BLK-SRV-UPD-004** | Bulk update fails - category not found | Product with updated category ID that does not exist | Error 404: Category not found | Negative |
| **CTL-BLK-SRV-UPD-005** | Bulk update fails - repo error | Valid inputs but repository updates fail | Error 500: DB update failure | Negative |
| **CTL-BLK-SRV-DEL-001** | Bulk delete successfully | List of product IDs to remove | Products deleted from repository | Positive |
| **CTL-BLK-SRV-DEL-002** | Bulk delete fails - empty ID list | Empty array of product IDs | Error 400: Invalid inputs | Negative |
| **CTL-BLK-SRV-DEL-003** | Bulk delete fails - repo error | Valid product IDs but deletion fails | Error 500: DB deletion failure | Negative |

### 2. HTTP Handler Integration & E2E Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **CTL-BLK-API-CRT-001** | Bulk create success | POST `/api/products/bulk` with valid JSON array | HTTP 200, confirmation message | Positive |
| **CTL-BLK-API-CRT-002** | Bulk create bad request | POST `/api/products/bulk` with empty body / invalid JSON | HTTP 400 Bad Request | Negative |
| **CTL-BLK-API-UPD-001** | Bulk update success | PUT `/api/products/bulk` with valid JSON array | HTTP 200, confirmation message | Positive |
| **CTL-BLK-API-UPD-002** | Bulk update bad request | PUT `/api/products/bulk` with invalid properties | HTTP 400 Bad Request | Negative |
| **CTL-BLK-API-DEL-001** | Bulk delete success | DELETE `/api/products/bulk` with query param `ids` or JSON body | HTTP 200, confirmation message | Positive |
| **CTL-BLK-API-DEL-002** | Bulk delete bad request | DELETE `/api/products/bulk` with empty `ids` | HTTP 400 Bad Request | Negative |
| **CTL-BLK-E2E-FLOW-001** | Full bulk operations flow | Sequence of Bulk Create -> Bulk Update -> Bulk Delete | Verifies state changes correctly reflect in the repository | Positive |
