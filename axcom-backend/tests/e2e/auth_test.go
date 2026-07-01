// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ecom-engine/tests/e2e/testutil"
)

// authCollections are the collections touched by auth tests — truncated before each test.
var authCollections = []string{"users", "refresh_tokens", "password_reset_tokens"}

func TestAuth_Register(t *testing.T) {
	harness.Truncate(t, authCollections...)

	t.Run("successful registration defaults to customer", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/auth/register", map[string]string{
			"email":    "reg_ok@example.com",
			"password": "Password123!",
		}, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Success bool `json:"success"`
			Data    struct {
				Email    string `json:"email"`
				Role     string `json:"role"`
				Password string `json:"password"` // must be absent
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.True(t, body.Success)
		assert.Equal(t, "reg_ok@example.com", body.Data.Email)
		assert.Equal(t, "customer", body.Data.Role)
		assert.Empty(t, body.Data.Password, "password must not be serialised")
	})

	t.Run("explicit merchant role accepted", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/auth/register", map[string]string{
			"email":    "merchant@example.com",
			"password": "Password123!",
			"role":     "merchant",
		}, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct{ Role string `json:"role"` } `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.Equal(t, "merchant", body.Data.Role)
	})

	t.Run("admin role is rejected", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/auth/register", map[string]string{
			"email":    "admin_attempt@example.com",
			"password": "Password123!",
			"role":     "admin",
		}, "")
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("weak password rejected", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/auth/register", map[string]string{
			"email":    "weak@example.com",
			"password": "weak",
		}, "")
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("duplicate email returns 409", func(t *testing.T) {
		payload := map[string]string{"email": "dup@example.com", "password": "Password123!"}
		r1 := harness.Do(t, http.MethodPost, "/api/auth/register", payload, "")
		require.Equal(t, http.StatusOK, r1.StatusCode)
		r1.Body.Close()

		r2 := harness.Do(t, http.MethodPost, "/api/auth/register", payload, "")
		assert.Equal(t, http.StatusConflict, r2.StatusCode)
		r2.Body.Close()
	})
}

func TestAuth_LoginAndSessionFlow(t *testing.T) {
	harness.Truncate(t, authCollections...)

	// Register a user to test against
	reg := harness.Do(t, http.MethodPost, "/api/auth/register", map[string]string{
		"email":    "session_user@example.com",
		"password": "Password123!",
	}, "")
	require.Equal(t, http.StatusOK, reg.StatusCode)
	reg.Body.Close()

	var refreshToken string

	t.Run("valid login returns tokens", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/auth/login", map[string]string{
			"email":    "session_user@example.com",
			"password": "Password123!",
		}, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.NotEmpty(t, body.Data.AccessToken)
		assert.NotEmpty(t, body.Data.RefreshToken)
		refreshToken = body.Data.RefreshToken
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/auth/login", map[string]string{
			"email":    "session_user@example.com",
			"password": "WrongPass999!",
		}, "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("unknown email returns 401", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/auth/login", map[string]string{
			"email":    "nobody@example.com",
			"password": "Password123!",
		}, "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("refresh token rotates session", func(t *testing.T) {
		require.NotEmpty(t, refreshToken, "depends on 'valid login' sub-test")

		resp := harness.Do(t, http.MethodPost, "/api/auth/refresh", map[string]string{
			"refresh_token": refreshToken,
		}, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		assert.NotEmpty(t, body.Data.AccessToken)
		assert.NotEmpty(t, body.Data.RefreshToken)
		assert.NotEqual(t, refreshToken, body.Data.RefreshToken, "refresh token must be rotated")
		refreshToken = body.Data.RefreshToken
	})

	t.Run("logout revokes refresh token", func(t *testing.T) {
		require.NotEmpty(t, refreshToken, "depends on 'refresh token rotates session' sub-test")

		logoutResp := harness.Do(t, http.MethodPost, "/api/auth/logout", map[string]string{
			"refresh_token": refreshToken,
		}, "")
		require.Equal(t, http.StatusOK, logoutResp.StatusCode)
		logoutResp.Body.Close()

		// The revoked token must no longer work
		retryResp := harness.Do(t, http.MethodPost, "/api/auth/refresh", map[string]string{
			"refresh_token": refreshToken,
		}, "")
		assert.Equal(t, http.StatusUnauthorized, retryResp.StatusCode)
		retryResp.Body.Close()
	})
}

func TestAuth_AccountLockout(t *testing.T) {
	harness.Truncate(t, authCollections...)

	// Register the target user
	reg := harness.Do(t, http.MethodPost, "/api/auth/register", map[string]string{
		"email":    "lockout@example.com",
		"password": "Password123!",
	}, "")
	require.Equal(t, http.StatusOK, reg.StatusCode)
	reg.Body.Close()

	// Exhaust the 5-attempt limit
	for i := range 5 {
		resp := harness.Do(t, http.MethodPost, "/api/auth/login", map[string]string{
			"email":    "lockout@example.com",
			"password": fmt.Sprintf("WrongPass%d!", i),
		}, "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	}

	// Even the correct password should now be rejected
	resp := harness.Do(t, http.MethodPost, "/api/auth/login", map[string]string{
		"email":    "lockout@example.com",
		"password": "Password123!",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestAuth_PasswordResetFlow(t *testing.T) {
	harness.Truncate(t, authCollections...)

	// Register the user
	reg := harness.Do(t, http.MethodPost, "/api/auth/register", map[string]string{
		"email":    "reset@example.com",
		"password": "OldPassword123!",
	}, "")
	require.Equal(t, http.StatusOK, reg.StatusCode)
	reg.Body.Close()

	var resetToken string

	t.Run("request reset returns token in non-production env", func(t *testing.T) {
		// APP_ENV is unset in tests → token is exposed in response
		resp := harness.Do(t, http.MethodPost, "/api/auth/password-reset", map[string]string{
			"email": "reset@example.com",
		}, "")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				ResetToken string `json:"reset_token"`
			} `json:"data"`
		}
		testutil.Decode(t, resp, &body)
		require.NotEmpty(t, body.Data.ResetToken, "reset_token must be present when APP_ENV is unset")
		resetToken = body.Data.ResetToken
	})

	t.Run("unknown email still returns 200 (no enumeration)", func(t *testing.T) {
		resp := harness.Do(t, http.MethodPost, "/api/auth/password-reset", map[string]string{
			"email": "ghost@example.com",
		}, "")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("confirm reset allows login with new password", func(t *testing.T) {
		require.NotEmpty(t, resetToken, "depends on 'request reset' sub-test")

		confirm := harness.Do(t, http.MethodPost, "/api/auth/password-reset/confirm", map[string]interface{}{
			"token":        resetToken,
			"new_password": "NewPassword456!",
		}, "")
		require.Equal(t, http.StatusOK, confirm.StatusCode)
		confirm.Body.Close()

		// Old password must be rejected
		oldLogin := harness.Do(t, http.MethodPost, "/api/auth/login", map[string]string{
			"email":    "reset@example.com",
			"password": "OldPassword123!",
		}, "")
		assert.Equal(t, http.StatusUnauthorized, oldLogin.StatusCode)
		oldLogin.Body.Close()

		// New password must work
		newLogin := harness.Do(t, http.MethodPost, "/api/auth/login", map[string]string{
			"email":    "reset@example.com",
			"password": "NewPassword456!",
		}, "")
		assert.Equal(t, http.StatusOK, newLogin.StatusCode)
		newLogin.Body.Close()
	})

	t.Run("same reset token cannot be reused", func(t *testing.T) {
		require.NotEmpty(t, resetToken, "depends on 'request reset' sub-test")

		resp := harness.Do(t, http.MethodPost, "/api/auth/password-reset/confirm", map[string]interface{}{
			"token":        resetToken,
			"new_password": "AnotherPass789!",
		}, "")
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestAuth_AdminSeedViaDirectInsert(t *testing.T) {
	harness.Truncate(t, authCollections...)

	// Admins can only be created by seeding directly — verifies the SeedUser helper.
	harness.SeedUser(t, "admin@example.com", "Admin123!", "admin")

	accessToken, _ := harness.LoginAs(t, "admin@example.com", "Admin123!")
	assert.NotEmpty(t, accessToken)
}
