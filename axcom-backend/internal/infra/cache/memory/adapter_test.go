// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package memory

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"ecom-engine/internal/infra/cache"
)

func TestMemoryAdapterSet(t *testing.T) {
	adapter := NewMemoryAdapter()
	defer func() { _ = adapter.Close() }()
	ctx := context.Background()

	err := adapter.Set(ctx, "test_key", "test_value", 10*time.Second)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := adapter.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Value is JSON serialized, so "test_value" becomes a quoted string
	expected := `"test_value"`
	if val != expected {
		t.Errorf("Expected %q, got %q", expected, val)
	}
}

func TestMemoryAdapterGet_Miss(t *testing.T) {
	adapter := NewMemoryAdapter()
	defer func() { _ = adapter.Close() }()
	ctx := context.Background()

	_, err := adapter.Get(ctx, "nonexistent")
	if err != cache.ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss, got %v", err)
	}
}

func TestMemoryAdapterDelete(t *testing.T) {
	adapter := NewMemoryAdapter()
	defer func() { _ = adapter.Close() }()
	ctx := context.Background()

	_ = adapter.Set(ctx, "test_key", "test_value", 0)
	_ = adapter.Delete(ctx, "test_key")

	_, err := adapter.Get(ctx, "test_key")
	if err != cache.ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss after delete, got %v", err)
	}
}

func TestMemoryAdapterExists(t *testing.T) {
	adapter := NewMemoryAdapter()
	defer func() { _ = adapter.Close() }()
	ctx := context.Background()

	// Should not exist
	exists, err := adapter.Exists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("Expected exists=false for nonexistent key")
	}

	// Set and check exists
	_ = adapter.Set(ctx, "test_key", "test_value", 0)
	exists, err = adapter.Exists(ctx, "test_key")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Expected exists=true for existing key")
	}
}

func TestMemoryAdapterIncrement(t *testing.T) {
	adapter := NewMemoryAdapter()
	defer func() { _ = adapter.Close() }()
	ctx := context.Background()

	// Increment non-existent key (should treat as 0)
	val, err := adapter.Increment(ctx, "counter", 1)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if val != 1 {
		t.Errorf("Expected 1, got %d", val)
	}

	// Increment again
	val, err = adapter.Increment(ctx, "counter", 5)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if val != 6 {
		t.Errorf("Expected 6, got %d", val)
	}

	// Test negative delta
	val, err = adapter.Increment(ctx, "counter", -3)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if val != 3 {
		t.Errorf("Expected 3, got %d", val)
	}
}

func TestMemoryAdapterIncrement_NonInteger(t *testing.T) {
	adapter := NewMemoryAdapter()
	defer func() { _ = adapter.Close() }()
	ctx := context.Background()

	// Set non-integer value
	_ = adapter.Set(ctx, "string_key", "not_a_number", 0)

	// Try to increment
	_, err := adapter.Increment(ctx, "string_key", 1)
	if err == nil {
		t.Error("Expected error when incrementing non-integer")
	}
	if !cache.IsCacheBackendError(err) {
		t.Errorf("Expected CacheBackendError, got %T", err)
	}
}

func TestMemoryAdapterHealthCheck(t *testing.T) {
	adapter := NewMemoryAdapter()
	defer func() { _ = adapter.Close() }()
	ctx := context.Background()

	err := adapter.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}

func TestMemoryAdapterExpiration(t *testing.T) {
	adapter := NewMemoryAdapter()
	defer func() { _ = adapter.Close() }()
	ctx := context.Background()

	// Set with 100ms TTL
	_ = adapter.Set(ctx, "expiring_key", "value", 100*time.Millisecond)

	// Should exist immediately
	exists, _ := adapter.Exists(ctx, "expiring_key")
	if !exists {
		t.Error("Expected key to exist immediately after Set")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not exist after expiration
	exists, _ = adapter.Exists(ctx, "expiring_key")
	if exists {
		t.Error("Expected key to expire after TTL")
	}

	// Should return miss on Get
	_, err := adapter.Get(ctx, "expiring_key")
	if err != cache.ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss for expired key, got %v", err)
	}
}

func TestMemoryAdapterConcurrency(t *testing.T) {
	adapter := NewMemoryAdapter()
	defer func() { _ = adapter.Close() }()
	ctx := context.Background()

	// Run multiple concurrent operations
	numGoroutines := 10
	numOperations := 100
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key_%d", id%5)
				_ = adapter.Set(ctx, key, j, 0)
				_, _ = adapter.Get(ctx, key)
				_, _ = adapter.Exists(ctx, key)
				_, _ = adapter.Increment(ctx, "counter", 1)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify counter was incremented correctly
	exists, _ := adapter.Exists(ctx, "counter")
	if !exists {
		t.Error("Counter should exist after concurrent increments")
	}
}

func TestMemoryAdapterEvictionLFU(t *testing.T) {
	ctx := context.Background()
	// Set small capacity of 3
	adapter := NewMemoryAdapter(WithMaxItems(3))
	defer func() { _ = adapter.Close() }()

	// Fill cache to capacity
	_ = adapter.Set(ctx, "k1", "v1", 0)
	_ = adapter.Set(ctx, "k2", "v2", 0)
	_ = adapter.Set(ctx, "k3", "v3", 0)

	// Make "k1" and "k3" frequently accessed
	_, _ = adapter.Get(ctx, "k1")
	_, _ = adapter.Get(ctx, "k1")
	_, _ = adapter.Get(ctx, "k3")

	// Set a 4th key. This should trigger eviction.
	// Since sample size is 10 and we have 3 keys, the whole map is evaluated.
	// "k2" has 1 hit (from Set), "k1" has 3 hits, "k3" has 2 hits.
	// "k2" should be evicted.
	_ = adapter.Set(ctx, "k4", "v4", 0)

	// Check that k2 is missing
	_, err := adapter.Get(ctx, "k2")
	if err != cache.ErrCacheMiss {
		t.Errorf("Expected k2 to be evicted, got error %v", err)
	}

	// Check that k1 and k3 are preserved
	_, err = adapter.Get(ctx, "k1")
	if err != nil {
		t.Errorf("k1 was evicted: %v", err)
	}
	_, err = adapter.Get(ctx, "k3")
	if err != nil {
		t.Errorf("k3 was evicted: %v", err)
	}
}

func TestMemoryAdapterEvictionLRU(t *testing.T) {
	ctx := context.Background()
	// Set small capacity of 3
	adapter := NewMemoryAdapter(WithMaxItems(3))
	defer func() { _ = adapter.Close() }()

	// Fill cache to capacity
	_ = adapter.Set(ctx, "k1", "v1", 0)
	_ = adapter.Set(ctx, "k2", "v2", 0)
	_ = adapter.Set(ctx, "k3", "v3", 0)

	// Sleep slightly to differentiate timestamps
	time.Sleep(5 * time.Millisecond)
	// Access k1, then k3. hits will be equal (2 for k1, 2 for k2, 2 for k3 - wait, k2 has 1 hit, k1 and k3 will have 2 hits if accessed).
	// Let's keep hits the same:
	// Set hits: k1 (1 hit), k2 (1 hit), k3 (1 hit)
	// If we access them, they get +1 hits.
	// Let's access k2, then k3.
	_, _ = adapter.Get(ctx, "k2")
	time.Sleep(2 * time.Millisecond)
	_, _ = adapter.Get(ctx, "k3")
	// Now:
	// k1: hits=1, lastAccess=older
	// k2: hits=2, lastAccess=medium
	// k3: hits=2, lastAccess=newest
	// Let's also access k1 to make it hits=2, but lastAccess is the oldest among all hits=2.
	time.Sleep(2 * time.Millisecond)
	_, _ = adapter.Get(ctx, "k1")
	// Now hits: k1=2, k2=2, k3=2
	// lastAccess order: k2 (oldest), k3 (medium), k1 (newest)
	// Adding k4 should evict k2 because it has the same hits but oldest lastAccess.
	time.Sleep(2 * time.Millisecond)
	_ = adapter.Set(ctx, "k4", "v4", 0)

	_, err := adapter.Get(ctx, "k2")
	if err != cache.ErrCacheMiss {
		t.Errorf("Expected k2 to be evicted via LRU tie-breaker, got error %v", err)
	}
}

func TestMemoryAdapterEviction_ExpiredItems(t *testing.T) {
	ctx := context.Background()
	// Set small capacity of 3
	adapter := NewMemoryAdapter(WithMaxItems(3))
	defer func() { _ = adapter.Close() }()

	// Fill cache with expiring items
	_ = adapter.Set(ctx, "k1", "v1", 10*time.Millisecond)
	_ = adapter.Set(ctx, "k2", "v2", 10*time.Millisecond)
	_ = adapter.Set(ctx, "k3", "v3", 10*time.Millisecond)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Set a 4th key. This should trigger the passive/active expiration check during eviction
	_ = adapter.Set(ctx, "k4", "v4", 0)

	// Verify that the expired keys are evicted/cleaned up
	_, err := adapter.Get(ctx, "k1")
	if err != cache.ErrCacheMiss {
		t.Errorf("Expected expired k1 to be cleaned up, got: %v", err)
	}
	_, err = adapter.Get(ctx, "k2")
	if err != cache.ErrCacheMiss {
		t.Errorf("Expected expired k2 to be cleaned up, got: %v", err)
	}
	_, err = adapter.Get(ctx, "k3")
	if err != cache.ErrCacheMiss {
		t.Errorf("Expected expired k3 to be cleaned up, got: %v", err)
	}

	// k4 should be present
	val, err := adapter.Get(ctx, "k4")
	if err != nil {
		t.Fatalf("Expected k4 to be present: %v", err)
	}
	if val != `"v4"` {
		t.Errorf("Expected value %q, got %q", `"v4"`, val)
	}
}

func TestMemoryAdapter_ConcurrentIncrement(t *testing.T) {
	adapter := NewMemoryAdapter()
	defer func() { _ = adapter.Close() }()
	ctx := context.Background()

	numGoroutines := 20
	numIncrements := 100
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numIncrements; j++ {
				_, err := adapter.Increment(ctx, "concurrent_counter", 1)
				if err != nil {
					t.Errorf("Increment failed: %v", err)
				}
			}
			done <- true
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Get final value
	val, err := adapter.Get(ctx, "concurrent_counter")
	if err != nil {
		t.Fatalf("Failed to get counter value: %v", err)
	}

	expectedVal := int64(numGoroutines * numIncrements)
	actualVal, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		t.Fatalf("Failed to parse counter value %q: %v", val, err)
	}

	if actualVal != expectedVal {
		t.Errorf("Expected final counter value to be %d, got %d", expectedVal, actualVal)
	}
}
