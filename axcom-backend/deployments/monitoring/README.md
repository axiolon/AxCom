8# ecom-engine Monitoring Stack

Full-stack observability for AxCom using the OpenTelemetry-native stack.

```
App (prometheus/client_golang + OTel SDK)
        │
        ├── /metrics  ──────────────────────────────► Prometheus
        │
        └── OTLP (gRPC :4317 / HTTP :4318) ────────► OTel Collector
                                                              │
                                                 ┌────────────┼────────────┐
                                                 ▼            ▼            ▼
                                               Loki         Tempo    Prometheus
                                               (logs)      (traces)  (remote_write)
                                                 └────────────┼────────────┘
                                                              ▼
                                                           Grafana
```

Two deployment options share the same OTel Collector pipeline — only the exporter config differs:

| Option        | Stack                                         | File                                              |
| ------------- | --------------------------------------------- | ------------------------------------------------- |
| Self-hosted   | OTelCol + Prometheus + Loki + Tempo + Grafana | `docker-compose.yml`                              |
| Grafana Cloud | OTelCol → Grafana Cloud OTLP endpoint         | `docker-compose.yml` + `docker-compose.cloud.yml` |

## Prerequisites

- Docker with the `ecom-net` network: `docker network create ecom-net`
- `.env` file: `cp .env.example .env` then edit as needed

## Self-hosted (Stack 1)

```bash
docker compose up -d
```

| Service    | Default URL           |
| ---------- | --------------------- |
| Grafana    | http://localhost:3000 |
| Prometheus | http://localhost:9090 |
| Loki       | http://localhost:3100 |
| Tempo      | http://localhost:3200 |

## Grafana Cloud (Stack 2)

Add to your `.env`:

```env
GRAFANA_CLOUD_OTLP_ENDPOINT=https://otlp-gateway-prod-us-east-0.grafana.net/otlp
GRAFANA_CLOUD_INSTANCE_ID=<your numeric stack ID>
GRAFANA_CLOUD_API_KEY=<API key with MetricsPublisher + LogsPublisher + TracesPublisher>
```

Then start with the cloud override:

```bash
docker compose -f docker-compose.yml -f docker-compose.cloud.yml up -d
```

This starts only the OTel Collector. All backends are provided by Grafana Cloud.

Find your endpoint and instance ID: **Grafana Cloud → Your Stack → OpenTelemetry tile → Configure**.

## Structure

```
monitoring/
├── .env.example
├── docker-compose.yml          # self-hosted stack
├── docker-compose.cloud.yml    # Grafana Cloud override
│
├── otelcol/
│   ├── otelcol.yml             # self-hosted: logs → Loki, traces → Tempo, metrics → Prometheus
│   └── otelcol-cloud.yml       # cloud: all signals → Grafana Cloud OTLP
│
├── loki/
│   └── loki.yml                # single-process Loki (replaces Elasticsearch)
│
├── tempo/
│   └── tempo.yml               # single-binary Tempo (replaces Jaeger)
│
├── prometheus/
│   ├── prometheus.yml          # scrapes app:8080/metrics
│   └── rules/
│       ├── recording-rules.yml # pre-computed rates (faster dashboard loads)
│       └── alerting-rules.yml  # Prometheus-native metric alerts
│
└── grafana/
    ├── provisioning/
    │   ├── dashboards/
    │   │   └── dashboards.yml          # auto-loads dashboards/ on startup
    │   ├── datasources/
    │   │   └── datasources.yml         # Prometheus + Loki + Tempo
    │   └── alerting/
    │       ├── api-alerts.yml          # HTTP error rate, latency, error log spike (Loki)
    │       ├── database-alerts.yml     # DB pool alerts (Prometheus)
    │       ├── cache-alerts.yml        # Redis hit rate, pool timeouts (Prometheus)
    │       ├── business-alerts.yml     # Payment errors, auth failures (Loki)
    │       └── runtime-alerts.yml      # CPU, memory, goroutines, GC (Prometheus)
    │
    └── dashboards/
        └── ecom-engine/
            ├── service-health.json     # incident entry point — top-line stats
            ├── http-traffic.json       # request rates, latency, status codes
            ├── database.json           # DB connection pool
            ├── cache.json              # L1 memory + L2 Redis cache
            ├── business-events.json    # orders, payments, cart, catalog
            ├── security.json           # auth events + rate limiting
            ├── logs.json               # live log stream (update to Loki datasource)
            └── runtime.json            # Go runtime metrics
```

## Signal Routing

### Metrics

Prometheus scrapes `app:8080/metrics` directly every 15s. The existing `prometheus/client_golang` metrics work with no app changes.

When the Go app adds the OTel SDK, SDK metrics can be pushed via OTLP to the collector → Prometheus remote_write (the `--web.enable-remote-write-receiver` flag is already set).

### Logs

The OTel Collector's `filelog` receiver tails Docker JSON log files from `/var/lib/docker/containers/**/*-json.log`. It:

1. Parses the Docker JSON envelope
2. Parses the inner structured JSON body (app structured logs)
3. Extracts `service.name` from the Docker Compose service label
4. Exports to Loki with `service_name` and `level` as stream labels

Query logs in Grafana with LogQL:

```logql
{service_name="ecom-engine"} | json | level="error"
```

### Traces

The OTel Collector's `otlp` receiver accepts traces on port 4317 (gRPC) or 4318 (HTTP) and forwards to Tempo. Configure the Go app's OTel SDK:

```env
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_SERVICE_NAME=ecom-engine
```

Traces are linked to logs (via `trace_id` field) and to metrics (via Tempo's span-metrics generator → Prometheus).

## Dashboards

Start with **Service Health** during an incident. Drill into focused dashboards from the dropdown.

| Dashboard       | Datasource        | When to use                                  |
| --------------- | ----------------- | -------------------------------------------- |
| Service Health  | Prometheus        | On-call first look, SLO monitoring           |
| HTTP Traffic    | Prometheus        | Request rate / latency / 5xx investigation   |
| Database        | Prometheus        | DB pool saturation, slow acquire times       |
| Cache           | Prometheus        | Cache hit rate drops, Redis pool issues      |
| Business Events | Prometheus + Loki | Order/payment drops, cart abandonment        |
| Security        | Prometheus + Loki | Auth failures, rate limiting health          |
| Logs            | Loki              | Ad-hoc log search with level/trace filtering |
| Runtime         | Prometheus        | Go heap, GC, goroutines, CPU/memory          |

> **Note:** `logs.json` and `business-events.json` were originally built for Elasticsearch.
> Open them in Grafana, switch the datasource to **Loki**, update queries to LogQL, then export and replace.

## Alerts

**Prometheus alerts** (`prometheus/rules/alerting-rules.yml`): metric-based, fire even if Grafana is down.
**Grafana alerts** (`grafana/provisioning/alerting/`): use both Prometheus and Loki for cross-signal conditions.

| Alert                      | Source     | Condition                        |
| -------------------------- | ---------- | -------------------------------- |
| `HighHttpErrorRate`        | Prometheus | 5xx rate > 5% for 5m             |
| `NoHttpTraffic`            | Prometheus | zero requests for 5m             |
| `HighP99Latency`           | Prometheus | p99 > 2s for 5m                  |
| `DbPoolExhausted`          | Prometheus | pool utilization > 90% for 2m    |
| `DbPoolEmptyAcquires`      | Prometheus | empty acquires > 0.5/s for 3m    |
| `LowCacheHitRate`          | Prometheus | Redis hit rate < 50% for 10m     |
| `RedisPoolTimeouts`        | Prometheus | pool timeouts > 0.1/s for 5m     |
| `RateLimitBackendFallback` | Prometheus | Redis → memory fallback detected |
| `HighCpuUsage`             | Prometheus | CPU > 80% for 5m                 |
| `HighMemoryRSS`            | Prometheus | RSS > 1 GiB for 10m              |
| `GoroutineLeak`            | Prometheus | goroutines > 500 for 15m         |
| `ErrorLogSpike`            | Loki       | > 50 error logs in 5m            |
| `PaymentErrorSpike`        | Loki       | > 10 payment error logs in 5m    |
| `AuthFailureSpike`         | Loki       | > 30 auth failure logs in 5m     |

## Dashboard Authoring

Dashboards are provisioned from Git — **do not edit in the Grafana UI**. `allowUiUpdates: false` prevents drift.

1. Set `allowUiUpdates: true` in `grafana/provisioning/dashboards/dashboards.yml` locally
2. Edit in Grafana UI
3. Export JSON (`Share → Export → Save to file`)
4. Replace the file in `grafana/dashboards/ecom-engine/`
5. Revert `allowUiUpdates` to `false`
6. Commit

## Prometheus Metrics Exposed by ecom-engine

| Metric                                          | Type          | Labels                   |
| ----------------------------------------------- | ------------- | ------------------------ |
| `ecom_engine_http_requests_total`               | Counter       | method, route, status    |
| `ecom_engine_http_request_duration_seconds`     | Histogram     | method, route            |
| `ecom_engine_http_requests_in_flight`           | Gauge         | —                        |
| `ecom_engine_db_pool_*`                         | Gauge/Counter | —                        |
| `ecom_engine_cache_requests_total`              | Counter       | layer, operation, result |
| `ecom_engine_cache_operation_duration_seconds`  | Histogram     | layer, operation         |
| `ecom_engine_cache_memory_items`                | Gauge         | —                        |
| `ecom_engine_cache_memory_evictions_total`      | Counter       | reason                   |
| `ecom_engine_cache_redis_pool_*`                | Gauge/Counter | —                        |
| `ecom_engine_ratelimit_requests_total`          | Counter       | bucket, decision         |
| `ecom_engine_ratelimit_backend_active`          | Gauge         | backend                  |
| `ecom_engine_ratelimit_backend_fallbacks_total` | Counter       | —                        |
| `ecom_engine_runtime_*`                         | Gauge/Counter | —                        |
| `ecom_engine_process_cpu_percent`               | Gauge         | —                        |
| `ecom_engine_process_memory_rss_bytes`          | Gauge         | —                        |
