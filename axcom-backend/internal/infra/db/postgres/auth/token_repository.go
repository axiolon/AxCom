// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package auth implements PostgreSQL persistence adapters for user accounts and token storage.
package auth

import (
	"context"
	"time"

	"ecom-engine/internal/core/auth"
	"ecom-engine/internal/infra/db"
	"ecom-engine/pkg/logger"

	"go.opentelemetry.io/otel"
)

// PostgresTokenRepository implements auth.TokenRepository using PostgreSQL.
// Use NewPostgresTokenRepository to initialize; zero value is invalid.
type PostgresTokenRepository struct {
	db db.Database
}

// NewPostgresTokenRepository creates a PostgresTokenRepository utilizing the provided Database interface.
func NewPostgresTokenRepository(database db.Database) *PostgresTokenRepository {
	return &PostgresTokenRepository{
		db: database,
	}
}

// SaveRefreshToken persists a refresh token record.
func (r *PostgresTokenRepository) SaveRefreshToken(ctx context.Context, token *auth.RefreshToken) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresTokenRepository.SaveRefreshToken")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Saving refresh token for user ID: %s", token.UserID)

	query := "INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at, revoked) VALUES ($1, $2, $3, $4, $5, $6)"
	_, err := r.db.ExecResult(ctx, query, token.ID, token.UserID, token.Token, token.ExpiresAt, token.CreatedAt, token.Revoked)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to save refresh token: %v", err)
	} else {
		logger.DebugCtx(ctx, "Postgres: Successfully saved refresh token")
	}
	return err
}

// GetRefreshToken retrieves a refresh token record matching the specified token value.
// It returns auth.ErrTokenRevoked if the token does not exist.
func (r *PostgresTokenRepository) GetRefreshToken(ctx context.Context, token string) (*auth.RefreshToken, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresTokenRepository.GetRefreshToken")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Fetching refresh token")

	query := "SELECT id, user_id, token, expires_at, created_at, revoked FROM refresh_tokens WHERE token = $1 LIMIT 1"
	rows, err := r.db.Query(ctx, query, token)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to fetch refresh token: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			span.RecordError(err)
			logger.ErrorCtx(ctx, "Postgres: Error during refresh token rows iteration: %v", err)
			return nil, err
		}
		logger.DebugCtx(ctx, "Postgres: Refresh token not found")
		return nil, auth.ErrTokenRevoked
	}

	var t auth.RefreshToken
	if err := rows.Scan(&t.ID, &t.UserID, &t.Token, &t.ExpiresAt, &t.CreatedAt, &t.Revoked); err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to scan refresh token: %v", err)
		return nil, err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully fetched refresh token")
	return &t, nil
}

// RevokeRefreshToken marks a refresh token record as revoked.
// It returns auth.ErrTokenRevoked if the token does not exist.
func (r *PostgresTokenRepository) RevokeRefreshToken(ctx context.Context, token string) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresTokenRepository.RevokeRefreshToken")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Revoking refresh token")

	query := "UPDATE refresh_tokens SET revoked = true WHERE token = $1"
	res, err := r.db.ExecResult(ctx, query, token)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to revoke refresh token: %v", err)
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return err
	}
	if rowsAffected == 0 {
		logger.DebugCtx(ctx, "Postgres: Revoke failed, token not found")
		return auth.ErrTokenRevoked
	}

	logger.DebugCtx(ctx, "Postgres: Successfully revoked refresh token")
	return nil
}

// RevokeAllUserTokens marks all active refresh token records belonging to a user as revoked.
func (r *PostgresTokenRepository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresTokenRepository.RevokeAllUserTokens")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Revoking all active tokens for user ID: %s", userID)

	query := "UPDATE refresh_tokens SET revoked = true WHERE user_id = $1 AND revoked = false"
	_, err := r.db.ExecResult(ctx, query, userID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to revoke user tokens: %v", err)
	} else {
		logger.DebugCtx(ctx, "Postgres: Successfully revoked all tokens for user ID: %s", userID)
	}
	return err
}

// SavePasswordResetToken persists a password reset token record.
func (r *PostgresTokenRepository) SavePasswordResetToken(ctx context.Context, token *auth.PasswordResetToken) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresTokenRepository.SavePasswordResetToken")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Saving password reset token for user ID: %s", token.UserID)

	query := "INSERT INTO password_reset_tokens (id, user_id, token, expires_at, used) VALUES ($1, $2, $3, $4, $5)"
	_, err := r.db.ExecResult(ctx, query, token.ID, token.UserID, token.Token, token.ExpiresAt, token.Used)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to save reset token: %v", err)
	} else {
		logger.DebugCtx(ctx, "Postgres: Successfully saved reset token")
	}
	return err
}

// GetPasswordResetToken retrieves a password reset token record matching the specified token value.
// It returns auth.ErrResetTokenInvalid if the token does not exist.
func (r *PostgresTokenRepository) GetPasswordResetToken(ctx context.Context, token string) (*auth.PasswordResetToken, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresTokenRepository.GetPasswordResetToken")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Fetching password reset token")

	query := "SELECT id, user_id, token, expires_at, used FROM password_reset_tokens WHERE token = $1 LIMIT 1"
	rows, err := r.db.Query(ctx, query, token)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to fetch reset token: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			span.RecordError(err)
			logger.ErrorCtx(ctx, "Postgres: Error during reset token rows iteration: %v", err)
			return nil, err
		}
		logger.DebugCtx(ctx, "Postgres: Password reset token not found")
		return nil, auth.ErrResetTokenInvalid
	}

	var t auth.PasswordResetToken
	if err := rows.Scan(&t.ID, &t.UserID, &t.Token, &t.ExpiresAt, &t.Used); err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to scan reset token: %v", err)
		return nil, err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully fetched reset token")
	return &t, nil
}

// MarkPasswordResetUsed marks a password reset token record as used.
// It returns auth.ErrResetTokenInvalid if the token does not exist.
func (r *PostgresTokenRepository) MarkPasswordResetUsed(ctx context.Context, token string) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresTokenRepository.MarkPasswordResetUsed")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Marking password reset token as used")

	query := "UPDATE password_reset_tokens SET used = true WHERE token = $1"
	res, err := r.db.ExecResult(ctx, query, token)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to update reset token status: %v", err)
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return err
	}
	if rowsAffected == 0 {
		logger.DebugCtx(ctx, "Postgres: Mark used failed, token not found")
		return auth.ErrResetTokenInvalid
	}

	logger.DebugCtx(ctx, "Postgres: Successfully marked reset token as used")
	return nil
}

// GetActiveRefreshTokens retrieves all non-revoked and non-expired refresh tokens for a user.
func (r *PostgresTokenRepository) GetActiveRefreshTokens(ctx context.Context, userID string) ([]*auth.RefreshToken, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresTokenRepository.GetActiveRefreshTokens")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Fetching active refresh tokens for user ID: %s", userID)

	query := `SELECT id, user_id, token, expires_at, created_at, revoked 
              FROM refresh_tokens 
              WHERE user_id = $1 AND revoked = false AND expires_at > $2
              ORDER BY created_at ASC`
	rows, err := r.db.Query(ctx, query, userID, time.Now())
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to query active refresh tokens: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tokens []*auth.RefreshToken
	for rows.Next() {
		var t auth.RefreshToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Token, &t.ExpiresAt, &t.CreatedAt, &t.Revoked); err != nil {
			span.RecordError(err)
			return nil, err
		}
		tokens = append(tokens, &t)
	}
	if err := rows.Err(); err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Error iterating active refresh tokens: %v", err)
		return nil, err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully fetched active refresh tokens")
	return tokens, nil
}

// InvalidateUserResetTokens marks all active password reset tokens for a user as used/expired.
func (r *PostgresTokenRepository) InvalidateUserResetTokens(ctx context.Context, userID string) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresTokenRepository.InvalidateUserResetTokens")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Invalidating active reset tokens for user ID: %s", userID)

	query := "UPDATE password_reset_tokens SET used = true WHERE user_id = $1 AND used = false"
	_, err := r.db.ExecResult(ctx, query, userID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to invalidate user reset tokens: %v", err)
	}
	return err
}
