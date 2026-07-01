# ADR-008: Modular Monolith and Kahn's Dependency Resolution

**Date:** 2026-06-27  
**Status:** accepted

## Context
As the application scales, functional domains (such as Catalog, Cart, Inventory, and Orders) grow in complexity. Coupling these modules directly to each other leads to a "spaghetti" codebase, while splitting them into microservices introduces high operational complexity, distributed transaction issues, and network latency.

Furthermore, dynamic module configuration (enabling/disabling specific modules via config) means we cannot rely on static initialization order. Ad-hoc manual ordering in main:
1. Leads to boot-time bugs (e.g. resolving a dependency that hasn't been initialized).
2. Fails to detect circular dependency deadlocks automatically.

## Decision
1. **Define a Standard Module Interface:** Every functional domain must implement the `engine.Module` contract:
   - `Name() string` for identification.
   - `Requires() []string` to declare dependencies.
   - `BasePaths() []string` to declare route ownership.
   - `Init(*Container) error` for wiring logic.
   - `RegisterRoutes(...)` for mounting HTTP handlers.
   - `Shutdown(context.Context) error` for resource cleanup.

2. **Topological Sorting at Boot Time:** Use Kahn's Algorithm at startup to validate and sort the enabled modules:
   - Fail fast if a required module is missing or disabled.
   - Parse dependencies to build an in-degree map and adjacency lists.
   - Resolve the execution path from independent modules (in-degree 0) to dependents.
   - Fail boot with a detailed error trace if a circular dependency is detected.

3. **Symmetric Graceful Shutdown:** Release module-owned resources (workers, connections) in reverse-topological order (dependents first, then dependencies).

## Alternatives Considered

| Option | Reason Rejected |
|--------|-----------------|
| Manual Ordering in `main.go` | Error-prone and brittle. As developers add modules, they must manually trace the correct boot order, leading to silent wiring failures or panic loops. |
| Microservices | High operational overhead, network latency, and complexity of maintaining consistency via Sagas/Outboxes for every single state mutation. |

## Why This Choice
A modular monolith architecture combines the operational simplicity of a single binary with the logical separation of microservices. Enforcing topological ordering via Kahn's algorithm guarantees that any dependency issues or cycles are caught immediately at boot time rather than manifesting as hard-to-debug runtime panics.

## Tradeoffs
**Gains:**
* Decoupled functional domains that can be developed independently and potentially extracted into microservices later.
* Immediate fail-fast verification of boot order and dependency loop issues.
* Safe graceful shutdown, ensuring dependencies are not torn down before their consumers.

**Accepts:**
* Boilerplate code for new modules implementing the lifecycle methods.
* Startup validation overhead (negligible).

## Consequences
* All new services must be packaged as types implementing `engine.Module` and registered in `internal/modules/registry`.
* Modules cannot create circular dependencies; doing so will result in a hard failure at startup.
