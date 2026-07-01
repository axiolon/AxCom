// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ecom-engine/internal/gateway/middleware"
	"ecom-engine/pkg/token"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// oidcEarlyReturnRouter builds a router with OIDCAuthMiddleware.
// A nil validator and nil authService are intentionally passed: the early-return
// paths (missing/malformed header) abort before either dependency is used.
func oidcEarlyReturnRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.OIDCAuthMiddleware((*token.OIDCValidator)(nil), nil))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestOIDCAuthMiddleware_MissingHeader_Returns401(t *testing.T) {
	r := oidcEarlyReturnRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOIDCAuthMiddleware_MalformedHeader_NoSpace_Returns401(t *testing.T) {
	r := oidcEarlyReturnRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "BearerTokenNoSpace")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOIDCAuthMiddleware_MalformedHeader_WrongScheme_Returns401(t *testing.T) {
	r := oidcEarlyReturnRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOIDCAuthMiddleware_Aborts_NextNotCalled(t *testing.T) {
	nextCalled := false
	r := gin.New()
	r.Use(middleware.OIDCAuthMiddleware((*token.OIDCValidator)(nil), nil))
	r.GET("/protected", func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil) // no header
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, nextCalled, "next handler must not run when OIDC auth aborts")
}

// Note: Full OIDC validation (valid external JWT → SyncOIDCUser) requires a live
// JWKS endpoint and is covered by integration/e2e tests.
