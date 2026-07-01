// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/metrics"

	"golang.org/x/sync/singleflight"
)

// nullSentinel is stored in cache when a fetchFn returns (nil, nil) — i.e. the
// record genuinely does not exist. Subsequent reads for the same key will return
// ErrNotFound without hitting the database, blocking cache penetration attacks.
const nullSentinel = "__null__"

// ErrNotFound is returned by GetOrFetch when the key maps to a null sentinel —
// meaning the record was looked up and confirmed absent in the source of truth.
var ErrNotFound = errors.New("cache: record not found")

// Manager orchestrates caching policies across L1 (Memory) and L2 (Redis) layers.
type Manager interface {
	// GetOrFetch attempts to retrieve the item from L1, then L2.
	// If both miss, it runs fetchFn, marshals the result, stores it in L2 and L1,
	// and unmarshals the result into target.
	// If fetchFn returns (nil, nil), a null sentinel is cached and ErrNotFound is returned.
	// Concurrent calls for the same key are deduplicated via singleflight.
	GetOrFetch(ctx context.Context, key string, target interface{}, ttl time.Duration, fetchFn func() (interface{}, error)) error

	// Invalidate deletes the key from both L1 and L2 caches.
	Invalidate(ctx context.Context, key string) error

	// Close closes both L1 and L2 cache backends.
	Close() error
}

type cacheManager struct {
	l1               Cache
	l2               Cache
	l1TTL            time.Duration
	negativeCacheTTL time.Duration
	ttlJitterPct     float64
	sf               singleflight.Group
}

// ManagerOption allows configuring settings on cacheManager.
type ManagerOption func(*cacheManager)

// WithL1TTL configures the TTL for the L1 cache.
func WithL1TTL(d time.Duration) ManagerOption {
	return func(m *cacheManager) {
		if d > 0 {
			m.l1TTL = d
		}
	}
}

// WithNegativeCacheTTL sets how long a null sentinel is cached after fetchFn
// returns (nil, nil). Prevents cache penetration on non-existent keys.
// Default: 30 seconds. Set to 0 to disable negative caching.
func WithNegativeCacheTTL(d time.Duration) ManagerOption {
	return func(m *cacheManager) {
		m.negativeCacheTTL = d
	}
}

// WithTTLJitter adds a random positive jitter to L2 TTLs to spread out expiry
// times and prevent cache avalanche. pct is the fraction of ttl to use as the
// jitter range, e.g. 0.1 = up to 10% extra. Set to 0 to disable.
// Default: 0.1 (10%).
func WithTTLJitter(pct float64) ManagerOption {
	return func(m *cacheManager) {
		if pct >= 0 {
			m.ttlJitterPct = pct
		}
	}
}

// NewCacheManager returns a Cache Manager configured with L1 and L2 cache providers.
func NewCacheManager(l1 Cache, l2 Cache, opts ...ManagerOption) Manager {
	m := &cacheManager{
		l1:               l1,
		l2:               l2,
		l1TTL:            1 * time.Minute,
		negativeCacheTTL: 30 * time.Second,
		ttlJitterPct:     0.1,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// jitterTTL adds a random positive duration up to pct*ttl to spread expiry times.
func jitterTTL(ttl time.Duration, pct float64) time.Duration {
	if pct <= 0 || ttl <= 0 {
		return ttl
	}
	// #nosec G404 -- Weak random number generator is safe and appropriate for non-security-critical cache TTL jitter
	jitter := time.Duration(rand.Int64N(int64(float64(ttl) * pct)))
	return ttl + jitter
}

func (m *cacheManager) GetOrFetch(ctx context.Context, key string, target interface{}, ttl time.Duration, fetchFn func() (interface{}, error)) error {
	// 1. Try L1 Memory Cache
	if m.l1 != nil {
		start := time.Now()
		val, err := m.l1.Get(ctx, key)
		metrics.CacheOperationDuration.WithLabelValues("L1", "get").Observe(time.Since(start).Seconds())
		switch {
		case err == nil:
			if val == nullSentinel {
				metrics.CacheRequestsTotal.WithLabelValues("L1", "get", "hit").Inc()
				return ErrNotFound
			}
			if jsonErr := json.Unmarshal([]byte(val), target); jsonErr == nil {
				metrics.CacheRequestsTotal.WithLabelValues("L1", "get", "hit").Inc()
				logger.InfoCtx(ctx, "Cache hit L1 (Memory) for key: %s", key)
				return nil
			}
		case IsCacheMiss(err):
			metrics.CacheRequestsTotal.WithLabelValues("L1", "get", "miss").Inc()
		default:
			metrics.CacheRequestsTotal.WithLabelValues("L1", "get", "error").Inc()
		}
	}

	// 2. Try L2 Redis Cache
	if m.l2 != nil {
		start := time.Now()
		val, err := m.l2.Get(ctx, key)
		metrics.CacheOperationDuration.WithLabelValues("L2", "get").Observe(time.Since(start).Seconds())
		switch {
		case err == nil:
			if val == nullSentinel {
				metrics.CacheRequestsTotal.WithLabelValues("L2", "get", "hit").Inc()
				// Backfill L1 with sentinel so the next read doesn't reach L2
				if m.l1 != nil {
					_ = m.l1.Set(ctx, key, nullSentinel, m.negativeCacheTTL)
				}
				return ErrNotFound
			}
			metrics.CacheRequestsTotal.WithLabelValues("L2", "get", "hit").Inc()
			logger.InfoCtx(ctx, "Cache hit L2 (Redis) for key: %s", key)
			if m.l1 != nil {
				if jsonErr := json.Unmarshal([]byte(val), target); jsonErr == nil {
					l1ttl := m.l1TTL
					if ttl > 0 && ttl < l1ttl {
						l1ttl = ttl
					}
					_ = m.l1.Set(ctx, key, json.RawMessage(val), l1ttl)
					return nil
				}
			} else {
				return json.Unmarshal([]byte(val), target)
			}
		case IsCacheMiss(err):
			metrics.CacheRequestsTotal.WithLabelValues("L2", "get", "miss").Inc()
		default:
			metrics.CacheRequestsTotal.WithLabelValues("L2", "get", "error").Inc()
		}
	}

	// 3. Cache Miss: deduplicate concurrent fetches for the same key via singleflight.
	// All concurrent callers for `key` share a single fetchFn invocation.
	logger.InfoCtx(ctx, "Cache miss L1/L2 for key: %s. Fetching from source...", key)

	type sfResult struct {
		bytes []byte
		found bool
	}

	v, err, shared := m.sf.Do(key, func() (interface{}, error) {
		data, fetchErr := fetchFn()
		if fetchErr != nil {
			return nil, fetchErr
		}

		// fetchFn confirmed the record does not exist
		if data == nil {
			return &sfResult{found: false}, nil
		}

		b, marshalErr := json.Marshal(data)
		if marshalErr != nil {
			return nil, marshalErr
		}
		return &sfResult{bytes: b, found: true}, nil
	})

	// shared=true means this caller received a deduplicated result (stampede protected)
	if shared {
		metrics.CacheStampedeDedupTotal.Inc()
	}

	if err != nil {
		return err
	}

	res := v.(*sfResult)

	// 3a. Record not found — cache a null sentinel to block future penetration
	if !res.found {
		logger.InfoCtx(ctx, "fetchFn returned nil for key: %s, caching null sentinel", key)
		metrics.CacheNegativeCacheTotal.Inc()
		if m.negativeCacheTTL > 0 {
			if m.l2 != nil {
				_ = m.l2.Set(ctx, key, nullSentinel, m.negativeCacheTTL)
			}
			if m.l1 != nil {
				_ = m.l1.Set(ctx, key, nullSentinel, m.negativeCacheTTL)
			}
		}
		return ErrNotFound
	}

	// 3b. Populate target from fetched bytes
	if err := json.Unmarshal(res.bytes, target); err != nil {
		return err
	}

	// 4. Persist to L2 (Redis) with jitter, then backfill L1
	rawMsg := json.RawMessage(res.bytes)
	if m.l2 != nil {
		l2ttl := jitterTTL(ttl, m.ttlJitterPct)
		start := time.Now()
		if err := m.l2.Set(ctx, key, rawMsg, l2ttl); err != nil {
			metrics.CacheRequestsTotal.WithLabelValues("L2", "set", "error").Inc()
			logger.ErrorCtx(ctx, "Failed to set L2 cache for key %s: %v", key, err)
		} else {
			metrics.CacheRequestsTotal.WithLabelValues("L2", "set", "ok").Inc()
		}
		metrics.CacheOperationDuration.WithLabelValues("L2", "set").Observe(time.Since(start).Seconds())
	}
	if m.l1 != nil {
		l1ttl := m.l1TTL
		if ttl > 0 && ttl < l1ttl {
			l1ttl = ttl
		}
		start := time.Now()
		if err := m.l1.Set(ctx, key, rawMsg, l1ttl); err != nil {
			metrics.CacheRequestsTotal.WithLabelValues("L1", "set", "error").Inc()
			logger.ErrorCtx(ctx, "Failed to set L1 cache for key %s: %v", key, err)
		} else {
			metrics.CacheRequestsTotal.WithLabelValues("L1", "set", "ok").Inc()
		}
		metrics.CacheOperationDuration.WithLabelValues("L1", "set").Observe(time.Since(start).Seconds())
	}

	return nil
}

func (m *cacheManager) Invalidate(ctx context.Context, key string) error {
	logger.InfoCtx(ctx, "Invalidating cache key: %s", key)
	var errs []error

	if m.l1 != nil {
		if err := m.l1.Delete(ctx, key); err != nil {
			logger.ErrorCtx(ctx, "Failed to invalidate L1 cache for key %s: %v", key, err)
			errs = append(errs, fmt.Errorf("L1: %w", err))
		}
	}

	if m.l2 != nil {
		if err := m.l2.Delete(ctx, key); err != nil {
			logger.ErrorCtx(ctx, "Failed to invalidate L2 cache for key %s: %v", key, err)
			errs = append(errs, fmt.Errorf("L2: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (m *cacheManager) Close() error {
	var errs []error
	if m.l1 != nil {
		if err := m.l1.Close(); err != nil {
			errs = append(errs, fmt.Errorf("L1: %w", err))
		}
	}
	if m.l2 != nil {
		if err := m.l2.Close(); err != nil {
			errs = append(errs, fmt.Errorf("L2: %w", err))
		}
	}
	return errors.Join(errs...)
}
