// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package main wires dependencies, starts the HTTP server,
// and handles graceful shutdown.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"ecom-engine/internal/engine"
	"ecom-engine/internal/gateway"
	"ecom-engine/internal/modules/registry"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/metrics"
	"ecom-engine/pkg/telemetry"

	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		logger.Error("Application terminated with error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// --- 1. Load Environment Variables ---
	// Determine the environment (e.g. dev, prod, staging) and load corresponding .env file.
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	envFile := ".env." + env
	if err := godotenv.Load(envFile); err != nil {
		// Try the default .env if the environment-specific file is missing.
		if errDefault := godotenv.Load(); errDefault != nil {
			logger.Warn("No custom env file (%s) or default .env file found: %v. Using system environment variables.", envFile, errDefault)
		} else {
			logger.Info("Loaded default .env file")
		}
	} else {
		logger.Info("Loaded configuration from environment file: %s", envFile)
	}

	// --- 2. Initialize Telemetry & Tracing ---
	// Read telemetry configurations from the environment and initialize tracing/telemetry adapters.
	telCfg := telemetry.ReadConfigFromEnv()
	telemetryShutdown, err := telemetry.Init(context.Background(), telCfg)
	if err != nil {
		logger.Error("Failed to initialize telemetry: %v", err)
	} else {
		// Reconfigure logger so the OTel slog bridge picks up the real LoggerProvider.
		logger.Reconfigure()
		// Ensure tracer and log providers shut down gracefully when main exits.
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if shutErr := telemetryShutdown(ctx); shutErr != nil {
				logger.Error("Error shutting down telemetry providers: %v", shutErr)
			}
		}()
	}

	// --- 3. Diagnostics Logging ---
	// Log core runtime environment diagnostics for supportability and debugging.
	logger.Info("Starting server environment diagnostics:\n"+
		"  - Go Version:                 %s\n"+
		"  - OS:                         %s\n"+
		"  - Arch:                       %s\n"+
		"  - PID:                        %d",
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
		os.Getpid(),
	)

	// --- 4. Load Configuration ---
	// Load structural configurations from yaml file and overlay dynamic env variables.
	cfg, err := engine.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Log loaded configurations (masking credentials like secrets/keys).
	logger.Info("Server Configuration loaded:\n"+
		"  - Port:                       %s\n"+
		"  - JWT Secret Configured:      %t\n"+
		"  - DB Type:                    %s\n"+
		"  - DB Conn String Configured:  %t\n"+
		"  - Cache Type:                 %s\n"+
		"  - Cache Address:              %q\n"+
		"  - Metrics Enabled:            %t\n"+
		"  - Metrics Addr:               %s\n"+
		"  - Payment Provider:           %s\n"+
		"  - Payment API Key Configured: %t",
		cfg.Port,
		cfg.Secret != "",
		cfg.DB.Type,
		cfg.DB.ConnectionString != "",
		cfg.Cache.Type,
		cfg.Cache.Addr,
		cfg.Metrics.Enabled,
		cfg.Metrics.Addr,
		cfg.Modules.Payments.Provider,
		cfg.Modules.Payments.APIKey != "",
	)

	logger.Info("Database pool configuration:\n"+
		"  - Max Pool Size:              %d\n"+
		"  - Min Pool Size:              %d\n"+
		"  - Max Conn Idle Time:         %s\n"+
		"  - Max Conn Lifetime:          %s\n"+
		"  - Conn Lifetime Jitter:       %s\n"+
		"  - Connect Timeout:            %s\n"+
		"  - Pool Acquire Timeout:       %s\n"+
		"  - Health Check Interval:      %s\n"+
		"  - Query Timeout:              %s\n"+
		"  - Transaction Timeout:        %s\n"+
		"  - Startup Retry Attempts:     %d",
		cfg.DB.MaxPoolSize,
		cfg.DB.MinPoolSize,
		cfg.DB.MaxConnIdleTime,
		cfg.DB.MaxConnLifetime,
		cfg.DB.MaxConnLifetimeJitter,
		cfg.DB.ConnectTimeout,
		cfg.DB.PoolAcquireTimeout,
		cfg.DB.HealthCheckInterval,
		cfg.DB.QueryTimeout,
		cfg.DB.TransactionTimeout,
		cfg.DB.RetryMaxAttempts,
	)

	// --- 5. Boot Engine ---
	// Collect registered modules and partition them into active/disabled sets based on config.
	active, disabled := registry.Collect(cfg)
	// Initialize the shared infrastructure (DB, Cache, Event Bus, DI Container),
	// validate dependencies, sort modules topologically, and execute module-specific Init functions.
	eng, err := engine.NewEngine(cfg, active, disabled)
	if err != nil {
		return fmt.Errorf("failed to initialize engine: %w", err)
	}

	// Ensure engine shuts down and cleans up resources (connections, pools) in reverse topological order on exit.
	defer func() {
		logger.Info("Shutting down engine...")
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer shutCancel()
		if err := eng.Shutdown(shutCtx); err != nil {
			logger.Error("Engine shutdown error: %v", err)
		} else {
			logger.Info("Engine shut down cleanly.")
		}
	}()

	// --- 6. Configure HTTP Server ---
	// Build HTTP router containing auth, rates, health probes, and active module routes.
	router := gateway.NewRouter(eng)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second, // Guard against Slowloris slow-header attacks.
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Channel to capture startup/runtime errors from the HTTP server.
	serverErrors := make(chan error, 1)

	// --- 7. Start HTTP Server ---
	// Run the HTTP server listener concurrently in a separate goroutine.
	go func() {
		logger.Info("Starting API server on port %s...", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// --- 7b. Start Internal Metrics Server ---
	// Serves /metrics on a separate port that is never exposed publicly.
	var metricsSrv *http.Server
	if cfg.Metrics.Enabled {
		metricsSrv = metrics.NewInternalServer(cfg.Metrics.Addr)
		go func() {
			logger.Info("Starting internal metrics server on %s...", cfg.Metrics.Addr)
			if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				serverErrors <- err
			}
		}()
	}

	// --- 8. Wait for Interrupt / Terminate Signals ---
	// Set up channel listener for OS signals to trigger a graceful shutdown sequence.
	shutdownChannel := make(chan os.Signal, 1)
	signal.Notify(shutdownChannel, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		// Server failed to bind or failed at runtime.
		logger.Error("Server failed to start or encountered a critical error: %v", err)
		return err

	case sig := <-shutdownChannel:
		// Termination signal captured (SIGINT or SIGTERM).
		logger.Info("Shutdown signal received: %v. Initiating graceful shutdown...", sig)

		// Set a deadline (15s) to allow active TCP connections/requests to finish before forcing close.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if metricsSrv != nil {
			if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
				logger.Error("Metrics server shutdown failed: %v", err)
			} else {
				logger.Info("Internal metrics server stopped.")
			}
		}

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("Server shutdown failed: %v. Forcing close.", err)
			if err := srv.Close(); err != nil {
				logger.Error("Forced server closure failed: %v", err)
			}
		} else {
			logger.Info("API server gracefully stopped accepting new connections.")
		}
	}

	logger.Info("Server shutdown sequence completed.")
	return nil
}
