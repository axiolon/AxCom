---
title: "Traces"
description: "Distributed tracing in AxCom — OpenTelemetry SDK setup, span naming, sampling, propagation, and trace-to-log correlation."
sidebar_position: 4
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

# Traces

AxCom uses the **OpenTelemetry SDK** for distributed tracing via the `pkg/telemetry` package. Traces are exported via OTLP/HTTP to the OTel Collector and forwarded to Tempo for storage and querying in Grafana.

> For Go API usage (initialising the SDK, shutdown hooks), see [pkg/telemetry](../pkg/telemetry.md).

---

## How Tracing Works

Each incoming HTTP request can carry a trace — a tree of timed spans representing the work done across services. Within a single AxCom process, a trace captures:

- HTTP handler execution (root span)
- Downstream database queries
- Cache lookups
- External HTTP calls

All spans carry **resource attributes** (service name, version, environment) and are linked by a shared `trace_id`. When `trace.id` appears in a log line, it points directly to the corresponding trace in Tempo.

---

## Configuration

All settings are read from environment variables at startup.

| Env Var                       | Values           | Default       | Description                                     |
| ----------------------------- | ---------------- | ------------- | ----------------------------------------------- |
| `OTEL_ENABLED`                | `true` / `false` | `false`       | Enables or disables the SDK                     |
| `OTEL_SERVICE_NAME`           | string           | `ecom-engine` | Service name on all spans                       |
| `OTEL_SERVICE_VERSION`        | string           | `1.0.0`       | Service version on all spans                    |
| `OTEL_ENVIRONMENT`            | string           | `production`  | Deployment environment                          |
| `OTEL_TRACE_SAMPLE`           | `0.0` – `1.0`    | `0.01`        | Fraction of traces to sample                    |
| `OTEL_EXPORTER`               | `otlp`, `none`   | `none`        | Trace exporter                                  |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | URL              | —             | Collector endpoint (e.g. `http://otelcol:4318`) |

### Enabling Traces

Add to your app environment (e.g. `.env.dev`):

```env
OTEL_ENABLED=true
OTEL_EXPORTER=otlp
OTEL_EXPORTER_OTLP_ENDPOINT=http://otelcol:4318
OTEL_TRACE_SAMPLE=0.1   # sample 10% in staging
```

In production, keep `OTEL_TRACE_SAMPLE` at `0.01` (1%) unless debugging a specific issue.

---

## Sampling

| `OTEL_TRACE_SAMPLE` value | Sampler                             | When to use                     |
| ------------------------- | ----------------------------------- | ------------------------------- |
| `<= 0`                    | `NeverSample` — no traces           | Disabled                        |
| `> 0` and `< 1`           | `TraceIDRatioBased` — probabilistic | Production (1%) / Staging (10%) |
| `>= 1`                    | `AlwaysSample` — every request      | Local development / debugging   |

Sampling is applied **at the root span**. If a trace is sampled, all child spans within the same trace are always included.

---

## Propagation

The SDK registers two standard propagators globally:

| Propagator           | Header                      | Purpose                                         |
| -------------------- | --------------------------- | ----------------------------------------------- |
| **W3C TraceContext** | `traceparent`, `tracestate` | Carry trace/span IDs between services           |
| **W3C Baggage**      | `baggage`                   | Carry key-value pairs across service boundaries |

When an upstream service (load balancer, API gateway, or another microservice) sends a `traceparent` header, the SDK automatically continues that trace rather than starting a new one. This enables end-to-end traces that span multiple services.

---

## Resource Attributes

Every span produced by AxCom includes these resource attributes:

| Attribute                     | Source                 |
| ----------------------------- | ---------------------- |
| `service.name`                | `OTEL_SERVICE_NAME`    |
| `service.version`             | `OTEL_SERVICE_VERSION` |
| `deployment.environment.name` | `OTEL_ENVIRONMENT`     |

These attributes appear in Tempo and can be used to filter traces by environment or version.

---

## Trace-to-Log Correlation

When a request is traced, the `trace.id` and `span.id` are automatically injected into every log line produced within that request's context (via `logger.*Ctx()` methods).

```json
{
  "@timestamp": "2026-06-28T14:32:05.456Z",
  "log.level": "error",
  "message": "checkout failed: payment timeout",
  "service.name": "ecom-engine",
  "trace.id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span.id": "00f067aa0ba902b7"
}
```

**Workflow in Grafana:**

1. See an error spike in **Service Health** or **HTTP Traffic** dashboard.
2. Open the **Logs** dashboard, filter by `error` level.
3. Find the relevant log line — copy `trace.id`.
4. Open Tempo in Grafana, search by trace ID.
5. Inspect the full trace: handler timing, DB query duration, cache hits.

---

## OTLP Exporter

When `OTEL_EXPORTER=otlp`, traces are exported using **OTLP/HTTP** (`/v1/traces`). The endpoint must be an OTel Collector (or compatible backend) that accepts OTLP HTTP.

In the self-hosted monitoring stack (Scenario 5), the OTel Collector is reachable at `http://otelcol:4318` over the `ecom-net` Docker network.

Compatible backends:

- OpenTelemetry Collector → Tempo
- Jaeger v2+
- Honeycomb
- Datadog Agent

---

## Current Instrumentation Status

| Layer                    | Instrumented | Notes                              |
| ------------------------ | ------------ | ---------------------------------- |
| HTTP handler (root span) | Planned      | OTel HTTP middleware not yet wired |
| Database queries         | Planned      | pgxpool OTel plugin not yet wired  |
| Cache operations         | Planned      | —                                  |
| External HTTP calls      | Planned      | —                                  |

The `pkg/telemetry` package and the OTel Collector pipeline are fully set up. The infrastructure is ready — adding instrumentation to individual layers is the next step.

> When `OTEL_ENABLED=false` (the current default), a no-op `TracerProvider` is registered. All downstream calls to `trace.SpanFromContext()` return a no-op span and are safe to call without nil checks.
