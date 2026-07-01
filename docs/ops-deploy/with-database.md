---
title: "With Database"
description: "Scenarios 1–4 — run the app and database together on the same host, with optional RabbitMQ and Redis."
sidebar_position: 4
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

# With Database

Scenarios 1–4 run the app and database together on the same host. Choose MongoDB or PostgreSQL, then optionally add RabbitMQ and Redis for async messaging and distributed caching.

---

## Scenario 1 — App + MongoDB

A single VPS running both the app and a MongoDB replica set.

### Architecture

```mermaid
graph TD
    Internet([Internet]) --> App["App :8080"]
    App --> MongoDB["MongoDB :27017\n(replica set rs0)"]
```

### Start

```bash
cd ecom-backend/deployments/mongodb
docker compose up -d
```

The compose file starts three containers:

- `app` — the AxCom API
- `db` — MongoDB with a single-member replica set (`rs0`)
- `db-init` — one-shot init job that calls `rs.initiate()` if not already done

App is available at `http://localhost:8080`.

### Why a replica set?

MongoDB transactions (used by the orders and payments modules) require replica set mode even on a single node. The `db-init` service handles this automatically.

---

## Scenario 2 — App + MongoDB + RabbitMQ + Redis

Add RabbitMQ for durable async event delivery and Redis as a distributed L2 cache. Required when running multiple app replicas or when you want events to survive app restarts.

### Architecture

```mermaid
graph TD
    Internet([Internet]) --> App["App :8080"]
    App --> MongoDB["MongoDB :27017"]
    App --> RMQ["RabbitMQ :5672"]
    App --> Redis["Redis :6379"]
```

### Start

```bash
# 1. Start the shared infra first
cd ecom-backend/deployments/rabbitmq-redis
docker compose up -d

# 2. Switch the app to full config (RabbitMQ events + Redis cache)
cd ../mongodb
# Edit docker-compose.yml — change the volume mount:
#   - ./config.yaml:/app/config.yaml:ro
# to:
#   - ./config.full.yaml:/app/config.yaml:ro
docker compose up -d
```

### RabbitMQ Management UI

Available at `http://localhost:15672` (default credentials: `guest` / `guest`).

---

## Scenario 3 — App + PostgreSQL

A single VPS running both the app and PostgreSQL.

### Architecture

```mermaid
graph TD
    Internet([Internet]) --> App["App :8080"]
    App --> PG["PostgreSQL :5432"]
```

### Start

```bash
cd ecom-backend/deployments/postgres
docker compose up -d
```

The compose file starts three containers:

- `app` — the AxCom API
- `migrate` — one-shot job that runs `migrate up` to apply all schema migrations
- `db` — PostgreSQL 16

App is available at `http://localhost:8080`.

### Running Migrations Manually

The `migrate` service runs automatically on first `docker compose up`. To run it again (e.g., after upgrading):

```bash
docker compose run --rm migrate
```

---

## Scenario 4 — App + PostgreSQL + RabbitMQ + Redis

Same as Scenario 2 but with PostgreSQL.

### Architecture

```mermaid
graph TD
    Internet([Internet]) --> App["App :8080"]
    App --> PG["PostgreSQL :5432"]
    App --> RMQ["RabbitMQ :5672"]
    App --> Redis["Redis :6379"]
```

### Start

```bash
# 1. Start the shared infra first
cd ecom-backend/deployments/rabbitmq-redis
docker compose up -d

# 2. Switch the app to full config
cd ../postgres
# Edit docker-compose.yml — change the volume mount:
#   - ./config.yaml:/app/config.yaml:ro
# to:
#   - ./config.full.yaml:/app/config.yaml:ro
docker compose up -d
```

---

## Config Files Explained

Each database deployment ships two app config files:

| File               | Events     | Cache  | Use when                       |
| ------------------ | ---------- | ------ | ------------------------------ |
| `config.yaml`      | `local`    | memory | Scenarios 1 and 3 (DB only)    |
| `config.full.yaml` | `rabbitmq` | redis  | Scenarios 2 and 4 (with infra) |

The difference in `config.full.yaml`:

```yaml
cache:
  type: redis
  addr: "redis:6379"

events:
  provider: rabbitmq
  rabbitmq:
    url: "amqp://guest:guest@rabbitmq:5672/"
    exchange_name: ecom_events
```

Switch configs by editing the volume mount in `docker-compose.yml`:

```yaml
volumes:
  - ./config.full.yaml:/app/config.yaml:ro # was config.yaml
```

---

## Common Operations

```bash
# View logs for app container
docker compose logs -f app

# View MongoDB / PostgreSQL logs
docker compose logs -f db

# Connect to MongoDB shell
docker compose exec db mongosh

# Connect to PostgreSQL
docker compose exec db psql -U postgres -d ecom_db

# Rebuild after code changes
docker compose up -d --build

# Stop stack (keeps volumes)
docker compose down

# Stop and delete all data
docker compose down -v
```

---

## Next Step

To add observability to any of these scenarios, see [Monitoring](./monitoring.md).
