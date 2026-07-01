---
title: "Configuration Reference"
description: "Complete reference for all options in config.yaml — the app's runtime configuration file."
sidebar_position: 7
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

# Configuration Reference

The app reads its runtime configuration from `config.yaml`, mounted into the container at `/app/config.yaml`. Secrets (JWT key, API keys, DB passwords) are provided via environment variables and override the corresponding config file values.

Each deployment folder ships two variants:

| File              | Events     | Cache  | Use when |
|-------------------|------------|--------|----------|
| `config.yaml`     | `local`    | memory | DB-only scenarios (no RabbitMQ/Redis) |
| `config.full.yaml`| `rabbitmq` | redis  | Full infra scenarios |

---

## Top-Level

```yaml
port: "8080"
secret: "your-jwt-secret"       # overridden by JWT_SECRET env var
service_name: "ecom-engine"
max_request_size: 5242880        # bytes (default: 5 MB)
```

| Key | Type | Description |
|-----|------|-------------|
| `port` | string | HTTP server listen port |
| `secret` | string | JWT signing secret (HS256). Set via `JWT_SECRET` env var in production |
| `service_name` | string | Service name used in logs and traces |
| `max_request_size` | int | Maximum request body size in bytes |

---

## auth

```yaml
auth:
  mode: local
```

| Key | Values | Description |
|-----|--------|-------------|
| `mode` | `local` | Authentication mode. `local` uses the built-in JWT auth module |

---

## db

```yaml
db:
  type: mongodb                  # or postgres
  connection_string: "mongodb://db:27017?replicaSet=rs0&directConnection=true"
  database: ecom_db              # MongoDB only
```

| Key | Description |
|-----|-------------|
| `type` | `mongodb` or `postgres` |
| `connection_string` | Full DB connection URI. Overridden by `DB_CONNECTION_STRING` env var |
| `database` | Database name (MongoDB only; PostgreSQL uses the name in the connection string) |

---

## cache

### Memory cache (default)

```yaml
cache:
  type: memory
  l1_ttl: 5m
  l1_max_items: 10000
```

### Redis cache

```yaml
cache:
  type: redis
  addr: "redis:6379"
  password: ""
  db: 0
  pool_size: 10
  l1_ttl: 5m           # in-process L1 TTL (sits in front of Redis)
  l1_max_items: 10000
```

| Key | Description |
|-----|-------------|
| `type` | `memory` or `redis` |
| `addr` | Redis address (Redis only) |
| `password` | Redis password, empty string if none |
| `db` | Redis database index |
| `pool_size` | Redis connection pool size |
| `l1_ttl` | TTL for the in-process L1 cache layer |
| `l1_max_items` | Maximum items in the L1 cache |

---

## storage

File storage is configured independently of the database and messaging stack. S3-compatible storage is recommended for production.

```yaml
storage:
  provider: local     # or s3
  bucket: products
  region: us-east-1
```

For S3 / Cloudflare R2:

```yaml
storage:
  provider: s3
  bucket: your-bucket-name
  region: us-east-1
  endpoint: ""           # leave empty for AWS S3; set for R2 or MinIO
  access_key: ""         # set via env var in production
  secret_key: ""         # set via env var in production
```

| Key | Description |
|-----|-------------|
| `provider` | `local` (container filesystem) or `s3` (AWS S3 / R2 / MinIO) |
| `bucket` | Bucket or folder name |
| `region` | AWS/R2 region |
| `endpoint` | Custom endpoint URL for non-AWS providers (Cloudflare R2, MinIO) |
| `access_key` / `secret_key` | S3 credentials |

---

## events

### Local (in-process)

```yaml
events:
  provider: local
  retry:
    max_retries: 3
    initial_backoff: 50ms
    max_backoff: 2s
  local:
    dlq_buffer_size: 100
```

### RabbitMQ

```yaml
events:
  provider: rabbitmq
  retry:
    max_retries: 3
    initial_backoff: 50ms
    max_backoff: 2s
  rabbitmq:
    url: "amqp://guest:guest@rabbitmq:5672/"
    exchange_name: ecom_events
    exchange_type: topic
    queue_name: ecom_queue
    dlq_exchange: ecom_events_dlq
    dlq_queue: ecom_queue_dlq
```

| Key | Description |
|-----|-------------|
| `provider` | `local` or `rabbitmq` |
| `retry.max_retries` | Number of delivery retries before sending to DLQ |
| `retry.initial_backoff` | First retry delay |
| `retry.max_backoff` | Maximum retry delay (exponential backoff cap) |
| `local.dlq_buffer_size` | In-process DLQ channel buffer size |
| `rabbitmq.url` | AMQP connection URL |
| `rabbitmq.exchange_name` | Main exchange name |
| `rabbitmq.exchange_type` | Exchange type (`topic`, `direct`, `fanout`) |
| `rabbitmq.queue_name` | Primary queue |
| `rabbitmq.dlq_exchange` / `dlq_queue` | Dead-letter exchange and queue |

---

## modules

Each module can be individually enabled or disabled. Features within a module can also be toggled.

```yaml
modules:
  catalog:
    enabled: true
    features:
      images: true
      variants: true
      discounts: true
      bulk: true
      reviews: true
  inventory:
    enabled: true
    features:
      bulk: true
      history: true
      reservation: true
      reports: true
      transfer: true
      adjustment: true
      sync: true
  cart:
    enabled: true
  orders:
    enabled: true
  payments:
    enabled: true
    provider: stripe
    api_key: "sk_test_your_key_here"   # overridden by PAYMENT_API_KEY env var
  shipping:
    enabled: true
    providers:
      - type: flatrate
        rate: 5.0
      - type: freeabove
        threshold: 50.0
  notifications:
    enabled: true
```

### Shipping providers

| Type | Config | Description |
|------|--------|-------------|
| `flatrate` | `rate` | Fixed shipping cost on every order |
| `freeabove` | `threshold` | Free shipping when order subtotal exceeds the threshold |
