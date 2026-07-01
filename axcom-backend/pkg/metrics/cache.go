// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
)

// ---------------------------------------------------------------------------
// Cache operational metrics
// ---------------------------------------------------------------------------

// CacheRequestsTotal counts every cache operation partitioned by layer (L1/L2),
// operation (get/set/delete/increment), and result (hit/miss/error).
// Use this to compute the cache hit rate per layer.
var CacheRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: ns,
		Name:      "cache_requests_total",
		Help:      "Total cache operations partitioned by layer, operation, and result.",
	},
	[]string{"layer", "operation", "result"},
)

// CacheOperationDuration tracks cache operation latency as a histogram.
// Buckets are skewed toward sub-millisecond ranges expected of L1/L2 caches.
var CacheOperationDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: ns,
		Name:      "cache_operation_duration_seconds",
		Help:      "Cache operation latency in seconds, partitioned by layer and operation.",
		Buckets:   []float64{.0001, .0005, .001, .005, .01, .025, .05, .1, .5},
	},
	[]string{"layer", "operation"},
)

// CacheStampedeDedupTotal counts requests that were collapsed by singleflight
// (i.e. concurrent callers for the same missing key that shared one fetchFn call).
// A high value means stampede protection is actively firing.
var CacheStampedeDedupTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Namespace: ns,
		Name:      "cache_stampede_dedup_total",
		Help:      "Number of GetOrFetch calls deduplicated by singleflight (concurrent callers for same key).",
	},
)

// CacheNegativeCacheTotal counts null sentinels written to cache, meaning a
// fetchFn returned (nil, nil). A sudden spike may indicate a cache penetration attack.
var CacheNegativeCacheTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Namespace: ns,
		Name:      "cache_negative_cache_total",
		Help:      "Number of null sentinels cached after fetchFn confirmed a record does not exist.",
	},
)

// ---------------------------------------------------------------------------
// Memory adapter gauges
// ---------------------------------------------------------------------------

// CacheMemoryItems is the current number of items in the L1 memory cache.
// Alert if this approaches the configured maxItems limit.
var CacheMemoryItems = promauto.NewGauge(
	prometheus.GaugeOpts{
		Namespace: ns,
		Name:      "cache_memory_items",
		Help:      "Current number of items stored in the L1 in-memory cache.",
	},
)

// CacheMemoryMaxItems exposes the configured maxItems capacity limit as a static gauge
// for use in dashboard ratio calculations (items / max_items).
var CacheMemoryMaxItems = promauto.NewGauge(
	prometheus.GaugeOpts{
		Namespace: ns,
		Name:      "cache_memory_max_items",
		Help:      "Configured maximum item capacity of the L1 in-memory cache.",
	},
)

// CacheMemoryEvictionsTotal counts evictions from the L1 memory cache partitioned
// by reason: "expired" (TTL expiry sweep) or "lfu" (capacity-based LFU eviction).
var CacheMemoryEvictionsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: ns,
		Name:      "cache_memory_evictions_total",
		Help:      "Total L1 memory cache evictions partitioned by reason (expired, lfu).",
	},
	[]string{"reason"},
)

// ---------------------------------------------------------------------------
// Redis connection pool collector
// ---------------------------------------------------------------------------

// CacheRedisPoolCollector implements prometheus.Collector and exposes Redis
// connection pool statistics sourced from go-redis PoolStats().
type CacheRedisPoolCollector struct {
	client *redis.Client

	totalConns   *prometheus.Desc
	idleConns    *prometheus.Desc
	staleConns   *prometheus.Desc
	poolHits     *prometheus.Desc
	poolMisses   *prometheus.Desc
	poolTimeouts *prometheus.Desc
}

// NewCacheRedisPoolCollector creates a collector wrapping the given Redis client.
func NewCacheRedisPoolCollector(client *redis.Client) *CacheRedisPoolCollector {
	fqn := func(name string) string {
		return prometheus.BuildFQName(ns, "cache_redis_pool", name)
	}
	return &CacheRedisPoolCollector{
		client: client,
		totalConns: prometheus.NewDesc(
			fqn("total_conns"),
			"Total connections currently in the Redis pool (active + idle).", nil, nil,
		),
		idleConns: prometheus.NewDesc(
			fqn("idle_conns"),
			"Connections currently idle and available in the Redis pool.", nil, nil,
		),
		staleConns: prometheus.NewDesc(
			fqn("stale_conns_total"),
			"Cumulative stale connections removed from the Redis pool.", nil, nil,
		),
		poolHits: prometheus.NewDesc(
			fqn("hits_total"),
			"Cumulative times a free connection was found in the Redis pool.", nil, nil,
		),
		poolMisses: prometheus.NewDesc(
			fqn("misses_total"),
			"Cumulative times a free connection was NOT found and a new one was dialed.", nil, nil,
		),
		poolTimeouts: prometheus.NewDesc(
			fqn("timeouts_total"),
			"Cumulative times a caller timed out waiting for a Redis pool connection.", nil, nil,
		),
	}
}

// Describe sends each metric descriptor to the Prometheus registry.
func (c *CacheRedisPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalConns
	ch <- c.idleConns
	ch <- c.staleConns
	ch <- c.poolHits
	ch <- c.poolMisses
	ch <- c.poolTimeouts
}

// Collect reads live pool stats from the Redis client and emits current values.
func (c *CacheRedisPoolCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.client.PoolStats()
	ch <- prometheus.MustNewConstMetric(c.totalConns, prometheus.GaugeValue, float64(s.TotalConns))
	ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(s.IdleConns))
	ch <- prometheus.MustNewConstMetric(c.staleConns, prometheus.CounterValue, float64(s.StaleConns))
	ch <- prometheus.MustNewConstMetric(c.poolHits, prometheus.CounterValue, float64(s.Hits))
	ch <- prometheus.MustNewConstMetric(c.poolMisses, prometheus.CounterValue, float64(s.Misses))
	ch <- prometheus.MustNewConstMetric(c.poolTimeouts, prometheus.CounterValue, float64(s.Timeouts))
}
