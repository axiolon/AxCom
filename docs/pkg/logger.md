---
title: logger
sidebar_label: logger
sidebar_position: 5
---

# logger

<DocBadge status="under-review" version="v0.1.0-alpha" />

The `logger` package provides structured logging built on Go's standard `log/slog`. It supports two output formats — a coloured terminal format for development and an **Elastic Common Schema (ECS) 8.11** JSON format for production. All context-aware methods automatically extract OpenTelemetry trace and span IDs, enabling log-to-trace correlation in observability backends.

**Import path:** `ecom-engine/pkg/logger`

> For the full ECS field schema, log-to-trace correlation guide, and Loki/Grafana usage, see [Observability → Logs](../observability/logs.md).

---

## Configuration

Controlled dynamically by the `APP_ENV` environment variable:

| Env Var | Values | Default | Description |
|---|---|---|---|
| `APP_ENV` | `production`, `staging`, `development`, `test` | `development` | Sets log level, log format, and Gin mode dynamically. |
| `SERVICE_NAME` | any string | `ecom-engine` | Injected as `service.name` in ECS JSON output |
| `GIN_MODE` | `release`, `debug`, `test` | *(dynamic)* | Sets Gin's mode. Overrides the default set by `APP_ENV`. |

### Environment Mapping

| Environment (`APP_ENV`) | Log Level | Format | Gin Mode |
|---|---|---|---|
| `production` / `prod` | `info` (logs info, warn, error) | `json` | `release` |
| `staging` / `stage` | `info` | `json` | `release` |
| `development` / `dev` / *(empty)* | `debug` | `text` | `debug` |
| `test` / `testing` | `debug` | `text` | `test` |

---

## Handlers

### text (development)

Uses [`lmittmann/tint`](https://github.com/lmittmann/tint) to produce colour-coded, human-readable terminal output with millisecond timestamps. Ideal for local development.

```
2026-06-27 14:32:01.123 INFO  server started on port 8080
2026-06-27 14:32:05.456 ERROR payment failed: connection refused  trace_id=abc123 span_id=def456
```

### json / ECS (production)

Uses the built-in `ECSHandler` to produce JSON conforming to Elastic Common Schema 8.11 — ready for direct ingestion by Filebeat or any log shipper targeting Elasticsearch.

```json
{
  "@timestamp": "2026-06-27T14:32:05.456Z",
  "log.level": "error",
  "message": "payment failed: connection refused",
  "service.name": "ecom-engine",
  "ecs.version": "8.11",
  "trace.id": "abc123...",
  "span.id": "def456..."
}
```

Note: `trace_id` and `span_id` slog attributes are remapped to `trace.id` and `span.id` automatically to conform to ECS field naming.

---

## Package-level functions (DefaultLogger)

The package exposes convenience functions that delegate to a shared `DefaultLogger` instance. This is the most common usage — no need to create a `Logger` instance manually.

```go
import "ecom-engine/pkg/logger"

logger.Info("server started on port %d", port)
logger.Warn("cache miss for key %s", key)
logger.Error("unhandled error: %v", err)
logger.Debug("processing item %d of %d", i, total)
```

### Context-aware variants

All `*Ctx` functions automatically extract `trace_id` and `span_id` from the active OTel span in the context, if one is present.

```go
logger.InfoCtx(ctx, "order placed: %s", orderID)
logger.WarnCtx(ctx, "retry attempt %d for %s", attempt, jobID)
logger.ErrorCtx(ctx, "payment failed: %v", err)
logger.DebugCtx(ctx, "cache lookup for key %s", cacheKey)
```

If no active span is found, these fall back to logging without trace fields.

---

## Logger instance

For components that need a persistent logger with fixed fields (e.g. a module name), create an instance with `With`:

```go
import "ecom-engine/pkg/logger"

log := logger.With("module", "checkout", "version", "v2")

log.InfoCtx(ctx, "order placed: %s", orderID)
// → message="order placed: ord_..." module=checkout version=v2 trace_id=... span_id=...
```

`With` returns a new `*Logger` without modifying the DefaultLogger.

---

## Logger struct methods

| Method | Description |
|---|---|
| `Info(format, v...)` | Log at Info level |
| `Warn(format, v...)` | Log at Warn level |
| `Error(format, v...)` | Log at Error level |
| `Debug(format, v...)` | Log at Debug level |
| `InfoCtx(ctx, format, v...)` | Info + OTel trace correlation |
| `WarnCtx(ctx, format, v...)` | Warn + OTel trace correlation |
| `ErrorCtx(ctx, format, v...)` | Error + OTel trace correlation |
| `DebugCtx(ctx, format, v...)` | Debug + OTel trace correlation |
| `With(args...)` | Returns new Logger with extra fields |

---

## Advanced: replacing the DefaultLogger

### SetDefault

```go
func SetDefault(l *Logger)
```

Replaces the package-level DefaultLogger. Useful for injecting a pre-configured logger (e.g. one with fixed `service`, `env` fields) at startup:

```go
log := logger.NewLogger().With("env", os.Getenv("APP_ENV"))
logger.SetDefault(log) // all future logger.Info/Warn/etc calls use this
```

### Reconfigure

```go
func Reconfigure()
```

Rebuilds the DefaultLogger from the current environment variables. Useful in tests that change `APP_ENV` mid-run.

```go
os.Setenv("APP_ENV", "production")
logger.Reconfigure()
// logger now outputs JSON logs at info level
```

---

## ECSHandler (advanced)

`ECSHandler` is the `slog.Handler` implementation used when `LOG_FORMAT=json`. It can be instantiated directly if you need a custom ECS handler with different options:

```go
import "ecom-engine/pkg/logger"

h := logger.NewECSHandler(os.Stdout, &logger.ECSHandlerOptions{
    Level:       slog.LevelWarn,
    ServiceName: "my-service",
})
```

The handler is safe for concurrent use — writes are protected by a mutex.
