# Core Catalog Feature Tests

This document tracks and describes the testing strategy and validation suite for the `core` catalog feature.

---

## Overview

The `core` catalog feature contains unit tests and HTTP handler integration tests to validate basic product and category management.
- **Unit Tests (`service_test.go`)**: Validates core business logic inside `catalogService`. It mocks the repository and event bus to isolate and check constraints, validation rules, and stock event handling.
- **Integration Tests (`controller_test.go`)**: Validates HTTP controllers, request bindings, status codes, and error translation using a local Gin HTTP router instance in `TestMode` and `httptest.ResponseRecorder` objects.

---

## Core Feature Test Suites

### 1. Service Layer Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **CTL-CORE-SRV-PRD-001** | Create product successfully | Valid product details, valid category ID | Product saved, auto-generated ID and variant IDs | Positive |
| **CTL-CORE-SRV-PRD-002** | Create product fails - invalid name | Product with empty name | Error: "product name is required" | Negative |
| **CTL-CORE-SRV-PRD-003** | Create product fails - negative price | Product with variant price < 0 | Error: "variant price must be non-negative" | Negative |
| **CTL-CORE-SRV-PRD-004** | Create product fails - category not found | Product with non-existent category ID | Error: category with ID ... not found | Negative |
| **CTL-CORE-SRV-PRD-005** | Update product successfully | Valid product update details | Product updated in repository | Positive |
| **CTL-CORE-SRV-PRD-006** | Update product fails - missing ID | Product update request with empty ID | Error: Product ID is required for update | Negative |
| **CTL-CORE-SRV-PRD-007** | Delete product successfully | Existing product ID | Product deleted from repository | Positive |
| **CTL-CORE-SRV-PRD-008** | Delete product fails - missing ID | Empty product ID | Error: Product ID is required for deletion | Negative |
| **CTL-CORE-SRV-CAT-001** | Create category successfully | Valid category details | Category created, slug auto-generated if empty | Positive |
| **CTL-CORE-SRV-CAT-002** | Create category fails - invalid name | Category with empty name | Error: "category name is required" | Negative |
| **CTL-CORE-SRV-CAT-003** | Create category fails - parent not found | Category referencing non-existent parent | Error: parent category not found | Negative |
| **CTL-CORE-SRV-CAT-004** | Update category successfully | Valid category details, unique slug | Category updated in repository | Positive |
| **CTL-CORE-SRV-CAT-005** | Update category fails - self parent | Category setting parent ID to its own ID | Error: Category cannot be its own parent | Negative |
| **CTL-CORE-SRV-CAT-006** | Delete category successfully | Existing category with no assigned products/children | Category deleted | Positive |
| **CTL-CORE-SRV-CAT-007** | Delete category fails - products assigned | Category ID used by active products | Conflict error: cannot delete category | Negative |
| **CTL-CORE-SRV-CAT-008** | Delete category fails - children exist | Category ID used as parent by subcategory | Conflict error: cannot delete category | Negative |
| **CTL-CORE-SRV-LST-001** | List products with query filtering | Filters like Category ID, InStock, Search query | List of matching ProductResponse models | Positive |
| **CTL-CORE-SRV-EVT-001** | Synchronize stock on event | InventoryStockChanged event published to bus | Updates matching variant stock in catalog repository | Positive |

### 2. HTTP Handler Integration Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **CTL-CORE-API-PRD-001** | Get products success | GET `/products` with / without query params | HTTP 200, array of products | Positive |
| **CTL-CORE-API-PRD-002** | Get single product success | GET `/products/:id` with existing ID | HTTP 200, product details | Positive |
| **CTL-CORE-API-PRD-003** | Get single product not found | GET `/products/:id` with invalid ID | HTTP 404 Not Found | Negative |
| **CTL-CORE-API-PRD-004** | Create product success (authenticated) | POST `/products` with valid JSON body | HTTP 200, created product | Positive |
| **CTL-CORE-API-PRD-005** | Create product bad request | POST `/products` with invalid JSON body | HTTP 400 Bad Request | Negative |
| **CTL-CORE-API-PRD-006** | Update product success (authenticated) | PUT `/products/:id` with valid JSON body | HTTP 200, updated product | Positive |
| **CTL-CORE-API-PRD-007** | Delete product success (authenticated) | DELETE `/products/:id` | HTTP 200, delete message | Positive |
| **CTL-CORE-API-CAT-001** | Get categories success | GET `/categories` | HTTP 200, list of categories | Positive |
| **CTL-CORE-API-CAT-002** | Create category success (authenticated) | POST `/categories` with valid JSON | HTTP 200, created category | Positive |
| **CTL-CORE-API-CAT-003** | Delete category conflict | DELETE `/categories/:id` when products assigned | HTTP 409 Conflict | Negative |
