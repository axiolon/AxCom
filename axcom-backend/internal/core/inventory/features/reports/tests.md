# Inventory Reports Feature Tests

This document tracks and describes the testing strategy and validation suite for the `reports` inventory feature.

---

## Overview

The `reports` inventory feature contains unit tests and HTTP handler integration/E2E tests to validate stock queries and CSV exports.
- **Unit Tests (`service_test.go`)**: Validates low stock retrieval filtering, CSV headers structure matching, and string formatting fields.
- **Integration & E2E Tests (`controller_test.go`)**: Validates router configurations, HTTP headers returned during attachments downloads, and E2E reports cycle flow on preset repository mock data.

---

## Reports Feature Test Suites

### 1. Service Layer Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-REP-SRV-LOW-001** | Get low stock items | Low stock items in repository | Slice of StockItem elements | Positive |
| **INV-REP-SRV-LOW-002** | Get low stock items error | Repository returning database error | Error 500: "failed to generate low stock report" | Negative |
| **INV-REP-SRV-CSV-001** | Export inventory CSV success | Populated repository state | Slice of bytes formatted as CSV, matching headers and values | Positive |
| **INV-REP-SRV-CSV-002** | Export inventory CSV empty | Empty repository state | Slice of bytes containing header row only | Positive |

### 2. HTTP Handler Integration & E2E Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **INV-REP-API-LOW-001** | Get low stock HTTP success | GET `/api/inventory/low-stock` | HTTP 200, matching items array envelope | Positive |
| **INV-REP-API-CSV-001** | Export CSV HTTP success | GET `/api/inventory/export` | HTTP 200, header `Content-Disposition: attachment; filename=inventory.csv` | Positive |
| **INV-REP-E2E-FLOW-001** | Complete E2E reports verify | Sequence of requests checking low-stock query and export bytes | Verifies CSV structure matches exact records | Positive |
