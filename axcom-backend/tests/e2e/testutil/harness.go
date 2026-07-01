// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

// Package testutil provides shared infrastructure for e2e tests.
// It spins up a real MongoDB container via testcontainers-go and boots
// the full engine+router so tests hit live HTTP endpoints with a real DB.
package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	tcmongo "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"ecom-engine/internal/engine"
	"ecom-engine/internal/events"
	"ecom-engine/internal/gateway"
	"ecom-engine/internal/modules/registry"
	"ecom-engine/pkg/idgen"
)

const (
	// DBName is the isolated test database — dropped or truncated between runs.
	DBName = "ecom_e2e_test"

	// TestSecret is the JWT signing key used across all e2e tests.
	TestSecret = "e2e-test-super-secret-signing-key-that-is-long"
)

// Harness holds the running httptest.Server and helper methods for e2e tests.
// Create one via New; tear it down via Shutdown.
type Harness struct {
	// Server is the live HTTP endpoint backed by a real MongoDB container.
	Server      *httptest.Server
	Engine      *engine.Engine
	MongoClient *mongodriver.Client
	container   *tcmongo.MongoDBContainer
}

// New starts a MongoDB testcontainer, boots the engine with the requested modules
// enabled (auth is always on regardless), and returns a Harness ready for tests.
// The caller is responsible for calling Shutdown when done.
func New(ctx context.Context, modules ...string) (*Harness, error) {
	gin.SetMode(gin.TestMode)

	// --- MongoDB container ---
	// WithReplicaSet enables multi-document transactions (required by auth session logic).
	ctr, err := tcmongo.Run(ctx, "mongo:7", tcmongo.WithReplicaSet("rs0"))
	if err != nil {
		return nil, err
	}

	uri, err := ctr.ConnectionString(ctx)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, err
	}

	// Append directConnection=true to prevent driver from routing through container's internal IP
	if !strings.Contains(uri, "directConnection=") {
		if strings.Contains(uri, "?") {
			uri += "&directConnection=true"
		} else {
			uri += "?directConnection=true"
		}
	}

	// --- Engine ---
	cfg := buildConfig(uri, modules...)
	active, disabled := registry.Collect(cfg)

	eng, err := engine.NewEngine(cfg, active, disabled)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, err
	}

	// --- HTTP server ---
	srv := httptest.NewServer(gateway.NewRouter(eng))

	// --- Mongo client for direct DB operations (seeding / truncation) ---
	mongoClient, err := mongodriver.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		srv.Close()
		_ = eng.Shutdown(ctx)
		_ = ctr.Terminate(ctx)
		return nil, err
	}

	return &Harness{
		Server:      srv,
		Engine:      eng,
		MongoClient: mongoClient,
		container:   ctr,
	}, nil
}

// Shutdown tears down the HTTP server, engine, Mongo client, and container.
// Safe to call more than once.
func (h *Harness) Shutdown(ctx context.Context) {
	if h.Server != nil {
		h.Server.Close()
	}
	if h.Engine != nil {
		_ = h.Engine.Shutdown(ctx)
	}
	if h.MongoClient != nil {
		_ = h.MongoClient.Disconnect(ctx)
	}
	if h.container != nil {
		_ = h.container.Terminate(ctx)
	}
}

// SeedUser inserts a user directly into MongoDB, bypassing the API role whitelist.
// This is the only way to create admin users in tests. Returns the generated user ID.
func (h *Harness) SeedUser(t *testing.T, email, password, role string) string {
	t.Helper()

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)

	id, err := idgen.Generate("usr_")
	require.NoError(t, err)

	now := time.Now()
	_, err = h.MongoClient.Database(DBName).Collection("users").InsertOne(context.Background(), bson.M{
		"_id":                   id,
		"email":                 email,
		"password":              string(hashed),
		"role":                  role,
		"created_at":            now,
		"updated_at":            now,
		"failed_login_attempts": 0,
	})
	require.NoError(t, err, "SeedUser: failed to insert %s (%s)", email, role)
	return id
}

// LoginAs issues POST /api/auth/login and returns the (accessToken, refreshToken) pair.
func (h *Harness) LoginAs(t *testing.T, email, password string) (accessToken, refreshToken string) {
	t.Helper()

	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	resp, err := http.Post(h.Server.URL+"/api/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "LoginAs: unexpected status for %s", email)

	var result struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.NotEmpty(t, result.Data.AccessToken, "LoginAs: empty access token")
	return result.Data.AccessToken, result.Data.RefreshToken
}

// Do sends an HTTP request to the test server.
// If token is non-empty it is sent as a Bearer Authorization header.
// body may be nil for GET/DELETE requests.
func (h *Harness) Do(t *testing.T, method, path string, body interface{}, token string) *http.Response {
	t.Helper()

	var buf *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewReader(b)
	} else {
		buf = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, h.Server.URL+path, buf)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// Truncate drops the named MongoDB collections in the test database.
// Call at the start of each test to prevent state leaking between tests.
func (h *Harness) Truncate(t *testing.T, collections ...string) {
	t.Helper()
	ctx := context.Background()
	db := h.MongoClient.Database(DBName)
	for _, col := range collections {
		if err := db.Collection(col).Drop(ctx); err != nil {
			t.Logf("Truncate: drop %q: %v (ignored)", col, err)
		}
	}
}

// Decode parses JSON from resp.Body into dst and closes the body.
func Decode(t *testing.T, resp *http.Response, dst interface{}) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(dst))
}

// ---------------------------------------------------------------------------
// internal config builder
// ---------------------------------------------------------------------------

func buildConfig(mongoURI string, modules ...string) engine.Config {
	enabled := make(map[string]bool, len(modules))
	for _, m := range modules {
		enabled[m] = true
	}

	return engine.Config{
		Port:           "0",
		Secret:         TestSecret,
		ServiceName:    "ecom-e2e",
		MaxRequestSize: 5 << 20,
		Auth:           engine.AuthConfig{Mode: "local"},
		DB: engine.DBConfig{
			Type:             "mongodb",
			ConnectionString: mongoURI,
			Database:         DBName,
		},
		Cache: engine.CacheConfig{
			Type:       "memory",
			L1TTL:      1 * time.Minute,
			L1MaxItems: 1000,
		},
		Storage: engine.StorageConfig{Provider: "local"},
		Events: events.BusConfig{
			Provider: "local",
			Retry: events.RetryConfig{
				MaxRetries:     1,
				InitialBackoff: 10 * time.Millisecond,
				MaxBackoff:     100 * time.Millisecond,
			},
			Local: events.LocalBusConfig{DLQBufferSize: 10},
		},
		Outbox: engine.OutboxConfig{Enabled: false},
		Modules: engine.ModulesConfig{
			Catalog: engine.CatalogModuleConfig{
				Enabled: enabled["catalog"],
				Features: engine.CatalogFeaturesConfig{
					Variants:  true, // required — products need at least one variant
					Images:    false,
					Discounts: false,
					Bulk:      false,
					Reviews:   false,
				},
			},
			Inventory: engine.InventoryModuleConfig{
				Enabled: enabled["inventory"],
				Features: engine.InventoryFeaturesConfig{
					Bulk:        true,
					History:     true,
					Reservation: true,
					Reports:     true,
					Transfer:    true,
					Adjustment:  true,
					Sync:        true,
				},
			},
			Cart:   engine.ModuleToggle{Enabled: enabled["cart"]},
			Orders: engine.ModuleToggle{Enabled: enabled["orders"]},
			Payments: engine.PaymentsModuleConfig{
				Enabled:  enabled["payments"],
				Provider: "stripe",
			},
			Shipping: engine.ShippingModuleConfig{
				Enabled: enabled["shipping"],
				Providers: []engine.ShippingProviderConfig{
					{Type: "flatrate", Rate: 5.99},
				},
			},
			Notifications: engine.ModuleToggle{Enabled: enabled["notifications"]},
			Dashboard: engine.DashboardConfig{
				Enabled:  enabled["dashboard"],
				Tier:     "small",
				CacheTTL: 1 * time.Minute,
			},
		},
	}
}
