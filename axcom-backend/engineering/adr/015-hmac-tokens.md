# ADR-015: Customized HMAC-SHA256 Signed Security Token Authentication

**Date:** 2026-06-27  
**Status:** accepted

## Context
API endpoints behind security gateways require stateless request authentication to retrieve the consumer's identity (`userID`) and permissions (`role`). 

Standard RFC 7519 JSON Web Tokens (JWT) are commonly used but require importing full-featured third-party JWT libraries. These libraries have historically suffered from security vulnerabilities, such as "algorithm swapping" attacks (where an attacker signs a token using the `none` algorithm or uses a public key to sign a token expected to be signed via HMAC). Additionally, full JWT libraries represent large dependency footprints for simple claims-signing tasks.

## Decision
1. **Custom HMAC-SHA256 Token Format:** Implement a lightweight, custom token signature manager in `pkg/token`. Tokens are serialized as a simple dot-separated string (`base64URL(payload).signature`) signed using HMAC-SHA256 with a server-side secret key.
2. **Hardcoded Signature Algorithm:** The token verification parser hardcodes HMAC-SHA256 validation. It completely bypasses parsing dynamic header algorithm parameters (`alg`), making algorithm-swapping and `none`-algorithm spoofing attacks physically impossible.
3. **Internal Gateway Scope:** Bypassing full RFC 7519 compliance is acceptable because authentication tokens are issued and validated exclusively by this gateway service (internal session scope).

## Alternatives Considered

| Option | Reason Rejected |
|--------|-----------------|
| Standard JWT Libraries | Increases dependency audit footprint and exposes the project to parsing vulnerabilities and algorithm manipulation exploits. |
| Database-Backed Sessions | Querying Postgres or Redis on every HTTP request increases endpoint latency and introduces a single point of dependency failure for the authentication middleware. |

## Why This Choice
Hardcoding the token validation to a single signature method (HMAC-SHA256) eliminates the entire class of algorithm-spoofing attacks that plague standard JWT parsers. It allows the token manager implementation to be less than 150 lines of clear, zero-dependency, auditable Go code.

## Tradeoffs
**Gains:**
* Immunity to algorithm-swapping and spoofing vulnerabilities.
* High-performance, stateless validation with zero database calls.
* Zero third-party package dependencies.

**Accepts:**
* Tokens cannot be verified by standard third-party identity providers without utilizing this custom package.

## Consequences
* Internal service-to-service and client-to-service sessions must use this custom HMAC signature layout.
* External OIDC verification (e.g. for external federated authentication) is separated and handled via dedicated validator adapters in `pkg/token/oidc`.
