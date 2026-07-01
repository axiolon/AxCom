---
title: "Modules Overview"
description: "Overview of all core business modules in AxCom — auth, cart, catalog, inventory, orders, payments, and shipping."
sidebar_label: Overview
sidebar_position: 0
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

# Modules Overview

AxCom is organized into independent core modules. Each module owns its domain — its data model, business logic, HTTP controllers, repository contract, and events. Modules communicate with each other exclusively through the event bus; they do not import each other directly.

---

## Core Modules

> For detailed development status of each module, see **[Project Status](../getting-started/status.md)**.

| Module | Status | Description | Source |
|:--|:--|:--|:--|
| [Authentication](./auth.md) | Under Review | User registration, login, JWT sessions, role-based access | `internal/core/auth` |
| [Cart](./cart.md) | Under Review | Cart management, item enrichment, guest cart merge | `internal/core/cart` |
| [Catalog](./catalog.md) | Under Review | Products, variants, images, discounts, bulk ops, reviews | `internal/core/catalog` |
| [Inventory](./inventory.md) | Under Review | Stock availability, reservations, history, adjustments | `internal/core/inventory` |
| [Orders](./orders.md) | Under Review | Order creation, lifecycle state machine, guest checkout | `internal/core/orders` |
| [Payments](./payments.md) | Under Review | Payment intents, refunds. **No gateway integration-tested.** | `internal/core/payments` |
| [Shipping](./shipping.md) | Under Review | Rate calculation, shipment creation. **No real provider tested.** | `internal/core/shipping` |

---

## Module Conventions

Every module follows the same internal structure:

```
internal/core/<module>/
├── controller.go     # HTTP handlers and request validation
├── service.go        # Business logic and Service interface
├── repository.go     # Storage port (interface)
├── model.go          # Domain models and DTOs
├── errors.go         # Sentinel error definitions
├── routes.go         # Route registration
├── README.md         # Lightweight orientation (links here)
└── tests.md          # Unit and integration test documentation
```

Larger modules (catalog, inventory) are further split into `features/` subdirectories, each with their own controller, service, and repository.

---

## Testing Convention

Each module keeps a `tests.md` file in its source directory alongside its code. This covers:

- Unit test cases and what they validate
- Integration test setup and dependencies
- How to run tests for that specific module

This keeps test documentation co-located with the code it describes, making it easy to find without leaving the module directory.

---

## Event Bus

Modules publish and subscribe to events through a shared event bus. Key cross-module events:

| Event               | Publisher | Subscribers                                          |
| :------------------ | :-------- | :--------------------------------------------------- |
| `order.created`     | Orders    | Inventory (reserve stock)                            |
| `payment.succeeded` | Payments  | Orders (mark paid), Inventory (finalize reservation) |
| `payment.failed`    | Payments  | Inventory (release reservation)                      |
| `order.shipped`     | Shipping  | Notifications (send tracking to customer)            |
