// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package engine provides core business logic abstractions, system initialization, and configuration parsing.
package engine

import (
	"context"
	"fmt"
	"time"

	"ecom-engine/internal/core/admin"
	"ecom-engine/internal/core/auth"
	"ecom-engine/internal/events"
	"ecom-engine/internal/gateway/middleware"
	"ecom-engine/internal/infra/cache"
	"ecom-engine/internal/infra/cache/memory"
	"ecom-engine/internal/infra/cache/redis"
	infradb "ecom-engine/internal/infra/db"
	mongodb "ecom-engine/internal/infra/db/mongodb"
	postgres "ecom-engine/internal/infra/db/postgres"
	"ecom-engine/internal/infra/storage"
	storageLocal "ecom-engine/internal/infra/storage/local"
	storageR2 "ecom-engine/internal/infra/storage/r2"
	storageS3 "ecom-engine/internal/infra/storage/s3"
	"ecom-engine/internal/migrate"
	pkgdb "ecom-engine/pkg/db"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/metrics"
	"ecom-engine/pkg/token"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

// Engine holds shared infrastructure and the ordered set of active modules.
// Use NewEngine to bootstrap the system.
type Engine struct {
	Config          Config
	DBConn          pkgdb.Connection
	Cache           cache.Cache
	AuthService     auth.Service
	AuthMiddleware  gin.HandlerFunc
	AdminMiddleware gin.HandlerFunc
	activeModules   []Module
	disabledModules []DisabledModuleInfo
	outboxRelay     *events.OutboxRelay
	PoolStats       infradb.PoolStatsProvider
}

// ActiveModules returns modules in topological (dependency-first) init order.
func (e *Engine) ActiveModules() []Module { return e.activeModules }

// DisabledModules returns metadata for modules that are registered but disabled.
func (e *Engine) DisabledModules() []DisabledModuleInfo { return e.disabledModules }

// Shutdown tears down modules in reverse init order, then closes the DB connection.
func (e *Engine) Shutdown(ctx context.Context) error {
	for i := len(e.activeModules) - 1; i >= 0; i-- {
		m := e.activeModules[i]
		if err := m.Shutdown(ctx); err != nil {
			logger.Error("module %s shutdown error: %v", m.Name(), err)
		}
	}
	if e.outboxRelay != nil {
		e.outboxRelay.Stop()
	}
	if e.DBConn != nil {
		if err := e.DBConn.Close(); err != nil {
			return fmt.Errorf("db close: %w", err)
		}
	}
	return nil
}

// NewEngine bootstraps infrastructure, wires the Container, then initialises
// the supplied modules in dependency order and returns a ready Engine.
// active contains the enabled modules; disabled carries metadata for route
// catch-alls. Call registry.Collect to build these slices from Config.
func NewEngine(cfg Config, active []Module, disabled []DisabledModuleInfo) (*Engine, error) {
	// --- Topological sort + dependency validation ---
	sorted, err := validateAndSort(active)
	if err != nil {
		return nil, fmt.Errorf("module wiring: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Create 10 sec timeout
	defer cancel()

	// --- Event bus ---
	eventBus, err := events.NewEventBus(cfg.Events)
	if err != nil {
		return nil, fmt.Errorf("event bus: %w", err)
	}

	// --- JWT / OIDC ---
	jwtManager := token.NewJWTManager(cfg.Secret)
	var oidcValidator *token.OIDCValidator
	if cfg.Auth.Mode == "oidc" {
		oidcValidator, err = token.NewOIDCValidator(cfg.Auth.OIDCIssuer, cfg.Auth.OIDCAudience, cfg.Auth.OIDCJwksURL)
		if err != nil {
			return nil, fmt.Errorf("OIDC validator: %w", err)
		}
		logger.Info("OIDC validator initialized with issuer=%s", cfg.Auth.OIDCIssuer)
	}

	// --- Cache ---
	l1 := memory.NewMemoryAdapter(memory.WithMaxItems(cfg.Cache.L1MaxItems))
	var l2 cache.Cache
	var l2Redis *redis.RedisAdapter
	var cacheInstance cache.Cache

	switch cfg.Cache.Type {
	case "redis":
		redisCfg := &redis.Config{
			Addr:         cfg.Cache.Addr,
			Password:     cfg.Cache.Password,
			DB:           cfg.Cache.DB,
			PoolSize:     cfg.Cache.PoolSize,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			DialTimeout:  5 * time.Second,
		}
		l2Redis, err = redis.NewRedisAdapter(ctx, redisCfg)
		if err != nil {
			return nil, fmt.Errorf("redis cache: %w", err)
		}
		l2 = l2Redis
		cacheInstance = l2Redis
	case "memory":
		cacheInstance = l1
	default:
		return nil, fmt.Errorf("unsupported cache type %q; supported: redis, memory", cfg.Cache.Type)
	}
	cacheManager := cache.NewCacheManager(l1, l2, cache.WithL1TTL(cfg.Cache.L1TTL))
	logger.Info("Cache layers initialized: L1=memory, L2=%s", cfg.Cache.Type)

	// Register Redis connection pool collector when L2 is Redis.
	if l2Redis != nil {
		prometheus.MustRegister(metrics.NewCacheRedisPoolCollector(l2Redis.Client()))
	}

	// Register process/runtime collector (CPU %, RSS/VMS memory, heap, GC, goroutines).
	prometheus.MustRegister(metrics.NewRuntimeCollector())

	// --- Database & RepoProvider ---
	var dbConn pkgdb.Connection
	var txManager infradb.TransactionManager
	var repos *RepoProvider
	var poolStats infradb.PoolStatsProvider

	dbName := cfg.DB.Database
	if dbName == "" {
		dbName = "ecom_db"
	}

	switch cfg.DB.Type {
	case "mongodb":
		var client *mongo.Client

		opts := options.Client().ApplyURI(cfg.DB.ConnectionString)

		// Pool Sizing
		if cfg.DB.MaxPoolSize > 0 {
			opts.SetMaxPoolSize(uint64(cfg.DB.MaxPoolSize)) //nolint:gosec
		}
		if cfg.DB.MinPoolSize > 0 {
			opts.SetMinPoolSize(uint64(cfg.DB.MinPoolSize)) //nolint:gosec
		}

		// Lifecycles and Timeouts
		if cfg.DB.MaxConnIdleTime > 0 {
			opts.SetMaxConnIdleTime(cfg.DB.MaxConnIdleTime)
		}
		if cfg.DB.ConnectTimeout > 0 {
			opts.SetConnectTimeout(cfg.DB.ConnectTimeout)
		}
		if cfg.DB.ServerSelectionTimeout > 0 {
			opts.SetServerSelectionTimeout(cfg.DB.ServerSelectionTimeout)
		}
		// Note: mongo-driver v2 removed SetSocketTimeout; per-operation timeouts
		// should be passed via context. SetTimeout sets a global operation timeout.
		if cfg.DB.SocketTimeout > 0 {
			opts.SetTimeout(cfg.DB.SocketTimeout)
		}

		// Retry Behavior
		if cfg.DB.RetryWrites != nil {
			opts.SetRetryWrites(*cfg.DB.RetryWrites)
		}
		if cfg.DB.RetryReads != nil {
			opts.SetRetryReads(*cfg.DB.RetryReads)
		}

		// Application Name
		appName := cfg.DB.ApplicationName
		if appName == "" {
			appName = cfg.ServiceName
		}
		opts.SetAppName(appName)

		// Write concern
		switch cfg.DB.WriteConcern {
		case "majority":
			opts.SetWriteConcern(writeconcern.Majority())
		case "1":
			opts.SetWriteConcern(writeconcern.W1())
		}

		// Read preference
		switch cfg.DB.ReadPreference {
		case "secondary":
			opts.SetReadPreference(readpref.Secondary())
		case "nearest":
			opts.SetReadPreference(readpref.Nearest())
		default:
			opts.SetReadPreference(readpref.Primary())
		}

		// Connect and ping using connectWithRetry
		err = connectWithRetry(context.Background(), cfg.DB, func() error {
			var connErr error
			client, connErr = mongo.Connect(opts)
			if connErr != nil {
				return connErr
			}
			pingCtx, pingCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer pingCancel()
			return client.Ping(pingCtx, nil)
		})
		if err != nil {
			return nil, fmt.Errorf("mongodb init failed: %w", err)
		}

		pingCtx, pingCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer pingCancel()

		mongoDB := client.Database(dbName)
		if _, indexErr := mongoDB.Collection("products").Indexes().CreateMany(pingCtx, []mongo.IndexModel{
			{Keys: bson.D{{Key: "category_id", Value: 1}}},
			{Keys: bson.D{{Key: "variants.id", Value: 1}}},
			{Keys: bson.D{{Key: "variants.sku", Value: 1}}},
		}); indexErr != nil {
			logger.Warn("Failed to ensure indexes on 'products' collection: %v", indexErr)
		}
		if _, indexErr := mongoDB.Collection("reviews").Indexes().CreateMany(pingCtx, []mongo.IndexModel{
			{Keys: bson.D{{Key: "product_id", Value: 1}}},
		}); indexErr != nil {
			logger.Warn("Failed to ensure indexes on 'reviews' collection: %v", indexErr)
		}
		dbConn = &pkgdb.MongoConnection{Client: client}
		txManager = mongodb.NewMongoAdapter(client, cfg.DB.TransactionTimeout)
		repos = newRepoProvider("mongodb", mongoDB, nil, txManager)
		logger.Info("MongoDB connected to database %q", dbName)

	case "postgres":
		adapter := postgres.NewPostgresAdapter(postgres.Config{
			MaxPoolSize:           cfg.DB.MaxPoolSize,
			MinPoolSize:           cfg.DB.MinPoolSize,
			MaxConnIdleTime:       cfg.DB.MaxConnIdleTime,
			MaxConnLifetime:       cfg.DB.MaxConnLifetime,
			MaxConnLifetimeJitter: cfg.DB.MaxConnLifetimeJitter,
			ConnectTimeout:        cfg.DB.ConnectTimeout,
			HealthCheckInterval:   cfg.DB.HealthCheckInterval,
			StatementTimeout:      cfg.DB.StatementTimeout,
			LockTimeout:           cfg.DB.LockTimeout,
			ApplicationName:       cfg.DB.ApplicationName,
			TransactionTimeout:    cfg.DB.TransactionTimeout,
		})
		err = connectWithRetry(context.Background(), cfg.DB, func() error {
			return adapter.Connect(context.Background(), cfg.DB.ConnectionString)
		})
		if err != nil {
			return nil, fmt.Errorf("postgres init failed: %w", err)
		}

		// Quick schema check — verifies migrations have been applied.
		// Run 'go run ./cmd/migrate up' if this fails.
		schemaCtx, schemaCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer schemaCancel()
		if migrationErr := migrate.QuickCheck(schemaCtx, cfg.DB.ConnectionString); migrationErr != nil {
			return nil, fmt.Errorf("schema not ready: %w", migrationErr)
		}
		dbConn = &pkgdb.MemoryConnection{}
		txManager = adapter
		poolStats = adapter
		repos = newRepoProvider("postgres", nil, adapter, txManager)
		logger.Info("Postgres connected")

	default:
		return nil, fmt.Errorf("unsupported database type %q; supported: mongodb, postgres", cfg.DB.Type)
	}

	// --- File storage ---
	var fileStorage storage.FileStorage
	switch cfg.Storage.Provider {
	case "s3":
		fileStorage, err = storageS3.NewS3Adapter(ctx, cfg.Storage.Region, cfg.Storage.AccessKeyID, cfg.Storage.SecretAccessKey)
		if err != nil {
			return nil, fmt.Errorf("s3 storage init: %w", err)
		}
	case "r2":
		fileStorage, err = storageR2.NewR2Adapter(ctx, cfg.Storage.AccountID, cfg.Storage.AccessKeyID, cfg.Storage.SecretAccessKey)
		if err != nil {
			return nil, fmt.Errorf("r2 storage init: %w", err)
		}
	default:
		fileStorage = storageLocal.NewLocalAdapter()
	}

	// --- Auth (infrastructure — always active, not a module) ---
	authSvc := auth.NewAuthService(repos.AuthUserRepo(), repos.AuthTokenRepo(), jwtManager, txManager)

	// --- Auth middleware (built before module Init so modules can store it from the Container) ---
	authMW := middleware.NewAuthMiddleware(cfg.Auth.Mode, jwtManager, oidcValidator, authSvc)
	adminMW := admin.AdminOnlyMiddleware()

	// --- Container ---
	container := &Container{
		Config:          cfg,
		EventBus:        eventBus,
		Cache:           cacheInstance,
		CacheManager:    cacheManager,
		DBConn:          dbConn,
		TxManager:       txManager,
		FileStorage:     fileStorage,
		Repos:           repos,
		AuthService:     authSvc,
		JWTManager:      jwtManager,
		OIDCValidator:   oidcValidator,
		AuthMiddleware:  authMW,
		AdminMiddleware: adminMW,
	}

	// --- Init each module in dependency order ---
	for _, m := range sorted {
		if err := m.Init(container); err != nil {
			return nil, fmt.Errorf("module %q init: %w", m.Name(), err)
		}
		logger.Info("Module %q initialized", m.Name())
	}

	// --- Outbox Relay ---
	var outboxRelay *events.OutboxRelay
	if cfg.Outbox.Enabled {
		outboxRepo := repos.OutboxRepo()
		dedupStore := repos.DedupStore()
		container.Outbox = outboxRepo
		container.DedupStore = dedupStore
		outboxRelay = events.NewOutboxRelay(outboxRepo, eventBus, cfg.Outbox.PollInterval, cfg.Outbox.BatchSize)
		outboxRelay.Start()
	}

	logger.Info("Engine initialized: %d active module(s), %d disabled", len(sorted), len(disabled))

	return &Engine{
		Config:          cfg,
		DBConn:          dbConn,
		Cache:           cacheInstance,
		AuthService:     authSvc,
		AuthMiddleware:  authMW,
		AdminMiddleware: adminMW,
		activeModules:   sorted,
		disabledModules: disabled,
		outboxRelay:     outboxRelay,
		PoolStats:       poolStats,
	}, nil
}

// connectWithRetry attempts fn with exponential backoff up to maxAttempts, and then
// continues to retry indefinitely at maxDelay until successful or context is cancelled.
func connectWithRetry(ctx context.Context, cfg DBConfig, fn func() error) error {
	maxAttempts := cfg.RetryMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	delay := cfg.RetryInitialDelay
	if delay <= 0 {
		delay = 1 * time.Second
	}
	maxDelay := cfg.RetryMaxDelay
	if maxDelay <= 0 {
		maxDelay = 30 * time.Second
	}

	attempt := 1
	for {
		if err := fn(); err != nil {
			logger.Warn("DB connection attempt %d failed: %v. Retrying in %s...",
				attempt, err, delay)

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return fmt.Errorf("connection retry cancelled: %w", ctx.Err())
			}

			// Exponential backoff up to maxAttempts, then cap at maxDelay and continue forever
			if attempt < maxAttempts {
				delay *= 2
				if delay > maxDelay {
					delay = maxDelay
				}
				attempt++
			} else {
				delay = maxDelay
				attempt++
			}
			continue
		}
		return nil // success
	}
}
