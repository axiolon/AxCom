// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"database/sql"
)

// Connection defines the standard database connection contract.
type Connection interface {
	// Ping checks if the database connection is alive.
	Ping(ctx context.Context) error

	// Close terminates the connection pool.
	Close() error
}

// SQLConnection wraps a standard database/sql connection pool.
type SQLConnection struct {
	DB *sql.DB
}

// Ping checks the health of the connection.
func (c *SQLConnection) Ping(ctx context.Context) error {
	return c.DB.PingContext(ctx)
}

// Close closes the underlying pool.
func (c *SQLConnection) Close() error {
	return c.DB.Close()
}

// MemoryConnection mock adapter for testing or local simulation.
type MemoryConnection struct{}

// Ping always succeeds for MemoryConnection.
func (c *MemoryConnection) Ping(_ context.Context) error {
	return nil
}

// Close is a no-op for MemoryConnection.
func (c *MemoryConnection) Close() error {
	return nil
}
