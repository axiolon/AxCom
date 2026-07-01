---
title: "Gateway Routing & Diagnostics"
description: "Routing structures, topological registrations, catch-alls for disabled modules, and diagnostic probes."
sidebar_position: 2
---

# Gateway Routing & Diagnostics

<DocBadge status="under-review" version="v0.1.0-alpha" />

The gateway routes HTTP traffic to core modules and diagnostic handlers. This document details how routes are structured, how active and disabled modules are isolated, and the structure of system probes.

---

## 1. Route Security Tiers

Routes under the `/api` prefix are segmented into three concentric security layers in `router.go`:

### A. Public Routes (No Authentication)
Open routes that do not require an authorization token to access.
* **Paths**:
  * **Authentication**: `/api/auth/register`, `/api/auth/login`, `/api/auth/refresh`
  * **Catalog (Read)**: `/api/products` (GET), `/api/products/:id` (GET), `/api/categories` (GET), `/api/categories/:id` (GET)
  * **Catalog Reviews (Read)**: `/api/products/:id/reviews` (GET), `/api/products/:id/reviews/rating` (GET)
  * **Guest Checkout**: `/api/orders/guest` (POST)
  * **Shipping (Rates & Track)**: `/api/shipping/rates` (POST), `/api/shipping/track/:tracking_number` (GET)
  * **Payments (Callbacks)**: `/api/payments/callback/:provider` (POST)
* **Registration**: Mounted directly onto the public `/api` router group (or registered with inline/local middleware handlers for conditional validation).

### B. Secured Routes (Authentication Required)
Routes requiring a valid token (local JWT or external OIDC token).
* **Paths**:
  * **Cart Operations**: `/api/cart` (GET/POST/DELETE), `/api/cart/count` (GET), `/api/cart/items/:variantId` (PUT/DELETE)
  * **Cart Merging**: `/api/cart/merge` (POST) — *Secured because it merges the guest cart session into the authenticated user's account*
  * **Customer Orders**: `/api/orders` (GET/POST), `/api/orders/:id` (GET), `/api/orders/:id/cancel` (POST)
  * **Customer Payments**: `/api/payments/intent` (POST), `/api/payments` (GET), `/api/payments/by-order/:orderID` (GET)
  * **Customer Shipping**: `/api/shipping/order/:order_id` (GET)
  * **Customer Reviews**: `/api/products/:id/reviews` (POST/PUT/DELETE)
  * **Notifications**: `/api/notifications/send` (POST)
* **Registration**: Mounted onto a sub-group that applies the configured Auth Middleware:
  ```go
  secured := api.Group("")
  secured.Use(eng.AuthMiddleware)
  ```

### C. Admin Routes (Admin Role Required)
Sensitive administrative tasks requiring both valid authentication and the `admin` role.
* **Paths**: `/api/admin/*`, `/api/inventory/*` (write operations), `/api/catalog/*` (write/create/delete catalog operations).
* **Registration**: Mounted onto a nested admin group:
  ```go
  adminGroup := secured.Group("")
  adminGroup.Use(eng.AdminMiddleware)
  ```

---

## 2. Topological Module Routing

Active domain modules are loaded at boot time by the core engine. Once sorted topologically to ensure dependencies are initialized first, the router registers each module's HTTP endpoints:

```go
// Register routes in topological / dependency-first order
for _, mod := range eng.ActiveModules() {
    mod.RegisterRoutes(api, secured, adminGroup)
}
```

This pattern ensures that:
- Modules do not register their own root routing engines.
- Domain modules mount endpoints directly to the central, pre-configured security groups (`api`, `secured`, `adminGroup`).

---

## 3. Disabled Module Catch-Alls

When a domain module is disabled via configuration (e.g. `modules.catalog.enabled: false`), the Gateway registers explicit catch-all routes matching the module's registered base paths. If a client attempts to access a disabled path, the system responds with a descriptive message rather than a generic `404 Not Found`:

```go
for _, info := range eng.DisabledModules() {
    for _, path := range info.BasePaths {
        name := info.Name // Capture loop variable
        r.Any(path+"/*path", func(c *gin.Context) {
            c.JSON(http.StatusServiceUnavailable, gin.H{
                "error": fmt.Sprintf(
                    "module %q is disabled; enable it in config.yaml under modules.%s.enabled: true",
                    name, name,
                ),
            })
        })
    }
}
```

---

## 4. Diagnostics & System Probes

Diagnostic endpoints are mounted outside the `/api` route group and bypass all rate limiting and authentication checks. This ensures orchestrators (e.g., Kubernetes, ALB) can always query system health even under high load.

### A. Liveness Probe (`GET /healthz`)
Verifies the process is alive.
* **Response Code**: `200 OK`
* **Payload**:
  ```json
  {
    "status": "UP",
    "time": "2026-06-27T16:00:00Z"
  }
  ```

### B. Readiness Probe (`GET /readyz`)
Performs active pings against backend systems with a strict **3-second timeout**. If database or cache connection checks fail, returns service unavailable.
* **Response Code**: `200 OK` (when all UP) or `503 Service Unavailable` (when any database or cache connection is DOWN).
* **Payload**:
  ```json
  {
    "status": "READY",
    "database": "UP",
    "cache": "UP",
    "time": "2026-06-27T16:00:00Z"
  }
  ```

### C. Prometheus Metrics (`GET /metrics`)
Exposes system-level Go runtime parameters, database connection pool statistics, HTTP request durations, and Redis adapter stats in standard Prometheus exposition format. Scraped dynamically every 15 seconds.

### D. JSON Metrics Snapshot (`GET /metricsz`)
Returns a human-readable JSON representation of active database connection pool statistics. Useful for debugging and terminal health inspections.
* **Response Code**: `200 OK`
* **Payload**:
  ```json
  {
    "time": "2026-06-27T16:00:00Z",
    "db_pool": {
      "max_conns": 50,
      "total_conns": 10,
      "acquired_conns": 2,
      "idle_conns": 8,
      "acquire_count": 521,
      "empty_acquire_count": 0,
      "acquire_duration_ms": 1
    }
  }
  ```
