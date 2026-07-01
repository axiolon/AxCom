---
title: telemetry
sidebar_label: telemetry
sidebar_position: 7
---

# telemetry

<DocBadge status="under-review" version="v0.1.0-alpha" />

The `telemetry` package bootstraps the **OpenTelemetry SDK** for distributed tracing. It registers a global `TracerProvider` and `TextMapPropagator` so that all downstream code (including the `logger` and `response` packages) can attach trace context without knowing the transport details.

**Import path:** `ecom-engine/pkg/telemetry`

> For span naming conventions, sampling strategy, propagation details, and trace-to-log correlation, see [Observability → Traces](../observability/traces.md).

---

## Configuration

All configuration is read from environment variables via `ReadConfigFromEnv`.

| Env Var | Values | Default | Description |
|---|---|---|---|
| `OTEL_ENABLED` | `true` / `false` | `false` | Enables or disables the SDK entirely |
| `OTEL_SERVICE_NAME` | string | `ecom-engine` | Service name attached to all spans |
| `OTEL_SERVICE_VERSION` | string | `1.0.0` | Service version attached to all spans |
| `OTEL_ENVIRONMENT` | string | `production` | Deployment environment (`production`, `staging`, etc.) |
| `OTEL_TRACE_SAMPLE` | `0.0` – `1.0` | `0.01` | Fraction of traces to sample (1% by default) |
| `OTEL_EXPORTER` | `otlp`, `none` | `none` | Trace exporter. `otlp` → OTLP/HTTP; `none` → no export |

### Sampling behaviour

| `OTEL_TRACE_SAMPLE` | Sampler used |
|---|---|
| `<= 0` | `NeverSample` — no traces recorded |
| `>= 1` | `AlwaysSample` — all traces recorded |
| between 0 and 1 | `TraceIDRatioBased` — probabilistic sampling |

---

## Config struct

```go
type Config struct {
    Enabled        bool
    ServiceName    string
    ServiceVersion string
    Environment    string
    TraceSample    float64
    Exporter       string // "none" or "otlp"
}
```

---

## Functions

### ReadConfigFromEnv

```go
func ReadConfigFromEnv() Config
```

Reads all env vars and returns a populated `Config` with defaults applied for any missing values.

### Init

```go
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error)
```

Initializes the global OTel `TracerProvider` and `TextMapPropagator`.

Returns a **shutdown function** that must be deferred to flush and close the exporter cleanly.

When `cfg.Enabled` is `false`, a no-op propagator is registered so that all downstream calls to `trace.SpanFromContext()` are safe (they return a no-op span rather than panicking).

---

## Usage

Call `Init` once at application startup, before starting the HTTP server:

```go
import (
    "context"
    "ecom-engine/pkg/telemetry"
    "ecom-engine/pkg/logger"
)

func main() {
    ctx := context.Background()

    cfg := telemetry.ReadConfigFromEnv()
    shutdown, err := telemetry.Init(ctx, cfg)
    if err != nil {
        logger.Error("failed to init telemetry: %v", err)
        os.Exit(1)
    }
    defer shutdown(ctx)

    // start HTTP server...
}
```

---

## OTLP exporter

When `OTEL_EXPORTER=otlp`, the package uses OTLP/HTTP transport. The endpoint is configured via the standard OTel SDK environment variables (not managed by this package):

| Env Var | Description |
|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Collector endpoint, e.g. `http://otel-collector:4318` |
| `OTEL_EXPORTER_OTLP_HEADERS` | Optional auth headers |

Compatible collectors: OpenTelemetry Collector, Jaeger (v2+), Grafana Tempo, Honeycomb, Datadog Agent.

---

## Resource attributes

All spans produced by this service include the following resource attributes:

| Attribute | Value |
|---|---|
| `service.name` | `OTEL_SERVICE_NAME` |
| `service.version` | `OTEL_SERVICE_VERSION` |
| `deployment.environment.name` | `OTEL_ENVIRONMENT` |
