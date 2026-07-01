---
title: "Authentication Module"
description: "Architecture, flows, and integration guide for the AxCom Authentication module — registration, login, JWT session management, and role-based access."
sidebar_label: Authentication
sidebar_position: 1
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

# Authentication Module

The Authentication module provides secure user authentication, registration, and session token generation for AxCom.

## Overview

- **User Registration**: Create new user accounts (defaults to the `customer` role if unspecified) with secure validation and password hashing.
- **Secure Password Hashing**: Passwords are encrypted using `bcrypt` before storage.
- **User Login**: Validate user credentials and return an active JWT session.
- **Role-Based Tokens**: Session tokens embed user roles to authorize operations downstream.
- **Input Validation**: Enforce validation rules (valid email format, minimum 8-character password with at least one letter and one number).
- **Custom Sentinel Errors**: Distinguish error conditions clearly (`ErrEmailAlreadyExists`, `ErrInvalidCredentials`, `ErrUserNotFound`).
- **Activity Logging**: Track request activities, successes, and failures using structured log levels.

---

## Architecture

```mermaid
flowchart LR
    A[HTTP Request] --> B[Auth Handler]
    B --> C[Auth Service]
    C --> D[UserRepository]
    C --> E[TokenRepository]
    C --> F[JWT Manager]
    D --> G[(Auth DB)]
    E --> G
    F --> H[JWT Token Generation / Validation]
    C --> I[Response]
```

- `Auth Handler` receives credentials and validates payloads.
- `Auth Service` contains business rules for registration, login, token refresh, password recovery, and logout.
- `UserRepository` and `TokenRepository` are storage contracts for user and token persistence.
- `JWT Manager` handles signing and verifying access tokens.

---

## Module Structure

| File            | Role                                                     |
| :-------------- | :------------------------------------------------------- |
| `handler.go`    | HTTP controllers — validates requests, encodes responses |
| `service.go`    | Core business logic; exposes the `Service` interface     |
| `model.go`      | Data schemas: `User`, `Session`                          |
| `repository.go` | `UserRepository` storage contract                        |
| `errors.go`     | Domain-specific sentinel errors                          |

---

## Database Design

```mermaid
erDiagram
    USERS {
        string id PK
        string email
        string password_hash
        string role
        datetime created_at
        datetime updated_at
    }
    REFRESH_TOKENS {
        string token PK
        string user_id FK
        datetime expires_at
        bool revoked
        datetime created_at
    }
    PASSWORD_RESET_TOKENS {
        string token PK
        string user_id FK
        datetime expires_at
        bool used
        datetime created_at
    }

    USERS ||--o{ REFRESH_TOKENS : has
    USERS ||--o{ PASSWORD_RESET_TOKENS : has
```

---

## What this module needs

- A `UserRepository` implementation to persist and retrieve users.
- A `TokenRepository` implementation for refresh tokens and password reset tokens.
- A secure password hashing mechanism (`bcrypt` or equivalent).
- A JWT manager for generating and validating access tokens.
- Structured application errors for invalid credentials, unauthorized access, and duplicate accounts.
- Request validation for email format, password strength, and required fields.

---

## Usage

Handlers rely on the `Service` interface for all tasks, allowing the service layer to be mocked or replaced easily.

```go
authService := auth.NewAuthService(userRepo, jwtManager)
authHandler := auth.NewAuthHandler(authService)
```
