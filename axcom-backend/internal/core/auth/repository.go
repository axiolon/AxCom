// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package auth provides user registration, credential validation, session tokens, and password recovery.
package auth

import "context"

// UserRepository defines the persistence contract for user records.
// Implementations must be safe for concurrent use.
type UserRepository interface {
	// Create persists a new user record in the storage system.
	Create(ctx context.Context, user *User) error

	// GetByEmail retrieves a user by their email address.
	// It returns an error if the user is not found.
	GetByEmail(ctx context.Context, email string) (*User, error)

	// GetByID retrieves a user by their unique identifier.
	// It returns an error if the user is not found.
	GetByID(ctx context.Context, id string) (*User, error)

	// UpdatePassword updates the hashed password of a user.
	UpdatePassword(ctx context.Context, userID string, hashedPassword string) error

	// Update updates user fields (e.g. lockout details, timestamps).
	Update(ctx context.Context, user *User) error

	// GetByOIDCSub retrieves a user by their IdP subject identifier.
	// Returns ErrUserNotFound if no match exists.
	GetByOIDCSub(ctx context.Context, sub string) (*User, error)
}

// TokenRepository defines the persistence contract for refresh and reset tokens.
// Implementations must be safe for concurrent use.
type TokenRepository interface {
	// SaveRefreshToken persists a new refresh token.
	SaveRefreshToken(ctx context.Context, token *RefreshToken) error

	// GetRefreshToken retrieves a refresh token by its value.
	GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error)

	// RevokeRefreshToken marks a refresh token as revoked to prevent reuse.
	RevokeRefreshToken(ctx context.Context, token string) error

	// RevokeAllUserTokens revokes all refresh tokens belonging to a specific user.
	RevokeAllUserTokens(ctx context.Context, userID string) error

	// GetActiveRefreshTokens retrieves all non-revoked and non-expired refresh tokens for a user.
	GetActiveRefreshTokens(ctx context.Context, userID string) ([]*RefreshToken, error)

	// SavePasswordResetToken persists a password reset token.
	SavePasswordResetToken(ctx context.Context, token *PasswordResetToken) error

	// GetPasswordResetToken retrieves a password reset token by its value.
	GetPasswordResetToken(ctx context.Context, token string) (*PasswordResetToken, error)

	// MarkPasswordResetUsed marks a password reset token as used to prevent replay attacks.
	MarkPasswordResetUsed(ctx context.Context, token string) error

	// InvalidateUserResetTokens marks all active password reset tokens for a user as used/expired.
	InvalidateUserResetTokens(ctx context.Context, userID string) error
}
