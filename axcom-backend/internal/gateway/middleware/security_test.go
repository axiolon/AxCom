// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ecom-engine/internal/gateway/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func securityTestRouter(csp string) *gin.Engine {
	r := gin.New()
	r.Use(middleware.SecurityHeadersMiddleware(csp))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func securityGET(r *gin.Engine) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)
	return w
}

func TestSecurityHeadersMiddleware_HSTS(t *testing.T) {
	w := securityGET(securityTestRouter(""))
	assert.Equal(t, "max-age=31536000; includeSubDomains",
		w.Header().Get("Strict-Transport-Security"))
}

func TestSecurityHeadersMiddleware_ContentTypeOptions(t *testing.T) {
	w := securityGET(securityTestRouter(""))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
}

func TestSecurityHeadersMiddleware_FrameOptions(t *testing.T) {
	w := securityGET(securityTestRouter(""))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
}

func TestSecurityHeadersMiddleware_XSSProtection(t *testing.T) {
	w := securityGET(securityTestRouter(""))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
}

func TestSecurityHeadersMiddleware_DefaultCSP(t *testing.T) {
	t.Setenv("CSP_DIRECTIVES", "")
	w := securityGET(securityTestRouter(""))
	assert.Equal(t, "default-src 'self'", w.Header().Get("Content-Security-Policy"))
}

func TestSecurityHeadersMiddleware_ReferrerPolicy(t *testing.T) {
	w := securityGET(securityTestRouter(""))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
}

func TestSecurityHeadersMiddleware_PermissionsPolicy_DisablesBrowserAPIs(t *testing.T) {
	w := securityGET(securityTestRouter(""))
	pp := w.Header().Get("Permissions-Policy")
	assert.Contains(t, pp, "camera=()")
	assert.Contains(t, pp, "microphone=()")
	assert.Contains(t, pp, "geolocation=()")
	assert.Contains(t, pp, "payment=()")
}

func TestSecurityHeadersMiddleware_AllSevenHeadersPresent(t *testing.T) {
	t.Setenv("CSP_DIRECTIVES", "")
	w := securityGET(securityTestRouter(""))
	required := []string{
		"Strict-Transport-Security",
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-XSS-Protection",
		"Content-Security-Policy",
		"Referrer-Policy",
		"Permissions-Policy",
	}
	for _, h := range required {
		assert.NotEmpty(t, w.Header().Get(h), "expected security header %q to be set", h)
	}
}

func TestSecurityHeadersMiddleware_CustomCSP_OverridesDefault(t *testing.T) {
	customCSP := "default-src 'self'; img-src *; script-src 'nonce-abc'"
	w := securityGET(securityTestRouter(customCSP))
	assert.Equal(t, customCSP, w.Header().Get("Content-Security-Policy"))
}

func TestSecurityHeadersMiddleware_EnvCSP_UsedWhenParamEmpty(t *testing.T) {
	envCSP := "default-src 'self'; script-src 'nonce-xyz'"
	t.Setenv("CSP_DIRECTIVES", envCSP)
	// Middleware must be constructed AFTER env var is set.
	w := securityGET(securityTestRouter(""))
	assert.Equal(t, envCSP, w.Header().Get("Content-Security-Policy"))
}

func TestSecurityHeadersMiddleware_ParamCSP_TakesPrecedenceOverEnv(t *testing.T) {
	t.Setenv("CSP_DIRECTIVES", "default-src 'self'")
	paramCSP := "default-src 'none'"
	w := securityGET(securityTestRouter(paramCSP))
	assert.Equal(t, paramCSP, w.Header().Get("Content-Security-Policy"),
		"explicit csp param must take precedence over env var")
}

func TestRequestSizeLimitMiddleware_SmallBody_Passes(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestSizeLimitMiddleware(1024))
	r.POST("/upload", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/upload", strings.NewReader("small body"))
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequestSizeLimitMiddleware_NilBody_Passes(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestSizeLimitMiddleware(10))
	r.GET("/probe", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/probe", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
