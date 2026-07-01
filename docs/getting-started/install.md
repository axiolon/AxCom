---
title: "Quick Setup"
description: "Clone the repo, start a local database with Docker Compose, run migrations, and start the server."
sidebar_position: 3
---

# Quick Setup

<DocBadge status="under-review" version="v0.1.0-alpha" />

This guide runs the server locally with `go run` and a containerized database. You will need **Go 1.25+** and **Docker** (with Docker Compose) installed.

If you want to run the full application stack in Docker, or deploy to a VPS or bare-metal server with PostgreSQL, see [Ops & Deploy](../ops-deploy/overview.md) instead.

---

## 1. Clone and Configure

```bash
git clone https://github.com/axiolon-labs/ecom-engine.git
cd ecom-engine/ecom-backend

# Copy both example config files
cp config.example.yaml config.yaml
cp .env.example .env.dev
```

Open `config.yaml` and set your database type. PostgreSQL is recommended for production ecommerce workloads. MongoDB is supported if your team prefers a document store.

**PostgreSQL** (recommended):

```yaml
db:
  type: postgres
  connection_string: "postgres://postgres:secret@localhost:5432/ecom_db?sslmode=disable"
  database: ecom_db
```

**MongoDB**:

```yaml
db:
  type: mongodb
  connection_string: "mongodb://localhost:27017?replicaSet=rs0&directConnection=true"
  database: ecom_db
```

Open `.env.dev` and set at minimum your `JWT_SECRET`. The database connection string in `.env.dev` overrides the one in `config.yaml` if both are set — for local `go run`, set it in `config.yaml` and leave the env file for secrets only.

See [Configuration](./configuration.md) for how the two files work together and all available settings.

---

## 2. Start the Database

The repo includes Docker Compose files for both databases under `deployments/`.

We create a shared Docker network (`ecom-net`) first. This allows your database container and any other local containers (such as the app server itself, if dockerized later) to securely communicate with each other using their container names.

**PostgreSQL:**

```bash
docker network create ecom-net
docker compose -f deployments/postgres/docker-compose.yml up -d db
```

**MongoDB:**

```bash
docker network create ecom-net
docker compose -f deployments/mongodb/docker-compose.yml up -d db db-init
```

The MongoDB setup starts a single-node replica set (required for transactions).

---

## 3. Run Migrations (PostgreSQL only)

PostgreSQL requires schema migrations before the server can start. MongoDB is schema-free — skip this step if you are using MongoDB.

```bash
go run ./cmd/migrate up
```

**See [Migrations](./migrations.md) for all migration commands (`down`, `status`, `verify`, `seed`).**

---

## 4. Start the Server

```bash
go run ./cmd/server/main.go
```

The engine boots all enabled modules in dependency order and starts listening on port **8080** by default.

```
http://localhost:8080
```

The startup logs will show each module that was initialized and the final listening address. If a required dependency is missing or misconfigured, the engine exits immediately with a clear error.

---

## What's Next

| Topic | Link |
|---|---|
| How the two config files work and all available settings | [Configuration](./configuration.md) |
| Migration commands and adding module schemas | [Migrations](./migrations.md) |
| Creating a new module from scratch | [Adding a Module](../engine/adding-a-module.md) |
| How the engine boots and manages modules | [Engine Overview](../engine/overview.md) |
| Docker Compose, VPS, bare-metal + PostgreSQL deployments | [Ops & Deploy](../ops-deploy/overview.md) |
