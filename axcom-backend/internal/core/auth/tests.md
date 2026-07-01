# Authentication Module Tests

This document tracks and describes the testing strategy and validation suite for the `auth` module.

---

## Overview

The `auth` module contains both unit tests and handler integration tests to ensure secure and correct operations.
- **Unit Tests (`service_test.go`)**: Validates core business logic inside `authService`. It mocks dependencies for user databases, token databases, and JWT generation, checking password hashing algorithms (`bcrypt`) and login validations directly.
- **Integration Tests (`controller_test.go`)**: Validates HTTP controllers and payload binding validation using a local Gin HTTP router instance in `TestMode` and `httptest.ResponseRecorder` objects.

---

## Test Suites

### 1. Service Layer Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **AUTH-SRV-REG-001** | Register new user successfully | Valid email, password, and role | User created, hashed password stored | Positive |
| **AUTH-SRV-REG-002** | Register duplicate email | Existing email address | Conflict error: "email already registered" | Negative |
| **AUTH-SRV-LOG-001** | Log in successfully | Correct email and password | Returns session with non-empty Access & Refresh tokens | Positive |
| **AUTH-SRV-LOG-002** | Log in with wrong password | Valid email, incorrect password | Unauthorized error: "invalid email or password" | Negative |
| **AUTH-SRV-LOG-003** | Log in with non-existent email | Unknown email | Unauthorized error: "invalid email or password" | Negative |
| **AUTH-SRV-OUT-001** | Log out successfully | Valid active refresh token | Refresh token marked revoked (revoked = true) | Positive |
| **AUTH-SRV-REF-001** | Refresh session successfully | Active refresh token | Returns new Access & Refresh tokens, old token revoked | Positive |
| **AUTH-SRV-REF-002** | Refresh with revoked token | Revoked refresh token | Unauthorized error: "refresh token has been revoked" | Negative |
| **AUTH-SRV-REF-003** | Refresh with expired token | Expired refresh token | Unauthorized error: "refresh token has expired" | Negative |
| **AUTH-SRV-RST-001** | Request password reset | Registered email address | Returns valid generated password reset token | Positive |
| **AUTH-SRV-RST-002** | Request reset for unknown email | Unregistered email | Error: "email address not registered" | Negative |
| **AUTH-SRV-RST-003** | Confirm password reset | Valid reset token + new password | User password updated, reset token marked used | Positive |
| **AUTH-SRV-RST-004** | Confirm reset with used token | Already consumed reset token | Error: "password reset token already used" | Negative |

### 2. HTTP Handler Integration Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **AUTH-API-REG-001** | Successful registration | Valid payload | HTTP 200, user details in envelope data | Positive |
| **AUTH-API-REG-002** | Validation error (weak password) | Weak/invalid password payload | HTTP 400 Bad Request | Negative |
| **AUTH-API-REG-003** | Duplicate conflict registration | Pre-existing email payload | HTTP 409 Conflict | Negative |
| **AUTH-API-LOG-001** | Successful login | Correct credentials | HTTP 200, returns active session tokens | Positive |
| **AUTH-API-LOG-002** | Invalid login credentials | Incorrect password | HTTP 401 Unauthorized | Negative |
| **AUTH-API-REF-001** | Successful token refresh | Valid rotated refresh token | HTTP 200, returns new session tokens | Positive |
| **AUTH-API-OUT-001** | Successful logout | Active refresh token | HTTP 200, invalidates session | Positive |
| **AUTH-API-RST-001** | Request password reset success | Registered email | HTTP 200, returns reset token in envelope | Positive |
| **AUTH-API-RST-002** | Confirm password reset success | Reset token + new password | HTTP 200, login works with new password | Positive |

---

## Running the Tests

To run the full suite for the auth package:

```bash
go test -v ./internal/core/auth/...
```

To run with coverage calculation:

```bash
go test -coverprofile=coverage.out ./internal/core/auth/...
go tool cover -func=coverage.out
```
