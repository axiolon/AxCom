// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package engine provides core business logic abstractions, system initialization, and configuration parsing.
package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ecom-engine/internal/events"

	"gopkg.in/yaml.v3"
)

// NewConfigError returns a configuration validation error.
func NewConfigError(msg string) error {
	return errors.New(msg)
}

// DBConfig holds configuration details for establishing a connection to the database layer.
type DBConfig struct {
	Type             string `yaml:"type"`              // "mongodb" or "postgres"
	ConnectionString string `yaml:"connection_string"` // full DSN
	Database         string `yaml:"database"`          // database / schema name

	// --- Pool Sizing ---
	MaxPoolSize int `yaml:"max_pool_size"` // max open connections (default: 25)
	MinPoolSize int `yaml:"min_pool_size"` // min idle connections kept warm (default: 5)

	// --- Connection Lifecycle ---
	MaxConnIdleTime       time.Duration `yaml:"max_conn_idle_time"`       // close idle conns after (default: 15m)
	MaxConnLifetime       time.Duration `yaml:"max_conn_lifetime"`        // max age of any conn (default: 1h)
	MaxConnLifetimeJitter time.Duration `yaml:"max_conn_lifetime_jitter"` // randomize expiry ±jitter (default: 2m)
	ConnectTimeout        time.Duration `yaml:"connect_timeout"`          // dial timeout (default: 10s)
	PoolAcquireTimeout    time.Duration `yaml:"pool_acquire_timeout"`     // wait for pool slot (default: 5s)
	HealthCheckInterval   time.Duration `yaml:"health_check_interval"`    // background liveness probe (default: 30s)

	// --- Query / Transaction Safety ---
	QueryTimeout       time.Duration `yaml:"query_timeout"`       // per-query context deadline (default: 30s)
	TransactionTimeout time.Duration `yaml:"transaction_timeout"` // per-tx context deadline (default: 60s)

	// --- Startup Retry ---
	RetryMaxAttempts  int           `yaml:"retry_max_attempts"`  // connection retries at startup (default: 20)
	RetryInitialDelay time.Duration `yaml:"retry_initial_delay"` // first retry delay (default: 1s)
	RetryMaxDelay     time.Duration `yaml:"retry_max_delay"`     // backoff cap (default: 30s)

	// --- PostgreSQL Specific ---
	SSLMode          string        `yaml:"ssl_mode"`          // "disable", "require", "verify-full", etc.
	ApplicationName  string        `yaml:"application_name"`  // shown in pg_stat_activity (default: service_name)
	StatementTimeout time.Duration `yaml:"statement_timeout"` // pg statement_timeout (default: 0 = none)
	LockTimeout      time.Duration `yaml:"lock_timeout"`      // pg lock_timeout (default: 0 = none)

	// --- MongoDB Specific ---
	ServerSelectionTimeout time.Duration `yaml:"server_selection_timeout"` // (default: 10s)
	SocketTimeout          time.Duration `yaml:"socket_timeout"`           // (default: 30s)
	RetryWrites            *bool         `yaml:"retry_writes"`             // (default: true)
	RetryReads             *bool         `yaml:"retry_reads"`              // (default: true)
	WriteConcern           string        `yaml:"write_concern"`            // "majority", "1", etc. (default: "majority")
	ReadPreference         string        `yaml:"read_preference"`          // "primary", "secondary", "nearest" (default: "primary")
}

// CacheConfig holds configuration details for initializing the caching layer.
type CacheConfig struct {
	Type       string        `yaml:"type"`         // "redis" or "memory"
	Addr       string        `yaml:"addr"`         // Redis host:port
	Password   string        `yaml:"password"`     // Redis password
	DB         int           `yaml:"db"`           // Redis database number (0-15)
	PoolSize   int           `yaml:"pool_size"`    // Redis max connections
	L1TTL      time.Duration `yaml:"l1_ttl"`       // L1 in-memory cache TTL
	L1MaxItems int           `yaml:"l1_max_items"` // L1 in-memory cache max entries
}

// Validate checks that the cache configuration is valid.
func (c *CacheConfig) Validate() error {
	if c.Type != "redis" && c.Type != "memory" {
		return NewConfigError("cache.type must be 'redis' or 'memory'")
	}
	return nil
}

// StorageConfig holds file storage settings.
type StorageConfig struct {
	Provider        string `yaml:"provider"` // "local", "s3", or "r2"
	Bucket          string `yaml:"bucket"`
	Region          string `yaml:"region"`
	AccountID       string `yaml:"account_id"`        // R2 account ID
	AccessKeyID     string `yaml:"access_key_id"`     // static credential (optional override)
	SecretAccessKey string `yaml:"secret_access_key"` // static credential (optional override)
}

// RateLimitConfig holds settings for the HTTP rate-limiter middleware.
type RateLimitConfig struct {
	// Backend selects the rate-limit store: "memory" (default, single-node) or "redis" (distributed).
	// When "redis" is chosen the limiter reuses the Cache connection settings (addr, password, db).
	// Redis mode automatically falls back to in-memory when Redis is unavailable and promotes back
	// after three consecutive successful probes (~90 s of stability).
	Backend string `yaml:"backend"`

	// GlobalRate is the token-bucket refill rate (tokens/sec) for the single global bucket
	// shared across all clients. Caps total API throughput regardless of per-IP limits.
	GlobalRate float64 `yaml:"global_rate"`

	// GlobalBurst is the maximum burst size for the global bucket.
	GlobalBurst float64 `yaml:"global_burst"`
}

// AuthConfig holds authentication / JWT settings.
type AuthConfig struct {
	Mode         string `yaml:"mode"` // "local" (default) or "oidc"
	OIDCIssuer   string `yaml:"oidc_issuer"`
	OIDCAudience string `yaml:"oidc_audience"`
	OIDCJwksURL  string `yaml:"oidc_jwks_url"`
}

// ---------------------------------------------------------------------------
// Module config
// ---------------------------------------------------------------------------

// ModuleToggle is the minimal config for a module that has no provider-specific settings.
type ModuleToggle struct {
	Enabled bool `yaml:"enabled"`
}

// CatalogFeaturesConfig lists optional features that build on the catalog core.
type CatalogFeaturesConfig struct {
	Images    bool `yaml:"images"`
	Variants  bool `yaml:"variants"`
	Discounts bool `yaml:"discounts"`
	Bulk      bool `yaml:"bulk"`
	Reviews   bool `yaml:"reviews"`
}

// CatalogModuleConfig holds settings for the catalog module.
type CatalogModuleConfig struct {
	Enabled  bool                  `yaml:"enabled"`
	Features CatalogFeaturesConfig `yaml:"features"`
}

// InventoryFeaturesConfig lists optional features that build on the inventory core.
type InventoryFeaturesConfig struct {
	Bulk        bool `yaml:"bulk"`
	History     bool `yaml:"history"`
	Reservation bool `yaml:"reservation"`
	Reports     bool `yaml:"reports"`
	Transfer    bool `yaml:"transfer"`
	Adjustment  bool `yaml:"adjustment"`
	Sync        bool `yaml:"sync"`
}

// InventoryModuleConfig holds settings for the inventory module.
type InventoryModuleConfig struct {
	Enabled  bool                    `yaml:"enabled"`
	Features InventoryFeaturesConfig `yaml:"features"`
}

// ShippingProviderConfig holds settings for a single shipping rate provider.
type ShippingProviderConfig struct {
	Type      string  `yaml:"type"` // "flatrate", "freeabove", "weightbased"
	Rate      float64 `yaml:"rate"`
	Threshold float64 `yaml:"threshold"`
	BaseRate  float64 `yaml:"base_rate"`
	PerKg     float64 `yaml:"per_kg"`
}

// ShippingModuleConfig holds settings for the shipping module.
type ShippingModuleConfig struct {
	Enabled   bool                     `yaml:"enabled"`
	Providers []ShippingProviderConfig `yaml:"providers"`
}

// PaymentsModuleConfig holds settings for the payments module.
type PaymentsModuleConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Provider string `yaml:"provider"` // "stripe", "paypal", "payhere"
	APIKey   string `yaml:"api_key"`
}

// ModulesConfig is the top-level module toggle config.
// All modules default to enabled when the modules section is absent from YAML.
type ModulesConfig struct {
	Catalog       CatalogModuleConfig   `yaml:"catalog"`
	Inventory     InventoryModuleConfig `yaml:"inventory"`
	Cart          ModuleToggle          `yaml:"cart"`
	Orders        ModuleToggle          `yaml:"orders"`
	Payments      PaymentsModuleConfig  `yaml:"payments"`
	Shipping      ShippingModuleConfig  `yaml:"shipping"`
	Notifications ModuleToggle          `yaml:"notifications"`
	Dashboard     DashboardConfig       `yaml:"dashboard"`
}

// MetricsConfig controls whether the internal Prometheus metrics server is started.
// When Enabled is false no metrics server is started and the DB pool collector
// is not registered — useful for lightweight deployments that do not run a
// Prometheus stack. Internal metric counters still increment (negligible cost)
// so no call sites need to change regardless of this setting.
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Addr    string `yaml:"addr"` // listen address for the internal metrics server (default ":9090")
}

// OutboxConfig holds settings for the transactional outbox + relay.
type OutboxConfig struct {
	Enabled      bool          `yaml:"enabled"`
	PollInterval time.Duration `yaml:"poll_interval"`
	BatchSize    int           `yaml:"batch_size"`
}

// DashboardConfig holds settings for the admin dashboard module.
type DashboardConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Tier     string        `yaml:"tier"`      // "small" (default) | "medium"
	CacheTTL time.Duration `yaml:"cache_ttl"` // medium tier response cache TTL, default 5m
}

// Config is the consolidated application configuration.
// Use LoadConfig (YAML + env overlay) or LoadConfigFromEnv (env-only) to initialize.
type Config struct {
	Port           string           `yaml:"port"`
	Secret         string           `yaml:"secret"`
	ServiceName    string           `yaml:"service_name"`
	MaxRequestSize int64            `yaml:"max_request_size"`
	Auth           AuthConfig       `yaml:"auth"`
	RateLimit      RateLimitConfig  `yaml:"rate_limit"`
	Metrics        MetricsConfig    `yaml:"metrics"`
	DB             DBConfig         `yaml:"db"`
	Cache          CacheConfig      `yaml:"cache"`
	Storage        StorageConfig    `yaml:"storage"`
	Events         events.BusConfig `yaml:"events"`
	Outbox         OutboxConfig     `yaml:"outbox"`
	Modules        ModulesConfig    `yaml:"modules"`
}

// Validate returns an error if the config is missing required values.
func (c *Config) Validate() error {
	if c.Secret == "" {
		return NewConfigError("secret (JWT_SECRET) must not be empty")
	}
	if c.DB.Type != "mongodb" && c.DB.Type != "postgres" {
		return fmt.Errorf("db.type must be 'mongodb' or 'postgres', got %q", c.DB.Type)
	}
	if c.DB.ConnectionString == "" {
		return NewConfigError("db.connection_string must not be empty")
	}
	if err := c.Cache.Validate(); err != nil {
		return err
	}
	if c.Auth.Mode != "local" && c.Auth.Mode != "oidc" {
		return fmt.Errorf("auth.mode must be 'local' or 'oidc', got %q", c.Auth.Mode)
	}
	return nil
}

// defaultModulesConfig returns a batteries-included ModulesConfig with everything enabled.
// This is used when the modules section is absent from the YAML file.
func defaultModulesConfig() ModulesConfig {
	return ModulesConfig{
		Catalog: CatalogModuleConfig{
			Enabled: true,
			Features: CatalogFeaturesConfig{
				Images:    true,
				Variants:  true,
				Discounts: true,
				Bulk:      true,
				Reviews:   true,
			},
		},
		Inventory: InventoryModuleConfig{
			Enabled: true,
			Features: InventoryFeaturesConfig{
				Bulk:        true,
				History:     true,
				Reservation: true,
				Reports:     true,
				Transfer:    true,
				Adjustment:  true,
				Sync:        true,
			},
		},
		Cart:          ModuleToggle{Enabled: true},
		Orders:        ModuleToggle{Enabled: true},
		Payments:      PaymentsModuleConfig{Enabled: true, Provider: "stripe"},
		Shipping:      ShippingModuleConfig{Enabled: true},
		Notifications: ModuleToggle{Enabled: true},
		Dashboard:     DashboardConfig{Enabled: true, Tier: "small", CacheTTL: 5 * time.Minute},
	}
}

// LoadConfig is the primary entry point.
// It checks the APP_CONFIG env var for a YAML file path, loads it, then overlays
// any environment variables on top. Falls back to pure env-var loading when APP_CONFIG
// is not set. All modules default to enabled when the modules section is absent.
func LoadConfig() (Config, error) {
	configPath := os.Getenv("APP_CONFIG")
	if configPath == "" {
		// Try well-known default locations.
		for _, candidate := range []string{"config.yaml", "config.yml"} {
			if _, err := os.Stat(candidate); err == nil {
				configPath = candidate
				break
			}
		}
	}

	var cfg Config
	if configPath != "" {
		var err error
		cfg, err = LoadConfigFromFile(configPath)
		if err != nil {
			return Config{}, err
		}
	} else {
		cfg = defaultConfig()
	}

	overlayEnv(&cfg)
	return cfg, nil
}

// LoadConfigFromFile reads a YAML config file and returns the resulting Config.
// Missing module sections default to all-enabled (batteries included).
func LoadConfigFromFile(path string) (Config, error) {
	cleanPath := filepath.Clean(path)
	// #nosec G304 G703 -- Config path is controlled by system administrator, not untrusted user inputs
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return Config{}, fmt.Errorf("config: cannot read %s: %w", path, err)
	}

	// Start with a fully-enabled default so that absent YAML sections keep sane values.
	cfg := defaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("config: cannot parse %s: %w", path, err)
	}
	return cfg, nil
}

// LoadConfigFromEnv reads configuration exclusively from environment variables.
// Kept for backwards compatibility and for deployments that do not use a config file.
func LoadConfigFromEnv() Config {
	cfg := defaultConfig()
	overlayEnv(&cfg)
	return cfg
}

// ---------------------------------------------------------------------------
// Internals
// ---------------------------------------------------------------------------

// defaultConfig returns a Config populated with safe, development-friendly defaults.
func defaultConfig() Config {
	return Config{
		Port:           "8080",
		Secret:         "default-engine-super-secret-key-that-is-very-long",
		ServiceName:    "ecom-engine",
		MaxRequestSize: 5 << 20, // 5 MB
		Auth: AuthConfig{
			Mode: "local",
		},
		RateLimit: RateLimitConfig{
			Backend:     "memory",
			GlobalRate:  10000,
			GlobalBurst: 15000,
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Addr:    ":9090",
		},
		DB: DBConfig{
			Type:                   "mongodb",
			ConnectionString:       "mongodb://localhost:27017",
			Database:               "ecom_db",
			MaxPoolSize:            25,
			MinPoolSize:            5,
			MaxConnIdleTime:        15 * time.Minute,
			MaxConnLifetime:        1 * time.Hour,
			MaxConnLifetimeJitter:  2 * time.Minute,
			ConnectTimeout:         10 * time.Second,
			PoolAcquireTimeout:     5 * time.Second,
			HealthCheckInterval:    30 * time.Second,
			QueryTimeout:           30 * time.Second,
			TransactionTimeout:     60 * time.Second,
			RetryMaxAttempts:       20,
			RetryInitialDelay:      1 * time.Second,
			RetryMaxDelay:          30 * time.Second,
			SSLMode:                "disable",
			ServerSelectionTimeout: 10 * time.Second,
			SocketTimeout:          30 * time.Second,
			RetryWrites:            boolPtr(true),
			RetryReads:             boolPtr(true),
			WriteConcern:           "majority",
			ReadPreference:         "primary",
		},
		Cache: CacheConfig{
			Type:       "memory",
			Addr:       "localhost:6379",
			DB:         0,
			PoolSize:   10,
			L1TTL:      5 * time.Minute,
			L1MaxItems: 10000,
		},
		Storage: StorageConfig{
			Provider: "local",
			Bucket:   "products",
			Region:   "us-east-1",
		},
		Events: events.BusConfig{
			Provider: "local",
			Retry: events.RetryConfig{
				MaxRetries:     3,
				InitialBackoff: 50 * time.Millisecond,
				MaxBackoff:     2 * time.Second,
			},
			Local: events.LocalBusConfig{DLQBufferSize: 100},
			Kafka: events.KafkaConfig{
				Brokers:       []string{"localhost:9092"},
				GroupID:       "ecom-engine-group",
				ClientID:      "ecom-engine-client",
				DLQTopic:      "events.dlq",
				AutoOffsetOld: true,
			},
			RabbitMQ: events.RabbitMQConfig{
				URL:          "amqp://localhost:5672/",
				ExchangeName: "ecom_events",
				ExchangeType: "topic",
				QueueName:    "ecom_queue",
				DLQExchange:  "ecom_events_dlq",
				DLQQueue:     "ecom_queue_dlq",
			},
		},
		Outbox: OutboxConfig{
			Enabled:      false,
			PollInterval: 5 * time.Second,
			BatchSize:    100,
		},
		Modules: defaultModulesConfig(),
	}
}

// overlayEnv applies environment variables on top of a Config.
// An env var overrides the corresponding YAML/default value only when it is non-empty
// (or, for numeric vars, only when the env var is set).
func overlayEnv(cfg *Config) {
	setStr := func(dest *string, env string) {
		if v := os.Getenv(env); v != "" {
			*dest = v
		}
	}
	setInt := func(dest *int, env string) {
		if v := os.Getenv(env); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				*dest = n
			}
		}
	}
	setInt64 := func(dest *int64, env string) {
		if v := os.Getenv(env); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				*dest = n
			}
		}
	}
	setFloat64 := func(dest *float64, env string) {
		if v := os.Getenv(env); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				*dest = f
			}
		}
	}
	setDuration := func(dest *time.Duration, env string) {
		if v := os.Getenv(env); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				*dest = d
			}
		}
	}

	setStr(&cfg.Port, "PORT")
	setStr(&cfg.Secret, "JWT_SECRET")
	setStr(&cfg.ServiceName, "SERVICE_NAME")
	if cfg.ServiceName == "ecom-engine" {
		setStr(&cfg.ServiceName, "OTEL_SERVICE_NAME")
	}
	setInt64(&cfg.MaxRequestSize, "MAX_REQUEST_SIZE_BYTES")

	// Metrics
	if v := os.Getenv("METRICS_ENABLED"); v == "false" {
		cfg.Metrics.Enabled = false
	}
	setStr(&cfg.Metrics.Addr, "METRICS_ADDR")

	// RateLimit
	setStr(&cfg.RateLimit.Backend, "RATE_LIMIT_BACKEND")
	setFloat64(&cfg.RateLimit.GlobalRate, "RATE_LIMIT_GLOBAL_RATE")
	setFloat64(&cfg.RateLimit.GlobalBurst, "RATE_LIMIT_GLOBAL_BURST")

	// Auth
	setStr(&cfg.Auth.Mode, "AUTH_MODE")
	setStr(&cfg.Auth.OIDCIssuer, "OIDC_ISSUER")
	setStr(&cfg.Auth.OIDCAudience, "OIDC_AUDIENCE")
	setStr(&cfg.Auth.OIDCJwksURL, "OIDC_JWKS_URL")

	// DB
	setStr(&cfg.DB.Type, "DB_TYPE")
	setStr(&cfg.DB.ConnectionString, "DB_CONNECTION_STRING")
	setStr(&cfg.DB.Database, "DB_DATABASE")
	setInt(&cfg.DB.MaxPoolSize, "DB_MAX_POOL_SIZE")
	setInt(&cfg.DB.MinPoolSize, "DB_MIN_POOL_SIZE")
	setDuration(&cfg.DB.MaxConnIdleTime, "DB_MAX_CONN_IDLE_TIME")
	setDuration(&cfg.DB.MaxConnLifetime, "DB_MAX_CONN_LIFETIME")
	setDuration(&cfg.DB.MaxConnLifetimeJitter, "DB_MAX_CONN_LIFETIME_JITTER")
	setDuration(&cfg.DB.ConnectTimeout, "DB_CONNECT_TIMEOUT")
	setDuration(&cfg.DB.PoolAcquireTimeout, "DB_POOL_ACQUIRE_TIMEOUT")
	setDuration(&cfg.DB.HealthCheckInterval, "DB_HEALTH_CHECK_INTERVAL")
	setDuration(&cfg.DB.QueryTimeout, "DB_QUERY_TIMEOUT")
	setDuration(&cfg.DB.TransactionTimeout, "DB_TRANSACTION_TIMEOUT")
	setInt(&cfg.DB.RetryMaxAttempts, "DB_RETRY_MAX_ATTEMPTS")
	setDuration(&cfg.DB.RetryInitialDelay, "DB_RETRY_INITIAL_DELAY")
	setDuration(&cfg.DB.RetryMaxDelay, "DB_RETRY_MAX_DELAY")
	setStr(&cfg.DB.SSLMode, "DB_SSL_MODE")
	setStr(&cfg.DB.ApplicationName, "DB_APPLICATION_NAME")
	setDuration(&cfg.DB.StatementTimeout, "DB_STATEMENT_TIMEOUT")
	setDuration(&cfg.DB.LockTimeout, "DB_LOCK_TIMEOUT")
	setDuration(&cfg.DB.ServerSelectionTimeout, "DB_SERVER_SELECTION_TIMEOUT")
	setDuration(&cfg.DB.SocketTimeout, "DB_SOCKET_TIMEOUT")
	if v := os.Getenv("DB_RETRY_WRITES"); v != "" {
		cfg.DB.RetryWrites = boolPtr(v == "true")
	}
	if v := os.Getenv("DB_RETRY_READS"); v != "" {
		cfg.DB.RetryReads = boolPtr(v == "true")
	}
	setStr(&cfg.DB.WriteConcern, "DB_WRITE_CONCERN")
	setStr(&cfg.DB.ReadPreference, "DB_READ_PREFERENCE")

	// Cache
	setStr(&cfg.Cache.Type, "CACHE_TYPE")
	setStr(&cfg.Cache.Addr, "CACHE_ADDR")
	setStr(&cfg.Cache.Password, "CACHE_PASSWORD")
	setInt(&cfg.Cache.DB, "CACHE_DB")
	setInt(&cfg.Cache.PoolSize, "CACHE_POOL_SIZE")
	setDuration(&cfg.Cache.L1TTL, "CACHE_L1_TTL")
	setInt(&cfg.Cache.L1MaxItems, "CACHE_L1_MAX_ITEMS")

	// Storage
	setStr(&cfg.Storage.Provider, "STORAGE_PROVIDER")
	setStr(&cfg.Storage.Bucket, "STORAGE_BUCKET")
	setStr(&cfg.Storage.Region, "STORAGE_REGION")
	setStr(&cfg.Storage.AccountID, "STORAGE_ACCOUNT_ID")
	setStr(&cfg.Storage.AccessKeyID, "STORAGE_ACCESS_KEY_ID")
	setStr(&cfg.Storage.SecretAccessKey, "STORAGE_SECRET_ACCESS_KEY")

	// Events
	setStr(&cfg.Events.Provider, "EVENT_PROVIDER")
	setInt(&cfg.Events.Retry.MaxRetries, "EVENT_MAX_RETRIES")
	setDuration(&cfg.Events.Retry.InitialBackoff, "EVENT_INITIAL_BACKOFF_MS") // kept for compat
	if v := os.Getenv("EVENT_INITIAL_BACKOFF_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Events.Retry.InitialBackoff = time.Duration(n) * time.Millisecond
		}
	}
	if v := os.Getenv("EVENT_MAX_BACKOFF_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Events.Retry.MaxBackoff = time.Duration(n) * time.Millisecond
		}
	}
	setInt(&cfg.Events.Local.DLQBufferSize, "EVENT_LOCAL_DLQ_SIZE")
	if v := os.Getenv("KAFKA_BROKERS"); v != "" {
		brokers := strings.Split(v, ",")
		for i := range brokers {
			brokers[i] = strings.TrimSpace(brokers[i])
		}
		cfg.Events.Kafka.Brokers = brokers
	}
	setStr(&cfg.Events.Kafka.GroupID, "KAFKA_GROUP_ID")
	setStr(&cfg.Events.Kafka.ClientID, "KAFKA_CLIENT_ID")
	setStr(&cfg.Events.Kafka.DLQTopic, "KAFKA_DLQ_TOPIC")
	if os.Getenv("KAFKA_AUTO_OFFSET_RESET_LATEST") == "true" {
		cfg.Events.Kafka.AutoOffsetOld = false
	}
	setStr(&cfg.Events.RabbitMQ.URL, "RABBITMQ_URL")
	setStr(&cfg.Events.RabbitMQ.ExchangeName, "RABBITMQ_EXCHANGE")
	setStr(&cfg.Events.RabbitMQ.ExchangeType, "RABBITMQ_EXCHANGE_TYPE")
	setStr(&cfg.Events.RabbitMQ.QueueName, "RABBITMQ_QUEUE")
	setStr(&cfg.Events.RabbitMQ.DLQExchange, "RABBITMQ_DLQ_EXCHANGE")
	setStr(&cfg.Events.RabbitMQ.DLQQueue, "RABBITMQ_DLQ_QUEUE")

	// Outbox
	if os.Getenv("OUTBOX_ENABLED") == "true" {
		cfg.Outbox.Enabled = true
	}
	setDuration(&cfg.Outbox.PollInterval, "OUTBOX_POLL_INTERVAL")
	setInt(&cfg.Outbox.BatchSize, "OUTBOX_BATCH_SIZE")

	// Module-level payments env override (mirrors legacy PAYMENT_* vars)
	setStr(&cfg.Modules.Payments.Provider, "PAYMENT_PROVIDER")
	setStr(&cfg.Modules.Payments.APIKey, "PAYMENT_API_KEY")

	// Dashboard
	setStr(&cfg.Modules.Dashboard.Tier, "DASHBOARD_TIER")
	setDuration(&cfg.Modules.Dashboard.CacheTTL, "DASHBOARD_CACHE_TTL")
}

func boolPtr(b bool) *bool {
	return &b
}
