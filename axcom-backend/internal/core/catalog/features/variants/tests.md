# Product Variants Feature Tests

This document tracks and describes the testing strategy and validation suite for the `variants` catalog feature.

---

## Overview

The `variants` catalog feature manages specific product variants (size, color, SKU, price, etc.) for existing catalog products.
- **Unit Tests (`service_test.go`)**: Validates variant mutations, stock requirements (cannot delete the last variant), duplicate SKU checks, and model validation logic.
- **Integration Tests (`controller_test.go`)**: Validates Gin router endpoint mappings, request parameters, JSON body bindings, and E2E flows using a local HTTP mock environment.

---

## Variants Feature Test Suites

### 1. Service Layer Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **CTL-VAR-SRV-GET-001** | Get variants successfully | Existing product ID | List of all variants associated with the product | Positive |
| **CTL-VAR-SRV-GET-002** | Get variants fails - product not found | Non-existent product ID | Error 404: "product not found" | Negative |
| **CTL-VAR-SRV-ADD-001** | Add variant successfully | Valid variant data (without ID) | Variant is assigned a random `var_` ID and updated in repository | Positive |
| **CTL-VAR-SRV-ADD-002** | Add variant fails - product not found | Non-existent product ID, valid variant | Error 404: "product not found" | Negative |
| **CTL-VAR-SRV-ADD-003** | Add variant fails - duplicate SKU | Variant with a SKU that already exists on the product | Error 400: "duplicate SKU" | Negative |
| **CTL-VAR-SRV-ADD-004** | Add variant fails - repo update error | Valid variant but repository fails to update | Error 500: "failed to add variant" | Negative |
| **CTL-VAR-SRV-UPD-001** | Update variant successfully | Existing product ID, valid updated variant details | Variant details updated in repository | Positive |
| **CTL-VAR-SRV-UPD-002** | Update variant fails - missing variant ID | Updated variant without an ID field | Error 400: "Variant ID is required" | Negative |
| **CTL-VAR-SRV-UPD-003** | Update variant fails - variant not found | Product exists but variant ID is missing | Error 404: "variant ... not found" | Negative |
| **CTL-VAR-SRV-DEL-001** | Delete variant successfully | Product ID with multiple variants, variant ID to delete | Variant removed from product in repository | Positive |
| **CTL-VAR-SRV-DEL-002** | Delete variant fails - last remaining | Product with only 1 variant left | Error 400: "must have at least one variant" | Negative |
| **CTL-VAR-SRV-DEL-003** | Delete variant fails - variant not found | Existing product but non-existent variant ID | Error 404: "variant ... not found" | Negative |

### 2. HTTP Handler Integration & E2E Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **CTL-VAR-API-GET-001** | Get variants success | GET `/api/products/:id/variants` with valid product ID | HTTP 200, array of variants | Positive |
| **CTL-VAR-API-GET-002** | Get variants not found | GET `/api/products/:id/variants` with invalid product ID | HTTP 404 Not Found | Negative |
| **CTL-VAR-API-ADD-001** | Add variant success | POST `/api/products/:id/variants` with valid body | HTTP 200, returns added variant with generated ID | Positive |
| **CTL-VAR-API-ADD-002** | Add variant bad request | POST `/api/products/:id/variants` with missing required fields | HTTP 400 Bad Request | Negative |
| **CTL-VAR-API-UPD-001** | Update variant success | PUT `/api/products/:id/variants/:variant_id` with valid body | HTTP 200, variant updated successfully | Positive |
| **CTL-VAR-API-DEL-001** | Delete variant success | DELETE `/api/products/:id/variants/:variant_id` | HTTP 200, variant deleted | Positive |
| **CTL-VAR-E2E-FLOW-001** | Complete variant operations cycle | Multi-step request sequence: GET -> ADD -> UPDATE -> DELETE | Verifies complete state mutation lifecycle through DB | Positive |
