// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package auth provides user registration, credential validation, session tokens, and password recovery.
package auth

import "time"

// User represents a user account containing identity and credential records.
type User struct {
	ID                  string     `json:"id"`
	Email               string     `json:"email"`
	Password            string     `json:"-"` // Omitted from JSON serialization to prevent credential leaks.
	OIDCSub             string     `json:"oidc_sub,omitempty"`
	Role                string     `json:"role"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	FailedLoginAttempts int        `json:"-"`
	LockedUntil         *time.Time `json:"-"`
}

// Session represents the active tokens issued upon successful authentication.
type Session struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// RefreshToken represents a persisted session key used to rotate access tokens.
type RefreshToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `json:"revoked"`
}

// PasswordResetToken represents a temporary token issued during password recovery.
type PasswordResetToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
}
