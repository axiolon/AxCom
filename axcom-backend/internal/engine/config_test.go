// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid configuration", func(t *testing.T) {
		t.Parallel()
		cfg := defaultConfig()
		// Overriding with fully valid items
		cfg.Secret = "valid-secret-key-that-is-long"
		cfg.DB.Type = "postgres"
		cfg.DB.ConnectionString = "postgres://user:pass@localhost:5432/db"
		cfg.Cache.Type = "redis"
		cfg.Auth.Mode = "local"

		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing secret", func(t *testing.T) {
		t.Parallel()
		cfg := defaultConfig()
		cfg.Secret = ""

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secret (JWT_SECRET) must not be empty")
	})

	t.Run("invalid db type", func(t *testing.T) {
		t.Parallel()
		cfg := defaultConfig()
		cfg.DB.Type = "sqlite"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db.type must be 'mongodb' or 'postgres'")
	})

	t.Run("missing db connection string", func(t *testing.T) {
		t.Parallel()
		cfg := defaultConfig()
		cfg.DB.ConnectionString = ""

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db.connection_string must not be empty")
	})

	t.Run("invalid cache type", func(t *testing.T) {
		t.Parallel()
		cfg := defaultConfig()
		cfg.Cache.Type = "invalid"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache.type must be 'redis' or 'memory'")
	})

	t.Run("invalid auth mode", func(t *testing.T) {
		t.Parallel()
		cfg := defaultConfig()
		cfg.Auth.Mode = "invalid"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "auth.mode must be 'local' or 'oidc'")
	})
}

func TestConfig_DefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "mongodb", cfg.DB.Type)
	assert.Equal(t, "memory", cfg.Cache.Type)
	assert.Equal(t, "local", cfg.Storage.Provider)
	assert.True(t, cfg.Modules.Catalog.Enabled)
	assert.True(t, cfg.Modules.Cart.Enabled)
	assert.True(t, cfg.Modules.Orders.Enabled)

	// New DB defaults assertions
	assert.Equal(t, 25, cfg.DB.MaxPoolSize)
	assert.Equal(t, 5, cfg.DB.MinPoolSize)
	assert.Equal(t, 15*time.Minute, cfg.DB.MaxConnIdleTime)
	assert.Equal(t, 1*time.Hour, cfg.DB.MaxConnLifetime)
	assert.Equal(t, 20, cfg.DB.RetryMaxAttempts)
	assert.Equal(t, 1*time.Second, cfg.DB.RetryInitialDelay)
	assert.Equal(t, 30*time.Second, cfg.DB.RetryMaxDelay)
	assert.Equal(t, "disable", cfg.DB.SSLMode)
	assert.Equal(t, true, *cfg.DB.RetryWrites)
	assert.Equal(t, "majority", cfg.DB.WriteConcern)
}

func TestConfig_OverlayEnv(t *testing.T) {
	t.Setenv("DB_MAX_POOL_SIZE", "99")
	t.Setenv("DB_MIN_POOL_SIZE", "11")
	t.Setenv("DB_QUERY_TIMEOUT", "45s")
	t.Setenv("DB_RETRY_WRITES", "false")

	cfg := defaultConfig()
	overlayEnv(&cfg)

	assert.Equal(t, 99, cfg.DB.MaxPoolSize)
	assert.Equal(t, 11, cfg.DB.MinPoolSize)
	assert.Equal(t, 45*time.Second, cfg.DB.QueryTimeout)
	assert.Equal(t, false, *cfg.DB.RetryWrites)
}

func TestConfig_LoadConfigFromFile(t *testing.T) {
	t.Parallel()

	t.Run("load valid config file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "config-test-*")
		assert.NoError(t, err)
		defer func() { _ = os.RemoveAll(tmpDir) }()

		yamlData := `
port: "9090"
secret: "yaml-loaded-secret-key-that-is-long"
db:
  type: "postgres"
  connection_string: "postgres://host"
cache:
  type: "memory"
  l1_ttl: 2m
`
		tmpFile := filepath.Join(tmpDir, "config.yaml")
		err = os.WriteFile(tmpFile, []byte(yamlData), 0600)
		assert.NoError(t, err)

		cfg, err := LoadConfigFromFile(tmpFile)
		assert.NoError(t, err)
		assert.Equal(t, "9090", cfg.Port)
		assert.Equal(t, "yaml-loaded-secret-key-that-is-long", cfg.Secret)
		assert.Equal(t, "postgres", cfg.DB.Type)
		assert.Equal(t, 2*time.Minute, cfg.Cache.L1TTL)
		// Check that unspecified modules default to true (batteries included)
		assert.True(t, cfg.Modules.Cart.Enabled)
	})

	t.Run("missing file returns error", func(t *testing.T) {
		_, err := LoadConfigFromFile("non-existent-file.yaml")
		assert.Error(t, err)
	})

	t.Run("invalid yaml content returns error", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "config-test-*")
		assert.NoError(t, err)
		defer func() { _ = os.RemoveAll(tmpDir) }()

		invalidYaml := `
port: "9090"
secret: : invalid yaml syntax
`
		tmpFile := filepath.Join(tmpDir, "invalid.yaml")
		err = os.WriteFile(tmpFile, []byte(invalidYaml), 0600)
		assert.NoError(t, err)

		_, err = LoadConfigFromFile(tmpFile)
		assert.Error(t, err)
	})
}
