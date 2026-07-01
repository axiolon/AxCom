// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ecom-engine/internal/gateway/middleware"
	"ecom-engine/pkg/token"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// selectorRouter wraps NewAuthMiddleware in a minimal test router.
func selectorRouter(mode string, jm *token.JWTManager, oidc *token.OIDCValidator) *gin.Engine {
	r := gin.New()
	r.Use(middleware.NewAuthMiddleware(mode, jm, oidc, nil))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestNewAuthMiddleware_LocalMode_ValidJWT_Passes(t *testing.T) {
	jm := token.NewJWTManager(testJWTSecret)
	tok, err := jm.Generate("u-1", "user", time.Hour)
	require.NoError(t, err)

	r := selectorRouter("local", jm, nil)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNewAuthMiddleware_LocalMode_NoJWT_Returns401(t *testing.T) {
	jm := token.NewJWTManager(testJWTSecret)
	r := selectorRouter("local", jm, nil)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNewAuthMiddleware_OIDCMode_NilValidator_FallsBackToJWT(t *testing.T) {
	// When mode is "oidc" but validator is nil, selector must fall back to JWT middleware.
	jm := token.NewJWTManager(testJWTSecret)
	tok, err := jm.Generate("u-1", "user", time.Hour)
	require.NoError(t, err)

	r := selectorRouter("oidc", jm, nil) // nil validator → JWT fallback
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "JWT fallback must work when OIDC validator is nil")
}

func TestNewAuthMiddleware_OIDCMode_NilValidator_NoToken_Returns401(t *testing.T) {
	jm := token.NewJWTManager(testJWTSecret)
	r := selectorRouter("oidc", jm, nil)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNewAuthMiddleware_UnknownMode_DefaultsToJWT(t *testing.T) {
	jm := token.NewJWTManager(testJWTSecret)
	tok, err := jm.Generate("u-1", "user", time.Hour)
	require.NoError(t, err)

	r := selectorRouter("saml", jm, nil) // unknown mode → JWT
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
