# Product Discounts Feature Tests

This document tracks and describes the testing strategy and validation suite for the `discounts` catalog feature.

---

## Overview

The `discounts` catalog feature supports applying percentage-based or fixed-value discounts directly to specific catalog products.
- **Unit Tests (`service_test.go`)**: Validates validation rules for discounts (valid types, value ranges such as percentage not exceeding 100%, non-negative values) and error mapping behavior.
- **Integration Tests (`controller_test.go`)**: Validates Gin HTTP handler endpoint routes, payloads binding, HTTP error statuses, and multi-step E2E discount life-cycles.

---

## Discounts Feature Test Suites

### 1. Service Layer Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **CTL-DSC-SRV-APP-001** | Apply percentage discount successfully | Product ID, discount payload (type: `percentage`, value: 15.5) | Product discount updated in repository | Positive |
| **CTL-DSC-SRV-APP-002** | Apply fixed discount successfully | Product ID, discount payload (type: `fixed`, value: 50.0) | Product discount updated in repository | Positive |
| **CTL-DSC-SRV-APP-003** | Apply discount fails - nil payload | Product ID, `nil` discount | Error 400: "Discount payload is required" | Negative |
| **CTL-DSC-SRV-APP-004** | Apply discount fails - invalid type | Product ID, discount payload (type: `invalid_type`, value: 10.0) | Error 400: invalid discount type | Negative |
| **CTL-DSC-SRV-APP-005** | Apply discount fails - negative value | Product ID, discount payload (type: `fixed`, value: -5.0) | Error 400: invalid/negative value | Negative |
| **CTL-DSC-SRV-APP-006** | Apply discount fails - percentage > 100 | Product ID, discount payload (type: `percentage`, value: 105.0) | Error 400: percentage value cannot exceed 100 | Negative |
| **CTL-DSC-SRV-APP-007** | Apply discount fails - product not found | Non-existent product ID, valid discount payload | Error 404: "product not found" | Negative |
| **CTL-DSC-SRV-APP-008** | Apply discount fails - repo update error | Valid discount payload but repository write fails | Error 500: Database update failure | Negative |
| **CTL-DSC-SRV-REM-001** | Remove discount successfully | Existing product ID | Product discount set to `nil` in repository | Positive |
| **CTL-DSC-SRV-REM-002** | Remove discount fails - product not found | Non-existent product ID | Error 404: "product not found" | Negative |
| **CTL-DSC-SRV-REM-003** | Remove discount fails - repo update error | Valid product ID but repository write fails | Error 500: Database update failure | Negative |

### 2. HTTP Handler Integration & E2E Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **CTL-DSC-API-APP-001** | Apply discount success | POST `/api/products/:id/discount` with valid body | HTTP 200, success message | Positive |
| **CTL-DSC-API-APP-002** | Apply discount bad request | POST `/api/products/:id/discount` with invalid body schema | HTTP 400 Bad Request | Negative |
| **CTL-DSC-API-REM-001** | Remove discount success | DELETE `/api/products/:id/discount` | HTTP 200, success message | Positive |
| **CTL-DSC-API-REM-002** | Remove discount not found | DELETE `/api/products/:id/discount` with non-existent product ID | HTTP 404 Not Found | Negative |
| **CTL-DSC-E2E-FLOW-001** | E2E Discount flow lifecycle | Sequence: POST (apply) -> Verify discount details -> DELETE (remove) -> Verify removal | Verifies state changes persist properly through mock database stack | Positive |
