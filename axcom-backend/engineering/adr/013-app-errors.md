# ADR-013: Domain-Aware Application Error Architecture

**Date:** 2026-06-27  
**Status:** accepted

## Context
Standard Go errors are simple strings that lack context. In a web API environment:
- We need to associate errors with specific HTTP status codes (e.g. 400 Bad Request, 404 Not Found).
- We must provide user-friendly error messages to clients without leaking sensitive system details (e.g. database schema errors, network timeouts).
- We need a standardized error format (such as RFC 7807 Problem Details) so client integrations can handle exceptions consistently.

Manually parsing and writing error responses in every controller leads to massive code duplication and inconsistent API behaviors.

## Decision
1. **Define a Custom AppError Struct:** Implement a standardized `AppError` type in `pkg/errors`:
   - `Code`: The HTTP status code written to the response header.
   - `Message`: A user-safe description of what went wrong.
   - `Err`: The underlying technical error (cause), retained server-side for logging.
   - `Type`: A URI identifying the problem type per [RFC 7807](https://datatracker.ietf.org/doc/html/rfc7807).
2. **Provide Standard Constructors:** Export predefined factories like `NewBadRequest`, `NewNotFound`, `NewForbidden`, and `NewInternal` to ensure uniform status-to-message mapping.
3. **Centralized Error Serialization:** Implement `WriteError` (for net/http) and `GinWriteError` (for Gin) in `pkg/response`:
   - If the error is an `*AppError`, write the custom message and HTTP code to the response and log the internal `Err` fields.
   - If it is a generic Go error, log it as an unexpected failure and return a secure, generic `500 Internal Server Error` to the client.

## Alternatives Considered

| Option | Reason Rejected |
|--------|-----------------|
| In-line Controller Error Writing | Writing JSON error payloads directly in Gin/HTTP handler files. This results in highly duplicate code and leads to drift in error formats across different modules. |
| Returning Raw Error Strings | Directly returning `err.Error()` to API clients. This exposes database credentials, table schemas, and internal stack traces, presenting a severe security vulnerability. |

## Why This Choice
This architecture guarantees that all API error responses follow the RFC 7807 standard, making client error-handling predictable. It keeps sensitive infrastructure details private while providing developers with full context in server-side logs via OTel trace-correlated logging.

## Tradeoffs
**Gains:**
* Consistent public API contract conforming to RFC 7807.
* Prevention of sensitive data leakage to API consumers.
* Simplified controller logic: controllers just pass errors to `response.GinWriteError`.

**Accepts:**
* Necessity of wrapping lower-layer errors into `AppError` at boundary points.

## Consequences
* Modules must utilize `pkg/errors` constructor helpers when generating client-facing errors.
* HTTP controllers must delegate error responses to `response.GinWriteError` or `response.WriteError` instead of rendering custom JSON error payloads.
