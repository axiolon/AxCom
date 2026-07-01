# ADR-014: Observability and Telemetry Architecture

**Date:** 2026-06-27  
**Status:** accepted

## Context
Running a high-performance transactional commerce engine requires deep runtime visibility. If tracing, logging, and metrics are isolated:
- Finding the root cause of a latency spike or transaction error is highly manual.
- We cannot correlate logs directly to specific API request traces.
- We lack visibility into cache hit rates and database/Redis connection pool saturation.

Ad-hoc logging libraries and custom APM agents introduce vendor lock-in and complicate local development setups.

## Decision
1. **OpenTelemetry-First Tracing:** Bootstrap the OpenTelemetry SDK (`pkg/telemetry`) globally. Support OTLP/HTTP export and configurable sampling rates (`OTEL_TRACE_SAMPLE`) to stream traces to OpenTelemetry-compliant collectors (e.g. Jaeger, Tempo).
2. **Context-Aware Structured Logging:** Wrap Go's standard `log/slog` structured logging (`pkg/logger`). All logging actions must support context-aware handlers (`InfoCtx`, `ErrorCtx`) which automatically extract `trace_id` and `span_id` from the active span context, enabling immediate trace-to-log correlation.
3. **Unified Prometheus Metrics:** Centralize Prometheus performance metrics registration (`pkg/metrics`):
   - Track HTTP request rate, status codes, and latency histograms.
   - Track cache operations (L1/L2 hits, misses, evictions, stampede singleflight deduplications).
   - Write custom Prometheus collectors implementing `prometheus.Collector` to expose Postgres pool (`DBPoolCollector`) and Redis pool (`CacheRedisPoolCollector`) statistics dynamically.

## Alternatives Considered

| Option | Reason Rejected |
|--------|-----------------|
| Third-Party Logging (Zap/Logrus) | Zap and Logrus are heavy dependencies. Standardizing on Go's built-in `log/slog` offers native structured logging, faster compile times, and clean context integration without third-party wrapper libraries. |
| Vendor APM Agents | High licensing costs and vendor lock-in. By adopting OpenTelemetry and Prometheus standards, the engine can export telemetry data to any modern backend (Grafana, Datadog, Honeycomb) via configuration. |

## Why This Choice
Standardizing on OpenTelemetry and Prometheus ensures vendor-neutral, lightweight, and unified observability. Correlating logs to traces via `trace_id` allows developers to view a trace and instantly pull up all logs emitted during that exact request execution path.

## Tradeoffs
**Gains:**
* Vendor-agnostic tracing, structured logging, and system metrics.
* Immediate correlation of log outputs to tracing spans.
* Direct insight into critical cache and database connection pool statistics.

**Accepts:**
* Requirement to pass `context.Context` through all helper functions, repositories, and logs.

## Consequences
* Developers must use context-aware logger methods (`InfoCtx`, `ErrorCtx`) in code paths where a request context is available.
* All DB connections and Redis clients must register their respective metrics collectors at startup.
