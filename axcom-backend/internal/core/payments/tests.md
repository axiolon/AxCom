# Payments Module Tests

This document tracks and describes the testing strategy and validation suite for the `payments` module, including both customer-facing and administrative payments routes.

---

## Overview

The `payments` module contains unit tests for core services and HTTP route handler integration tests.
- **Service Unit Tests (`service_*_test.go`)**: Validates payment intent creation, confirmation, refunds, fetching, and listing across multiple focused test files, verifying state machine transitions and event publishing.
- **HTTP Handler Integration Tests (`controller_test.go` and `admin/controller_test.go`)**: Validates incoming payload parsing, route parameter binding, customer context extraction, error mapping, and authentication.

---

## Test Suites

### 1. Payments Service (`service_*_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **PMT-SRV-CRT-001** | Create payment intent successfully | Valid OrderID, CustomerID, Provider | Payment record created with status pending, provider intent ID set | Positive |
| **PMT-SRV-CRT-002** | Create payment intent order not found | Invalid OrderID | ErrOrderNotFound | Negative |
| **PMT-SRV-CRT-003** | Create payment intent order not pending | Order in paid or shipped status | ErrInvalidOrderStatus | Negative |
| **PMT-SRV-CRT-004** | Create payment intent provider not found | Invalid provider name | ErrProviderNotFound | Negative |
| **PMT-SRV-CRT-005** | Create payment intent provider error | Provider API error | Error returned, payment not created | Negative |
| **PMT-SRV-CNF-001** | Confirm payment success | Valid Provider, IntentID, Success=true | Payment status set to succeeded, success event published | Positive |
| **PMT-SRV-CNF-002** | Confirm payment failure | Valid Provider, IntentID, Success=false | Payment status set to failed, failure event published | Positive |
| **PMT-SRV-CNF-003** | Confirm payment not found | Invalid provider/intent ID | ErrPaymentNotFound | Negative |
| **PMT-SRV-CNF-004** | Confirm payment already finalized | Already Succeeded/Refunded | No-op, returns existing payment directly | Positive |
| **PMT-SRV-CNF-005** | Confirm payment provider error | Provider API error on confirmation | Error returned | Negative |
| **PMT-SRV-RFD-001** | Refund payment successfully | Valid OrderID for Succeeded payment | Payment status set to refunded | Positive |
| **PMT-SRV-RFD-002** | Refund payment not found | Order has no payment record | ErrPaymentNotFound | Negative |
| **PMT-SRV-RFD-003** | Refund payment not succeeded | Payment status is pending/failed | Error returned ("cannot refund payment in status...") | Negative |
| **PMT-SRV-RFD-004** | Refund payment provider error | Provider API error on refund | Error returned | Negative |
| **PMT-SRV-GET-001** | Get payment by OrderID success | Existing Order ID | Returns payment record | Positive |
| **PMT-SRV-GET-002** | Get payment by OrderID not found | Non-existent Order ID | ErrPaymentNotFound | Negative |
| **PMT-SRV-LST-001** | List all payments success | Any call | Returns slice of all payments | Positive |

### 2. Customer Payments API (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **PMT-API-CRT-001** | Create intent successfully | Valid payload, authenticated customer | HTTP 200, returns payment response | Positive |
| **PMT-API-CRT-002** | Create intent unauthorized | Authenticated customer missing | HTTP 401 Unauthorized | Negative |
| **PMT-API-CRT-003** | Create intent validation error | Missing or invalid OrderID | HTTP 400 Bad Request | Negative |
| **PMT-API-CRT-004** | Create intent order not found | Order not found in catalog | HTTP 404 Not Found | Negative |
| **PMT-API-CRT-005** | Create intent order not pending | Order already paid/shipped | HTTP 400 Bad Request | Negative |
| **PMT-API-CRT-006** | Create intent internal service error | Service returns unexpected error | HTTP 500 Internal Server Error | Negative |
| **PMT-API-CBK-001** | Process callback successfully | Valid provider in path, intent ID, success flag | HTTP 200, callback processed response | Positive |
| **PMT-API-CBK-002** | Process callback validation error | Missing intent_id in payload | HTTP 400 Bad Request | Negative |
| **PMT-API-CBK-003** | Process callback payment not found | Payment not matching provider/intent ID | HTTP 404 Not Found | Negative |
| **PMT-API-CBK-004** | Process callback service error | Service returns unexpected error | HTTP 500 Internal Server Error | Negative |

### 3. Admin Payments API (`admin/controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **PMT-ADM-API-LST-001** | List all payments successfully | GET request | HTTP 200, returns payment list and count | Positive |
| **PMT-ADM-API-LST-002** | List all payments service error | GET request, service fails | HTTP 500 Internal Server Error | Negative |
| **PMT-ADM-API-RFD-001** | Refund payment successfully | Valid OrderID payload | HTTP 200, refund success message and payment | Positive |
| **PMT-ADM-API-RFD-002** | Refund payment validation error | Missing order_id in payload | HTTP 400 Bad Request | Negative |
| **PMT-ADM-API-RFD-003** | Refund payment not found | No payment associated with order ID | HTTP 404 Not Found | Negative |
| **PMT-ADM-API-RFD-004** | Refund payment service error | Service returns other error (e.g. status not succeeded, provider error) | HTTP 500 Internal Server Error | Negative |

---

## Running the Tests

To run the full suite for the payments package:

```bash
go test -v ./internal/core/payments/...
```

To run with coverage calculation:

```bash
go test -coverprofile=coverage.out ./internal/core/payments/...
go tool cover -func=coverage.out
```
