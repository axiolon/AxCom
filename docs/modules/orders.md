---
title: "Orders Module"
description: "Architecture, flows, and integration guide for the AxCom Orders module — order creation, state machine, guest checkout, and event publishing."
sidebar_label: Orders
sidebar_position: 5
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

# Orders Module

The Orders module implements the `orders` domain for AxCom. It contains the business logic, HTTP controllers, state machine, repository contract, and guest checkout support for order processing.

---

## Capabilities

- Creates and validates orders for authenticated users and guests.
- Manages order lifecycle transitions using a state machine.
- Exposes customer-facing and admin-facing HTTP endpoints.
- Persists orders via the `OrderRepository` interface.
- Publishes events for order creation, payment success, and order status changes.

---

## Module Structure

| File/Dir          | Role                                                                      |
| :---------------- | :------------------------------------------------------------------------ |
| `service.go`      | Core order business service — creation, transitions, retrieval, listing   |
| `repository.go`   | Repository contract for saving, retrieving, and listing orders            |
| `model.go`        | Re-exports domain types: `Order`, `OrderItem`, `OrderStatus`, `GuestInfo` |
| `statemachine.go` | Order state machine adapter                                               |
| `errors.go`       | Order-specific error definitions                                          |
| `admin/`          | Admin HTTP handlers and DTOs                                              |
| `user/`           | Authenticated customer order handlers and DTOs                            |
| `guest/`          | Guest checkout handler, request/response models, and repository           |
| `domain/`         | Order domain model, validation, and state machine logic                   |

---

## Key Components

### `service.go`

- Implements order creation, status transitions, retrieval, and listing.
- Uses `OrderRepository` to persist and retrieve orders.
- Uses `OrderStateMachine` to validate state transitions.
- Subscribes to payment success events and transitions orders to `paid` when payments complete.

### `statemachine.go`

Delegates transition logic to the domain implementation:

| From               | To         |
| :----------------- | :--------- |
| `pending`          | `paid`     |
| `paid`             | `shipped`  |
| `shipped`          | `done`     |
| `pending` / `paid` | `canceled` |

---

## API Routes

### Customer Endpoints (`user/`)

| Method | Route         | Description                                    |
| :----- | :------------ | :--------------------------------------------- |
| `POST` | `/orders`     | Create an authenticated order                  |
| `GET`  | `/orders/:id` | Get a specific order (own orders only)         |
| `GET`  | `/orders`     | List all orders for the authenticated customer |

### Guest Endpoints (`guest/`)

| Method | Route           | Description                                 |
| :----- | :-------------- | :------------------------------------------ |
| `POST` | `/orders/guest` | Create a guest order without authentication |

### Admin Endpoints (`admin/`)

| Method | Route                          | Description                        |
| :----- | :----------------------------- | :--------------------------------- |
| `GET`  | `/admin/orders`                | List all orders                    |
| `GET`  | `/admin/orders/:id`            | Get any order by ID                |
| `POST` | `/admin/orders/:id/transition` | Trigger an order status transition |

---

## Order Status Lifecycle

```
pending → paid → shipped → done
pending → canceled
paid    → canceled
```

Status constants exposed by the module:

- `StatusPending`
- `StatusPaid`
- `StatusShipped`
- `StatusDone`
- `StatusCanceled`

These are aliases of the internal domain constants defined in `internal/core/orders/domain`.

---

## Key Flows

### Authenticated Order Creation

```mermaid
sequenceDiagram
  participant User as Customer
  participant API as API Gateway
  participant UserCtrl as orders/user controller
  participant Service as orders.Service
  participant Repo as OrderRepository
  participant Events as EventBus

  User->>API: POST /orders
  API->>UserCtrl: create order request
  UserCtrl->>Service: CreateOrder(customerID, items)
  Service->>Repo: Create(order)
  Repo-->>Service: order saved
  Service->>Events: publish OrderCreated
  Service-->>UserCtrl: order created
  UserCtrl-->>API: 200 OK
  API-->>User: order response
```

### Guest Order Creation

```mermaid
sequenceDiagram
  participant Guest as Guest Customer
  participant API as API Gateway
  participant GuestCtrl as orders/guest controller
  participant Service as orders.Service
  participant Repo as OrderRepository
  participant GuestRepo as GuestCustomerRepository

  Guest->>API: POST /orders/guest
  API->>GuestCtrl: create guest order request
  GuestCtrl->>Service: CreateOrder("", items)
  Service->>Repo: Create(order)
  Service-->>GuestCtrl: order created
  GuestCtrl->>GuestRepo: Save(orderID, guestInfo)
  GuestRepo-->>GuestCtrl: guest info saved
  GuestCtrl-->>API: 200 OK
  API-->>Guest: guest order response
```

### Admin Order Transition

```mermaid
sequenceDiagram
  participant Admin as Admin UI
  participant API as API Gateway
  participant AdminCtrl as orders/admin controller
  participant Service as orders.Service
  participant Repo as OrderRepository

  Admin->>API: POST /admin/orders/:id/transition
  API->>AdminCtrl: transition request
  AdminCtrl->>Service: TransitionOrder(id, action)
  Service->>Repo: GetByID(id)
  Repo-->>Service: order loaded
  Service->>StateMachine: Transition(order.Status, action)
  StateMachine-->>Service: next status
  Service->>Repo: Update(order)
  Repo-->>Service: updated
  Service-->>AdminCtrl: order transitioned
  AdminCtrl-->>API: 200 OK
  API-->>Admin: transition response
```

---

## Integrating the Orders Module

1. Implement `orders.OrderRepository` for the chosen database.
2. Register the order service with the event bus.
3. Create `admin`, `user`, and `guest` controllers with the service.
4. Mount the controllers in the router under `/admin/orders`, `/orders`, and `/orders/guest`.

---

## Notes

- The service emits events for created orders and payment success.
- Guest checkout stores contact data separately from the main order record.
- Order lifecycle rules are enforced by the state machine; invalid transitions return errors.
- Admin controllers can resolve guest details using `GuestInfoProvider`.
