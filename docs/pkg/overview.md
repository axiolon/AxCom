---
title: pkg - Shared Utility Libraries
sidebar_label: Overview
sidebar_position: 1
---

# pkg — Shared Utility Libraries

<DocBadge status="under-review" version="v0.1.0-alpha" />

The `pkg/` directory contains **framework-agnostic, reusable Go packages** that provide cross-cutting infrastructure to the rest of the application. These packages have no knowledge of domain logic and can be imported by any layer (`internal/core`, `internal/gateway`, adapters, etc.) without creating circular dependencies.

---

## Package Index

| Package                    | Purpose                                        | Key Exports                                                |
| -------------------------- | ---------------------------------------------- | ---------------------------------------------------------- |
| [`ctxkeys`](./ctxkeys)     | Type-safe context key constants                | `UserIDKey`, `UserRoleKey`, `CorrelationIDKey`             |
| [`errors`](./errors)       | Domain-aware HTTP error type                   | `AppError`, `NewBadRequest`, `NewInternal`, …              |
| [`idgen`](./idgen)         | Cryptographically secure UUIDv7 ID generation  | `Generate`, `MustGenerate`, `ToUUID`, `FromUUID`           |
| [`logger`](./logger)       | Structured logging with OTel trace correlation | `Logger`, `NewLogger`, `ECSHandler`, context-aware methods |
| [`response`](./response)   | JSON API envelope + RFC 7807 Problem Details   | `APIResponse`, `ProblemDetail`, `GinOK`, `GinWriteError`   |
| [`telemetry`](./telemetry) | OpenTelemetry tracing bootstrap                | `Config`, `ReadConfigFromEnv`, `Init`                      |
| [`token`](./token)         | Internal HMAC token manager + OIDC validator   | `JWTManager`, `OIDCValidator`                              |
| [`validator`](./validator) | Input validation helpers                       | `ValidateStruct`, `ValidateEmail`, `ValidatePassword`      |
| [`db`](./db)               | Database connection interface and adapters     | `Connection`, `SQLConnection`, `MongoConnection`           |
| [`metrics`](./metrics)     | Prometheus metrics registration                | HTTP metrics, DB pool collector, cache metrics             |

---

## Design Principles

1. **No domain imports** — packages under `pkg/` never import from `internal/core` or `internal/infra` (with the exception of `metrics`, which references an infra interface for pool stats — see the [metrics doc](./metrics) for details).
2. **Minimal coupling** — each package is independently importable with no hard dependencies on sibling packages.
3. **Env-driven config** — runtime behaviour (log format, telemetry sampling, etc.) is controlled via environment variables.
4. **OTel-first observability** — logging and response writing integrate natively with OpenTelemetry trace context, attaching `trace_id` and `span_id` to every log record and error response automatically.
