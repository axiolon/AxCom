---
title: "Engine Overview"
description: "How the application engine boots, wires dependencies, and starts serving requests."
sidebar_position: 1
---

# Engine Overview

<DocBadge status="under-review" version="v0.1.0-alpha" />

The engine is the central orchestrator of the application. It owns the full startup pipeline: loading configuration, initializing shared infrastructure, sorting and booting modules in dependency order, mounting routes, and managing graceful shutdown.

---

## Boot Sequence

The boot sequence is orchestrated by `main.go` in a linear pipeline. Each step produces output consumed by the next - no step begins until the previous one succeeds.

```mermaid
sequenceDiagram
    autonumber
    participant Main as main.go
    participant Registry as registry.Collect()
    participant Engine as engine.NewEngine()
    participant DepGraph as depgraph.go (Kahn's Sort)
    participant Module as Module.Init()
    participant Router as gateway.NewRouter()

    Main->>Main: Load Environment (.env)
    Main->>Main: Load Configuration (YAML + Env Overlays)
    Main->>Registry: Partition modules into active/disabled
    Registry-->>Main: Return active, disabled modules
    Main->>Engine: NewEngine(cfg, active, disabled)
    Engine->>Engine: Initialize Shared Infra (DB, Cache, Event Bus, Auth)
    Engine->>Engine: Build Dependency Container
    Engine->>DepGraph: Validate and sort active modules
    DepGraph-->>Engine: Modules in topological order
    loop For each sorted Module
        Engine->>Module: Init(Container)
        Module->>Engine: Provide services to Container
    end
    Engine-->>Main: Return ready Engine instance
    Main->>Router: NewRouter(Engine)
    Router->>Router: Mount active routes & catch-alls for disabled routes
    Main->>Main: Start HTTP Server Listener
```

---

## Key Concepts

Each stage of the boot sequence corresponds to a dedicated concept. Refer to these pages for the full details:

| Concept              | Description                                                                                        | Doc                                               |
| -------------------- | -------------------------------------------------------------------------------------------------- | ------------------------------------------------- |
| **Dependency Graph** | Topological sort via Kahn's algorithm - ensures modules boot in the right order and detects cycles | [Dependency Graph](./dependency-graph.md)         |
| **DI Container**     | Shared registry of infrastructure and module services, passed to every `Init()` call               | [Dependency Injection](./dependency-injection.md) |
| **Module Lifecycle** | The `Module` interface - `Init`, `RegisterRoutes`, `Shutdown`, and the directory layout            | [Module Lifecycle](./module-lifecycle.md)         |
| **Repository Layer** | Database-agnostic `RepoProvider` that routes domain interfaces to the configured DB driver         | [Repository Layer](./repository-layer.md)         |
