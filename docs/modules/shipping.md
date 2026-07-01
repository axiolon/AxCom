---
title: "Shipping Module"
description: "Architecture, flows, and integration guide for the AxCom Shipping module — rate calculation, shipment creation, status tracking, and event publishing."
sidebar_label: Shipping
sidebar_position: 7
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

# Shipping Module

The Shipping module handles package rate calculation, shipping carrier integration, tracking registration, and shipment status management.

---

## Overview

The shipping subsystem allows querying rates across multiple configured shipping providers (e.g., Flat Rate, Free Above X, Weight-Based) and manages the lifecycle of order shipments. It supports:

- **Rate Calculation**: Estimating shipping costs based on weight and order value.
- **Shipment Creation**: Logging tracking information, assigning carrier details, and initiating shipping logs.
- **Status Tracking**: Transitioning shipment statuses (e.g., from pending to in-transit or delivered).
- **Event Dispatching**: Publishing `order.shipped` events when packages transition to the transit stage, enabling external modules to trigger notifications.

---

## Architecture

```mermaid
flowchart TD
  Client["Client (Web/Mobile)"] -->|Calculate Rates / Get tracking| Gateway["API Gateway / Router"]
  Gateway -->|Auth / Route| Ctrl[Shipping Controller]
  Gateway -->|Admin Auth / Route| AdminCtrl[Admin Shipping Controller]
  Ctrl --> Service[Shipping Service]
  AdminCtrl --> Service

  Service --> Repo[(Shipping DB Repository)]
  Service --> Providers[Shipping Providers List]

  Providers --> FlatRate[Flat Rate Provider]
  Providers --> FreeAbove[Free Above X Provider]
  Providers --> WeightBased[Weight-Based Provider]

  Service -->|Publish Events| EventBus["Event Bus / PubSub"]
```

---

## API Routes

### Customer Endpoints

| Method | Route                              | Description                                                | Auth           |
| :----- | :--------------------------------- | :--------------------------------------------------------- | :------------- |
| `POST` | `/shipping/rates`                  | Calculates delivery rates across all providers             | No (Public)    |
| `GET`  | `/shipping/order/:order_id`        | Fetches shipment and tracking details for a specific order | Yes (Customer) |
| `GET`  | `/shipping/track/:tracking_number` | Public tracking lookup by tracking number                  | No (Public)    |

### Admin Endpoints

| Method | Route                 | Description                                 | Auth        |
| :----- | :-------------------- | :------------------------------------------ | :---------- |
| `GET`  | `/admin/shipping`     | Lists all shipments in the database         | Yes (Admin) |
| `POST` | `/admin/shipping`     | Creates a new shipment record for an order  | Yes (Admin) |
| `PUT`  | `/admin/shipping/:id` | Updates tracking number and shipment status | Yes (Admin) |

---

## Data Model

| Status       | Description                                              |
| :----------- | :------------------------------------------------------- |
| `pending`    | Shipment record created, package not yet dispatched      |
| `in_transit` | Package handed over to carrier; tracking number assigned |
| `delivered`  | Successfully received by the customer                    |
| `returned`   | Package returned to origin                               |

---

## Key Flows

### Calculate Shipping Rates

```mermaid
sequenceDiagram
  participant Client as Customer Client
  participant Ctrl as Shipping Controller
  participant Svc as Shipping Service
  participant Prov as Shipping Provider (Impl)

  Client->>Ctrl: POST /shipping/rates {weight, value}
  Ctrl->>Svc: CalculateRates(weight, value)

  loop Each Provider
    Svc->>Prov: CalculateRate(package)
    Prov-->>Svc: rate cost
  end

  Svc-->>Ctrl: Array of rates and provider names
  Ctrl-->>Client: 200 OK (rates list)
```

### Admin Create Shipment

```mermaid
sequenceDiagram
  participant Admin as Admin UI
  participant Ctrl as Admin Shipping Controller
  participant Svc as Shipping Service
  participant Repo as DB Repository
  participant Bus as Event Bus

  Admin->>Ctrl: POST /admin/shipping {order_id, carrier, tracking_number, weight, value}
  Ctrl->>Svc: CreateShipment(...)
  Svc->>Repo: Create(Shipment Record)

  alt tracking_number is provided (status: in_transit)
    Svc->>Bus: Publish 'order.shipped' event
  end

  Svc-->>Ctrl: Shipment Details
  Ctrl-->>Admin: 200 OK
```

### Update Shipment Status

```mermaid
sequenceDiagram
  participant Admin as Admin UI
  participant Ctrl as Admin Shipping Controller
  participant Svc as Shipping Service
  participant Repo as DB Repository
  participant Bus as Event Bus

  Admin->>Ctrl: PUT /admin/shipping/:id {status, tracking_number}
  Ctrl->>Svc: UpdateShipmentStatus(id, status, tracking)
  Svc->>Repo: GetByID(id)
  Repo-->>Svc: Shipment Record
  Svc->>Repo: Update Status & Tracking Info

  alt transition from 'pending' to 'in_transit'
    Svc->>Bus: Publish 'order.shipped' event
  end

  Svc-->>Ctrl: Updated Shipment Details
  Ctrl-->>Admin: 200 OK
```

### Track Shipment by Tracking Number

```mermaid
sequenceDiagram
  participant Client as Customer Client
  participant Ctrl as Shipping Controller
  participant Svc as Shipping Service
  participant Repo as DB Repository

  Client->>Ctrl: GET /shipping/track/:tracking_number
  Ctrl->>Svc: TrackShipment(tracking_number)
  Svc->>Repo: GetByTrackingNumber(tracking_number)
  Repo-->>Svc: Shipment Record
  Svc-->>Ctrl: Shipment (Filtered)
  Ctrl-->>Client: 200 OK (TrackingResponse DTO)
```

---

## Event Subscriptions

| Event           | Trigger                            | Consumers                                                |
| :-------------- | :--------------------------------- | :------------------------------------------------------- |
| `order.shipped` | Shipment enters `in_transit` phase | Notifications module (send tracking details to customer) |
