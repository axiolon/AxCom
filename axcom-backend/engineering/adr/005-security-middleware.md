# ADR-005: Security Headers Middleware Configuration

**Date:** 2026-06-27  
**Status:** accepted

## Context
The backend gateway implements `SecurityHeadersMiddleware` to attach standard HTTP security headers (HSTS, CSP, X-Frame-Options, etc.) to all outgoing HTTP responses. 

Since this gateway functions primarily as a JSON REST API rather than an interactive website, many of these browser-oriented security headers (like Content Security Policy (CSP), `X-Frame-Options`, and `Permissions-Policy`) are technically ignored by API clients (mobile apps, direct backend requests). However:
1. They are critical for **defense-in-depth** (e.g., if a browser user opens an endpoint directly, or if an endpoint accidentally returns user-controlled HTML/script content).
2. Standard security compliance scanners flag the absence of these headers as a vulnerability.

Additionally, specifying browser restrictions such as `connect-src 'self' https://api.example.com;` inside Go code would be rigid and environment-dependent, breaking when moved between local development, staging, and production environments.

## Decision
1. **Retain Defense-in-Depth Headers:** Continue injecting HSTS, X-Content-Type-Options, X-Frame-Options, CSP, Referrer-Policy, and Permissions-Policy globally on all router endpoints.
2. **Externalize CSP Directives:** Do not hardcode specific domains or external resources (e.g. `connect-src`) in the codebase.
3. **Environment-Driven Configuration:** Bind CSP dynamically via the `CSP_DIRECTIVES` environment variable, falling back to a strict local-only policy (`default-src 'self'`) if undefined:
   ```go
   if csp == "" {
       csp = os.Getenv("CSP_DIRECTIVES")
       if csp == "" {
           csp = "default-src 'self'"
       }
   }
   ```

## Alternatives Considered

| Option | Reason Rejected |
|--------|-----------------|
| Remove browser-centric headers for pure APIs | Exposes endpoints to vulnerabilities if they serve user-supplied assets/HTML by accident. Fails automated security audits. |
| Hardcode environment-specific CSP strings in Go code | Brittle. Requires modifying and building source code to run in different deployment regions or environments. |

## Why This Choice
This approach secures the gateway against edge-case web vulnerabilities and satisfies automated security tools, while keeping configuration separate from application code in accordance with the Twelve-Factor App methodology.

## Tradeoffs
**Gains:**
* Defense-in-depth protection for browser-facing actions.
* Zero code changes required when deploying to a new environment or using a new external API domain.
* Standard compliance alignment out of the box.

**Accepts:**
* Small network payload overhead (attaching security headers on every single JSON response).

## Consequences
* Operators must configure `CSP_DIRECTIVES` in the target deployment environment (e.g. via Kubernetes config, Docker Compose, or `.env` files) if the gateway needs to load or link to external assets/APIs.
