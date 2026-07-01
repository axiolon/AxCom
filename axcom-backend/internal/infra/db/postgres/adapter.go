// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package postgres implements the PostgreSQL database adapter using pgx/v5.
package postgres

import (
	"context"
	"ecom-engine/internal/infra/db"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ============================================================================
// pool Configuration & Initialization
// ============================================================================

// Config contains the configuration parameters for the PostgreSQL connection pool.
type Config struct {
	MaxPoolSize           int           // Maximum number of active connections in the pool.
	MinPoolSize           int           // Minimum number of idle connections maintained in the pool.
	MaxConnIdleTime       time.Duration // Maximum amount of time an idle connection can remain in the pool.
	MaxConnLifetime       time.Duration // Maximum lifetime of a connection in the pool.
	MaxConnLifetimeJitter time.Duration // Maximum jitter to add to connection lifetime to prevent simultaneous reconnects.
	ConnectTimeout        time.Duration // Timeout for establishing a new connection.
	HealthCheckInterval   time.Duration // Interval at which connections are checked for health.
	StatementTimeout      time.Duration // Session-level statement execution timeout.
	LockTimeout           time.Duration // Session-level database lock acquisition timeout.
	ApplicationName       string        // Application name reported to PostgreSQL for tracing.
	TransactionTimeout    time.Duration // Timeout enforced on transactions managed by RunInTx.
}

// PostgresAdapter adapts pgxpool.Pool to satisfy the db.Connection interface.
type PostgresAdapter struct { //nolint:revive // Name is intentionally explicit for the public API.
	cfg  Config
	pool *pgxpool.Pool
}

// NewPostgresAdapter creates a new PostgresAdapter with the specified config.
func NewPostgresAdapter(cfg Config) *PostgresAdapter {
	return &PostgresAdapter{cfg: cfg}
}

// ============================================================================
// Database Connection Lifecycle
// ============================================================================

// Connect parses the connection string and initializes the underlying pgx connection pool.
func (a *PostgresAdapter) Connect(ctx context.Context, connectionString string) error {
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return fmt.Errorf("parse pool config: %w", err)
	}

	// Pool sizing
	if a.cfg.MaxPoolSize > 0 && a.cfg.MaxPoolSize <= math.MaxInt32 {
		config.MaxConns = int32(a.cfg.MaxPoolSize) //nolint:gosec
	}
	if a.cfg.MinPoolSize > 0 && a.cfg.MinPoolSize <= math.MaxInt32 {
		config.MinConns = int32(a.cfg.MinPoolSize) //nolint:gosec
	}

	// Connection lifecycle
	if a.cfg.MaxConnLifetime > 0 {
		config.MaxConnLifetime = a.cfg.MaxConnLifetime
	}
	if a.cfg.MaxConnLifetimeJitter > 0 {
		config.MaxConnLifetimeJitter = a.cfg.MaxConnLifetimeJitter
	}
	if a.cfg.MaxConnIdleTime > 0 {
		config.MaxConnIdleTime = a.cfg.MaxConnIdleTime
	}
	if a.cfg.HealthCheckInterval > 0 {
		config.HealthCheckPeriod = a.cfg.HealthCheckInterval
	}

	// Timeouts
	if a.cfg.ConnectTimeout > 0 {
		config.ConnConfig.ConnectTimeout = a.cfg.ConnectTimeout
	}

	// AfterConnect hook: set session-level Postgres parameters.
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		var stmts []string
		if a.cfg.StatementTimeout > 0 {
			stmts = append(stmts, fmt.Sprintf("SET statement_timeout = '%dms'",
				a.cfg.StatementTimeout.Milliseconds()))
		}
		if a.cfg.LockTimeout > 0 {
			stmts = append(stmts, fmt.Sprintf("SET lock_timeout = '%dms'",
				a.cfg.LockTimeout.Milliseconds()))
		}
		if a.cfg.ApplicationName != "" {
			stmts = append(stmts, fmt.Sprintf("SET application_name = '%s'",
				a.cfg.ApplicationName))
		}
		for _, stmt := range stmts {
			if _, err = conn.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("afterConnect SET: %w", err)
			}
		}
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return err
	}
	a.pool = pool
	return nil
}

// Close gracefully shuts down the connection pool.
func (a *PostgresAdapter) Close(_ context.Context) error {
	if a.pool != nil {
		a.pool.Close()
	}
	return nil
}

// ============================================================================
// Query Execution (db.Connection Implementation)
// ============================================================================

// Exec executes a query that does not return rows (e.g., INSERT, UPDATE, DELETE).
// It automatically detects and uses a transaction if present in the context.
func (a *PostgresAdapter) Exec(ctx context.Context, query string, args ...any) error {
	if tx := txFromContext(ctx); tx != nil {
		_, err := tx.Exec(ctx, query, args...)
		return err
	}
	_, err := a.pool.Exec(ctx, query, args...)
	return err
}

// ExecResult executes a query and returns a db.Result indicating the rows affected.
// It automatically detects and uses a transaction if present in the context.
func (a *PostgresAdapter) ExecResult(ctx context.Context, query string, args ...any) (db.Result, error) {
	if tx := txFromContext(ctx); tx != nil {
		tag, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		return &pgResult{rowsAffected: tag.RowsAffected()}, nil
	}
	tag, err := a.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgResult{rowsAffected: tag.RowsAffected()}, nil
}

// Query executes a query that returns rows (e.g., SELECT).
// It automatically detects and uses a transaction if present in the context.
func (a *PostgresAdapter) Query(ctx context.Context, query string, args ...any) (db.Rows, error) {
	if tx := txFromContext(ctx); tx != nil {
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		return &pgRows{Rows: rows}, nil
	}
	rows, err := a.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgRows{Rows: rows}, nil
}

// ============================================================================
// Transaction Support
// ============================================================================

// BeginTx starts a new transaction manually. Nested transactions are not supported.
func (a *PostgresAdapter) BeginTx(ctx context.Context) (db.Transaction, error) {
	if tx := txFromContext(ctx); tx != nil {
		return nil, fmt.Errorf("nested transactions not supported")
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &PostgresTx{adapter: a, tx: tx}, nil
}

// RunInTx executes the given function within a transaction.
// It handles commit on success, rollback on error, and recovery on panic.
func (a *PostgresAdapter) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if tx := txFromContext(ctx); tx != nil {
		return fn(ctx)
	}

	// Apply transaction timeout if configured.
	if a.cfg.TransactionTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.cfg.TransactionTimeout)
		defer cancel()
	}

	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()
	txCtx := contextWithTx(ctx, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// PostgresTx implements the db.Transaction interface.
type PostgresTx struct { //nolint:revive // Name is intentionally explicit for the public API.
	adapter *PostgresAdapter
	tx      pgx.Tx
}

// Connect is not supported within a transaction context and returns an error.
func (t *PostgresTx) Connect(_ context.Context, _ string) error {
	return fmt.Errorf("Connect not supported in transaction")
}

// Close is not supported within a transaction context and returns an error.
func (t *PostgresTx) Close(_ context.Context) error {
	return fmt.Errorf("Close not supported in transaction")
}

// Exec executes a query within the transaction.
func (t *PostgresTx) Exec(ctx context.Context, query string, args ...any) error {
	_, err := t.tx.Exec(ctx, query, args...)
	return err
}

// ExecResult executes a query within the transaction and returns a db.Result.
func (t *PostgresTx) ExecResult(ctx context.Context, query string, args ...any) (db.Result, error) {
	tag, err := t.tx.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgResult{rowsAffected: tag.RowsAffected()}, nil
}

// Query executes a query within the transaction that returns rows.
func (t *PostgresTx) Query(ctx context.Context, query string, args ...any) (db.Rows, error) {
	rows, err := t.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgRows{Rows: rows}, nil
}

// BeginTx returns an error as nested transactions are not supported.
func (t *PostgresTx) BeginTx(_ context.Context) (db.Transaction, error) {
	return nil, fmt.Errorf("nested transactions not supported")
}

// Commit finalizes the transaction.
func (t *PostgresTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback aborts the transaction and discards any changes.
func (t *PostgresTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// ============================================================================
// Pool Statistics
// ============================================================================

// PoolStats retrieves point-in-time statistics about the connection pool.
func (a *PostgresAdapter) PoolStats() db.PoolStats {
	if a.pool == nil {
		return db.PoolStats{}
	}
	s := a.pool.Stat()
	return db.PoolStats{
		MaxConns:          s.MaxConns(),
		TotalConns:        s.TotalConns(),
		AcquiredConns:     s.AcquiredConns(),
		IdleConns:         s.IdleConns(),
		AcquireCount:      s.AcquireCount(),
		EmptyAcquireCount: s.EmptyAcquireCount(),
		AcquireDuration:   s.AcquireDuration(),
	}
}

// ============================================================================
// Internal Types & Context Helpers
// ============================================================================

// pgResult implements the db.Result interface.
type pgResult struct {
	rowsAffected int64
}

// RowsAffected returns the number of rows affected by the query.
func (r *pgResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

// pgRows wraps pgx.Rows to satisfy the db.Rows interface.
type pgRows struct {
	pgx.Rows
}

// Close releases the resources associated with the rows cursor.
func (r *pgRows) Close() error {
	r.Rows.Close()
	return nil
}

// txKey is a unique context key to store and retrieve active transactions.
type txKey struct{}

// contextWithTx returns a new context containing the active pgx.Tx.
func contextWithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// txFromContext retrieves the active pgx.Tx from the context if it exists.
func txFromContext(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return nil
}
