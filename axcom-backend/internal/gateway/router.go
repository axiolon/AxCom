// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package gateway configures the HTTP router, endpoints, and middleware.
package gateway

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"ecom-engine/internal/core/admin"
	"ecom-engine/internal/core/auth"
	"ecom-engine/internal/engine"
	"ecom-engine/internal/gateway/middleware"
	"ecom-engine/pkg/metrics"
	"ecom-engine/pkg/token"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	goredis "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// NewRouter builds the HTTP router, registers global middleware, mounts auth
// routes (infrastructure — always active), then delegates all domain routes to
// the active modules. Disabled modules get a helpful catch-all error response.
func NewRouter(eng *engine.Engine) *gin.Engine {
	if os.Getenv("GIN_MODE") == "" {
		env := strings.ToLower(os.Getenv("APP_ENV"))
		switch env {
		case "production", "prod":
			gin.SetMode(gin.ReleaseMode)
		case "staging", "stage":
			gin.SetMode(gin.ReleaseMode)
		case "test", "testing":
			gin.SetMode(gin.TestMode)
		case "development", "dev", "":
			fallthrough
		default:
			if gin.Mode() != gin.TestMode {
				gin.SetMode(gin.DebugMode)
			}
		}
	}

	r := gin.New()

	// --- DB pool metrics (Postgres only; MongoDB driver exposes no pool stats) ---
	// Only registered when metrics are enabled to avoid holding a reference to
	// the pool collector in deployments that do not run Prometheus.
	if eng.Config.Metrics.Enabled && eng.PoolStats != nil {
		prometheus.MustRegister(metrics.NewDBPoolCollector(eng.PoolStats))
	}

	// --- Global middleware ---
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware(eng.Config.ServiceName))
	if eng.Config.Metrics.Enabled {
		r.Use(middleware.PrometheusMiddleware())
	}
	r.Use(middleware.SecurityHeadersMiddleware(""))
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RequestSizeLimitMiddleware(eng.Config.MaxRequestSize))

	// --- Health probes (outside /api — never rate-limited) ---
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "UP",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		dbStatus := "UP"
		overallOK := true
		if err := eng.DBConn.Ping(ctx); err != nil {
			dbStatus = "DOWN: " + err.Error()
			overallOK = false
		}

		cacheStatus := "UP"
		if err := eng.Cache.HealthCheck(ctx); err != nil {
			cacheStatus = "DOWN: " + err.Error()
			overallOK = false
		}

		status := "READY"
		httpCode := http.StatusOK
		if !overallOK {
			status = "NOT READY"
			httpCode = http.StatusServiceUnavailable
		}

		c.JSON(httpCode, gin.H{
			"status":   status,
			"database": dbStatus,
			"cache":    cacheStatus,
			"time":     time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Prometheus /metrics is served on a separate internal port (pkg/metrics.NewInternalServer)
	// and is never exposed on the public API port.

	// Serve locally uploaded product images.
	r.Static("/uploads", "./uploads")

	api := r.Group("/api")

	// Rate limiter scoped to /api — health probes are never throttled.
	jwtManager := token.NewJWTManager(eng.Config.Secret)

	var rlStore middleware.Store
	if eng.Config.RateLimit.Backend == "redis" {
		rc := goredis.NewClient(&goredis.Options{
			Addr:     eng.Config.Cache.Addr,
			Password: eng.Config.Cache.Password,
			DB:       eng.Config.Cache.DB,
		})
		rlStore = middleware.NewFallbackStore(rc)
	} else {
		rlStore = middleware.NewMemoryStore()
	}

	globalCfg := middleware.RateConfig{
		Rate:  eng.Config.RateLimit.GlobalRate,
		Burst: eng.Config.RateLimit.GlobalBurst,
	}
	api.Use(middleware.RateLimitMiddleware(jwtManager, rlStore, globalCfg))

	// --- Auth routes (infrastructure — always registered) ---
	auth.RegisterRoutes(api, auth.NewController(eng.AuthService))

	// --- Secured and admin route groups ---
	secured := api.Group("")
	secured.Use(eng.AuthMiddleware)

	adminGroup := secured.Group("")
	adminGroup.Use(eng.AdminMiddleware)

	// Admin meta-routes (always present).
	admin.RegisterRoutes(adminGroup, admin.NewAdminHandler())

	// --- Module routes (registered in topological / dependency-first order) ---
	for _, mod := range eng.ActiveModules() {
		mod.RegisterRoutes(api, secured, adminGroup)
	}

	// --- Disabled module catch-alls ---
	// Returns a descriptive error instead of a generic 404.
	for _, info := range eng.DisabledModules() {
		for _, path := range info.BasePaths {
			name := info.Name // capture loop variable
			r.Any(path+"/*path", func(c *gin.Context) {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error": fmt.Sprintf(
						"module %q is disabled; enable it in config.yaml under modules.%s.enabled: true",
						name, name,
					),
				})
			})
		}
	}

	return r
}
