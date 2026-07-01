// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ecom-engine/internal/infra/cache"
	"ecom-engine/internal/infra/cache/memory"
)

type TestItem struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestCacheManager_GetOrFetch(t *testing.T) {
	ctx := context.Background()

	t.Run("L1 Hit", func(t *testing.T) {
		l1 := memory.NewMemoryAdapter()
		defer func() { _ = l1.Close() }()
		l2 := memory.NewMemoryAdapter()
		defer func() { _ = l2.Close() }()

		mgr := cache.NewCacheManager(l1, l2)
		defer func() { _ = mgr.Close() }()

		// Pre-populate L1
		item := TestItem{Name: "L1Item", Value: 42}
		_ = l1.Set(ctx, "k1", item, 10*time.Second)

		var result TestItem
		called := false
		fetchFn := func() (interface{}, error) {
			called = true
			return &TestItem{Name: "Fetched", Value: 99}, nil
		}

		err := mgr.GetOrFetch(ctx, "k1", &result, 10*time.Second, fetchFn)
		if err != nil {
			t.Fatalf("GetOrFetch failed: %v", err)
		}

		if called {
			t.Error("fetchFn should not have been called on L1 hit")
		}
		if result.Name != "L1Item" || result.Value != 42 {
			t.Errorf("Unexpected result: %+v", result)
		}
	})

	t.Run("L2 Hit populates L1", func(t *testing.T) {
		l1 := memory.NewMemoryAdapter()
		defer func() { _ = l1.Close() }()
		l2 := memory.NewMemoryAdapter()
		defer func() { _ = l2.Close() }()

		mgr := cache.NewCacheManager(l1, l2)
		defer func() { _ = mgr.Close() }()

		// Pre-populate L2
		item := TestItem{Name: "L2Item", Value: 100}
		_ = l2.Set(ctx, "k2", item, 10*time.Second)

		var result TestItem
		called := false
		fetchFn := func() (interface{}, error) {
			called = true
			return &TestItem{Name: "Fetched", Value: 99}, nil
		}

		err := mgr.GetOrFetch(ctx, "k2", &result, 10*time.Second, fetchFn)
		if err != nil {
			t.Fatalf("GetOrFetch failed: %v", err)
		}

		if called {
			t.Error("fetchFn should not have been called on L2 hit")
		}
		if result.Name != "L2Item" || result.Value != 100 {
			t.Errorf("Unexpected result: %+v", result)
		}

		// Verify L1 is now populated
		val, err := l1.Get(ctx, "k2")
		if err != nil {
			t.Fatalf("Expected key to be in L1: %v", err)
		}
		expectedJSON := `{"name":"L2Item","value":100}`
		if val != expectedJSON {
			t.Errorf("Expected L1 value to be %q, got %q", expectedJSON, val)
		}
	})

	t.Run("Full Miss calls fetchFn and populates L1 and L2", func(t *testing.T) {
		l1 := memory.NewMemoryAdapter()
		defer func() { _ = l1.Close() }()
		l2 := memory.NewMemoryAdapter()
		defer func() { _ = l2.Close() }()

		mgr := cache.NewCacheManager(l1, l2)
		defer func() { _ = mgr.Close() }()

		var result TestItem
		called := false
		fetchFn := func() (interface{}, error) {
			called = true
			return &TestItem{Name: "DatabaseItem", Value: 777}, nil
		}

		err := mgr.GetOrFetch(ctx, "k3", &result, 10*time.Second, fetchFn)
		if err != nil {
			t.Fatalf("GetOrFetch failed: %v", err)
		}

		if !called {
			t.Error("Expected fetchFn to be called on full miss")
		}
		if result.Name != "DatabaseItem" || result.Value != 777 {
			t.Errorf("Unexpected result: %+v", result)
		}

		// Verify L1 and L2 are populated
		val1, err1 := l1.Get(ctx, "k3")
		if err1 != nil {
			t.Fatalf("Expected key to be in L1: %v", err1)
		}
		val2, err2 := l2.Get(ctx, "k3")
		if err2 != nil {
			t.Fatalf("Expected key to be in L2: %v", err2)
		}

		expectedJSON := `{"name":"DatabaseItem","value":777}`
		if val1 != expectedJSON || val2 != expectedJSON {
			t.Errorf("Expected L1/L2 to hold %q, got L1=%q, L2=%q", expectedJSON, val1, val2)
		}
	})

	t.Run("FetchFn error propagation", func(t *testing.T) {
		l1 := memory.NewMemoryAdapter()
		defer func() { _ = l1.Close() }()
		l2 := memory.NewMemoryAdapter()
		defer func() { _ = l2.Close() }()

		mgr := cache.NewCacheManager(l1, l2)
		defer func() { _ = mgr.Close() }()

		var result TestItem
		expectedErr := errors.New("db connection failure")
		fetchFn := func() (interface{}, error) {
			return nil, expectedErr
		}

		err := mgr.GetOrFetch(ctx, "k4", &result, 10*time.Second, fetchFn)
		if !errors.Is(err, expectedErr) {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})
}

type errorCache struct {
	cache.Cache
	err error
}

func (e *errorCache) Delete(_ context.Context, _ string) error {
	return e.err
}

func (e *errorCache) Close() error {
	return e.err
}

func TestCacheManager_InvalidateAndCloseErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("Invalidate error accumulation", func(t *testing.T) {
		err1 := errors.New("L1 delete failed")
		err2 := errors.New("L2 delete failed")

		l1 := &errorCache{err: err1}
		l2 := &errorCache{err: err2}

		mgr := cache.NewCacheManager(l1, l2)
		err := mgr.Invalidate(ctx, "test_key")
		if err == nil {
			t.Fatal("Expected error on invalidate, got nil")
		}

		if !errors.Is(err, err1) || !errors.Is(err, err2) {
			t.Errorf("Expected combined errors, got: %v", err)
		}
	})

	t.Run("Close error accumulation", func(t *testing.T) {
		err1 := errors.New("L1 close failed")
		err2 := errors.New("L2 close failed")

		l1 := &errorCache{err: err1}
		l2 := &errorCache{err: err2}

		mgr := cache.NewCacheManager(l1, l2)
		err := mgr.Close()
		if err == nil {
			t.Fatal("Expected error on close, got nil")
		}

		if !errors.Is(err, err1) || !errors.Is(err, err2) {
			t.Errorf("Expected combined errors, got: %v", err)
		}
	})
}
