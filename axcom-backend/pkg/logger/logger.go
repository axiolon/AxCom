// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/trace"
)

// Logger wraps slog.Logger and provides trace/span context correlation.
type Logger struct {
	slog *slog.Logger
}

// DefaultLogger is the global default logger instance.
var DefaultLogger = NewLogger()

// multiHandler fans out log records to multiple slog.Handler implementations.
type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r.Clone()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}

// NewLogger creates a new Logger instance based on env vars (LOG_FORMAT, LOG_LEVEL).
// When OTEL_ENABLED=true and telemetry.Init has been called, logs are also forwarded
// to the OTel Collector via the global LoggerProvider.
func NewLogger() *Logger {
	env := strings.ToLower(os.Getenv("APP_ENV"))
	if env == "" {
		env = "development"
	}

	var format string
	var level slog.Level

	isTest := env == "test" || env == "testing"

	if isTest {
		level = slog.LevelDebug
		format = "text"
	} else {
		switch env {
		case "production", "prod":
			level = slog.LevelInfo
			format = "json"
		case "staging", "stage":
			level = slog.LevelInfo
			format = "json"
		case "development", "dev":
			fallthrough
		default:
			level = slog.LevelDebug
			format = "text"
		}
	}

	var stdoutHandler slog.Handler
	if format == "json" {
		stdoutHandler = NewECSHandler(os.Stdout, &ECSHandlerOptions{
			Level:       level,
			ServiceName: os.Getenv("SERVICE_NAME"),
		})
	} else {
		stdoutHandler = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      level,
			TimeFormat: "2006-01-02 15:04:05.000",
			NoColor:    false,
		})
	}

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "ecom-engine"
	}

	handler := slog.Handler(&multiHandler{
		handlers: []slog.Handler{
			stdoutHandler,
			otelslog.NewHandler(serviceName),
		},
	})

	return &Logger{
		slog: slog.New(handler),
	}
}

// With returns a new Logger containing the given attributes.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		slog: l.slog.With(args...),
	}
}

// Info logs messages at Info level.
func (l *Logger) Info(format string, v ...any) {
	l.slog.Info(fmt.Sprintf(format, v...))
}

// Error logs messages at Error level.
func (l *Logger) Error(format string, v ...any) {
	l.slog.Error(fmt.Sprintf(format, v...))
}

// Warn logs messages at Warn level.
func (l *Logger) Warn(format string, v ...any) {
	l.slog.Warn(fmt.Sprintf(format, v...))
}

// Debug logs messages at Debug level.
func (l *Logger) Debug(format string, v ...any) {
	l.slog.Debug(fmt.Sprintf(format, v...))
}

// getTraceAttrs extracts OTel trace and span IDs if present in the context.
func getTraceAttrs(ctx context.Context) []slog.Attr {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return nil
	}
	sc := span.SpanContext()
	return []slog.Attr{
		slog.String("trace_id", sc.TraceID().String()),
		slog.String("span_id", sc.SpanID().String()),
	}
}

// InfoCtx logs messages at Info level with trace_id/span_id extracted from context.
func (l *Logger) InfoCtx(ctx context.Context, format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	if attrs := getTraceAttrs(ctx); len(attrs) > 0 {
		l.slog.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
	} else {
		l.slog.InfoContext(ctx, msg)
	}
}

// ErrorCtx logs messages at Error level with trace_id/span_id extracted from context.
func (l *Logger) ErrorCtx(ctx context.Context, format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	if attrs := getTraceAttrs(ctx); len(attrs) > 0 {
		l.slog.LogAttrs(ctx, slog.LevelError, msg, attrs...)
	} else {
		l.slog.ErrorContext(ctx, msg)
	}
}

// WarnCtx logs messages at Warn level with trace_id/span_id extracted from context.
func (l *Logger) WarnCtx(ctx context.Context, format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	if attrs := getTraceAttrs(ctx); len(attrs) > 0 {
		l.slog.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
	} else {
		l.slog.WarnContext(ctx, msg)
	}
}

// DebugCtx logs messages at Debug level with trace_id/span_id extracted from context.
func (l *Logger) DebugCtx(ctx context.Context, format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	if attrs := getTraceAttrs(ctx); len(attrs) > 0 {
		l.slog.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
	} else {
		l.slog.DebugContext(ctx, msg)
	}
}

// Global helper functions redirecting to DefaultLogger

func Info(format string, v ...any) {
	DefaultLogger.Info(format, v...)
}

func Error(format string, v ...any) {
	DefaultLogger.Error(format, v...)
}

func Warn(format string, v ...any) {
	DefaultLogger.Warn(format, v...)
}

func Debug(format string, v ...any) {
	DefaultLogger.Debug(format, v...)
}

func InfoCtx(ctx context.Context, format string, v ...any) {
	DefaultLogger.InfoCtx(ctx, format, v...)
}

func ErrorCtx(ctx context.Context, format string, v ...any) {
	DefaultLogger.ErrorCtx(ctx, format, v...)
}

func WarnCtx(ctx context.Context, format string, v ...any) {
	DefaultLogger.WarnCtx(ctx, format, v...)
}

func DebugCtx(ctx context.Context, format string, v ...any) {
	DefaultLogger.DebugCtx(ctx, format, v...)
}

func With(args ...any) *Logger {
	return DefaultLogger.With(args...)
}

// SetDefault sets the package-level default logger.
func SetDefault(l *Logger) {
	if l != nil {
		DefaultLogger = l
	}
}

// Reconfigure reloads the package-level default logger from the current environment variables.
// Call this after telemetry.Init() so the OTel log bridge picks up the real LoggerProvider.
func Reconfigure() {
	DefaultLogger = NewLogger()
}
