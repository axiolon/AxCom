---
title: "Configuration"
description: "How to configure ecom-engine: database, modules, secrets, and environment variables."
sidebar_position: 4
---

# Configuration

<DocBadge status="under-review" version="v0.1.0-alpha" />

The engine is configured through two files that serve different purposes:

- **`.env.dev`** (or `.env.prod`, `.env.stage`) — secrets and per-environment values: database connection string, JWT secret, API keys. Copy `.env.example` to create one.
- **`config.yaml`** — structural settings: which modules are on or off, shipping rates, cache type, connection pool sizing. The same across environments and safe to version control (no secrets here). Copy `config.example.yaml` to create one.

```bash
cp .env.example .env.dev
cp config.example.yaml config.yaml
```

The engine has built-in defaults for every setting — you only need to change what differs.

---

## How Configuration Loads

Settings are resolved in three stages, each layer overriding the previous:

1. **Built-in defaults** — the engine starts with development-friendly values for every setting.
2. **`config.yaml`** — any key you set here overrides the default. Keys you omit keep their built-in values.
3. **Environment variables** — always win. On startup the server loads `.env.<APP_ENV>` (defaulting to `.env.dev` if `APP_ENV` is not set or is empty) and applies those on top. Use env vars for all secrets — never put credentials in `config.yaml`.

To use a config file from a custom path, set `APP_CONFIG=/path/to/config.yaml`. To see every available setting with comments, open `config.example.yaml`.

### Setting the Environment (`APP_ENV`)

You specify the active environment by setting the `APP_ENV` environment variable before starting the application. 

* **PowerShell (Windows default):**
  ```powershell
  $env:APP_ENV="prod"
  go run ./cmd/server
  ```
* **Windows Command Prompt (cmd):**
  ```cmd
  set APP_ENV=prod
  go run ./cmd/server
  ```
* **Bash / Linux / macOS:**
  ```bash
  APP_ENV=prod go run ./cmd/server
  ```

If no `APP_ENV` is specified, it defaults to `dev` (which loads `.env.dev`).

---

## In Docker Deployments

When running via Docker Compose, the same two-file system applies but the files come from different locations:

- **`.env.dev`** at `ecom-backend/` — Docker Compose reads this via `env_file` and injects the values as container environment variables. The file does not need to be inside the container.
- **`deployments/<stack>/config.yaml`** — each deployment scenario has its own `config.yaml` that is mounted into the container at `/app/config.yaml`. When running with Docker Compose, you edit the file in the deployment folder, not the root one.

The `environment:` block in each `docker-compose.yml` adds deployment-specific overrides on top — for example, changing the database hostname from `localhost` to `db` (the Docker service name on the shared `ecom-net` network).

For full setup instructions covering Docker Compose, bare-metal + PostgreSQL, and managed database scenarios, see [Ops & Deploy](../ops-deploy/overview.md).

---

## Database

Set your database in `config.yaml`. For local bare-metal development, use `localhost`. For Docker Compose, use the service name (see the deployment-specific `config.yaml` in `deployments/<stack>/`).

**PostgreSQL** (recommended for production):

```yaml
db:
  type: postgres
  connection_string: "postgres://user:password@localhost:5432/ecom_db?sslmode=disable"
  database: ecom_db
```

**MongoDB** (for teams on a MERN stack):

```yaml
db:
  type: mongodb
  connection_string: "mongodb://localhost:27017?replicaSet=rs0&directConnection=true"
  database: ecom_db
```

In production, set `DB_CONNECTION_STRING` as an environment variable instead of putting credentials in `config.yaml`.

---

## Secret Key

Used to sign JWT tokens. Set this in `.env.dev` (or via `JWT_SECRET` env var in production):

```bash
JWT_SECRET=a-long-random-string
```

The engine will refuse to start with an empty secret.

---

## Modules

All modules are **enabled by default**. To turn one off, set `enabled: false` in `config.yaml`:

```yaml
modules:
  cart:
    enabled: false
  orders:
    enabled: false
```

A disabled module returns `503` on all its routes — it is not removed from the binary.

Some modules have additional settings beyond the on/off toggle:

**Catalog** — optional feature flags:

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
```

**Payments** — provider and API key:

```yaml
modules:
  payments:
    enabled: true
    provider: stripe    # "stripe" | "paypal" | "payhere"
    api_key: ""         # use PAYMENT_API_KEY env var in production
```

**Shipping** — rate calculation providers:

```yaml
modules:
  shipping:
    enabled: true
    providers:
      - type: flatrate
        rate: 5.0
      - type: freeabove
        threshold: 50.0
      - type: weightbased
        base_rate: 2.0
        per_kg: 1.5
```

---

## Cache

For local development, the in-memory cache works without any external services:

```yaml
cache:
  type: memory
```

For production, switch to Redis:

```yaml
cache:
  type: redis
  addr: "localhost:6379"
  password: ""
```

---

## Environment Variable Reference

Environment variables always win over `config.yaml`. Set these in `.env.<APP_ENV>` for local use or as system env vars for production and cloud deployments.

| Environment Variable | What it sets | Default |
|---|---|---|
| `PORT` | HTTP listen port | `8080` |
| `JWT_SECRET` | JWT signing secret | *(must set in production)* |
| `DB_TYPE` | `postgres` or `mongodb` | `mongodb` |
| `DB_CONNECTION_STRING` | Full database DSN | localhost dev string |
| `DB_DATABASE` | Database name | `ecom_db` |
| `CACHE_TYPE` | `redis` or `memory` | `memory` |
| `CACHE_ADDR` | Redis address | `localhost:6379` |
| `CACHE_PASSWORD` | Redis password | *(empty)* |
| `STORAGE_PROVIDER` | `local`, `s3`, or `r2` | `local` |
| `PAYMENT_PROVIDER` | `stripe`, `paypal`, or `payhere` | `stripe` |
| `PAYMENT_API_KEY` | Payment provider API key | *(empty)* |
| `EVENT_PROVIDER` | `local`, `kafka`, or `rabbitmq` | `local` |
| `AUTH_MODE` | `local` (JWT) or `oidc` | `local` |
| `METRICS_ADDR` | Prometheus metrics listen address | `:9090` |
| `APP_ENV` | Environment name (`production`, `staging`, `development`, `test`) | `development` |
| `GIN_MODE` | Gin mode override (`release`, `debug`, or `test`) | *(dynamic based on APP_ENV)* |

For the full list of env vars, see `overlayEnv()` in `internal/engine/config.go`.

---

For adding configuration when building a new custom module, see [Adding a Module](../engine/adding-a-module.md#3-add-configuration).
