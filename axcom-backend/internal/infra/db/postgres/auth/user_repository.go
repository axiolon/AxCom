// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package auth implements PostgreSQL persistence adapters for user accounts and token storage.
package auth

import (
	"context"
	"ecom-engine/internal/core/auth"
	"ecom-engine/internal/infra/db"
	"ecom-engine/pkg/logger"
	"strings"

	"go.opentelemetry.io/otel"
)

// PostgresUserRepository implements auth.UserRepository using PostgreSQL.
// Use NewPostgresUserRepository to initialize; zero value is invalid.
type PostgresUserRepository struct {
	db db.Database
}

// NewPostgresUserRepository creates a PostgresUserRepository utilizing the provided Database interface.
func NewPostgresUserRepository(database db.Database) *PostgresUserRepository {
	return &PostgresUserRepository{
		db: database,
	}
}

// Create inserts a user record into PostgreSQL.
// It returns auth.ErrEmailAlreadyExists if the email is already taken.
func (r *PostgresUserRepository) Create(ctx context.Context, user *auth.User) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresUserRepository.Create")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Inserting user document for ID: %s", user.ID)

	query := "INSERT INTO users (id, email, password, oidc_sub, role, created_at, updated_at, failed_login_attempts, locked_until) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"
	_, err := r.db.ExecResult(ctx, query, user.ID, user.Email, user.Password, user.OIDCSub, user.Role, user.CreatedAt, user.UpdatedAt, user.FailedLoginAttempts, user.LockedUntil)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to insert user: %v", err)
		errStr := err.Error()
		if strings.Contains(errStr, "23505") || strings.Contains(errStr, "duplicate key") || strings.Contains(errStr, "unique constraint") {
			return auth.ErrEmailAlreadyExists
		}
		return err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully inserted user document for ID: %s", user.ID)
	return nil
}

// GetByEmail retrieves a user record matching the specified email address.
// It returns auth.ErrUserNotFound if no record is matched.
func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*auth.User, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresUserRepository.GetByEmail")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding user by email: %s", email)

	query := "SELECT id, email, password, oidc_sub, role, created_at, updated_at, failed_login_attempts, locked_until FROM users WHERE email = $1 LIMIT 1"
	rows, err := r.db.Query(ctx, query, email)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to query user by email %s: %v", email, err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			span.RecordError(err)
			logger.ErrorCtx(ctx, "Postgres: Error during user rows iteration: %v", err)
			return nil, err
		}
		logger.DebugCtx(ctx, "Postgres: No user found for email: %s", email)
		return nil, auth.ErrUserNotFound
	}

	var u auth.User
	if err := rows.Scan(&u.ID, &u.Email, &u.Password, &u.OIDCSub, &u.Role, &u.CreatedAt, &u.UpdatedAt, &u.FailedLoginAttempts, &u.LockedUntil); err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to scan user: %v", err)
		return nil, err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully found user by email: %s", email)
	return &u, nil
}

// GetByID retrieves a user record matching the specified unique identifier.
// It returns auth.ErrUserNotFound if no record is matched.
func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*auth.User, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresUserRepository.GetByID")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding user by ID: %s", id)

	query := "SELECT id, email, password, oidc_sub, role, created_at, updated_at, failed_login_attempts, locked_until FROM users WHERE id = $1 LIMIT 1"
	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to query user by ID %s: %v", id, err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			span.RecordError(err)
			logger.ErrorCtx(ctx, "Postgres: Error during user rows iteration: %v", err)
			return nil, err
		}
		logger.DebugCtx(ctx, "Postgres: No user found for ID: %s", id)
		return nil, auth.ErrUserNotFound
	}

	var u auth.User
	if err := rows.Scan(&u.ID, &u.Email, &u.Password, &u.OIDCSub, &u.Role, &u.CreatedAt, &u.UpdatedAt, &u.FailedLoginAttempts, &u.LockedUntil); err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to scan user: %v", err)
		return nil, err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully found user by ID: %s", id)
	return &u, nil
}

// UpdatePassword updates the hashed password value of the matched user.
// It returns auth.ErrUserNotFound if the user ID does not exist.
func (r *PostgresUserRepository) UpdatePassword(ctx context.Context, userID string, hashedPassword string) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresUserRepository.UpdatePassword")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Updating password for user ID: %s", userID)

	query := "UPDATE users SET password = $1 WHERE id = $2"
	res, err := r.db.ExecResult(ctx, query, hashedPassword, userID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to update password for user %s: %v", userID, err)
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return err
	}
	if rowsAffected == 0 {
		logger.DebugCtx(ctx, "Postgres: UpdatePassword failed, user ID not found: %s", userID)
		return auth.ErrUserNotFound
	}

	logger.DebugCtx(ctx, "Postgres: Successfully updated password for user ID: %s", userID)
	return nil
}

// Update updates user fields (e.g. lockout details, timestamps).
func (r *PostgresUserRepository) Update(ctx context.Context, user *auth.User) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresUserRepository.Update")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Updating user document for ID: %s", user.ID)

	query := "UPDATE users SET email = $1, password = $2, oidc_sub = $3, role = $4, updated_at = $5, failed_login_attempts = $6, locked_until = $7 WHERE id = $8"
	res, err := r.db.ExecResult(ctx, query, user.Email, user.Password, user.OIDCSub, user.Role, user.UpdatedAt, user.FailedLoginAttempts, user.LockedUntil, user.ID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to update user %s: %v", user.ID, err)
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return err
	}
	if rowsAffected == 0 {
		logger.DebugCtx(ctx, "Postgres: Update failed, user ID not found: %s", user.ID)
		return auth.ErrUserNotFound
	}

	logger.DebugCtx(ctx, "Postgres: Successfully updated user document for ID: %s", user.ID)
	return nil
}

// GetByOIDCSub retrieves a user record matching the specified IdP subject identifier.
// It returns auth.ErrUserNotFound if no record is matched.
func (r *PostgresUserRepository) GetByOIDCSub(ctx context.Context, sub string) (*auth.User, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresUserRepository.GetByOIDCSub")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding user by OIDC sub: %s", sub)

	query := "SELECT id, email, password, oidc_sub, role, created_at, updated_at, failed_login_attempts, locked_until FROM users WHERE oidc_sub = $1 LIMIT 1"
	rows, err := r.db.Query(ctx, query, sub)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to query user by OIDC sub %s: %v", sub, err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			span.RecordError(err)
			logger.ErrorCtx(ctx, "Postgres: Error during user rows iteration: %v", err)
			return nil, err
		}
		logger.DebugCtx(ctx, "Postgres: No user found for OIDC sub: %s", sub)
		return nil, auth.ErrUserNotFound
	}

	var u auth.User
	if err := rows.Scan(&u.ID, &u.Email, &u.Password, &u.OIDCSub, &u.Role, &u.CreatedAt, &u.UpdatedAt, &u.FailedLoginAttempts, &u.LockedUntil); err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to scan user: %v", err)
		return nil, err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully found user by OIDC sub: %s", sub)
	return &u, nil
}
