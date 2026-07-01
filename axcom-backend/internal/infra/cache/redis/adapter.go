// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"ecom-engine/internal/infra/cache"
	"ecom-engine/pkg/logger"

	"github.com/redis/go-redis/v9"
)

const (
	// maxKeyLength is the maximum allowed key length in bytes.
	maxKeyLength = 512
)

// RedisAdapter is a production-grade Redis cache implementation.
type RedisAdapter struct { //nolint:revive // Name is intentionally explicit for the public API.
	client *redis.Client
	cfg    *Config
}

// NewRedisAdapter creates a new Redis cache adapter and connects to Redis.
func NewRedisAdapter(ctx context.Context, cfg *Config) (*RedisAdapter, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Warn if running without auth — safe to start but not safe for production.
	if cfg.Password == "" {
		logger.Warn("Redis adapter started without a password. Set Config.Password for production deployments.")
	}

	opts := &redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		DialTimeout:  cfg.DialTimeout,
	}

	// Wire TLS if enabled.
	if cfg.TLSEnabled {
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}

		if cfg.TLSCA != "" {
			caPEM, err := os.ReadFile(cfg.TLSCA)
			if err != nil {
				return nil, cache.NewCacheBackendError("connect", "", fmt.Errorf("read TLS CA: %w", err))
			}
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(caPEM) {
				return nil, cache.NewCacheBackendError("connect", "", fmt.Errorf("parse TLS CA certificate"))
			}
			tlsCfg.RootCAs = pool
		}

		if cfg.TLSCert != "" && cfg.TLSKey != "" {
			cert, err := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
			if err != nil {
				return nil, cache.NewCacheBackendError("connect", "", fmt.Errorf("load TLS client cert: %w", err))
			}
			tlsCfg.Certificates = []tls.Certificate{cert}
		}

		opts.TLSConfig = tlsCfg
	}

	client := redis.NewClient(opts)

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, cache.NewCacheBackendError("connect", "", err)
	}

	return &RedisAdapter{
		client: client,
		cfg:    cfg,
	}, nil
}

// Set stores a value with an optional TTL.
func (a *RedisAdapter) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if len(key) > maxKeyLength {
		return cache.NewCacheBackendError("set", key, cache.ErrKeyTooLong)
	}

	jsonData, err := json.Marshal(value)
	if err != nil {
		return cache.NewCacheBackendError("set", key, fmt.Errorf("json marshal: %w", err))
	}

	if a.cfg.MaxValueBytes > 0 && len(jsonData) > a.cfg.MaxValueBytes {
		return cache.NewCacheBackendError("set", key, cache.ErrValueTooLarge)
	}

	if err := a.client.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		return cache.NewCacheBackendError("set", key, err)
	}

	return nil
}

// Get retrieves a value by key.
func (a *RedisAdapter) Get(ctx context.Context, key string) (string, error) {
	val, err := a.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", cache.ErrCacheMiss
		}
		return "", cache.NewCacheBackendError("get", key, err)
	}

	return val, nil
}

// Delete removes a key from the cache.
func (a *RedisAdapter) Delete(ctx context.Context, key string) error {
	if err := a.client.Del(ctx, key).Err(); err != nil {
		return cache.NewCacheBackendError("delete", key, err)
	}

	return nil
}

// Exists checks if a key exists without retrieving the value.
func (a *RedisAdapter) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := a.client.Exists(ctx, key).Result()
	if err != nil {
		return false, cache.NewCacheBackendError("exists", key, err)
	}

	return exists > 0, nil
}

// Increment atomically increments an integer value at the given key.
func (a *RedisAdapter) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	val, err := a.client.IncrBy(ctx, key, delta).Result()
	if err != nil {
		return 0, cache.NewCacheBackendError("increment", key, err)
	}

	return val, nil
}

// HealthCheck verifies Redis is operational.
func (a *RedisAdapter) HealthCheck(ctx context.Context) error {
	if err := a.client.Ping(ctx).Err(); err != nil {
		return cache.NewCacheBackendError("healthcheck", "", err)
	}

	return nil
}

// Close closes the Redis connection.
func (a *RedisAdapter) Close() error {
	return a.client.Close()
}

// Client returns the underlying go-redis client.
// Use this to register a metrics.CacheRedisPoolCollector with the Prometheus registry.
func (a *RedisAdapter) Client() *redis.Client {
	return a.client
}
