// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package logger_test

import (
	"context"
	"ecom-engine/pkg/logger"
	"testing"

	"go.opentelemetry.io/otel"
)

func TestLoggerFunctions(_ *testing.T) {
	// Verify standard logs compile and execute
	logger.Info("Hello info: %s", "world")
	logger.Warn("Hello warn")
	logger.Error("Hello error")
	logger.Debug("Hello debug")

	ctx := context.Background()
	logger.InfoCtx(ctx, "Info with context: %d", 42)

	// Verify contextual logging with a mock trace span
	tp := otel.GetTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	logger.InfoCtx(ctx, "Info with active OTel span")
	logger.ErrorCtx(ctx, "Error with active OTel span")
}

func TestLoggerWith(_ *testing.T) {
	subLogger := logger.With("component", "testing")
	subLogger.Info("Sublogger info message")
}
