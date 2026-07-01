// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJWTManager_GenerateAndValidate(t *testing.T) {
	secret := "my-secret-key-that-is-very-long-and-secure" //nolint:gosec // This is a mock secret key for testing
	manager := NewJWTManager(secret)

	t.Run("valid token", func(t *testing.T) {
		userID := "user-123:with:colons"
		role := "admin"
		duration := 15 * time.Minute

		tok, err := manager.Generate(userID, role, duration)
		assert.NoError(t, err)
		assert.NotEmpty(t, tok)

		claims, err := manager.Validate(tok)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, role, claims.Role)
	})

	t.Run("expired token", func(t *testing.T) {
		userID := "user-123"
		role := "user"
		duration := -1 * time.Minute // expired

		tok, err := manager.Generate(userID, role, duration)
		assert.NoError(t, err)

		_, err = manager.Validate(tok)
		assert.Error(t, err)
		assert.Equal(t, "token expired", err.Error())
	})

	t.Run("invalid signature", func(t *testing.T) {
		userID := "user-123"
		role := "user"
		duration := 15 * time.Minute

		tok, err := manager.Generate(userID, role, duration)
		assert.NoError(t, err)

		// Tamper with the token
		tamperedTok := tok + "extra"
		_, err = manager.Validate(tamperedTok)
		assert.Error(t, err)
		assert.Equal(t, "invalid signature", err.Error())
	})

	t.Run("invalid token format", func(t *testing.T) {
		_, err := manager.Validate("invalid-format")
		assert.Error(t, err)
		assert.Equal(t, "invalid token format", err.Error())
	})
}
