# Ecom Engine End-to-End Test Suite

This directory contains the production-grade end-to-end (E2E) test suite for the Ecom Engine API. Tests run against a **real MongoDB container** spun up on demand via [testcontainers-go](https://golang.testcontainers.org/), executing live HTTP requests against the fully-booted engine and router. No mocking is involved at any layer.

---

## Directory Structure

```
tests/e2e/
├── README.md               # This file
├── main_test.go            # TestMain — starts/stops the shared harness for the entire run
├── auth_test.go            # Auth module tests (register, login, refresh, logout, lockout, password reset)
├── catalog_test.go         # Catalog module tests (category CRUD, product CRUD, search/filter)
├── cart_test.go            # Cart module tests (add, get, update, remove, clear, count)
├── inventory_test.go       # Inventory module tests (update, check, list, adjust, reserve, bulk, transfer, history, delete)
├── orders_test.go          # Orders module tests (customer flow, guest checkout, admin management)
├── shipping_test.go        # Shipping module tests (rate calculation, admin CRUD, tracking, customer lookup)
├── dashboard_test.go       # Dashboard module tests (admin stats, RBAC)
└── testutil/
    └── harness.go          # Shared test harness (container, HTTP server, helpers)
```

---

## Prerequisites

| Requirement | Version | Notes |
| :--- | :--- | :--- |
| Go | ≥ 1.22 | Uses `range N` syntax |
| Docker | ≥ 20.10 | Must be running and accessible to the test process |
| MongoDB image | `mongo:7` | Pulled automatically by testcontainers on first run |

> **Note:** Docker Desktop on Windows/macOS or Docker Engine on Linux must be running before executing the suite. The container is pulled, started, and terminated automatically — no manual setup required.

---

## Running the Tests

E2E tests are gated behind the `e2e` build tag so they never run as part of the regular `go test ./...` command.

### Run all E2E tests

```bash
cd ecom-backend
go test -tags e2e -v ./tests/e2e/... -timeout 300s
```

### Run a specific module's tests

```bash
# Auth only
go test -tags e2e -v -run TestAuth ./tests/e2e/... -timeout 120s

# Catalog only
go test -tags e2e -v -run TestCatalog ./tests/e2e/... -timeout 120s

# Cart only
go test -tags e2e -v -run TestCart ./tests/e2e/... -timeout 120s

# Inventory only
go test -tags e2e -v -run TestInventory ./tests/e2e/... -timeout 120s

# Orders only
go test -tags e2e -v -run TestOrders ./tests/e2e/... -timeout 120s

# Shipping only
go test -tags e2e -v -run TestShipping ./tests/e2e/... -timeout 120s

# Dashboard only
go test -tags e2e -v -run TestDashboard ./tests/e2e/... -timeout 120s
```

### Run a single test function

```bash
go test -tags e2e -v -run TestOrders_GuestCheckout ./tests/e2e/... -timeout 120s
```

---

## Architecture

### Shared Harness

A single `testutil.Harness` is created once in `TestMain` and shared across the entire test binary run. It holds:

- A `mongo:7` replica-set container (required for multi-document transactions)
- A `httptest.Server` backed by the fully-wired `engine.Engine` + `gateway.Router`
- A direct `mongo.Client` for seeding and truncation operations that bypass the API

Starting the harness (container pull + engine boot) typically takes **15–30 seconds** on the first run. Subsequent runs reuse the pulled image and are faster.

### State Isolation

Every top-level `TestXxx` function calls `harness.Truncate(t, collections...)` at the start, which drops the relevant MongoDB collections. This guarantees:

- No state leaks between test functions
- Tests can run in any order
- Sub-tests within a function share state intentionally (sequential dependencies are declared with `require.NotEmpty`)

### Module Activation

The harness boots the engine with **all modules enabled** simultaneously. This mirrors a real production deployment and catches cross-module integration issues (e.g., cart referencing inventory stock, orders publishing events consumed by shipping).

| Module | Enabled in E2E | Features active |
| :--- | :---: | :--- |
| Auth | Always | — |
| Catalog | ✓ | Variants |
| Inventory | ✓ | Bulk, History, Reservation, Reports, Transfer, Adjustment, Sync |
| Cart | ✓ | — |
| Orders | ✓ | — |
| Shipping | ✓ | Flatrate provider (5.99) |
| Dashboard | ✓ | Small tier |
| Notifications | — | Off (no email/SMS infra in tests) |
| Payments | — | Off (no Stripe sandbox in tests) |

---

## Test Coverage

### Auth (`auth_test.go`)

| Test | What it verifies |
| :--- | :--- |
| `TestAuth_Register` | Customer/merchant registration, admin role rejection, weak password, duplicate email 409 |
| `TestAuth_LoginAndSessionFlow` | Token issuance, wrong password 401, refresh token rotation, logout + token revocation |
| `TestAuth_AccountLockout` | 5 failed attempts locks the account; correct password still rejected |
| `TestAuth_PasswordResetFlow` | Reset token returned in non-prod, unknown email still returns 200, confirm + re-login, token reuse rejected |
| `TestAuth_AdminSeedViaDirectInsert` | Admin users can only be created via direct DB seed (bypasses API role whitelist) |

### Catalog (`catalog_test.go`)

| Test | What it verifies |
| :--- | :--- |
| `TestCatalog_CategoryCRUD` | Public list, unauthed/non-admin create rejection, admin create/update/delete, list reflects changes |
| `TestCatalog_ProductCRUD` | Admin product create with variants, variant required validation, get by ID, 404 for unknown, list, category filter, update, delete |
| `TestCatalog_SearchProducts` | Full-text search by name, price range filter |

### Cart (`cart_test.go`)

| Test | What it verifies |
| :--- | :--- |
| `TestCart_AddAndGet` | Empty cart initially, unauthed add returns 401, add item, cart count (total + distinct) |
| `TestCart_UpdateAndRemove` | Update item quantity, remove item clears list |
| `TestCart_Clear` | Clear endpoint empties cart; subsequent GET returns empty items |

### Inventory (`inventory_test.go`)

| Test | What it verifies |
| :--- | :--- |
| `TestInventory_UpdateAndCheck` | Stock starts at zero, unauthed update returns 401, admin sets stock, public check reflects new quantity |
| `TestInventory_ListAndConfigure` | Admin list returns seeded stock, configure sets low-stock threshold |
| `TestInventory_Adjust` | Negative adjustment decrements, positive adjustment increments |
| `TestInventory_Reservation` | Reserve reduces available stock, release restores it |
| `TestInventory_BulkUpdate` | Bulk update sets multiple variants; check verifies final quantity |
| `TestInventory_Transfer` | Stock transfer from one location to another succeeds |
| `TestInventory_History` | History endpoint responds 200 after stock operations |
| `TestInventory_Delete` | Admin deletes stock record |

### Orders (`orders_test.go`)

| Test | What it verifies |
| :--- | :--- |
| `TestOrders_CustomerFlow` | Unauthed create returns 401, customer create (checks total), list own orders, get by ID, cancel transitions status to `cancelled` |
| `TestOrders_GuestCheckout` | Guest order requires `guest_info`, successful creation returns `order_id`, missing info returns 400 |
| `TestOrders_AdminManagement` | Admin lists all orders, gets any order, transitions status (`approve`), non-admin returns 403 |

### Shipping (`shipping_test.go`)

| Test | What it verifies |
| :--- | :--- |
| `TestShipping_CalculateRates` | Public rate calculation returns provider name + rate > 0 |
| `TestShipping_AdminCRUD` | Admin create shipment with tracking number, list shipments, update status to `in_transit`, public tracking by number reflects status, customer gets their order's shipment, non-admin create returns 403, admin delete |

### Dashboard (`dashboard_test.go`)

| Test | What it verifies |
| :--- | :--- |
| `TestDashboard_AdminStats` | Admin gets stats (`tier`, `revenue_today`, `orders_by_status`, `recent_orders`), unauthed returns 401, customer returns 403 |

---

## Harness Helper API

The `testutil.Harness` type exposes the following helpers for use in tests:

```go
// Do sends an HTTP request to the test server.
// token is optional; if non-empty it is sent as Bearer Authorization.
harness.Do(t, method, path, body, token) *http.Response

// Truncate drops the named collections to isolate test state.
harness.Truncate(t, "users", "orders", ...)

// SeedUser inserts a user directly into MongoDB, bypassing API role restrictions.
// The only way to create admin users in tests.
harness.SeedUser(t, email, password, role) string // returns user ID

// LoginAs issues POST /api/auth/login and returns (accessToken, refreshToken).
harness.LoginAs(t, email, password) (string, string)
```

```go
// Decode parses JSON from resp.Body into dst and closes the body.
testutil.Decode(t, resp, &dst)
```

---

## CI/CD Integration

The suite is designed to run inside CI pipelines with Docker-in-Docker (DinD) support. Add the following step to your pipeline after unit tests pass:

```yaml
# GitHub Actions example
- name: Run E2E Tests
  run: |
    cd ecom-backend
    go test -tags e2e -v ./tests/e2e/... -timeout 300s
  env:
    DOCKER_HOST: unix:///var/run/docker.sock
```

> **Timeout guidance:** Allow at least **5 minutes** (`-timeout 300s`). The first run on a cold CI runner may need longer if the `mongo:7` image is not cached.

### Recommended pipeline order

```
Unit Tests (go test ./...) → E2E Tests (-tags e2e) → Load Tests (k6)
```

---

## Adding Tests for a New Module

1. Create `tests/e2e/<module>_test.go` with build tag `//go:build e2e` and `package e2e`.
2. Declare the collections your module touches as a package-level `var` slice.
3. Call `harness.Truncate(t, yourCollections...)` at the top of every top-level `TestXxx` function.
4. If the module requires a new flag in the harness config, update `testutil/harness.go` — `buildConfig` reads from the `enabled` map.
5. Enable the module in `main_test.go` by adding its name to the `testutil.New(ctx, ...)` call.

---

## Common Issues

| Symptom | Likely cause | Fix |
| :--- | :--- | :--- |
| `Cannot connect to the Docker daemon` | Docker is not running | Start Docker Desktop / Engine |
| `context deadline exceeded` during container start | Slow network pulling `mongo:7` | Pre-pull with `docker pull mongo:7`, or increase `-timeout` |
| `failed to start harness` | Port conflict or replica-set init failure | Ensure no other MongoDB is using the same random port; retry |
| Test panics on nil `harness` | Build tag missing — test ran without `-tags e2e` | Always include `-tags e2e` |
| 401 on seeded admin endpoints | `SeedUser` called before `Truncate` in a previous test leaked the user | Check `Truncate` call includes `"users"` collection |
