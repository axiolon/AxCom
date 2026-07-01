// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ecom-engine/internal/gateway/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// buildCORSRouter constructs a fresh middleware instance AFTER the caller has
// set any environment variables, so the closure captures the correct state.
func buildCORSRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.CORSMiddleware())
	r.Any("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func corsRequest(r *gin.Engine, method, origin string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, "/", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	r.ServeHTTP(w, req)
	return w
}

func TestCORSMiddleware_NoOriginHeader_NoCORSHeaders(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")
	r := buildCORSRouter()
	w := corsRequest(r, http.MethodGet, "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Methods"))
}

func TestCORSMiddleware_Wildcard_WhenNoAllowedOriginsEnv(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")
	r := buildCORSRouter()
	w := corsRequest(r, http.MethodGet, "https://example.com")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"))
	assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Headers"))
}

func TestCORSMiddleware_AllowedOrigin_SetsHeadersAndVary(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://shop.example.com,https://admin.example.com")
	r := buildCORSRouter()
	w := corsRequest(r, http.MethodGet, "https://shop.example.com")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://shop.example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", w.Header().Get("Vary"))
	assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"))
	assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Headers"))
}

func TestCORSMiddleware_SecondAllowedOrigin_AlsoAllowed(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://shop.example.com, https://admin.example.com")
	r := buildCORSRouter()
	w := corsRequest(r, http.MethodGet, "https://admin.example.com")

	assert.Equal(t, "https://admin.example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_DisallowedOrigin_NoCORSHeaders(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://shop.example.com")
	r := buildCORSRouter()
	w := corsRequest(r, http.MethodGet, "https://evil.attacker.com")

	// The request still proceeds (CORS enforcement is client-side).
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"), "disallowed origin must not receive Allow-Origin")
}

func TestCORSMiddleware_Preflight_AllowedOrigin_Returns204WithMaxAge(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://shop.example.com")
	r := buildCORSRouter()
	w := corsRequest(r, http.MethodOptions, "https://shop.example.com")

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
	assert.Equal(t, "https://shop.example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_Preflight_DisallowedOrigin_NoAllowOriginHeader(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://shop.example.com")
	r := buildCORSRouter()
	w := corsRequest(r, http.MethodOptions, "https://evil.attacker.com")

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"),
		"preflight for disallowed origin must not carry Allow-Origin")
}

func TestCORSMiddleware_Preflight_Wildcard_Returns204(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")
	r := buildCORSRouter()
	w := corsRequest(r, http.MethodOptions, "https://any.com")

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_AllowedMethods_IncludesCRUD(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")
	r := buildCORSRouter()
	w := corsRequest(r, http.MethodGet, "https://any.com")

	methods := w.Header().Get("Access-Control-Allow-Methods")
	assert.Contains(t, methods, "GET")
	assert.Contains(t, methods, "POST")
	assert.Contains(t, methods, "PUT")
	assert.Contains(t, methods, "DELETE")
}

func TestCORSMiddleware_AllowedHeaders_IncludesAuthAndUserHeaders(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")
	r := buildCORSRouter()
	w := corsRequest(r, http.MethodGet, "https://any.com")

	headers := w.Header().Get("Access-Control-Allow-Headers")
	assert.Contains(t, headers, "Authorization")
	assert.Contains(t, headers, "X-User-ID")
	assert.Contains(t, headers, "X-User-Role")
}
