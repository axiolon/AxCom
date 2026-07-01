// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"time"
)

// Cache defines the contract for all cache backends (Redis, Memory, etc.)
//
// All operations accept a cancellable context.
// Errors are typed: ErrCacheMiss for missing keys, CacheBackendError for backend failures.
// All operations are safe to call concurrently.
//
// Value serialization, data storage location, and failure handling (fail-open vs fail-closed)
// are implementation details, decided by each module using the cache.
type Cache interface {
	// Set stores a value with an optional time-to-live (TTL).
	// If ttl <= 0, the value is stored indefinitely.
	// ErrCacheMiss is never returned; only CacheBackendError on failure.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Get retrieves a value by key.
	// Returns ErrCacheMiss if the key is not found.
	// Returns CacheBackendError on backend failure.
	Get(ctx context.Context, key string) (string, error)

	// Delete removes a key from the cache.
	// Deleting a non-existent key is not an error.
	// Returns CacheBackendError on backend failure.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in the cache without retrieving the value.
	// Returns true if the key exists and is not expired; false otherwise.
	// Returns CacheBackendError on backend failure.
	Exists(ctx context.Context, key string) (bool, error)

	// Increment atomically increments the integer value at the given key by delta.
	// If the key does not exist, it is treated as 0.
	// Returns the new value after the increment.
	// Returns CacheBackendError on backend failure or if the value is not an integer.
	Increment(ctx context.Context, key string, delta int64) (int64, error)

	// HealthCheck verifies the cache backend is operational.
	// Returns nil if healthy, CacheBackendError if unhealthy.
	HealthCheck(ctx context.Context) error

	// Close closes the cache backend connection and cleans up resources.
	Close() error
}
