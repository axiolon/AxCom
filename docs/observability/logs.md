---
title: "Logs"
description: "Structured logging in AxCom — ECS JSON schema, log levels, structured fields, and log-to-trace correlation."
sidebar_position: 3
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

# Logs

AxCom uses the `pkg/logger` package for all structured logging. Every log line produced in production is a JSON object conforming to **Elastic Common Schema (ECS) 8.11**, ready for ingestion by the OTel Collector → Loki pipeline.

> For Go API usage (how to call the logger from application code), see [pkg/logger](../pkg/logger.md).

---

## Log Formats

Two formats are available, resolved dynamically based on the `APP_ENV` environment variable.

### `text` - Development

Human-readable, colour-coded terminal output. Not suitable for log shippers.

```
2026-06-28 14:32:01.123 INFO  server started on port 8080
2026-06-28 14:32:05.456 ERROR payment failed: connection refused  trace_id=abc123 span_id=def456
```

### `json` — Production (ECS)

Each log line is a self-contained JSON object. Produced when `LOG_FORMAT=json`.

```json
{
  "@timestamp": "2026-06-28T14:32:05.456Z",
  "log.level": "error",
  "message": "payment failed: connection refused",
  "service.name": "ecom-engine",
  "ecs.version": "8.11",
  "trace.id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span.id": "00f067aa0ba902b7"
}
```

---

## ECS Field Reference

| Field             | Type            | Always present            | Description                                              |
| ----------------- | --------------- | ------------------------- | -------------------------------------------------------- |
| `@timestamp`      | ISO 8601 string | Yes                       | Log emission time in UTC                                 |
| `log.level`       | string          | Yes                       | `debug`, `info`, `warn`, `error`                         |
| `message`         | string          | Yes                       | The log message                                          |
| `service.name`    | string          | Yes                       | Value of `SERVICE_NAME` env var (default: `ecom-engine`) |
| `ecs.version`     | string          | Yes                       | Always `8.11`                                            |
| `trace.id`        | hex string      | When span active          | W3C TraceID of the active OTel span                      |
| `span.id`         | hex string      | When span active          | W3C SpanID of the active OTel span                       |
| additional fields | any             | When logged with `With()` | Module names, IDs, or any extra key-value pairs          |

> The `slog` attributes `trace_id` and `span_id` are automatically remapped to ECS field names `trace.id` and `span.id` by the ECS handler.

---

## Log Levels

| Level   | When to use                                                                                                     |
| ------- | --------------------------------------------------------------------------------------------------------------- |
| `debug` | Detailed internal state - cache keys, SQL queries, loop counters. Off by default (`LOG_LEVEL=debug` to enable). |
| `info`  | Normal operational events - server start, request handled, order placed.                                        |
| `warn`  | Degraded state that does not require immediate action - cache miss rate elevated, retry attempted.              |
| `error` | Action required - unhandled error, external service failure, data inconsistency.                                |

The minimum log level is set automatically based on the `APP_ENV` environment variable.

---

## Log-to-Trace Correlation

When a request has an active OpenTelemetry span (i.e. `OTEL_ENABLED=true`), the `*Ctx` logger methods automatically extract the trace and span IDs and attach them to the log line.

```go
// In a handler — ctx carries the active OTel span
logger.ErrorCtx(ctx, "payment failed: %v", err)
```

This produces:

```json
{
  "@timestamp": "2026-06-28T14:32:05.456Z",
  "log.level": "error",
  "message": "payment failed: connection timeout",
  "service.name": "ecom-engine",
  "ecs.version": "8.11",
  "trace.id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span.id": "00f067aa0ba902b7"
}
```

In Grafana:

1. Open the **Logs** dashboard and find the error log line.
2. Click the `trace.id` value — Grafana opens the corresponding Tempo trace.
3. From the trace, click any span to inspect its timing and attributes.

If `OTEL_ENABLED=false` or no active span exists, `trace.id` and `span.id` are simply omitted logging continues normally.

---

## Structured Fields with `With()`

Attach fixed context fields to every log line from a component:

```go
log := logger.With("module", "checkout", "order_id", orderID)

log.InfoCtx(ctx, "payment initiated")
// → { "message": "payment initiated", "module": "checkout", "order_id": "ord_..." }

log.ErrorCtx(ctx, "payment failed: %v", err)
// → { "message": "payment failed: ...", "module": "checkout", "order_id": "ord_..." }
```

---

## Log Pipeline

```
App (stdout JSON) → Docker log driver → OTel Collector filelog receiver
                                      → Loki (push)
                                      → Grafana (query)
```

The OTel Collector uses a `filelog` receiver to tail Docker JSON log files. Structured fields in the JSON log are parsed automatically — no additional parsing configuration is needed.

---

## Configuration Reference

| Env Var        | Values                           | Default       | Description                              |
| -------------- | -------------------------------- | ------------- | ---------------------------------------- |
| `APP_ENV`      | `production`, `staging`, `development`, `test` | `development` | Sets log level, log format, and Gin mode dynamically. |
| `SERVICE_NAME` | string                           | `ecom-engine` | Injected as `service.name` in ECS output |
| `GIN_MODE`     | `release`, `debug`, `test`       | *(dynamic)*   | Sets Gin's mode. Overrides the default set by `APP_ENV`. |

### Environment Mapping

| Environment (`APP_ENV`) | Log Level | Format | Gin Mode |
|---|---|---|---|
| `production` / `prod` | `info` | `json` | `release` |
| `staging` / `stage` | `info` | `json` | `release` |
| `development` / `dev` / *(empty)* | `debug` | `text` | `debug` |
| `test` / `testing` | `debug` | `text` | `test` |

---

## Loki in Grafana

The **Logs** dashboard (`ecom-engine-logs`) provides:

- **Log Events by Level** - timeseries panel showing `error`/`warn`/`info`/`debug` volume over time
- **Log Level Distribution** - pie chart of proportions across the selected time range
- **Live Log Stream** - raw log panel with multi-select level filter and trace ID search

Use the **Log Level** variable at the top of the dashboard to filter to `error` or `warn` during an incident.

For the full dashboard description, see [Dashboards → Logs](./dashboards.md#logs).
