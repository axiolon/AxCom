// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ecom-engine/internal/core/admin"
	"ecom-engine/internal/engine"
	"ecom-engine/internal/gateway/middleware"
	"ecom-engine/internal/infra/cache/memory"
	"ecom-engine/pkg/db"
	"ecom-engine/pkg/token"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const routerTestSecret = "test-secret-key-that-is-very-long-and-secure"

// --- Failing infrastructure stubs ---

type failingDB struct{ db.MemoryConnection }

func (f *failingDB) Ping(_ context.Context) error { return errors.New("connection refused") }

type failingCache struct{ *memory.MemoryAdapter }

func (f *failingCache) HealthCheck(_ context.Context) error {
	return errors.New("cache unreachable")
}

// --- Engine factory ---

func minimalEngine() *engine.Engine {
	jm := token.NewJWTManager(routerTestSecret)
	return &engine.Engine{
		Config: engine.Config{
			Secret:         routerTestSecret,
			ServiceName:    "test-service",
			MaxRequestSize: 1024 * 1024,
		},
		DBConn:          &db.MemoryConnection{},
		Cache:           memory.NewMemoryAdapter(memory.WithMaxItems(100)),
		AuthMiddleware:  middleware.AuthMiddleware(jm),
		AdminMiddleware: admin.AdminOnlyMiddleware(),
	}
}

func probeRequest(router *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	router.ServeHTTP(w, req)
	return w
}

// --- Route registration ---

func TestNewRouter_RegistersProbeRoutes(t *testing.T) {
	router := NewRouter(minimalEngine())
	require.NotNil(t, router)

	routeSet := make(map[string]bool)
	for _, route := range router.Routes() {
		routeSet[route.Method+" "+route.Path] = true
	}

	for _, expected := range []string{
		"GET /healthz",
		"GET /readyz",
	} {
		assert.True(t, routeSet[expected], "expected probe route %q to be registered", expected)
	}
}

func TestNewRouter_RegistersAuthRoutes(t *testing.T) {
	router := NewRouter(minimalEngine())

	routeSet := make(map[string]bool)
	for _, route := range router.Routes() {
		routeSet[route.Method+" "+route.Path] = true
	}

	for _, expected := range []string{
		"POST /api/auth/register",
		"POST /api/auth/login",
		"POST /api/auth/logout",
		"POST /api/auth/refresh",
		"POST /api/auth/password-reset",
		"POST /api/auth/password-reset/confirm",
	} {
		assert.True(t, routeSet[expected], "auth route %q must be registered", expected)
	}
}

// --- /healthz ---

func TestHealthz_AlwaysReturns200(t *testing.T) {
	w := probeRequest(NewRouter(minimalEngine()), "/healthz")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"UP"`)
	assert.Contains(t, w.Body.String(), `"time"`)
}

// --- /readyz ---

func TestReadyz_AllHealthy_Returns200(t *testing.T) {
	w := probeRequest(NewRouter(minimalEngine()), "/readyz")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"READY"`)
	assert.Contains(t, w.Body.String(), `"database":"UP"`)
	assert.Contains(t, w.Body.String(), `"cache":"UP"`)
}

func TestReadyz_FailingDB_Returns503(t *testing.T) {
	eng := minimalEngine()
	eng.DBConn = &failingDB{}
	w := probeRequest(NewRouter(eng), "/readyz")

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"status":"NOT READY"`)
	assert.Contains(t, body, `"database":"DOWN:`)
	assert.Contains(t, body, `"cache":"UP"`)
}

func TestReadyz_FailingCache_Returns503(t *testing.T) {
	eng := minimalEngine()
	eng.Cache = &failingCache{memory.NewMemoryAdapter(memory.WithMaxItems(1))}
	w := probeRequest(NewRouter(eng), "/readyz")

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"status":"NOT READY"`)
	assert.Contains(t, body, `"database":"UP"`)
	assert.Contains(t, body, `"cache":"DOWN:`)
}

func TestReadyz_BothDown_Returns503WithBothStatuses(t *testing.T) {
	eng := minimalEngine()
	eng.DBConn = &failingDB{}
	eng.Cache = &failingCache{memory.NewMemoryAdapter(memory.WithMaxItems(1))}
	w := probeRequest(NewRouter(eng), "/readyz")

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"status":"NOT READY"`)
	assert.Contains(t, body, `"database":"DOWN:`)
	assert.Contains(t, body, `"cache":"DOWN:`)
}

// --- Security boundary: auth middleware fires on secured group ---

func TestSecuredGroup_ValidJWT_PassesThrough(t *testing.T) {
	jm := token.NewJWTManager(routerTestSecret)
	tok, err := jm.Generate("user-1", "user", time.Hour)
	require.NoError(t, err)

	// Wire a canary endpoint directly behind the auth middleware.
	r := gin.New()
	secured := r.Group("")
	secured.Use(middleware.AuthMiddleware(jm))
	secured.GET("/canary", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/canary", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSecuredGroup_NoJWT_Returns401(t *testing.T) {
	jm := token.NewJWTManager(routerTestSecret)

	r := gin.New()
	secured := r.Group("")
	secured.Use(middleware.AuthMiddleware(jm))
	secured.GET("/canary", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/canary", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Probes are outside /api and therefore exempt from rate limiting ---

func TestProbes_NotUnderAPIGroup(t *testing.T) {
	router := NewRouter(minimalEngine())
	for _, route := range router.Routes() {
		path := route.Path
		if path == "/healthz" || path == "/readyz" {
			require.False(t,
				len(path) >= 4 && path[:4] == "/api",
				"probe %q must not be registered under /api", path,
			)
		}
	}
}

// --- Auth routes are public (no JWT required) ---

func TestAuthRoutes_ArePublic_NoJWTRequired(t *testing.T) {
	router := NewRouter(minimalEngine())

	// POST /api/auth/login is public — without a JWT it must not 401.
	// It will return a non-401 status (likely 400/422 since the body is empty,
	// or 500 if the auth service is nil), but never 401.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", nil)
	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusUnauthorized, w.Code,
		"public auth route must not require a JWT")
}
