# ADR-007: Distributed Rate Limiting and Fail-Open Failover Architecture

**Date:** 2026-06-27  
**Status:** accepted

## Context
Following the acceptance of [ADR-006](../../../ecom-backend/docs/adr/006-rate-limiting.md) (which established the Token Bucket algorithm and IP-based tiering using a local memory store), we must address horizontal scaling. 

In multi-replica deployments, a purely in-memory rate limiter allows clients to exceed their aggregate quotas because their requests are load-balanced across multiple backend instances. To enforce strict global quotas, a distributed rate limiting store is required. However, introducing a centralized cache (Redis) creates a single point of failure (SPOF) for all API requests. If Redis encounters latency or downtime, the gateway must not drop legitimate customer traffic.

## Decision
1. **Centralized Storage:** Implement a distributed `redisStore` using Redis as the central state provider.
2. **Atomic Token Updates:** Perform all token evaluations using an **atomic Lua script** within the Redis engine to prevent concurrent read-modify-write race conditions across horizontally scaled application replicas.
3. **Fallback Resiliency (`FallbackStore`):** Wrap the Redis store with a local, in-memory backup store. If a Redis socket or query operation returns an error, the system must immediately and transparently transition to tracking that client's rate limits in the local `memoryStore`.
4. **Fail-Open Policy:** If both Redis and the fallback memory store experience errors, the request must fail open (allow the traffic) rather than throwing a service outage error (`500` or `503`) to the client.
5. **Anti-Flapping Health Checks:** Keep a background health checker routine that pings Redis every 30 seconds. To prevent connection flapping, only promote Redis back to primary after **3 consecutive successful pings** (~90 seconds of stable connectivity).

## Alternatives Considered

| Approach | Availability | Complexity | Global Accuracy | Reason Rejected |
|---|---|---|---|---|
| **Sticky Sessions** | Medium | High | Medium | Load balancer routing overhead; uneven server load distribution; state is lost if a backend replica restarts. |
| **Fail-Closed Redis** | Low | Low | High | Enforces rates strictly, but if Redis is down, all API requests are rejected. Dropping user traffic due to a cache outage is unacceptable. |
| **Direct Fail-Open (No local memory fallback)** | High | Low | Low | Bypasses all rate limiting entirely when Redis is down. Leaves the service unprotected against DoS/brute-force attacks during Redis maintenance window. |

## Why This Choice
* **Strict Global Enforcement:** Redis ensures that client limits are synchronized accurately across all horizontal replicas.
* **Race Condition Safety:** The Lua script execution is single-threaded inside Redis, ensuring atomic checks and writes without locking overhead on the application side.
* **Robust Fail-Open/Graceful Degradation:** During Redis outages, we degrade to per-instance tracking via the local `memoryStore` instead of disabling protection completely or crashing the server.
* **Flapping Mitigation:** The 3-ping recovery gate prevents the gateway from continually thrashing between Redis and local memory under shaky network conditions.

## Tradeoffs
**Gains:**
* Enforces strict, cluster-wide rate limit quotas.
* High availability: A Redis outage does not affect user experience.
* Zero state-sync lag during normal operations.

**Accepts:**
* Temporary quota drift during fallbacks: When running on the local `memoryStore` fallback, client quotas are tracked per-instance, temporarily allowing higher collective rates until Redis recovers. This is an acceptable tradeoff to keep the service online.

## Consequences
* The `middleware.Store` interface must accommodate both Redis and Memory backends.
* The API Gateway requires Redis connectivity parameters in `config.yaml` to enable distributed mode.
* Operational dashboards and alerts must track fallback events and total Redis downtime duration to alert operators of underlying database/cache issues.
