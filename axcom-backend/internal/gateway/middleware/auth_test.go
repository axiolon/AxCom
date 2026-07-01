// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ecom-engine/internal/gateway/middleware"
	"ecom-engine/pkg/ctxkeys"
	"ecom-engine/pkg/token"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const testJWTSecret = "test-secret-key-that-is-long-enough-for-hmac"

func newTestJWTManager() *token.JWTManager {
	return token.NewJWTManager(testJWTSecret)
}

func makeValidToken(t *testing.T, jm *token.JWTManager, userID, role string) string {
	t.Helper()
	tok, err := jm.Generate(userID, role, time.Hour)
	require.NoError(t, err)
	return tok
}

func makeExpiredToken(t *testing.T, jm *token.JWTManager) string {
	t.Helper()
	tok, err := jm.Generate("user-1", "user", -time.Second)
	require.NoError(t, err)
	return tok
}

// authTestRouter wires a single protected route behind AuthMiddleware.
// The handler echoes Gin context values and downstream headers so tests can
// assert what the middleware injected.
func authTestRouter(jm *token.JWTManager) *gin.Engine {
	r := gin.New()
	r.Use(middleware.AuthMiddleware(jm))
	r.GET("/protected", func(c *gin.Context) {
		ginUserID, _ := c.Get(string(ctxkeys.UserIDKey))
		ginRole, _ := c.Get(string(ctxkeys.UserRoleKey))
		ctxUserID := c.Request.Context().Value(ctxkeys.UserIDKey)
		ctxRole := c.Request.Context().Value(ctxkeys.UserRoleKey)
		c.JSON(http.StatusOK, gin.H{
			"gin_user_id": ginUserID,
			"gin_role":    ginRole,
			"ctx_user_id": ctxUserID,
			"ctx_role":    ctxRole,
			"x_user_id":   c.Request.Header.Get("X-User-ID"),
			"x_user_role": c.Request.Header.Get("X-User-Role"),
		})
	})
	return r
}

func TestAuthMiddleware_MissingHeader_Returns401(t *testing.T) {
	r := authTestRouter(newTestJWTManager())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_MalformedHeader_NoSpace_Returns401(t *testing.T) {
	r := authTestRouter(newTestJWTManager())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "BearerTokenNoSpace")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_MalformedHeader_WrongScheme_Returns401(t *testing.T) {
	r := authTestRouter(newTestJWTManager())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_MalformedHeader_TooManyParts_Returns401(t *testing.T) {
	r := authTestRouter(newTestJWTManager())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer token extra-part")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidSignature_Returns401(t *testing.T) {
	// Token signed with a different secret.
	otherManager := token.NewJWTManager("completely-different-secret-key")
	tok := makeValidToken(t, otherManager, "user-1", "user")

	r := authTestRouter(newTestJWTManager())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_ExpiredToken_Returns401(t *testing.T) {
	jm := newTestJWTManager()
	tok := makeExpiredToken(t, jm)

	r := authTestRouter(jm)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_GarbageToken_Returns401(t *testing.T) {
	r := authTestRouter(newTestJWTManager())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-token-at-all")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_ValidToken_Returns200_AndInjectsIdentity(t *testing.T) {
	jm := newTestJWTManager()
	tok := makeValidToken(t, jm, "user-42", "admin")

	r := authTestRouter(jm)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))

	assert.Equal(t, "user-42", body["gin_user_id"], "Gin context must carry user_id")
	assert.Equal(t, "admin", body["gin_role"], "Gin context must carry role")
	assert.Equal(t, "user-42", body["ctx_user_id"], "standard context must carry user_id")
	assert.Equal(t, "admin", body["ctx_role"], "standard context must carry role")
	assert.Equal(t, "user-42", body["x_user_id"], "X-User-ID downstream header must be set")
	assert.Equal(t, "admin", body["x_user_role"], "X-User-Role downstream header must be set")
}

func TestAuthMiddleware_Aborts_NextHandlerNotCalled(t *testing.T) {
	jm := newTestJWTManager()
	nextCalled := false

	r := gin.New()
	r.Use(middleware.AuthMiddleware(jm))
	r.GET("/protected", func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req) // no Authorization header

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, nextCalled, "handler must not be invoked when auth middleware aborts")
}
