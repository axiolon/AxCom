---
title: "Why AxCom?"
description: "Introduction to AxCom - a compiled, modular commerce engine built with Go and Gin."
sidebar_position: 1
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

:::caution Alpha — Not Production-Ready
AxCom is in early alpha (`v0.1.0-alpha`). Modules are at different stages of completeness and third-party integrations (payment gateways, shipping providers) have not been tested against live services. See **[Project Status](./status.md)** for a detailed breakdown of what works and what does not.
:::

## What is AxCom?

AxCom (ecom-engine) is a modular, compiled commerce backend written in pure Go. It ships as a single static binary with pre-built domain modules - catalog, inventory, cart, orders, payments, shipping, and more - that you enable, disable, or swap through configuration alone.

The engine uses [Gin](https://github.com/gin-gonic/gin) for HTTP routing, [pgx](https://github.com/jackc/pgx) for PostgreSQL, the official [MongoDB Go driver](https://github.com/mongodb/mongo-go-driver) for MongoDB, and standard library patterns everywhere else. No ORMs, no heavy frameworks.

---

## Why Go for Commerce?

Standard commerce platforms rely on interpreted runtimes (Node.js, PHP) and large framework abstractions. Under peak traffic - flash sales, limited drops - these hit garbage-collection pauses, slow queries, and lock contention.

Go compiles to a native binary with a small memory footprint and predictable latency. The engine starts in milliseconds, handles high concurrency natively via goroutines, and runs anywhere - Docker, bare metal, or a single VPS.

---

## Core Design Principles

- **Strict domain isolation** - Each module owns its business logic with zero cross-module imports. Modules communicate through the engine's dependency injection container.
- **Swappable modules** - Enable or disable any module in `config.yaml`. Disabled modules return clear `503` responses on their routes instead of breaking the system.
- **Database flexibility** - PostgreSQL is the recommended database for production ecommerce (ACID transactions, relational integrity). MongoDB is fully supported as an alternative for teams already on a MERN stack.
- **Minimal complexity** - No auto-magic. Configuration loads in three explicit stages (defaults, YAML, environment variables). Dependencies are declared and validated at startup.

---

## Documentation Guide

These docs are organized into sections that cover different aspects of the engine. Here is a quick map to help you find what you need:

| Section                                             | What it covers                                                                                    |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| **[Project Status](./status.md)**                   | What works, what is untested, and what is planned — check here first                              |
| **[Getting Started](./install.md)**                 | Local setup, configuration, migrations, and adding modules                                        |
| **[Engine](../engine/overview.md)**                 | How the module system works - lifecycle, dependency injection, dependency graph, repository layer |
| **[Modules](../modules/overview.md)**               | Per-module docs - catalog, inventory, cart, orders, payments, shipping                            |
| **[Gateway](../gateway/gateway.md)**                | HTTP routing, authentication, CORS, and rate limiting                                             |
| **[Infrastructure](../infrastructure/database.md)** | Database, cache, event bus, and file storage internals                                            |
| **[Ops & Deploy](../ops-deploy/overview.md)**       | Docker Compose scenarios from single-VPS to full-stack with monitoring                            |
| **[Observability](../observability/overview.md)**   | Metrics, traces, logs, dashboards, and alerts                                                     |
| **[Testing](../testing/overview.md)**               | Unit tests, E2E tests, and load testing                                                           |
| **[Packages](../pkg/overview.md)**                  | Shared utility packages - errors, validation, ID generation, logging                              |

Start with [Quick Setup](./install.md) to get a local instance running, then explore the sections relevant to your work.
