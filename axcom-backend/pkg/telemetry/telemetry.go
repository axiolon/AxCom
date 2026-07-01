// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otellog "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// Config holds Telemetry configuration details.
type Config struct {
	Enabled        bool
	ServiceName    string
	ServiceVersion string
	Environment    string
	TraceSample    float64
	Exporter       string // "none", "otlp"
}

// ReadConfigFromEnv initializes Config from environment variables.
func ReadConfigFromEnv() Config {
	enabled, _ := strconv.ParseBool(os.Getenv("OTEL_ENABLED"))
	sampleRate, err := strconv.ParseFloat(os.Getenv("OTEL_TRACE_SAMPLE"), 64)
	if err != nil {
		sampleRate = 0.01 // default 1%
	}
	exporter := os.Getenv("OTEL_EXPORTER")
	if exporter == "" {
		exporter = "none"
	}
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "ecom-engine"
	}
	serviceVersion := os.Getenv("OTEL_SERVICE_VERSION")
	if serviceVersion == "" {
		serviceVersion = "1.0.0"
	}
	environment := os.Getenv("OTEL_ENVIRONMENT")
	if environment == "" {
		environment = "production"
	}

	return Config{
		Enabled:        enabled,
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		Environment:    environment,
		TraceSample:    sampleRate,
		Exporter:       exporter,
	}
}

// Init initializes the global TracerProvider, LoggerProvider, and TextMapPropagator.
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if !cfg.Enabled {
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
		return func(_ context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			attribute.String("deployment.environment.name", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var sampler sdktrace.Sampler
	switch {
	case cfg.TraceSample <= 0:
		sampler = sdktrace.NeverSample()
	case cfg.TraceSample >= 1:
		sampler = sdktrace.AlwaysSample()
	default:
		sampler = sdktrace.TraceIDRatioBased(cfg.TraceSample)
	}

	var tp *sdktrace.TracerProvider
	if cfg.Exporter == "otlp" {
		traceExporter, err := otlptracehttp.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
		}
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sampler),
			sdktrace.WithResource(res),
			sdktrace.WithBatcher(traceExporter),
		)
	} else {
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sampler),
			sdktrace.WithResource(res),
		)
	}

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// ── Log provider ───────────────────────────────────────────────────────────
	var lp *sdklog.LoggerProvider
	if cfg.Exporter == "otlp" {
		logExporter, err := otlploghttp.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP log exporter: %w", err)
		}
		lp = sdklog.NewLoggerProvider(
			sdklog.WithResource(res),
			sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		)
		otellog.SetLoggerProvider(lp)
	}

	shutdown := func(shutdownCtx context.Context) error {
		if err := tp.Shutdown(shutdownCtx); err != nil {
			return err
		}
		if lp != nil {
			if err := lp.Shutdown(shutdownCtx); err != nil {
				return err
			}
		}
		return nil
	}

	return shutdown, nil
}
