// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package auth provides user registration, credential validation, session tokens, and password recovery.
package auth

import "errors"

var (
	// ErrUserNotFound indicates the user does not exist.
	ErrUserNotFound = errors.New("user not found")

	// ErrInvalidCredentials indicates the credentials provided do not match.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrEmailAlreadyExists indicates the email is already in use.
	ErrEmailAlreadyExists = errors.New("email already exists")

	// ErrTokenExpired indicates the token validity period has passed.
	ErrTokenExpired = errors.New("token has expired")

	// ErrTokenRevoked indicates the token is no longer valid.
	ErrTokenRevoked = errors.New("token has been revoked")

	// ErrResetTokenUsed indicates the reset token has already been consumed.
	ErrResetTokenUsed = errors.New("reset token already used")

	// ErrResetTokenInvalid indicates the reset token is unrecognized.
	ErrResetTokenInvalid = errors.New("invalid reset token")
)
