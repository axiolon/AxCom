# ADR-006: Rate Limiting Strategy and Implementation

**Date:** 2026-06-27  
**Status:** accepted

## Context
The API Gateway needs to handle traffic surges, protect against Denial-of-Service (DoS) / brute-force attacks, and prevent resource starvation. Since different client types have different usage patterns (e.g., public guests vs. logged-in customers vs. administrators), the rate limiter must support dynamic tiers and accommodate legitimate client request bursts while keeping memory consumption low.

## Decision
1. **Algorithm:** Implement the rate limiting mechanism using the **Token Bucket** algorithm.
2. **Identification:** Throttle requests on a per-Client-IP basis.
3. **Dynamic Tiering:** Select the rate configuration dynamically based on the JWT claims/role extracted from the `Authorization` header:
   * **Public Guest:** 30 req/min (Rate: 0.5 tokens/sec, Burst: 10)
   * **Authenticated User:** 120 req/min (Rate: 2.0 tokens/sec, Burst: 20)
   * **Admin User:** 300 req/min (Rate: 5.0 tokens/sec, Burst: 50)
4. **Memory Management:** Use an in-memory map protected by a mutex, backed by a background janitor routine that runs every minute to prune inactive clients (idle for >5 minutes) to prevent memory leaks.
5. **Testing Flexibility:** Skip rate limiting in Gin test mode (`gin.Mode() == gin.TestMode`) or when the `SKIP_RATE_LIMIT` environment variable is explicitly set to `true`.

## Alternatives Considered

| Strategy | Burst | Accuracy | Memory | Reason Rejected |
|---|---|---|---|---|
| **Fixed Window** | Yes | Low | Low | Vulnerable to boundary bursts (double the rate limit could be consumed at the window boundary). |
| **Sliding Log** | No | High | High | Keeps all request timestamps in memory; high memory overhead makes it unsuitable for high-traffic endpoints. |
| **Sliding Counter** | Limited | Medium | Low | Offers only limited burst capacity and is complex to implement compared to Token Bucket. |
| **Leaky Bucket** | No | High | Low | Emits requests at a constant, smooth rate by queuing them. Disallows any client-side burst traffic, which degrades user experience for interactive e-commerce applications. |

## Why This Choice
* **Burst Handling:** The Token Bucket algorithm naturally allows for bursts of requests up to the bucket's capacity. This is critical for modern web applications that fetch multiple resources or make concurrent API requests during page loads.
* **Resource Efficiency:** It only requires storing three fields per client (current tokens, last refill timestamp, and last seen timestamp), keeping memory footprint minimal.
* **Accuracy:** Refilling tokens proportionally based on elapsed time achieves high accuracy without requiring timestamp logs.

## Tradeoffs
**Gains:**
* Native, high-performance support for bursty client traffic without artificial latency.
* Safe, automatic memory cleanups via the background janitor.
* Low memory overhead per active client.

**Accepts:**
* Distributed deployment issues: Since the bucket states are stored in-memory per-instance, clients may get higher aggregate limits if requests are load-balanced across multiple instances (unless sticky sessions or a centralized store like Redis is introduced later).

## Consequences
* For horizontally scaled environments, actual limits per client will scale with the number of gateway instances. If strict global enforcement becomes necessary in the future, the rate-limiting interface can be refactored to use a Redis-backed token bucket.
