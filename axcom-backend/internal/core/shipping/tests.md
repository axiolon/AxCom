# Shipping Module Tests

This document tracks and describes the testing strategy and validation suite for the `shipping` module, including both client-facing and administrative shipping routes.

---

## Overview

The `shipping` module contains unit tests for core services and HTTP route handler integration tests.
- **Service Unit Tests (`service_test.go`)**: Validates shipping rate calculations, shipment creation, tracking updates, and provider matching logic.
- **HTTP Handler Integration Tests (`controller_test.go` and `admin/controller_test.go`)**: Validates incoming payload parsing, query parameter binding, user context extraction, and error propagation.

---

## Test Suites

### 1. Customer Shipping API (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **SHPM-API-RAT-001** | Calculate rates successfully | Valid request payload (weight + value) | HTTP 200, returns list of shipping rates | Positive |
| **SHPM-API-RAT-002** | Calculate rates validation error | Missing or invalid weight | HTTP 400 Bad Request | Negative |
| **SHPM-API-RAT-003** | Calculate rates service error | Valid payload, service fails | HTTP 500 Internal Server Error | Negative |
| **SHPM-API-ORD-001** | Get order shipment successfully | Valid order ID and authorized customer context | HTTP 200, returns active shipment details | Positive |
| **SHPM-API-ORD-002** | Get order shipment unauthorized | Missing customer ID in context | HTTP 401 Unauthorized | Negative |
| **SHPM-API-ORD-003** | Get order shipment not found | Non-existent order ID | HTTP 404 Not Found | Negative |

### 2. Admin Shipping API (`admin/controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **SHPM-ADM-API-LST-001** | List all shipments successfully | GET request | HTTP 200, returns list of all shipments | Positive |
| **SHPM-ADM-API-LST-002** | List all shipments service error | GET request, service fails | HTTP 500 Internal Server Error | Negative |
| **SHPM-ADM-API-CRT-001** | Create shipment successfully | Valid OrderID, Carrier, Weight, Value | HTTP 200, returns created shipment details | Positive |
| **SHPM-ADM-API-CRT-002** | Create shipment validation error | Missing OrderID, Carrier, or Weight | HTTP 400 Bad Request | Negative |
| **SHPM-ADM-API-CRT-003** | Create shipment service error | Valid payload, service fails | HTTP 500 Internal Server Error | Negative |
| **SHPM-ADM-API-UPD-001** | Update shipment status successfully | Valid shipment ID and Status | HTTP 200, returns updated shipment details | Positive |
| **SHPM-ADM-API-UPD-002** | Update shipment status validation error | Missing status in request payload | HTTP 400 Bad Request | Negative |
| **SHPM-ADM-API-UPD-003** | Update shipment status not found | Unknown shipment ID | HTTP 404 Not Found | Negative |
| **SHPM-ADM-API-UPD-004** | Update shipment status service error | Valid input, service fails | HTTP 500 Internal Server Error | Negative |

---

## Running the Tests

To run the full suite for the shipping package:

```bash
go test -v ./internal/core/shipping/...
```

To run with coverage calculation:

```bash
go test -coverprofile=coverage.out ./internal/core/shipping/...
go tool cover -func=coverage.out
```
