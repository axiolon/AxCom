// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"errors"
	"fmt"
)

// CacheMiss represents a cache key not found error (expected condition, not app error)
var ErrCacheMiss = errors.New("cache miss")

// ErrValueTooLarge is returned by Set when the serialized value exceeds the
// configured maximum size for the adapter. Prevents memory exhaustion attacks.
var ErrValueTooLarge = errors.New("cache value exceeds maximum allowed size")

// ErrKeyTooLong is returned by Set when the key length exceeds 512 bytes.
// Redis itself allows up to 512 MB but excessively long keys waste memory and
// slow lookups; this enforces a sane upper bound across all adapters.
var ErrKeyTooLong = errors.New("cache key exceeds maximum allowed length")

// CacheBackendError represents a failure in the cache backend (connection, I/O, etc.)
type CacheBackendError struct { //nolint:revive // Name is intentionally explicit for the public API.
	Operation string // "get", "set", "delete", etc.
	Key       string
	Err       error
}

// Error returns the string representation of CacheBackendError
func (e *CacheBackendError) Error() string {
	return fmt.Sprintf("cache backend error [%s %s]: %v", e.Operation, e.Key, e.Err)
}

// Unwrap returns the underlying error for errors.Is/As chaining
func (e *CacheBackendError) Unwrap() error {
	return e.Err
}

// IsCacheMiss returns true if the error is a cache miss
func IsCacheMiss(err error) bool {
	return errors.Is(err, ErrCacheMiss)
}

// IsCacheBackendError returns true if the error is a cache backend error
func IsCacheBackendError(err error) bool {
	var target *CacheBackendError
	return errors.As(err, &target)
}

// NewCacheBackendError wraps a backend error with context about the operation
func NewCacheBackendError(operation, key string, err error) *CacheBackendError {
	return &CacheBackendError{
		Operation: operation,
		Key:       key,
		Err:       err,
	}
}
