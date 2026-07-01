# Cart & Cart Merge Testing Documentation

This document covers details on executing and understanding the test suites written for the `cart` and `cart/merge` modules.

## Overview

We have implemented a comprehensive, thread-safe, and parallel unit/integration test suite covering:
1. **Cart Service**: Core business logic (adding, updating, removing, clearing, counting, and calculating product discounts/image fallbacks).
2. **Cart HTTP Controller**: Routes integration verifying request mapping, parameters, payload validations, and authentication contexts.
3. **Merge Service**: Guest cart merging behavior, item combination logic, and clean up.
4. **Merge HTTP Controller**: Merge request binding and workflow validations.

## Test Suites

### 1. Cart Service Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **CART-SRV-GET-001** | Get cart empty for new customer | Customer ID with no saved cart | Returns empty cart response with correct CustomerID | Positive |
| **CART-SRV-ADD-001** | Add new item successfully | CartItem with valid variant & quantity | Returns cart containing item; 10% percentage discount applied | Positive |
| **CART-SRV-ADD-002** | Add item missing variant ID | Empty VariantID | Validation error returned | Negative |
| **CART-SRV-ADD-003** | Add item with quantity <= 0 | Quantity = 0 | Validation error returned | Negative |
| **CART-SRV-ADD-004** | Add item exceeding stock | Quantity = 11 (max stock is 10) | Insufficient stock error returned | Negative |
| **CART-SRV-UPD-001** | Update item quantity successfully | Valid variant & quantity within stock | Item quantity updated in cart | Positive |
| **CART-SRV-UPD-002** | Update item exceeding stock | Quantity = 15 | Insufficient stock error returned | Negative |
| **CART-SRV-RMV-001** | Remove variant from cart | Valid variant ID | Variant item is deleted from the cart | Positive |
| **CART-SRV-CLR-001** | Clear cart successfully | Customer ID | Saved cart deleted; subsequent reads return empty cart | Positive |
| **CART-SRV-CNT-001** | Count items in cart | Cart with multiple items/quantities | Returns correct sum of all item quantities | Positive |
| **CART-SRV-ENR-001** | enrichCart fixed discount fallback | Product with $15 fixed discount | Discounted price resolves to original price minus $15 | Positive |
| **CART-SRV-ENR-002** | enrichCart image fallback | Product with fallback/non-primary images | Correctly selects first available image | Positive |

### 2. Cart HTTP Controller Integration Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **CART-API-GET-001** | Get cart unauthorized | Request without UserID context | HTTP 401 Unauthorized | Negative |
| **CART-API-GET-002** | Get cart successful | Request with UserID context | HTTP 200, returns envelope with active cart | Positive |
| **CART-API-ADD-001** | Add item successfully | Valid body with variant + qty | HTTP 200, item added | Positive |
| **CART-API-ADD-002** | Add item validation error | Missing variant ID or qty <= 0 | HTTP 400 Bad Request | Negative |
| **CART-API-UPD-001** | Update item successfully | Valid path param + body qty | HTTP 200, item quantity updated | Positive |
| **CART-API-RMV-001** | Remove item successfully | Valid variant path parameter | HTTP 200, item removed | Positive |
| **CART-API-CLR-001** | Clear cart successfully | Authorized request | HTTP 200, returns "cart cleared" message | Positive |
| **CART-API-CNT-001** | Get count successfully | Authorized request | HTTP 200, returns total item count | Positive |

### 3. Cart Merge Service Unit Tests (`merge/service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **MRG-SRV-MRG-001** | Merge guest cart - guest not found | Non-existent guest cart ID | Returns account cart unchanged | Positive |
| **MRG-SRV-MRG-002** | Merge guest cart - new account cart | Existing guest cart ID | Creates new account cart filled with guest items | Positive |
| **MRG-SRV-MRG-003** | Merge guest cart - overlapping items | Guest & account carts have same variant | Quantities of shared variant are summed | Positive |

### 4. Cart Merge HTTP Controller Integration Tests (`merge/controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **MRG-API-MRG-001** | Merge cart unauthorized | Request without UserID context | HTTP 401 Unauthorized | Negative |
| **MRG-API-MRG-002** | Merge cart missing guest ID | Empty guest_cart_id body | HTTP 400 Bad Request | Negative |
| **MRG-API-MRG-003** | Merge cart successful | Valid UserID + guest_cart_id | HTTP 200, merges and returns merged cart | Positive |

## Running Tests

Run the full cart test suite with the following command:

```bash
go test -v ./internal/core/cart/...
```

To run with coverage profiling:

```bash
go test -v -coverprofile=coverage.out ./internal/core/cart/...
```

## Structure of Tests

- **Parallelism**: Every test suite and individual subtest leverages `t.Parallel()` to run concurrently.
- **Thread Safety**: All mock services and mock repositories use `sync.RWMutex` to guard state during concurrent assertions.
- **Assertions**: Handled using `github.com/stretchr/testify/assert` and `github.com/stretchr/testify/require`.
