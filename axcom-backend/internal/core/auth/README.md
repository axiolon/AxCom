# Authentication Module

Provides secure user registration, login, JWT session management, and role-based token generation.

## Quick Links

- [Full Documentation](../../../../../docs/modules/auth.md)
- [Tests](./tests.md)

## Directory Layout

| File | Role |
| :--- | :--- |
| `handler.go` | HTTP controllers — validation and response encoding |
| `service.go` | Core business logic; `Service` interface |
| `model.go` | `User`, `Session` data schemas |
| `repository.go` | `UserRepository` storage contract |
| `errors.go` | Domain-specific sentinel errors |