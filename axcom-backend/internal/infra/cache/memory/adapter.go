// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"ecom-engine/internal/infra/cache"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/metrics"
)

type cacheItem struct {
	value      interface{} // Stored as JSON string or raw value
	expiration int64       // Unix nano timestamp, 0 = no expiration
	hits       int64       // Track access count for LFU eviction
	lastAccess int64       // Track last access UnixNano for LRU eviction
}

const (
	// defaultMaxItems is the default capacity of the in-memory cache.
	defaultMaxItems = 10000
	// defaultMaxValueBytes is the default max serialized value size (1 MiB).
	defaultMaxValueBytes = 1 << 20 // 1 MiB
	// maxKeyLength is the maximum allowed key length in bytes (matches Redis convention).
	maxKeyLength = 512
)

// MemoryAdapter is a thread-safe, in-memory cache implementation with capacity limit and eviction.
type MemoryAdapter struct { //nolint:revive // Name is intentionally explicit for the public API.
	mu            sync.RWMutex
	items         map[string]*cacheItem
	maxItems      int
	maxValueBytes int
	stopChan      chan struct{}
}

// Option defines a configuration option for MemoryAdapter.
type Option func(*MemoryAdapter)

// WithMaxItems sets the maximum item capacity.
func WithMaxItems(maxItems int) Option {
	return func(a *MemoryAdapter) {
		if maxItems > 0 {
			a.maxItems = maxItems
		}
	}
}

// WithMaxValueBytes sets the maximum allowed serialized value size in bytes.
// Set attempts with larger values will return ErrValueTooLarge.
func WithMaxValueBytes(maxValueBytes int) Option {
	return func(a *MemoryAdapter) {
		if maxValueBytes > 0 {
			a.maxValueBytes = maxValueBytes
		}
	}
}

// NewMemoryAdapter creates a new in-memory cache adapter.
func NewMemoryAdapter(opts ...Option) *MemoryAdapter {
	a := &MemoryAdapter{
		items:         make(map[string]*cacheItem),
		maxItems:      defaultMaxItems,
		maxValueBytes: defaultMaxValueBytes,
		stopChan:      make(chan struct{}),
	}
	for _, opt := range opts {
		opt(a)
	}

	// Expose static capacity limit for dashboard ratio calculations.
	metrics.CacheMemoryMaxItems.Set(float64(a.maxItems))

	// Start the active background cleanup goroutine
	go a.startCleanupLoop(1 * time.Minute)

	return a
}

func (a *MemoryAdapter) startCleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			a.cleanupExpired()
		case <-a.stopChan:
			return
		}
	}
}

// Close closes the cache backend and stops the background cleanup loop.
func (a *MemoryAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	select {
	case <-a.stopChan:
		// already closed
	default:
		close(a.stopChan)
	}
	return nil
}

// Set stores a value with an optional TTL.
func (a *MemoryAdapter) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if len(key) > maxKeyLength {
		return cache.NewCacheBackendError("set", key, cache.ErrKeyTooLong)
	}

	// Serialize to JSON for consistency with Redis adapter
	jsonData, err := json.Marshal(value)
	if err != nil {
		logger.ErrorCtx(ctx, "Memory cache set serialization failed for key %s: %v", key, err)
		return cache.NewCacheBackendError("set", key, fmt.Errorf("json marshal: %w", err))
	}

	if len(jsonData) > a.maxValueBytes {
		logger.ErrorCtx(ctx, "Memory cache set rejected for key %s: value size %d exceeds limit %d", key, len(jsonData), a.maxValueBytes)
		return cache.NewCacheBackendError("set", key, cache.ErrValueTooLarge)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Handle eviction if capacity limit is reached
	if len(a.items) >= a.maxItems {
		// 1. Passive/Active cleanup: check a sample of items and delete expired ones
		scanned := 0
		expiredDeleted := 0
		for k, item := range a.items {
			scanned++
			if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
				delete(a.items, k)
				expiredDeleted++
				metrics.CacheMemoryEvictionsTotal.WithLabelValues("expired").Inc()
				logger.DebugCtx(ctx, "Memory cache key %s evicted (expired)", k)
			}
			if scanned >= 50 {
				break
			}
		}

		// 2. Sample-based LFU (Least Frequently Used) eviction if still at capacity
		if len(a.items) >= a.maxItems {
			var bestKey string
			var bestItem *cacheItem
			sampleCount := 0
			// Go maps have randomized iteration, which acts as a natural random sampler
			for k, item := range a.items {
				if bestItem == nil {
					bestKey = k
					bestItem = item
				} else {
					itemHits := atomic.LoadInt64(&item.hits)
					bestHits := atomic.LoadInt64(&bestItem.hits)
					if itemHits < bestHits {
						bestKey = k
						bestItem = item
					} else if itemHits == bestHits {
						// Tie-breaker: oldest access time (LRU)
						itemAccess := atomic.LoadInt64(&item.lastAccess)
						bestAccess := atomic.LoadInt64(&bestItem.lastAccess)
						if itemAccess < bestAccess {
							bestKey = k
							bestItem = item
						}
					}
				}
				sampleCount++
				if sampleCount >= 10 { // sample size
					break
				}
			}
			if bestKey != "" {
				delete(a.items, bestKey)
				metrics.CacheMemoryEvictionsTotal.WithLabelValues("lfu").Inc()
				logger.InfoCtx(ctx, "Memory cache capacity limit reached (%d items). Evicted key: %s (hits: %d)", a.maxItems, bestKey, atomic.LoadInt64(&bestItem.hits))
			}
		}
	}

	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}

	// Preserve stats if item already exists
	var hits int64 = 1
	if existing, found := a.items[key]; found {
		hits = atomic.LoadInt64(&existing.hits) + 1
	}

	a.items[key] = &cacheItem{
		value:      string(jsonData),
		expiration: exp,
		hits:       hits,
		lastAccess: time.Now().UnixNano(),
	}

	metrics.CacheMemoryItems.Set(float64(len(a.items)))
	logger.DebugCtx(ctx, "Memory cache set for key: %s, TTL: %v", key, ttl)
	return nil
}

// Get retrieves a value by key.
func (a *MemoryAdapter) Get(ctx context.Context, key string) (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	item, found := a.items[key]
	if !found {
		logger.DebugCtx(ctx, "Memory cache miss for key: %s", key)
		return "", cache.ErrCacheMiss
	}

	// Check expiration
	if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
		a.mu.RUnlock()
		a.mu.Lock()
		// Double check under write lock
		if it, ok := a.items[key]; ok && it.expiration > 0 && time.Now().UnixNano() > it.expiration {
			delete(a.items, key)
		}
		a.mu.Unlock()
		a.mu.RLock() // Relock to satisfy the defer RUnlock()
		logger.DebugCtx(ctx, "Memory cache miss for key: %s (expired)", key)
		return "", cache.ErrCacheMiss
	}

	// Record hits and last access atomically under read lock
	atomic.AddInt64(&item.hits, 1)
	atomic.StoreInt64(&item.lastAccess, time.Now().UnixNano())

	valStr, ok := item.value.(string)
	if !ok {
		logger.ErrorCtx(ctx, "Memory cache value for key %s is not a string", key)
		return "", cache.NewCacheBackendError("get", key, fmt.Errorf("stored value is not a string"))
	}

	logger.DebugCtx(ctx, "Memory cache hit for key: %s (hits: %d)", key, atomic.LoadInt64(&item.hits))
	return valStr, nil
}

// Delete removes a key from the cache.
func (a *MemoryAdapter) Delete(ctx context.Context, key string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.items, key)
	metrics.CacheMemoryItems.Set(float64(len(a.items)))
	logger.DebugCtx(ctx, "Memory cache key deleted: %s", key)
	return nil
}

// Exists checks if a key exists without retrieving the value.
func (a *MemoryAdapter) Exists(ctx context.Context, key string) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	item, found := a.items[key]
	if !found {
		logger.DebugCtx(ctx, "Memory cache Exists check miss for key: %s", key)
		return false, nil
	}

	// Check expiration
	if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
		logger.DebugCtx(ctx, "Memory cache Exists check miss for key: %s (expired)", key)
		return false, nil
	}

	logger.DebugCtx(ctx, "Memory cache Exists check hit for key: %s", key)
	return true, nil
}

// Increment atomically increments an integer value at the given key.
func (a *MemoryAdapter) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	item, found := a.items[key]

	var currentVal int64
	var exp int64
	if found {
		// Check expiration
		if item.expiration == 0 || time.Now().UnixNano() <= item.expiration {
			// Try to parse as integer
			valStr, ok := item.value.(string)
			if !ok {
				logger.ErrorCtx(ctx, "Memory cache increment parse failed for key %s: value is not a string", key)
				return 0, cache.NewCacheBackendError("increment", key, fmt.Errorf("value is not a string"))
			}
			val, err := strconv.ParseInt(valStr, 10, 64)
			if err != nil {
				logger.ErrorCtx(ctx, "Memory cache increment parse failed for key %s: %v", key, err)
				return 0, cache.NewCacheBackendError("increment", key, fmt.Errorf("value is not an integer: %w", err))
			}
			currentVal = val
			exp = item.expiration
		}
	}

	newVal := currentVal + delta

	var hits int64 = 1
	if found {
		hits = atomic.LoadInt64(&item.hits) + 1
	}

	// Store back as integer string (not JSON)
	a.items[key] = &cacheItem{
		value:      strconv.FormatInt(newVal, 10),
		expiration: exp,
		hits:       hits,
		lastAccess: time.Now().UnixNano(),
	}

	logger.DebugCtx(ctx, "Memory cache increment key: %s by %d, new value: %d", key, delta, newVal)
	return newVal, nil
}

// HealthCheck always returns nil for in-memory cache (always healthy).
func (a *MemoryAdapter) HealthCheck(_ context.Context) error {
	return nil
}

// cleanupExpired removes expired items (optional maintenance).
func (a *MemoryAdapter) cleanupExpired() {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now().UnixNano()
	deletedCount := 0
	for key, item := range a.items {
		if item.expiration > 0 && now > item.expiration {
			delete(a.items, key)
			deletedCount++
			metrics.CacheMemoryEvictionsTotal.WithLabelValues("expired").Inc()
		}
	}
	if deletedCount > 0 {
		metrics.CacheMemoryItems.Set(float64(len(a.items)))
		logger.Info("Memory cache cleaned up %d expired items", deletedCount)
	}
}
