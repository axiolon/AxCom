// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"bytes"
	"context"
	"ecom-engine/internal/core/auth/dto"
	"ecom-engine/pkg/token"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func performRequest(r http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	var jsonBody []byte
	if body != nil {
		jsonBody, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestController_Register(t *testing.T) {
	t.Parallel()

	userRepo := NewMockUserRepository()
	tokenRepo := NewMockTokenRepository()
	jwtManager := token.NewJWTManager("secret_key_123")
	service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})
	controller := NewController(service)

	router := gin.New()
	rg := router.Group("/api")
	RegisterRoutes(rg, controller)

	t.Run("successful registration", func(t *testing.T) {
		reqBody := dto.AuthRequest{
			Email:    "testreg_success@example.com",
			Password: "Password123!",
			Role:     "customer",
		}

		w := performRequest(router, "POST", "/api/auth/register", reqBody)
		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		respData, ok := resp["data"].(map[string]interface{})
		require.True(t, ok, "expected data field in response")

		assert.Equal(t, "testreg_success@example.com", respData["email"])
		assert.NotContains(t, respData, "password")
	})

	t.Run("validation error - weak password", func(t *testing.T) {
		reqBody := dto.AuthRequest{
			Email:    "testreg_weak@example.com",
			Password: "weak",
		}

		w := performRequest(router, "POST", "/api/auth/register", reqBody)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("duplicate registration conflict", func(t *testing.T) {
		// Pre-register user first to guarantee conflict
		hashed, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)
		require.NoError(t, err)
		err = userRepo.Create(context.Background(), &User{
			ID:        "usr_dup",
			Email:     "testreg_dup@example.com",
			Password:  string(hashed),
			Role:      "customer",
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)

		reqBody := dto.AuthRequest{
			Email:    "testreg_dup@example.com",
			Password: "Password123!",
		}

		w := performRequest(router, "POST", "/api/auth/register", reqBody)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("role whitelist rejection", func(t *testing.T) {
		reqBody := dto.AuthRequest{
			Email:    "adminreg@example.com",
			Password: "Password123!",
			Role:     "admin",
		}

		w := performRequest(router, "POST", "/api/auth/register", reqBody)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestController_LoginAndRefreshWorkflow(t *testing.T) {
	t.Parallel()

	userRepo := NewMockUserRepository()
	tokenRepo := NewMockTokenRepository()
	jwtManager := token.NewJWTManager("secret_key_123")
	service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})
	controller := NewController(service)

	router := gin.New()
	rg := router.Group("/api")
	RegisterRoutes(rg, controller)

	// Pre-register user
	hashed, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)
	require.NoError(t, err)
	err = userRepo.Create(context.Background(), &User{
		ID:        "usr_123",
		Email:     "flow@example.com",
		Password:  string(hashed),
		Role:      "customer",
		CreatedAt: time.Now(),
	})
	require.NoError(t, err)

	var refreshToken string

	t.Run("successful login", func(t *testing.T) {
		reqBody := dto.AuthRequest{
			Email:    "flow@example.com",
			Password: "Password123!",
		}

		w := performRequest(router, "POST", "/api/auth/login", reqBody)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool    `json:"success"`
			Data    Session `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.NotEmpty(t, resp.Data.AccessToken)
		assert.NotEmpty(t, resp.Data.RefreshToken)

		refreshToken = resp.Data.RefreshToken
	})

	t.Run("invalid login credentials", func(t *testing.T) {
		reqBody := dto.AuthRequest{
			Email:    "flow@example.com",
			Password: "WrongPassword123!",
		}

		w := performRequest(router, "POST", "/api/auth/login", reqBody)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("successful token refresh", func(t *testing.T) {
		require.NotEmpty(t, refreshToken, "RefreshToken must be set by preceding login test")
		reqBody := dto.RefreshRequest{
			RefreshToken: refreshToken,
		}

		w := performRequest(router, "POST", "/api/auth/refresh", reqBody)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool    `json:"success"`
			Data    Session `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.NotEmpty(t, resp.Data.AccessToken)
		assert.NotEmpty(t, resp.Data.RefreshToken)

		refreshToken = resp.Data.RefreshToken // update to rotated refresh token
	})

	t.Run("successful logout", func(t *testing.T) {
		require.NotEmpty(t, refreshToken, "RefreshToken must be set by preceding refresh test")
		reqBody := dto.LogoutRequest{
			RefreshToken: refreshToken,
		}

		w := performRequest(router, "POST", "/api/auth/logout", reqBody)
		require.Equal(t, http.StatusOK, w.Code)

		// Try to refresh again, should fail because token was revoked
		refreshReq := dto.RefreshRequest{
			RefreshToken: refreshToken,
		}
		w2 := performRequest(router, "POST", "/api/auth/refresh", refreshReq)
		assert.Equal(t, http.StatusUnauthorized, w2.Code)
	})
}

func TestController_PasswordResetWorkflow(t *testing.T) {
	t.Parallel()

	userRepo := NewMockUserRepository()
	tokenRepo := NewMockTokenRepository()
	jwtManager := token.NewJWTManager("secret_key_123")
	service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})
	controller := NewController(service)

	router := gin.New()
	rg := router.Group("/api")
	RegisterRoutes(rg, controller)

	// Pre-register user
	hashed, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)
	require.NoError(t, err)
	err = userRepo.Create(context.Background(), &User{
		ID:        "usr_123",
		Email:     "resetflow@example.com",
		Password:  string(hashed),
		Role:      "customer",
		CreatedAt: time.Now(),
	})
	require.NoError(t, err)

	var resetToken string

	t.Run("request password reset success", func(t *testing.T) {
		reqBody := dto.PasswordResetRequest{
			Email: "resetflow@example.com",
		}

		w := performRequest(router, "POST", "/api/auth/password-reset", reqBody)
		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		respData, ok := resp["data"].(map[string]interface{})
		require.True(t, ok, "expected data field in response")

		tokenVal, ok := respData["reset_token"].(string)
		require.True(t, ok, "reset_token missing or not a string")
		resetToken = tokenVal
	})

	t.Run("confirm password reset success", func(t *testing.T) {
		require.NotEmpty(t, resetToken, "resetToken must be set by preceding request password reset test")
		reqBody := dto.PasswordResetConfirmRequest{
			Token:       resetToken,
			NewPassword: "NewSecretPassword123!",
		}

		w := performRequest(router, "POST", "/api/auth/password-reset/confirm", reqBody)
		require.Equal(t, http.StatusOK, w.Code)

		// Verify we can login with the new password now
		loginReq := dto.AuthRequest{
			Email:    "resetflow@example.com",
			Password: "NewSecretPassword123!",
		}
		w2 := performRequest(router, "POST", "/api/auth/login", loginReq)
		assert.Equal(t, http.StatusOK, w2.Code)
	})
}
