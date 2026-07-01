// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package middleware provides HTTP middleware handlers for request processing, authentication, and security.
package middleware

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/metrics"
	"ecom-engine/pkg/response"
	"ecom-engine/pkg/token"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

// ---------------------------------------------------------------------------
// Rate configs
// ---------------------------------------------------------------------------

// RateConfig defines the tokens refilled per second and maximum burst capacity.
type RateConfig struct {
	Rate  float64 // Tokens per second
	Burst float64 // Maximum burst size
}

var (
	// Per-IP tier limits — role-based, applied to every /api request.

	// TierPublic: 30 req/min (0.5 tokens/sec), burst of 10.
	TierPublic = RateConfig{Rate: 0.5, Burst: 10}
	// TierAuth: 120 req/min (2.0 tokens/sec), burst of 20.
	TierAuth = RateConfig{Rate: 2.0, Burst: 20}
	// TierAdmin: 300 req/min (5.0 tokens/sec), burst of 50.
	TierAdmin = RateConfig{Rate: 5.0, Burst: 50}

	// Endpoint-specific limits — tighter caps on sensitive routes, applied per-IP
	// in addition to the tier limit above.

	// EndpointAuth: 5 req/min, burst 3 — brute-force protection for login/register.
	EndpointAuth = RateConfig{Rate: 0.083, Burst: 3}
	// EndpointCheckout: 10 req/min, burst 5 — bot/abuse protection for checkout.
	EndpointCheckout = RateConfig{Rate: 0.167, Burst: 5}
	// EndpointPayments: 10 req/min, burst 5 — duplicate-charge prevention for payments.
	EndpointPayments = RateConfig{Rate: 0.167, Burst: 5}
)

// ---------------------------------------------------------------------------
// Store interface
// ---------------------------------------------------------------------------

// Store is the rate-limit backend interface.
// Allow returns true if the request identified by key is within the configured limit.
// Implementations must be safe for concurrent use.
type Store interface {
	Allow(key string, cfg RateConfig) bool
}

// ---------------------------------------------------------------------------
// memoryStore — in-process token bucket
// ---------------------------------------------------------------------------

type clientBucket struct {
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

type memoryStore struct {
	mu      sync.Mutex
	buckets map[string]*clientBucket
	once    sync.Once
}

func newMemoryStore() *memoryStore {
	ms := &memoryStore{buckets: make(map[string]*clientBucket)}
	ms.initJanitor()
	return ms
}

// NewMemoryStore returns a standalone in-process token-bucket Store.
func NewMemoryStore() Store {
	return newMemoryStore()
}

func (ms *memoryStore) initJanitor() {
	ms.once.Do(func() {
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				ms.mu.Lock()
				cutoff := time.Now().Add(-5 * time.Minute)
				for key, b := range ms.buckets {
					if b.lastSeen.Before(cutoff) {
						delete(ms.buckets, key)
					}
				}
				ms.mu.Unlock()
			}
		}()
	})
}

// Allow implements Store using a token-bucket algorithm keyed by the caller-provided key.
func (ms *memoryStore) Allow(key string, cfg RateConfig) bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	b, exists := ms.buckets[key]
	now := time.Now()

	if !exists {
		b = &clientBucket{
			tokens:     cfg.Burst,
			lastRefill: now,
			lastSeen:   now,
		}
		ms.buckets[key] = b
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	b.lastRefill = now
	b.lastSeen = now

	b.tokens += elapsed * cfg.Rate
	if b.tokens > cfg.Burst {
		b.tokens = cfg.Burst
	}

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// redisStore — distributed token bucket via Lua script
//
// Redis key scheme:
//   Caller-provided key (e.g. "rl:global", "rl:ip:1.2.3.4", "rl:ep:auth:1.2.3.4")
//
// Each key is a Redis hash with two fields:
//   tok  — current token count (float, stored as string)
//   ref  — Unix timestamp of last refill in microseconds (int64, stored as string)
//
// TTL: ceil(burst/rate) + 60 seconds — long enough that active clients never
//      lose bucket state between requests; automatic cleanup for idle ones.
// ---------------------------------------------------------------------------

// tokenBucketLua atomically reads, refills, and writes a token-bucket entry.
//
// KEYS[1]  = bucket key
// ARGV[1]  = rate   (tokens/sec, float string)
// ARGV[2]  = burst  (max tokens, float string)
// ARGV[3]  = now_us (current Unix time in microseconds)
// ARGV[4]  = ttl    (key TTL in seconds)
//
// Returns 1 if the request is allowed, 0 if denied.
var tokenBucketLua = goredis.NewScript(`
local key    = KEYS[1]
local rate   = tonumber(ARGV[1])
local burst  = tonumber(ARGV[2])
local now_us = tonumber(ARGV[3])
local ttl    = tonumber(ARGV[4])

local data   = redis.call('HMGET', key, 'tok', 'ref')
local tokens = tonumber(data[1])
local ref    = tonumber(data[2])

if tokens == nil then
    tokens = burst
    ref    = now_us
end

local elapsed = (now_us - ref) / 1000000
tokens = math.min(burst, tokens + elapsed * rate)
ref    = now_us

local allowed = 0
if tokens >= 1.0 then
    tokens  = tokens - 1.0
    allowed = 1
end

redis.call('HMSET', key, 'tok', tostring(tokens), 'ref', tostring(ref))
redis.call('EXPIRE', key, ttl)
return allowed
`)

type redisStore struct {
	client *goredis.Client
}

// Allow implements Store. On Redis error it fails open (returns true) so a
// Redis hiccup never rejects legitimate traffic. The fallbackStore layer calls
// allowE directly to detect and handle errors.
func (rs *redisStore) Allow(key string, cfg RateConfig) bool {
	ok, _ := rs.allowE(key, cfg)
	return ok
}

func (rs *redisStore) allowE(key string, cfg RateConfig) (bool, error) {
	ttl := int64(math.Ceil(cfg.Burst/cfg.Rate)) + 60
	nowUs := time.Now().UnixMicro()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := tokenBucketLua.Run(
		ctx, rs.client,
		[]string{key},
		fmt.Sprintf("%f", cfg.Rate),
		fmt.Sprintf("%f", cfg.Burst),
		fmt.Sprintf("%d", nowUs),
		fmt.Sprintf("%d", ttl),
	).Int64()
	if err != nil {
		return true, err // fail open
	}
	return result == 1, nil
}

// ---------------------------------------------------------------------------
// fallbackStore — Redis primary with automatic in-memory fallback
//
// On any Redis error the store:
//   1. Switches immediately to memoryStore (per-request latency unaffected).
//   2. Logs a WARN with the error and records the downtime start time.
//   3. Starts counting consecutive successful PING probes (every 30 s).
//   4. Promotes Redis back only after recoveryThreshold (3) consecutive successes,
//      i.e. ~90 s of stable Redis, to prevent flapping on an intermittent connection.
//   5. Logs a WARN on recovery with the total downtime duration.
//
// Per-request Redis errors while already in fallback mode are counted in metrics
// but not logged individually — the probe handles status logging to avoid spam.
// ---------------------------------------------------------------------------

const (
	recoveryThreshold = 3
	probeInterval     = 30 * time.Second
)

type fallbackStore struct {
	primary   *redisStore
	secondary *memoryStore

	healthy  atomic.Bool
	consecOK atomic.Int32

	mu        sync.Mutex
	downSince time.Time
}

// NewFallbackStore returns a Store backed by Redis with automatic in-memory
// fallback. It is only constructed when backend = "redis".
func NewFallbackStore(client *goredis.Client) Store {
	fs := &fallbackStore{
		primary:   &redisStore{client: client},
		secondary: newMemoryStore(),
	}
	fs.healthy.Store(true)
	metrics.RateLimitBackendActive.WithLabelValues("redis").Set(1)
	metrics.RateLimitBackendActive.WithLabelValues("memory").Set(0)
	go fs.runProbe()
	return fs
}

// Allow tries Redis first; on error it marks Redis unhealthy and serves the
// request from the in-memory store so no traffic is dropped during an outage.
func (fs *fallbackStore) Allow(key string, cfg RateConfig) bool {
	if !fs.healthy.Load() {
		return fs.secondary.Allow(key, cfg)
	}

	allowed, err := fs.primary.allowE(key, cfg)
	if err != nil {
		fs.markUnhealthy(err)
		metrics.RateLimitRedisErrorsTotal.Inc()
		return fs.secondary.Allow(key, cfg)
	}
	return allowed
}

// markUnhealthy transitions to fallback mode exactly once per outage episode.
func (fs *fallbackStore) markUnhealthy(err error) {
	if fs.healthy.CompareAndSwap(true, false) {
		fs.mu.Lock()
		fs.downSince = time.Now()
		fs.mu.Unlock()
		fs.consecOK.Store(0)
		logger.Warn("ratelimit: Redis unavailable, switched to in-memory fallback — error: %v", err)
		metrics.RateLimitFallbacksTotal.Inc()
		metrics.RateLimitBackendActive.WithLabelValues("redis").Set(0)
		metrics.RateLimitBackendActive.WithLabelValues("memory").Set(1)
	}
}

// runProbe pings Redis every 30 s and promotes it back after recoveryThreshold
// consecutive successes. It runs for the lifetime of the process.
func (fs *fallbackStore) runProbe() {
	ticker := time.NewTicker(probeInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := fs.primary.client.Ping(ctx).Err()
		cancel()

		if err != nil {
			fs.consecOK.Store(0)
			if !fs.healthy.Load() {
				logger.Debug("ratelimit: Redis probe failed (still in fallback) — error: %v", err)
			}
			metrics.RateLimitRedisErrorsTotal.Inc()
			continue
		}

		newOK := fs.consecOK.Add(1)
		if !fs.healthy.Load() {
			logger.Info("ratelimit: Redis probe succeeded (consecutive_ok=%d/%d)", newOK, recoveryThreshold)
		}

		if int(newOK) >= recoveryThreshold && !fs.healthy.Load() {
			fs.mu.Lock()
			downtime := time.Since(fs.downSince)
			fs.mu.Unlock()

			fs.healthy.Store(true)
			logger.Warn("ratelimit: Redis recovered after %s, resuming distributed rate limiting", downtime.Round(time.Second))
			metrics.RateLimitRecoveriesTotal.Inc()
			metrics.RateLimitBackendActive.WithLabelValues("redis").Set(1)
			metrics.RateLimitBackendActive.WithLabelValues("memory").Set(0)
		}
	}
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

// RateLimitMiddleware applies three token-bucket checks on every /api request:
//
//  1. Global bucket ("rl:global") — shared across all IPs; caps total API throughput.
//  2. Per-IP tier bucket ("rl:ip:{addr}") — role-based limits (public/auth/admin).
//  3. Endpoint bucket ("rl:ep:{name}:{addr}") — tighter per-IP limits on sensitive
//     paths (/api/auth/*, /api/checkout/*, /api/payments/*).
//
// Rate limiting is skipped entirely in gin.TestMode or when SKIP_RATE_LIMIT=true.
func RateLimitMiddleware(jwtManager *token.JWTManager, store Store, globalCfg RateConfig) gin.HandlerFunc {
	if gin.Mode() == gin.TestMode || os.Getenv("SKIP_RATE_LIMIT") == "true" {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		// 1. Global bucket.
		if !store.Allow("rl:global", globalCfg) {
			metrics.RateLimitRequestsTotal.WithLabelValues("global", "denied").Inc()
			response.GinError(c, http.StatusTooManyRequests, "rate limit exceeded: server capacity reached")
			return
		}
		metrics.RateLimitRequestsTotal.WithLabelValues("global", "allowed").Inc()

		// 2. Per-IP tier bucket.
		ip := c.ClientIP()
		tier := getClientTier(c, jwtManager)

		if !store.Allow("rl:ip:"+ip, tier) {
			metrics.RateLimitRequestsTotal.WithLabelValues(tierLabel(tier), "denied").Inc()
			response.GinError(c, http.StatusTooManyRequests, "rate limit exceeded: too many requests")
			return
		}
		metrics.RateLimitRequestsTotal.WithLabelValues(tierLabel(tier), "allowed").Inc()

		// 3. Endpoint-specific bucket (sensitive paths only).
		if epName, epCfg, ok := endpointConfig(c.Request.URL.Path); ok {
			if !store.Allow("rl:ep:"+epName+":"+ip, epCfg) {
				metrics.RateLimitRequestsTotal.WithLabelValues("ep:"+epName, "denied").Inc()
				response.GinError(c, http.StatusTooManyRequests, "rate limit exceeded: too many requests to "+epName)
				return
			}
			metrics.RateLimitRequestsTotal.WithLabelValues("ep:"+epName, "allowed").Inc()
		}

		c.Next()
	}
}

// EndpointRateLimit returns a standalone per-IP token-bucket middleware for a
// named endpoint group. Use this when you need fine-grained control outside of
// the paths already handled by RateLimitMiddleware.
//
// Key: rl:ep:{endpoint}:{clientIP}
func EndpointRateLimit(store Store, cfg RateConfig, endpoint string) gin.HandlerFunc {
	if gin.Mode() == gin.TestMode || os.Getenv("SKIP_RATE_LIMIT") == "true" {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := "rl:ep:" + endpoint + ":" + ip

		if !store.Allow(key, cfg) {
			metrics.RateLimitRequestsTotal.WithLabelValues("ep:"+endpoint, "denied").Inc()
			response.GinError(c, http.StatusTooManyRequests, "rate limit exceeded: too many requests to "+endpoint)
			return
		}
		metrics.RateLimitRequestsTotal.WithLabelValues("ep:"+endpoint, "allowed").Inc()

		c.Next()
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// endpointConfig returns the rate config and label for a sensitive API path,
// or (_, _, false) for all other paths.
func endpointConfig(path string) (string, RateConfig, bool) {
	switch {
	case strings.HasPrefix(path, "/api/auth"):
		return "auth", EndpointAuth, true
	case strings.HasPrefix(path, "/api/checkout"):
		return "checkout", EndpointCheckout, true
	case strings.HasPrefix(path, "/api/payments"):
		return "payments", EndpointPayments, true
	default:
		return "", RateConfig{}, false
	}
}

func getClientTier(c *gin.Context, jwtManager *token.JWTManager) RateConfig {
	if jwtManager == nil {
		return TierPublic
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return TierPublic
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return TierPublic
	}

	claims, err := jwtManager.Validate(parts[1])
	if err != nil {
		return TierPublic
	}

	switch claims.Role {
	case "admin":
		return TierAdmin
	case "user":
		return TierAuth
	default:
		return TierAuth
	}
}

func tierLabel(cfg RateConfig) string {
	switch cfg {
	case TierAdmin:
		return "tier:admin"
	case TierAuth:
		return "tier:auth"
	default:
		return "tier:public"
	}
}
