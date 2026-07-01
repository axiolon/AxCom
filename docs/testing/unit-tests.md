---
title: "Unit Tests"
description: "Go unit test conventions — in-package mocks, domain tests, controller tests with httptest, and how to run them."
sidebar_position: 7
---

# Unit Tests

<DocBadge status="under-review" version="v0.1.0-alpha" />

Unit tests are co-located with the code they test — a `*_test.go` file sits next to every service, controller, and domain package. They use only the standard library, `testify`, and hand-rolled in-package mocks. No containers, no network, no external dependencies.

---

## Running

```bash
# Run all unit tests (excludes e2e — no build tag)
go test ./...

# Run with verbose output
go test -v ./...

# Run a specific package
go test -v ./internal/core/auth/...

# Run a single test by name
go test -v -run TestController_Register ./internal/core/auth/...

# Run in parallel (already the default — all tests call t.Parallel())
go test -count=1 ./...
```

> The `-count=1` flag disables the test result cache, useful when verifying a fix.

---

## Test Layout

```
internal/core/<module>/
├── service.go
├── service_test.go       ← service unit tests + in-package mocks
├── controller.go
├── controller_test.go    ← controller unit tests (gin + httptest.NewRecorder)
└── domain/
    ├── stock.go
    └── stock_test.go     ← pure domain logic tests (table-driven)
```

Tests live in the **same package** as the code under test. This gives them access to unexported types and makes mocks trivial to define without an additional mock-generation step.

---

## Three Layers of Unit Tests

### 1. Domain Tests

Test pure business rules with no dependencies. These are the fastest tests in the suite.

```go
// internal/core/inventory/domain/stock_test.go
func TestReserve(t *testing.T) {
    tests := []struct {
        name        string
        initialQty  int
        reserveQty  int
        expectError error
        expectedQty int
    }{
        {"successful reservation", 10, 4, nil, 6},
        {"reserve exact stock quantity", 5, 5, nil, 0},
        {"insufficient stock", 5, 6, ErrInsufficientStock, 5},
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) { ... })
    }
}
```

All domain tests are **table-driven** — one `tests []struct` slice with named cases covers happy paths, edge cases, and error conditions in a single function.

---

### 2. Service Tests

Test service logic with in-package mock repositories. Mocks implement the repository interface declared in the same package.

```go
// In service_test.go — mock defined alongside tests
type MockUserRepository struct {
    mu           sync.RWMutex
    usersByID    map[string]*User
    usersByEmail map[string]*User
}

func (m *MockUserRepository) Create(ctx context.Context, user *User) error { ... }
func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*User, error) { ... }
```

Service tests wire the mock directly into the service constructor:

```go
func TestAuthService_Register(t *testing.T) {
    repo := NewMockUserRepository()
    svc  := NewAuthService(repo, NewMockTokenRepository(), jwtManager, &MockTxManager{})
    // test against svc...
}
```

---

### 3. Controller Tests

Test HTTP routing, request binding, and response serialisation using `gin.TestMode` + `httptest.NewRecorder`. The controller is wired against the same in-package mock service.

```go
func init() { gin.SetMode(gin.TestMode) }

func TestController_Register(t *testing.T) {
    t.Parallel()

    service    := NewAuthService(NewMockUserRepository(), ...)
    controller := NewController(service)

    router := gin.New()
    RegisterRoutes(router.Group("/api"), controller)

    w := performRequest(router, http.MethodPost, "/api/auth/register", map[string]string{
        "email":    "test@example.com",
        "password": "Password123!",
    })

    assert.Equal(t, http.StatusOK, w.Code)
}

// shared helper — marshal body, fire request, return recorder
func performRequest(r http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
    b, _ := json.Marshal(body)
    req, _ := http.NewRequest(method, path, bytes.NewBuffer(b))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    return w
}
```

---

## Conventions

| Convention | Detail |
|---|---|
| Package | Same package as production code (white-box) |
| Parallelism | All tests call `t.Parallel()` |
| Mocks | Hand-rolled in-package structs implementing the repository interface |
| Assertions | `testify/require` for fatal conditions; `testify/assert` for non-fatal checks |
| Table-driven | Domain and service tests use `[]struct` test tables with `t.Run` sub-tests |
| No mock gen | No `mockery` or `gomock` — mocks are small enough to write inline |

---

## Coverage

```bash
# Generate a coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Focus coverage on the service and domain layers. Controller tests validate binding and status codes; full business logic coverage belongs in service/domain tests.
