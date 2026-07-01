// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ecom-engine/internal/infra/cache"
)

// TestRedisAdapterIntegration tests require a running Redis instance on localhost:6379
// Set SKIP_REDIS_TESTS=1 to skip these tests if Redis is not available
func TestRedisAdapterSet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &Config{
		Addr: "localhost:6379",
	}

	adapter, err := NewRedisAdapter(ctx, cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	err = adapter.Set(ctx, "test_key", "test_value", 10*time.Second)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	defer func() { _ = adapter.Delete(ctx, "test_key") }()

	val, err := adapter.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	expected := `"test_value"`
	if val != expected {
		t.Errorf("Expected %q, got %q", expected, val)
	}
}

func TestRedisAdapterGet_Miss(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &Config{Addr: "localhost:6379"}
	adapter, err := NewRedisAdapter(ctx, cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	_, err = adapter.Get(ctx, fmt.Sprintf("nonexistent_key_%d", time.Now().UnixNano()))
	if err != cache.ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss, got %v", err)
	}
}

func TestRedisAdapterDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &Config{Addr: "localhost:6379"}
	adapter, err := NewRedisAdapter(ctx, cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	_ = adapter.Set(ctx, "delete_test", "value", 10*time.Second)
	defer func() { _ = adapter.Delete(ctx, "delete_test") }()
	_ = adapter.Delete(ctx, "delete_test")

	_, err = adapter.Get(ctx, "delete_test")
	if err != cache.ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss after delete, got %v", err)
	}
}

func TestRedisAdapterExists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &Config{Addr: "localhost:6379"}
	adapter, err := NewRedisAdapter(ctx, cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	// Should not exist
	nonexistentKey := fmt.Sprintf("nonexistent_key_%d", time.Now().UnixNano())
	exists, err := adapter.Exists(ctx, nonexistentKey)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("Expected exists=false for nonexistent key")
	}

	// Set and check exists
	_ = adapter.Set(ctx, "exists_test", "value", 10*time.Second)
	defer func() { _ = adapter.Delete(ctx, "exists_test") }()
	exists, err = adapter.Exists(ctx, "exists_test")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Expected exists=true for existing key")
	}
}

func TestRedisAdapterIncrement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &Config{Addr: "localhost:6379"}
	adapter, err := NewRedisAdapter(ctx, cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	counterKey := fmt.Sprintf("counter_%d", time.Now().UnixNano())
	defer func() { _ = adapter.Delete(ctx, counterKey) }()

	// Increment non-existent key (should treat as 0)
	val, err := adapter.Increment(ctx, counterKey, 1)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if val != 1 {
		t.Errorf("Expected 1, got %d", val)
	}

	// Increment again
	val, err = adapter.Increment(ctx, counterKey, 5)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if val != 6 {
		t.Errorf("Expected 6, got %d", val)
	}
}

func TestRedisAdapterHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &Config{Addr: "localhost:6379"}
	adapter, err := NewRedisAdapter(ctx, cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	err = adapter.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}
