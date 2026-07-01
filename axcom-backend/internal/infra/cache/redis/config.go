// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"time"
)

// Config holds Redis-specific configuration.
type Config struct {
	Addr          string        // Host:port, e.g., "localhost:6379"
	Password      string        // Redis password (empty if no auth)
	DB            int           // Database number (0-15), default 0
	PoolSize      int           // Max connections in pool, default 10
	MinIdleConns  int           // Minimum idle connections to keep in the pool
	ReadTimeout   time.Duration // Socket read timeout, default 3s
	WriteTimeout  time.Duration // Socket write timeout, default 3s
	DialTimeout   time.Duration // Connection timeout, default 5s
	MaxValueBytes int           // Max serialized value size in bytes, 0 = unlimited (default 5 MiB)

	// TLS configuration — required for production deployments.
	TLSEnabled bool   // Enable TLS for the Redis connection
	TLSCert    string // Path to client certificate file (mTLS)
	TLSKey     string // Path to client key file (mTLS)
	TLSCA      string // Path to CA certificate file
}

// DefaultConfig returns a reasonable default Redis configuration.
func DefaultConfig() *Config {
	return &Config{
		Addr:          "localhost:6379",
		Password:      "",
		DB:            0,
		PoolSize:      10,
		MinIdleConns:  2,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		DialTimeout:   5 * time.Second,
		MaxValueBytes: 5 << 20, // 5 MiB
	}
}
