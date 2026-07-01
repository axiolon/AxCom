---
title: "Prerequisites"
description: "One-time setup steps required before running any AxCom deployment stack."
sidebar_position: 2
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

# Prerequisites

Before running any deployment stack, complete the steps on this page once per machine.

---

## 1. Install Docker

You need Docker Engine with the Compose plugin (v2).

- **Linux:** [docs.docker.com/engine/install](https://docs.docker.com/engine/install/)
- **macOS / Windows:** Install [Docker Desktop](https://www.docker.com/products/docker-desktop/)

Verify:

```bash
docker --version          # Docker version 26+
docker compose version    # Docker Compose version v2+
```

> If `docker compose` (with a space) does not work but `docker-compose` does, you have Compose v1. Upgrade to v2.

---

## 2. Create the Shared Network

All stacks communicate over a single external Docker network called `ecom-net`. Create it once:

```bash
docker network create ecom-net
```

You can verify it exists at any time:

```bash
docker network ls | grep ecom-net
```

---

## 3. Configure Environment Files

AxCom uses **two separate env files** depending on which stacks you run.

### 3a. App env file (always required)

The app container reads secrets from `ecom-backend/.env.{environment}`. Create it from the example:

```bash
cd ecom-backend
cp .env.example .env.dev
```

Edit `.env.dev` and set at minimum:

| Variable          | Description                      | Example                        |
| ----------------- | -------------------------------- | ------------------------------ |
| `JWT_SECRET`      | HS256 signing key (min 32 chars) | `your-very-long-random-secret` |
| `PAYMENT_API_KEY` | Payment provider secret key      | `sk_live_...`                  |

All other variables have safe defaults for local development. See `.env.example` for the full list with comments.

#### Switching environments (`APP_ENV`)

The Compose files load the env file matching `APP_ENV` (default: `dev`):

```bash
# Loads ecom-backend/.env.dev (default)
docker compose up -d

# Loads ecom-backend/.env.prod
APP_ENV=prod docker compose up -d

# Loads ecom-backend/.env.stage
APP_ENV=stage docker compose up -d
```

Create separate files for each environment and keep secrets out of version control. Only `.env.example` should be committed.

#### What to set per scenario

Most variables have working defaults. The ones that actually differ per scenario:

| Variable                      | Scenario 0 (external DB) | Scenarios 1–4 (local DB)                   | Scenarios 5–6 (+ monitoring) |
| ----------------------------- | ------------------------ | ------------------------------------------ | ---------------------------- |
| `DB_TYPE`                     | `postgres` or `mongodb`  | Set by Compose — no change needed          | Same                         |
| `DB_CONNECTION_STRING`        | Your managed DB URL      | Set by Compose — no change needed          | Same                         |
| `OTEL_ENABLED`                | `false`                  | `false`                                    | `true`                       |
| `OTEL_EXPORTER`               | `none`                   | `none`                                     | `otlp`                       |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | —                        | —                                          | `http://otelcol:4318`        |
| `EVENT_PROVIDER`              | `local`                  | `local` (Scenario 1/3) or `rabbitmq` (2/4) | Same                         |

> **Note:** The Compose files for Scenarios 1–4 inject `DB_TYPE` and `DB_CONNECTION_STRING` directly via `environment:`, overriding whatever is in your `.env` file. You only need to set these in your env file for Scenario 0 (external managed DB).

---

### 3b. Monitoring env file (Scenario 5 and 6 only)

The monitoring stack has its own env file at `deployments/monitoring/.env`:

```bash
cd ecom-backend/deployments/monitoring
cp .env.example .env
```

Edit `.env` and change at minimum:

| Variable           | Default    | Description                                |
| ------------------ | ---------- | ------------------------------------------ |
| `GRAFANA_PASSWORD` | `changeme` | Grafana admin password — change before use |

For Scenario 6 (Grafana Cloud), also set:

| Variable                      | Description                                      |
| ----------------------------- | ------------------------------------------------ |
| `GRAFANA_CLOUD_OTLP_ENDPOINT` | Your Grafana Cloud OTLP endpoint URL             |
| `GRAFANA_CLOUD_INSTANCE_ID`   | Grafana Cloud stack instance ID                  |
| `GRAFANA_CLOUD_API_KEY`       | Grafana Cloud API key with MetricsPublisher role |

The monitoring env file controls **only** the monitoring stack (ports, Grafana credentials, Grafana Cloud keys). It is independent of the app env file.

---

## 4. Build the App Image (first run only)

The Compose files build the app image from the repo root `Dockerfile`. The first build compiles the Go binary and may take 1–2 minutes. Subsequent builds are fast due to layer caching.

```bash
cd deployments/mongodb   # or any other folder
docker compose build
```

---

## Port Reference

Ports used across all stacks (all overridable via environment variables):

| Variable             | Default | Service             |
| -------------------- | ------- | ------------------- |
| `PORT`               | 8080    | App API             |
| `MONGO_PORT`         | 27017   | MongoDB             |
| `POSTGRES_PORT`      | 5432    | PostgreSQL          |
| `RABBITMQ_PORT`      | 5672    | RabbitMQ AMQP       |
| `RABBITMQ_MGMT_PORT` | 15672   | RabbitMQ Management |
| `REDIS_PORT`         | 6379    | Redis               |
| `PROMETHEUS_PORT`    | 9090    | Prometheus          |
| `GRAFANA_PORT`       | 3000    | Grafana             |
| `LOKI_PORT`          | 3100    | Loki                |
| `TEMPO_PORT`         | 3200    | Tempo               |
| `OTEL_GRPC_PORT`     | 4317    | OTel Collector gRPC |
| `OTEL_HTTP_PORT`     | 4318    | OTel Collector HTTP |

To change a port, export the variable before running Compose:

```bash
PORT=9000 GRAFANA_PORT=3001 docker compose up -d
```

---

## Firewall / Security Notes

- Expose **only port 8080** (or your custom `PORT`) to the public internet.
- All other ports (databases, monitoring) should be bound to `127.0.0.1` or kept behind a firewall / VPN.
- In production, put the app behind a reverse proxy (nginx, Caddy, Traefik) and terminate TLS there.
- Set `OTEL_ENABLED=false` and `LOG_FORMAT=text` for local development without monitoring.
