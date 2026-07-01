// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"time"

	"ecom-engine/internal/core/auth"
)

// This contain three collections: users, refresh_tokens, and password_reset_tokens

// userDoc represents the MongoDB document schema for Users.
type userDoc struct {
	ID                  string     `bson:"_id"`
	Email               string     `bson:"email"`
	Password            string     `bson:"password"`
	OIDCSub             string     `bson:"oidc_sub,omitempty"`
	Role                string     `bson:"role"`
	CreatedAt           time.Time  `bson:"created_at"`
	UpdatedAt           time.Time  `bson:"updated_at"`
	FailedLoginAttempts int        `bson:"failed_login_attempts"`
	LockedUntil         *time.Time `bson:"locked_until,omitempty"`
}

// toUserDoc maps a domain User into a userDoc database entity.
func toUserDoc(u *auth.User) *userDoc {
	if u == nil {
		return nil
	}
	return &userDoc{
		ID:                  u.ID,
		Email:               u.Email,
		Password:            u.Password,
		OIDCSub:             u.OIDCSub,
		Role:                u.Role,
		CreatedAt:           u.CreatedAt,
		UpdatedAt:           u.UpdatedAt,
		FailedLoginAttempts: u.FailedLoginAttempts,
		LockedUntil:         u.LockedUntil,
	}
}

// toDomainUser maps a userDoc database entity into a domain User.
func toDomainUser(doc *userDoc) *auth.User {
	if doc == nil {
		return nil
	}
	return &auth.User{
		ID:                  doc.ID,
		Email:               doc.Email,
		Password:            doc.Password,
		OIDCSub:             doc.OIDCSub,
		Role:                doc.Role,
		CreatedAt:           doc.CreatedAt,
		UpdatedAt:           doc.UpdatedAt,
		FailedLoginAttempts: doc.FailedLoginAttempts,
		LockedUntil:         doc.LockedUntil,
	}
}

// refreshTokenDoc represents the MongoDB document schema for RefreshTokens.
type refreshTokenDoc struct {
	ID        string    `bson:"_id"`
	UserID    string    `bson:"user_id"`
	Token     string    `bson:"token"`
	ExpiresAt time.Time `bson:"expires_at"`
	CreatedAt time.Time `bson:"created_at"`
	Revoked   bool      `bson:"revoked"`
}

// toRefreshTokenDoc maps a domain RefreshToken into a refreshTokenDoc database entity.
func toRefreshTokenDoc(t *auth.RefreshToken) *refreshTokenDoc {
	if t == nil {
		return nil
	}
	return &refreshTokenDoc{
		ID:        t.ID,
		UserID:    t.UserID,
		Token:     t.Token,
		ExpiresAt: t.ExpiresAt,
		CreatedAt: t.CreatedAt,
		Revoked:   t.Revoked,
	}
}

// toDomainRefreshToken maps a refreshTokenDoc database entity into a domain RefreshToken.
func toDomainRefreshToken(doc *refreshTokenDoc) *auth.RefreshToken {
	if doc == nil {
		return nil
	}
	return &auth.RefreshToken{
		ID:        doc.ID,
		UserID:    doc.UserID,
		Token:     doc.Token,
		ExpiresAt: doc.ExpiresAt,
		CreatedAt: doc.CreatedAt,
		Revoked:   doc.Revoked,
	}
}

// passwordResetTokenDoc represents the MongoDB document schema for PasswordResetTokens.
type passwordResetTokenDoc struct {
	ID        string    `bson:"_id"`
	UserID    string    `bson:"user_id"`
	Token     string    `bson:"token"`
	ExpiresAt time.Time `bson:"expires_at"`
	Used      bool      `bson:"used"`
}

// toPasswordResetTokenDoc maps a domain PasswordResetToken into a passwordResetTokenDoc database entity.
func toPasswordResetTokenDoc(t *auth.PasswordResetToken) *passwordResetTokenDoc {
	if t == nil {
		return nil
	}
	return &passwordResetTokenDoc{
		ID:        t.ID,
		UserID:    t.UserID,
		Token:     t.Token,
		ExpiresAt: t.ExpiresAt,
		Used:      t.Used,
	}
}

// toDomainPasswordResetToken maps a passwordResetTokenDoc database entity into a domain PasswordResetToken.
func toDomainPasswordResetToken(doc *passwordResetTokenDoc) *auth.PasswordResetToken {
	if doc == nil {
		return nil
	}
	return &auth.PasswordResetToken{
		ID:        doc.ID,
		UserID:    doc.UserID,
		Token:     doc.Token,
		ExpiresAt: doc.ExpiresAt,
		Used:      doc.Used,
	}
}
