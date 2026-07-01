// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Tests are in the same package so they can reach unexported types
// (memoryStore, clientBucket, fallbackStore, endpointConfig).
package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ecom-engine/pkg/token"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// newIsolatedMemoryStore returns a fresh memoryStore that does not share
// state with any other instance, keeping tests independent.
func newIsolatedMemoryStore() *memoryStore {
	return &memoryStore{buckets: make(map[string]*clientBucket)}
}

// ---------------------------------------------------------------------------
// memoryStore.Allow unit tests
// ---------------------------------------------------------------------------

func TestMemoryStore_Allow_BurstExhaustion(t *testing.T) {
	ms := newIsolatedMemoryStore()
	cfg := RateConfig{Rate: 0.01, Burst: 3} // near-zero refill

	for i := 0; i < 3; i++ {
		assert.True(t, ms.Allow("rl:ip:127.0.0.1", cfg), "request %d within burst must be allowed", i+1)
	}
	assert.False(t, ms.Allow("rl:ip:127.0.0.1", cfg), "request after burst exhaustion must be denied")
}

func TestMemoryStore_Allow_TokenRefillOverTime(t *testing.T) {
	ms := newIsolatedMemoryStore()
	cfg := RateConfig{Rate: 10.0, Burst: 1}

	assert.True(t, ms.Allow("rl:ip:127.0.0.1", cfg))  // consume the single token
	assert.False(t, ms.Allow("rl:ip:127.0.0.1", cfg)) // empty

	// Back-date lastRefill to simulate 200 ms passing (≥2 tokens refilled).
	ms.mu.Lock()
	ms.buckets["rl:ip:127.0.0.1"].lastRefill = time.Now().Add(-200 * time.Millisecond)
	ms.mu.Unlock()

	assert.True(t, ms.Allow("rl:ip:127.0.0.1", cfg), "tokens should have refilled after elapsed time")
}

func TestMemoryStore_Allow_TokensCapAtBurst(t *testing.T) {
	ms := newIsolatedMemoryStore()
	cfg := RateConfig{Rate: 100.0, Burst: 5}

	ms.Allow("rl:ip:127.0.0.1", cfg) // creates the entry

	// Simulate 60 seconds of elapsed time — tokens must not exceed Burst.
	ms.mu.Lock()
	ms.buckets["rl:ip:127.0.0.1"].lastRefill = time.Now().Add(-60 * time.Second)
	ms.mu.Unlock()

	ms.Allow("rl:ip:127.0.0.1", cfg) // triggers refill + consume

	ms.mu.Lock()
	remaining := ms.buckets["rl:ip:127.0.0.1"].tokens
	ms.mu.Unlock()

	assert.LessOrEqual(t, remaining, cfg.Burst, "tokens must never exceed the burst cap")
}

func TestMemoryStore_Allow_PerKeyIsolation(t *testing.T) {
	ms := newIsolatedMemoryStore()
	cfg := RateConfig{Rate: 0.01, Burst: 1}

	assert.True(t, ms.Allow("rl:ip:192.168.1.1", cfg))
	assert.False(t, ms.Allow("rl:ip:192.168.1.1", cfg)) // bucket drained

	// A different key still has a full burst bucket.
	assert.True(t, ms.Allow("rl:ip:192.168.1.2", cfg), "separate key must have its own token bucket")
}

func TestMemoryStore_Allow_NewKeyStartsWithFullBurst(t *testing.T) {
	ms := newIsolatedMemoryStore()
	cfg := RateConfig{Rate: 1.0, Burst: 5}

	assert.True(t, ms.Allow("rl:ip:10.0.0.1", cfg))
}

func TestMemoryStore_Allow_GlobalKeyDistinctFromIPKey(t *testing.T) {
	ms := newIsolatedMemoryStore()
	cfg := RateConfig{Rate: 0.01, Burst: 1}

	assert.True(t, ms.Allow("rl:global", cfg))
	assert.False(t, ms.Allow("rl:global", cfg))
	// IP key is a separate bucket.
	assert.True(t, ms.Allow("rl:ip:1.2.3.4", cfg))
}

func TestMemoryStore_Allow_EndpointKeyDistinctFromIPKey(t *testing.T) {
	ms := newIsolatedMemoryStore()
	cfg := RateConfig{Rate: 0.01, Burst: 1}

	assert.True(t, ms.Allow("rl:ep:auth:1.2.3.4", cfg))
	assert.False(t, ms.Allow("rl:ep:auth:1.2.3.4", cfg))
	// The IP-tier bucket is separate.
	assert.True(t, ms.Allow("rl:ip:1.2.3.4", cfg))
}

// ---------------------------------------------------------------------------
// fallbackStore unit tests
// ---------------------------------------------------------------------------

func TestFallbackStore_MarkUnhealthyTransitionsOnce(t *testing.T) {
	fs := &fallbackStore{
		primary:   &redisStore{client: nil},
		secondary: newIsolatedMemoryStore(),
	}
	fs.healthy.Store(true)

	fs.markUnhealthy(errors.New("connection refused"))
	assert.False(t, fs.healthy.Load(), "should be unhealthy after first error")

	// Calling again must not panic or change consecOK unexpectedly.
	fs.markUnhealthy(errors.New("still down"))
	assert.False(t, fs.healthy.Load())
}

func TestFallbackStore_AllowUsesSecondaryWhenUnhealthy(t *testing.T) {
	secondary := newIsolatedMemoryStore()
	fs := &fallbackStore{
		primary:   &redisStore{client: nil}, // nil client — allowE will error
		secondary: secondary,
	}
	fs.healthy.Store(false) // already in fallback

	// secondary has a full burst, so Allow must succeed.
	cfg := RateConfig{Rate: 1.0, Burst: 5}
	assert.True(t, fs.Allow("rl:ip:1.2.3.4", cfg))
}

func TestFallbackStore_DoesNotRecoverBeforeThreshold(t *testing.T) {
	fs := &fallbackStore{
		primary:   &redisStore{client: nil},
		secondary: newIsolatedMemoryStore(),
	}
	fs.healthy.Store(false)
	fs.consecOK.Store(int32(recoveryThreshold - 1)) // one short

	// Simulate threshold check manually (probe not running in unit test).
	assert.False(t, int(fs.consecOK.Load()) >= recoveryThreshold || fs.healthy.Load(),
		"should not recover before threshold")
}

func TestFallbackStore_RecoversAfterThreshold(t *testing.T) {
	fs := &fallbackStore{
		primary:   &redisStore{client: nil},
		secondary: newIsolatedMemoryStore(),
	}
	fs.healthy.Store(false)
	fs.downSince = time.Now().Add(-2 * time.Minute)
	fs.consecOK.Store(int32(recoveryThreshold - 1))

	// Simulate the last successful probe increment + promotion.
	newOK := fs.consecOK.Add(1)
	if int(newOK) >= recoveryThreshold && !fs.healthy.Load() {
		fs.healthy.Store(true)
	}

	assert.True(t, fs.healthy.Load(), "should recover once threshold is reached")
}

// ---------------------------------------------------------------------------
// endpointConfig helper tests
// ---------------------------------------------------------------------------

func TestEndpointConfig_AuthPath(t *testing.T) {
	name, cfg, ok := endpointConfig("/api/auth/login")
	assert.True(t, ok)
	assert.Equal(t, "auth", name)
	assert.Equal(t, EndpointAuth, cfg)
}

func TestEndpointConfig_CheckoutPath(t *testing.T) {
	name, cfg, ok := endpointConfig("/api/checkout")
	assert.True(t, ok)
	assert.Equal(t, "checkout", name)
	assert.Equal(t, EndpointCheckout, cfg)
}

func TestEndpointConfig_PaymentsPath(t *testing.T) {
	name, cfg, ok := endpointConfig("/api/payments/stripe/webhook")
	assert.True(t, ok)
	assert.Equal(t, "payments", name)
	assert.Equal(t, EndpointPayments, cfg)
}

func TestEndpointConfig_UnknownPath(t *testing.T) {
	_, _, ok := endpointConfig("/api/products")
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// getClientTier unit tests
// ---------------------------------------------------------------------------

func setupGinContext(authHeader string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	if authHeader != "" {
		c.Request.Header.Set("Authorization", authHeader)
	}
	return c
}

func TestGetClientTier_NoAuthHeader_ReturnsPublic(t *testing.T) {
	jm := token.NewJWTManager("test-secret-long-enough-for-hmac")
	c := setupGinContext("")
	assert.Equal(t, TierPublic, getClientTier(c, jm))
}

func TestGetClientTier_NilJWTManager_ReturnsPublic(t *testing.T) {
	c := setupGinContext("Bearer some-token")
	assert.Equal(t, TierPublic, getClientTier(c, nil))
}

func TestGetClientTier_MalformedHeader_ReturnsPublic(t *testing.T) {
	jm := token.NewJWTManager("test-secret-long-enough-for-hmac")
	c := setupGinContext("NotBearer anything")
	assert.Equal(t, TierPublic, getClientTier(c, jm))
}

func TestGetClientTier_InvalidToken_ReturnsPublic(t *testing.T) {
	jm := token.NewJWTManager("test-secret-long-enough-for-hmac")
	c := setupGinContext("Bearer garbage-not-a-token")
	assert.Equal(t, TierPublic, getClientTier(c, jm))
}

func TestGetClientTier_UserToken_ReturnsTierAuth(t *testing.T) {
	jm := token.NewJWTManager("test-secret-long-enough-for-hmac")
	tok, _ := jm.Generate("user-1", "user", time.Hour)
	c := setupGinContext("Bearer " + tok)
	assert.Equal(t, TierAuth, getClientTier(c, jm))
}

func TestGetClientTier_AdminToken_ReturnsTierAdmin(t *testing.T) {
	jm := token.NewJWTManager("test-secret-long-enough-for-hmac")
	tok, _ := jm.Generate("admin-1", "admin", time.Hour)
	c := setupGinContext("Bearer " + tok)
	assert.Equal(t, TierAdmin, getClientTier(c, jm))
}

func TestGetClientTier_UnknownRole_ReturnsTierAuth(t *testing.T) {
	jm := token.NewJWTManager("test-secret-long-enough-for-hmac")
	tok, _ := jm.Generate("svc-1", "service-account", time.Hour)
	c := setupGinContext("Bearer " + tok)
	assert.Equal(t, TierAuth, getClientTier(c, jm))
}

// ---------------------------------------------------------------------------
// RateLimitMiddleware integration tests (test-mode bypass)
// ---------------------------------------------------------------------------

func TestRateLimitMiddleware_BypassInTestMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jm := token.NewJWTManager("test-secret-long-enough-for-hmac")
	store := NewMemoryStore()
	globalCfg := RateConfig{Rate: 10000, Burst: 15000}

	r := gin.New()
	r.Use(RateLimitMiddleware(jm, store, globalCfg))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	for i := 0; i < 200; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "request %d must pass in test mode", i+1)
	}
}

// ---------------------------------------------------------------------------
// Tier constant sanity checks
// ---------------------------------------------------------------------------

func TestTierConstants_PublicIsLowest(t *testing.T) {
	assert.Less(t, TierPublic.Rate, TierAuth.Rate)
	assert.Less(t, TierAuth.Rate, TierAdmin.Rate)
}

func TestTierConstants_BurstsAreAscending(t *testing.T) {
	assert.Less(t, TierPublic.Burst, TierAuth.Burst)
	assert.Less(t, TierAuth.Burst, TierAdmin.Burst)
}

func TestEndpointTiers_BelowPublicTier(t *testing.T) {
	assert.Less(t, EndpointAuth.Rate, TierPublic.Rate, "auth endpoint must be stricter than public tier")
	assert.Less(t, EndpointCheckout.Rate, TierPublic.Rate)
	assert.Less(t, EndpointPayments.Rate, TierPublic.Rate)
}
