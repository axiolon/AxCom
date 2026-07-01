// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package db defines interfaces representing the database ports/adapters.
package db

import (
	"context"
	"time"
)

// Database defines the behaviors required to interact with a database engine.
type Database interface {
	// Connect initializes a connection pool to the database using the connectionString.
	Connect(ctx context.Context, connectionString string) error

	// Close gracefully terminates all active connections to the database.
	Close(ctx context.Context) error

	// Exec executes a query that does not return rows (e.g. INSERT, UPDATE, DELETE).
	Exec(ctx context.Context, query string, args ...any) error

	// ExecResult executes a query and returns Result metadata (e.g. RowsAffected).
	ExecResult(ctx context.Context, query string, args ...any) (Result, error)

	// Query executes a query that returns rows (e.g. SELECT).
	Query(ctx context.Context, query string, args ...any) (Rows, error)

	// BeginTx starts a database transaction.
	BeginTx(ctx context.Context) (Transaction, error)
}

// Transaction represents a database transaction context.
type Transaction interface {
	Database
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// TransactionManager coordinates transactions database-agnostically.
type TransactionManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Result represents the result of a database query execution.
type Result interface {
	// RowsAffected returns the number of rows affected by the query execution.
	RowsAffected() (int64, error)
}

// Rows represents an active cursor over results returned by a Database Query.
type Rows interface {
	// Next prepares the next result row for reading. It returns true on success,
	// or false if there are no more rows or an error occurred.
	Next() bool

	// Scan copies the columns in the current row into the values pointed at by dest.
	Scan(dest ...any) error

	// Err returns the error, if any, that was encountered during iteration.
	Err() error

	// Close releases any resources associated with the rows cursor.
	Close() error
}

// PoolStats holds point-in-time connection pool metrics.
type PoolStats struct {
	MaxConns          int32         // configured max pool size
	TotalConns        int32         // current total connections (active + idle)
	AcquiredConns     int32         // connections currently in use
	IdleConns         int32         // connections sitting idle in pool
	AcquireCount      int64         // cumulative successful acquires
	EmptyAcquireCount int64         // acquires that had to create a new conn
	AcquireDuration   time.Duration // cumulative time spent acquiring
}

// PoolStatsProvider exposes connection pool metrics.
// Implemented by database adapters that manage a connection pool.
type PoolStatsProvider interface {
	PoolStats() PoolStats
}
