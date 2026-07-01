---
title: "Security & CORS Configurations"
description: "Details security headers, CORS origin mapping, and transport security controls."
sidebar_position: 3
---

# Security & CORS Configurations

<DocBadge status="under-review" version="v0.1.0-alpha" />

The API Gateway enforces security headers, payload boundaries, read timeouts, and Cross-Origin Resource Sharing (CORS) rules to secure the application.

---

## 1. HTTP Server Header Protections

At the server bootstrapping layer in `cmd/server/main.go`, specific boundaries are set on HTTP headers to prevent denial-of-service (DoS) attacks:

```go
srv := &http.Server{
    Addr:              ":" + cfg.Port,
    Handler:           router,
    ReadHeaderTimeout: 5 * time.Second,  // Guard against Slowloris attacks
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      15 * time.Second,
    IdleTimeout:       60 * time.Second,
}
```

* **`ReadHeaderTimeout` (5 Seconds)**: Limits the amount of time the server allows to read incoming request headers. This guards against **Slowloris** slow-header attacks where an attacker holds connections open by sending header lines extremely slowly.
* **`MaxHeaderBytes` (1 MB)**: Uses Go's default `http.DefaultMaxHeaderBytes` limit of 1 MB. Any incoming request containing headers exceeding this size is immediately rejected by the runtime to prevent memory exhaustion.

---

## 2. Request Payload Boundaries

To prevent buffer overflows and denial-of-service attempts via massive JSON uploads, the gateway applies a request size limit middleware globally:

```go
r.Use(middleware.RequestSizeLimitMiddleware(eng.Config.MaxRequestSize))
```

The `RequestSizeLimitMiddleware` wraps the incoming request body using Go's `http.MaxBytesReader`. If a client transmits a payload larger than `MaxRequestSize` (default: 5 MB), the reader stops parsing, releases the network socket connection, and returns an HTTP error.

---

## 3. Security Headers Middleware

The `SecurityHeadersMiddleware` appends defensive HTTP headers to every outgoing response to guide browser behavior and mitigate client-side vulnerabilities:

| HTTP Header | Setting / Value | Mitigated Vulnerability | Description |
| :--- | :--- | :--- | :--- |
| **Strict-Transport-Security** (HSTS) | `max-age=31536000; includeSubDomains` | Man-in-the-Middle (MitM) | Forces the browser to connect exclusively over HTTPS for the next 365 days (including all subdomains). |
| **X-Content-Type-Options** | `nosniff` | MIME Sniffing / XSS | Prevents the browser from executing files based on guessed MIME types rather than the declared Content-Type. |
| **X-Frame-Options** | `DENY` | Clickjacking | Blocks browsers from rendering API responses inside iframe or frame elements, preventing clickjacking attacks. |
| **X-XSS-Protection** | `1; mode=block` | Cross-Site Scripting (XSS) | Activates built-in XSS filters in legacy browsers, blocking the page from rendering if XSS is detected. |
| **Content-Security-Policy** (CSP) | Default: `default-src 'self'` | Script Injection / Data Theft | Limits the origins from which scripts, styles, and assets can be loaded (configurable via `CSP_DIRECTIVES`). |
| **Referrer-Policy** | `strict-origin-when-cross-origin` | Credential Leakage | Limits the referrer header sent on cross-origin requests, protecting sensitive URL tokens. |
| **Permissions-Policy** | `camera=(), microphone=(), ...` | Client Feature Hijacking | Disables browser-specific hardware APIs (camera, microphone, geolocation, payment APIs) for the domain. |

---

## 4. CORS Middleware

The `CORSMiddleware` manages Cross-Origin Resource Sharing based on the `ALLOWED_ORIGINS` environment configuration.

### A. Origin Allow-List Verification
Instead of performing sequential slice scans, the middleware parses the comma-separated `ALLOWED_ORIGINS` environment variable at startup and populates a hash map:

```go
allowedOrigins := make(map[string]struct{})
```

For every request, the gateway checks the `Origin` header using an **$O(1)$ map lookup**. If the origin matches, it is written to the `Access-Control-Allow-Origin` response header alongside the `Vary: Origin` directive to instruct caching proxies to partition cache keys by origin.

:::note
If `ALLOWED_ORIGINS` is unset, the gateway defaults to wildcards (`*`) to facilitate local development. In production, always configure explicit, restricted origins.
:::

### B. Preflight Options Caching
CORS preflight checks (`OPTIONS` requests) are handled directly at the middleware layer. To reduce connection overhead and response latency, preflight responses are cached in the user's browser for **24 hours** before another preflight verification is required:

```go
if c.Request.Method == "OPTIONS" {
    c.Header("Access-Control-Max-Age", "86400") // Cache preflight for 24h
    c.AbortWithStatus(http.StatusNoContent)
    return
}
```
