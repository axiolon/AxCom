---
title: "Dependency Graph"
description: "How the engine validates and orders module initialization using Kahn's topological sort algorithm."
sidebar_position: 4
---

# Dependency Graph

<DocBadge status="under-review" version="v0.1.0-alpha" />

Modules declare which other modules they depend on. The engine validates these declarations at startup and uses them to determine a safe boot order. This prevents two classes of bugs: boot-order failures (a module initializing before its dependency) and circular dependencies (which would deadlock the boot sequence entirely).

This logic lives in `depgraph.go`.

---

## Declaring Dependencies

Each module returns its required modules from `Requires()`:

```
[Catalog Module]   →  Requires: (none)
      ↓
[Cart Module]      →  Requires: "catalog"
      ↓
[Orders Module]    →  Requires: "catalog", "cart"
```

The engine reads these declarations to build a directed acyclic graph (DAG) before any module's `Init()` is called.

---

## Kahn's Algorithm

The engine sorts the module DAG topologically using **Kahn's Algorithm**, which processes nodes layer by layer starting from those with no dependencies.

### Steps

**1. Validation**
Check that every name in every module's `Requires()` list is present in the active module set. A missing or disabled dependency is a hard error - the engine exits before attempting any boot.

**2. In-Degree Calculation**
Count how many dependencies each module has. This number is the module's _in-degree_.

**3. Adjacency Mapping**
Build an adjacency list: for each module, record which other modules depend on it (its _dependents_).

**4. Seed the Queue**
Place all modules with in-degree `0` (no dependencies) into a processing queue. These are safe to initialize immediately.

**5. Topological Loop**

```
Queue: [Catalog]

Pop Catalog → sorted = [Catalog]
  Decrement in-degree of Cart (catalog removed as dep) → Cart in-degree = 0
  Decrement in-degree of Orders → Orders in-degree = 1
  Push Cart to queue

Queue: [Cart]

Pop Cart → sorted = [Catalog, Cart]
  Decrement in-degree of Orders → Orders in-degree = 0
  Push Orders to queue

Queue: [Orders]

Pop Orders → sorted = [Catalog, Cart, Orders]
```

**6. Cycle Detection**
After the loop, if `len(sorted) < len(active modules)`, a cycle exists among the remaining unsorted nodes. The engine runs a depth-first search via `findCycle()` and exits with a descriptive error:

```
circular dependency detected: cart → orders → cart
```

---

## Result

The engine receives modules in safe initialization order. Each module's dependencies are guaranteed to be fully initialized and their services registered in the container before the dependent module's `Init()` is called.
