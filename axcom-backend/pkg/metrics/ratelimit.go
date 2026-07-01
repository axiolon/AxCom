// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ---------------------------------------------------------------------------
// Rate-limit metrics
// ---------------------------------------------------------------------------

// RateLimitRequestsTotal counts every rate-limit evaluation, partitioned by
// bucket label (global, tier:public, tier:auth, tier:admin, ep:auth,
// ep:checkout, ep:payments) and decision (allowed or denied).
var RateLimitRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: ns,
		Subsystem: "ratelimit",
		Name:      "requests_total",
		Help:      "Total rate-limit evaluations, partitioned by bucket and decision.",
	},
	[]string{"bucket", "decision"}, // decision: "allowed" | "denied"
)

// RateLimitBackendActive is a gauge set to 1 for the currently active store
// (redis or memory) and 0 for the inactive one.
var RateLimitBackendActive = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: ns,
		Subsystem: "ratelimit",
		Name:      "backend_active",
		Help:      "1 if this backend is currently the active rate-limit store, 0 otherwise.",
	},
	[]string{"backend"}, // "redis" | "memory"
)

// RateLimitFallbacksTotal counts transitions from Redis to the in-memory fallback store.
var RateLimitFallbacksTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Namespace: ns,
		Subsystem: "ratelimit",
		Name:      "backend_fallbacks_total",
		Help:      "Times the rate-limiter switched from Redis to in-memory fallback.",
	},
)

// RateLimitRecoveriesTotal counts successful restorations of the Redis backend
// after an outage episode.
var RateLimitRecoveriesTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Namespace: ns,
		Subsystem: "ratelimit",
		Name:      "backend_recoveries_total",
		Help:      "Times the rate-limiter recovered from in-memory fallback back to Redis.",
	},
)

// RateLimitRedisErrorsTotal accumulates all raw Redis errors encountered by
// the rate-limiter (per-request failures and probe failures).
var RateLimitRedisErrorsTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Namespace: ns,
		Subsystem: "ratelimit",
		Name:      "redis_errors_total",
		Help:      "Total Redis errors encountered by the rate-limiter (per-request and probe failures).",
	},
)
