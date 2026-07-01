---
title: "App Only (External Database)"
description: "Scenario 0 — run the app on a single VPS and connect to an external managed database. The cheapest production-ready option."
sidebar_position: 3
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

# App Only (External Database)

**Scenario 0** — The cheapest production-ready option. The app runs in Docker on your VPS; the database is provided by an external managed service. No local database container to maintain.

---

## When to Use This

- You want the lowest possible VPS footprint (1 vCPU / 512 MB RAM is enough for the app alone)
- You are using a managed database service that handles backups, scaling, and failover
- You want to get something live quickly without setting up and maintaining a database

### Managed database options

| Database   | Free-tier managed service     |
|------------|-------------------------------|
| MongoDB    | MongoDB Atlas (512 MB free)   |
| PostgreSQL | Supabase (500 MB free), Neon, Railway |

---

## Architecture

```mermaid
graph LR
    Internet([Internet]) --> App["App :8080"]
    App -->|DB connection string| ExtDB["External DB\n(Atlas / Supabase / etc.)"]
```

The app uses:
- **In-memory event bus** — domain events are processed synchronously in-process
- **Memory cache (L1 only)** — no Redis; cache is per-container and cleared on restart
- **File storage** — configured separately via `storage` in `config.yaml`; defaults to `local` (stored in the container). S3-compatible storage (R2, AWS S3) is recommended for production — see [Config Reference](./config-yaml.md#storage).

---

## Setup

### 1. Prerequisites

Complete [Prerequisites](./prerequisites.md) first (Docker, `ecom-net` network, `.env.dev`).

### 2. Get your database connection string

**MongoDB Atlas:**

1. Create a free cluster at [cloud.mongodb.com](https://cloud.mongodb.com)
2. Under **Database → Connect → Drivers**, copy the connection string
3. Replace `<password>` with your database user's password
4. It will look like: `mongodb+srv://user:pass@cluster0.abc123.mongodb.net/ecom_db?retryWrites=true&w=majority`

**Supabase (PostgreSQL):**

1. Create a project at [supabase.com](https://supabase.com)
2. Go to **Project Settings → Database → Connection string → URI**
3. It will look like: `postgresql://postgres:pass@db.abc123.supabase.co:5432/postgres`

### 3. Configure environment

Add to `ecom-backend/.env.dev`:

```env
DB_TYPE=mongodb                  # or postgres
DB_CONNECTION_STRING=mongodb+srv://user:pass@cluster0.abc123.mongodb.net/ecom_db
```

### 4. Start

```bash
cd ecom-backend/deployments/app-only
docker compose up -d
```

The app is available at `http://localhost:8080`.

---

## PostgreSQL: Run Migrations

When using PostgreSQL, run the migration runner once after the first start (or any time you upgrade):

```bash
docker compose run --rm app ./migrate up
```

> MongoDB does not require migrations — schemas are created on first write.

---

## Adding Monitoring Later

If you later want to add Grafana Cloud monitoring (the cheapest observability option), see [Monitoring](./monitoring.md#scenario-6--grafana-cloud).

Set these in your `.env.dev` before restarting the app:

```env
OTEL_ENABLED=true
OTEL_EXPORTER=otlp
OTEL_EXPORTER_OTLP_ENDPOINT=http://otelcol:4318
```

---

## Common Operations

```bash
# View logs
docker compose logs -f app

# Restart app (e.g., after config change)
docker compose restart app

# Rebuild after code changes
docker compose up -d --build

# Stop
docker compose down
```

---

## Limitations

| Feature | Status |
|---------|--------|
| In-memory events | Async events are processed in-process; no retry queue survives restart |
| Memory cache | Cache is cleared on each restart |
| Single instance | Horizontal scaling requires switching to Redis + RabbitMQ (scenario 2 or 4) |
| File storage | Defaults to local filesystem; configure S3/R2 for durable file storage |

These are acceptable trade-offs for a low-traffic store. When you outgrow them, switch to [Scenario 2 or 4](./with-database.md).
